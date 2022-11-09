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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/strings/slices"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	api "github.com/jeewangue/postgres-indb-operator/api/v1alpha1"
	apiutil "github.com/jeewangue/postgres-indb-operator/api/v1alpha1/util"
	ctlerrors "github.com/jeewangue/postgres-indb-operator/internal/errors"
	"github.com/jeewangue/postgres-indb-operator/internal/postgres"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PgDatabaseReconciler reconciles a PgDatabase object
type PgDatabaseReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	logger   logr.Logger
	database *api.PgDatabase
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
	r.logger = setupLogger(ctx)
	r.logger.Info("Reconciling PgDatabase")

	result, err := r.handleResult(r.reconcile(ctx, req))
	r.logger.Info("Finished reconciling PgDatabase")
	return result, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *PgDatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.PgDatabase{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1,
			RateLimiter:             DefaultControllerRateLimiter(),
		}).
		Complete(r)
}

func (r *PgDatabaseReconciler) reconcile(ctx context.Context, req reconcile.Request) error {
	// Fetch the PgDatabase instance
	database := &api.PgDatabase{}
	{
		err := r.Client.Get(ctx, req.NamespacedName, database)
		if err != nil {
			if errors.IsNotFound(err) {
				// Request object not found, could have been deleted after reconcile request.
				// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
				// Return and don't requeue
				r.logger.Info("Object not found")
				return nil
			}
			// Error reading the object - requeue the request.
			return ctlerrors.NewTemporary(err)
		}

		r.database = database
	}

	// PgDatabase instance created or updated
	r.logger = r.logger.WithValues("database", database.Name)
	r.logger.Info("Reconciling found PgDatabase resource")

	if database.GetDeletionTimestamp() != nil {
		if !slices.Contains(database.Finalizers, operatorFinalizer) {
			return nil
		}
		// Run finalization logic. If the
		// finalization logic fails, don't remove the finalizer so
		// that we can retry during the next reconciliation.
		if err := r.finalize(database); err != nil {
			return err
		}

		// Remove finalizer. Once all finalizers have been
		// removed, the object will be deleted.
		controllerutil.RemoveFinalizer(database, operatorFinalizer)
		if err := r.Update(ctx, database); err != nil {
			return ctlerrors.NewTemporary(err)
		}

		return nil
	}

	// Add finalizer for this CR
	if !slices.Contains(database.Finalizers, operatorFinalizer) {
		controllerutil.AddFinalizer(database, operatorFinalizer)

		if err := r.Update(ctx, database); err != nil {
			return ctlerrors.NewTemporary(err)
		}
	}

	// Connect to database
	hostCred, err := apiutil.PgHostCredentialByName(r.Client, database.Namespace, database.Spec.HostCredential)
	if err != nil {
		r.logger.Error(err, "Failed to get host credential from the access spec. Skipping '"+database.Spec.HostCredential+"'")
		return ctlerrors.NewTemporary(err)
	}

	{
		connStr, err := apiutil.GetConnectionString(hostCred, r.Client)
		if err != nil {
			r.logger.Error(err, "Failed to get connection string from the access spec. Skipping '"+database.Spec.HostCredential+"'")
			return ctlerrors.NewTemporary(err)
		}

		db, err := postgres.NewClient(ctx, r.logger, connStr)
		if err != nil {
			r.logger.Error(err, "Failed to open database connection")
			return ctlerrors.NewTemporary(err)
		}

		if err := db.EnsureDatabase(database.Spec.Name); err != nil {
			return ctlerrors.NewTemporary(err)
		}

	}

	{
		connStr, err := apiutil.GetConnectionStringWithDatabase(hostCred, r.Client, database.Spec.Name)
		if err != nil {
			r.logger.Error(err, "Failed to get connection string from the access spec. Skipping '"+database.Spec.HostCredential+"'")
			return ctlerrors.NewTemporary(err)
		}

		db, err := postgres.NewClient(ctx, r.logger, connStr)
		if err != nil {
			r.logger.Error(err, "Failed to open database connection")
			return ctlerrors.NewTemporary(err)
		}

		if err := db.EnsureDatabaseAccessRoles(database.Spec.Name); err != nil {
			return ctlerrors.NewTemporary(err)
		}

	}

	return nil
}

func (r *PgDatabaseReconciler) finalize(database *api.PgDatabase) error {
	r.logger.Info("Successfully finalized PgDatabase")
	return nil
}

func (r *PgDatabaseReconciler) handleResult(err error) (ctrl.Result, error) {
	var phase api.Phase
	var errorMessage string

	switch {
	case err == nil:
		phase = api.PhaseAvailable
		errorMessage = ""
	case ctlerrors.IsTemporary(err):
		phase = api.PhaseFailed
		errorMessage = err.Error()
	case ctlerrors.IsInvalid(err):
		phase = api.PhaseInvalid
		errorMessage = err.Error()
	default:
		phase = api.PhaseInvalid
		errorMessage = err.Error()
	}

	if r.database != nil {
		r.database.Status.Phase = phase
		r.database.Status.PhaseUpdated = metav1.Now()
		r.database.Status.Error = errorMessage
	}

	if err := r.Status().Update(context.Background(), r.database); err != nil {
		r.logger.Error(err, "Failed to update the status")
	}

	isRequeue := (phase == api.PhaseFailed)

	return ctrl.Result{Requeue: isRequeue}, err
}
