/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"database/sql"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/strings/slices"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	postgresv1alpha1 "github.com/jeewangue/postgres-indb-operator/api/v1alpha1"
)

// PgDatabaseReconciler reconciles a PgDatabase object
type PgDatabaseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=postgres.jeewangue.com,resources=pgdatabases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=postgres.jeewangue.com,resources=pgdatabases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=postgres.jeewangue.com,resources=pgdatabases/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PgDatabase object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *PgDatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)

	requestID, err := uuid.NewRandom()
	if err != nil {
		reqLogger.Error(err, "Failed to pick a request ID. Continuing without")
	}
	reqLogger = reqLogger.WithValues("requestId", requestID.String())
	reqLogger.Info("Reconciling PgDatabase")

	result, err := r.reconcile(ctx, reqLogger, req)
	if err != nil {
		reqLogger.Error(err, "Failed to reconcile PgDatabase object")
	}

	return result, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *PgDatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&postgresv1alpha1.PgDatabase{}).
		Complete(r)
}

func (r *PgDatabaseReconciler) reconcile(ctx context.Context, reqLogger logr.Logger, request reconcile.Request) (ctrl.Result, error) {
	// Fetch the PgDatabase instance
	database := &postgresv1alpha1.PgDatabase{}
	err := r.Client.Get(ctx, request.NamespacedName, database)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("Object not found")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// PgDatabase instance created or updated
	reqLogger = reqLogger.WithValues("database", database.Spec.Name)
	reqLogger.Info("Reconciling found PgDatabase resource")

	if database.GetDeletionTimestamp() != nil {
		if !slices.Contains(database.Finalizers, operatorFinalizer) {
			return ctrl.Result{}, nil
		}
		// Run finalization logic. If the
		// finalization logic fails, don't remove the finalizer so
		// that we can retry during the next reconciliation.
		if err := r.finalize(reqLogger, database); err != nil {
			return ctrl.Result{}, err
		}

		// Remove finalizer. Once all finalizers have been
		// removed, the object will be deleted.
		controllerutil.RemoveFinalizer(database, operatorFinalizer)
		err := r.Update(ctx, database)
		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !slices.Contains(database.Finalizers, operatorFinalizer) {
		controllerutil.AddFinalizer(database, operatorFinalizer)

		err = r.Update(ctx, database)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// Connect to database
	connStr := "postgresql://postgres:password@127.0.0.1:5432/postgres?sslmode=disable"
	db, err := pgx.Connect(context.Background(), connStr)
	// db, err := sql.Open("postgres", connStr)
	defer db.Close(context.Background())
	if err != nil {
		reqLogger.Error(err, "Failed to open database connection")
		return ctrl.Result{}, err
	}

	userExists := false
	var usename string
	var usesysid uint32
	err = db.QueryRow(context.Background(), "SELECT usename, usesysid FROM pg_user WHERE usename = $1", database.Spec.Name).Scan(&usename, &usesysid)
	switch {
	case err == pgx.ErrNoRows:
		userExists = false
		reqLogger.Info(fmt.Sprintf("No user with name %s. Creating...", database.Spec.Name))
	case err != nil:
		reqLogger.Error(err, "Failed to query from pg_user")
		return ctrl.Result{}, nil
	default:
		userExists = true
		reqLogger.Info(fmt.Sprintf("Found user with name '%s' from pg_user", database.Spec.Name), "usename", usename, "usesysid", usesysid)
	}

	if !userExists {
		if _, err := db.Exec(
			context.Background(),
			fmt.Sprintf("CREATE ROLE %s WITH "+
				"LOGIN "+
				"NOSUPERUSER "+
				"NOCREATEDB "+
				"NOCREATEROLE "+
				"INHERIT "+
				"NOREPLICATION "+
				"CONNECTION LIMIT -1", database.Spec.Name)); err != nil {
			reqLogger.Error(err, "Failed to create user")
			return ctrl.Result{}, nil
		}
		reqLogger.Info("Successfully created a user")
	}

	return ctrl.Result{}, nil
}

func (r *PgDatabaseReconciler) finalize(reqLogger logr.Logger, user *postgresv1alpha1.PgDatabase) error {
	// Connect to database
	connStr := "postgresql://postgres:password@127.0.0.1:5432/postgres?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	defer db.Close()
	if err != nil {
		reqLogger.Error(err, "Failed to open database connection")
	}

	row := db.QueryRow("select * from pg_roles where rolname=$1", user.Spec.Name)
	if row.Err() != nil {
		reqLogger.Error(row.Err(), "Failed to query pg_roles")
	}
	reqLogger.Info("Successfully finalized PgUser")

	return nil
}
