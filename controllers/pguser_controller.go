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
	api "github.com/jeewangue/postgres-indb-operator/api/v1alpha1"
	_ "github.com/lib/pq"
)

// PgUserReconciler reconciles a PgUser object
type PgUserReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=postgres.jeewangue.com,resources=pgusers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=postgres.jeewangue.com,resources=pgusers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=postgres.jeewangue.com,resources=pgusers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PgUser object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *PgUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)

	requestID, err := uuid.NewRandom()
	if err != nil {
		reqLogger.Error(err, "Failed to pick a request ID. Continuing without")
	}
	reqLogger = reqLogger.WithValues("requestId", requestID.String())
	reqLogger.Info("Reconciling PgUser")

	status, err := r.reconcile(ctx, reqLogger, req)
	if err != nil {
		reqLogger.Error(err, "Failed to reconcile PgUser object")
	}
	status.Persist(ctx, err)

	reqLogger.Info("Result of reconcilation", "status", status, "err", err)
	return ctrl.Result{Requeue: status.IsRequeue()}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *PgUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.PgUser{}).
		Complete(r)
}

func (r *PgUserReconciler) reconcile(ctx context.Context, reqLogger logr.Logger, request reconcile.Request) (status *Status, err error) {
	// Fetch the PgUser instance
	user := &api.PgUser{}
	err = r.Client.Get(ctx, request.NamespacedName, user)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("Object not found")
			return nil, nil
		}
		// Error reading the object - requeue the request.
		return nil, err
	}

	// PgUser instance created or updated
	reqLogger = reqLogger.WithValues("user", user.Spec.Name)
	reqLogger.Info("Reconciling found PgUser resource")

	status = NewStatus(reqLogger, r.Client, user)

	if user.GetDeletionTimestamp() != nil {
		if !slices.Contains(user.Finalizers, operatorFinalizer) {
			return status, nil
		}
		// Run finalization logic. If the
		// finalization logic fails, don't remove the finalizer so
		// that we can retry during the next reconciliation.
		if err := r.finalize(reqLogger, user); err != nil {
			return status, err
		}

		// Remove finalizer. Once all finalizers have been
		// removed, the object will be deleted.
		controllerutil.RemoveFinalizer(user, operatorFinalizer)
		err := r.Update(ctx, user)
		if err != nil {
			return status, err
		}

		return status, nil
	}

	// Add finalizer for this CR
	if !slices.Contains(user.Finalizers, operatorFinalizer) {
		controllerutil.AddFinalizer(user, operatorFinalizer)

		err = r.Update(ctx, user)
		if err != nil {
			return status, err
		}
	}

	// We need to sanitize the user.Spec.Name to be a valid PostgreSQL role name
	// sanitizedRole := sanitizedRole(role)

	// Connect to database
	connStr := "postgresql://postgres:password@127.0.0.1:5432/postgres?sslmode=disable"
	db, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		reqLogger.Error(err, "Failed to open database connection")
		return status, err
	}
	defer db.Close(ctx)

	userExists := false
	var usename string
	var usesysid uint32
	err = db.QueryRow(context.Background(), "SELECT usename, usesysid FROM pg_user WHERE usename = $1", user.Spec.Name).Scan(&usename, &usesysid)
	switch {
	case err == pgx.ErrNoRows:
		userExists = false
		reqLogger.Info(fmt.Sprintf("No user with name %s. Creating...", user.Spec.Name))
	case err != nil:
		reqLogger.Error(err, "Failed to query from pg_user")
		return status, nil
	default:
		userExists = true
		reqLogger.Info(fmt.Sprintf("Found user with name '%s' from pg_user", user.Spec.Name), "usename", usename, "usesysid", usesysid)
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
				"CONNECTION LIMIT -1", user.Spec.Name)); err != nil {
			reqLogger.Error(err, "Failed to create user")
			return status, nil
		}
		reqLogger.Info("Successfully created a user")
	}

	if user.Spec.Password != nil {
		if _, err := db.Exec(
			context.Background(),
			fmt.Sprintf("ALTER ROLE %s PASSWORD '%s'",
				user.Spec.Name,
				*user.Spec.Password)); err != nil {
			reqLogger.Error(err, "Failed to setup user")
			return status, nil
		}
		reqLogger.Info("Successfully set password for the user")
	}

	return status, nil
}

func (r *PgUserReconciler) finalize(reqLogger logr.Logger, user *api.PgUser) error {
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
