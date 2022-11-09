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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
)

// PgUserReconciler reconciles a PgUser object
type PgUserReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	logger logr.Logger
	user   *api.PgUser
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
	r.logger = setupLogger(ctx)
	r.logger.Info("Reconciling PgUser")

	result, err := r.handleResult(r.reconcile(ctx, req))
	r.logger.Info("Finished reconciling PgUser")
	return result, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *PgUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.PgUser{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1,
			RateLimiter:             DefaultControllerRateLimiter(),
		}).
		Complete(r)
}

func (r *PgUserReconciler) reconcile(ctx context.Context, req reconcile.Request) error {
	// Fetch the PgUser instance
	user := &api.PgUser{}
	{
		err := r.Client.Get(ctx, req.NamespacedName, user)
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

		r.user = user
	}

	// PgUser instance created or updated
	r.logger = r.logger.WithValues("user", user.Name)
	r.logger.Info("Reconciling found PgUser resource")

	if user.GetDeletionTimestamp() != nil {
		if !slices.Contains(user.Finalizers, operatorFinalizer) {
			return nil
		}
		// Run finalization logic. If the
		// finalization logic fails, don't remove the finalizer so
		// that we can retry during the next reconciliation.
		if err := r.finalize(user); err != nil {
			return err
		}

		// Remove finalizer. Once all finalizers have been
		// removed, the object will be deleted.
		controllerutil.RemoveFinalizer(user, operatorFinalizer)
		if err := r.Update(ctx, user); err != nil {
			return ctlerrors.NewTemporary(err)
		}

		return nil
	}

	// Add finalizer for this CR
	if !slices.Contains(user.Finalizers, operatorFinalizer) {
		controllerutil.AddFinalizer(user, operatorFinalizer)

		if err := r.Update(ctx, user); err != nil {
			return ctlerrors.NewTemporary(err)
		}
	}

	username, err := apiutil.ResourceValue(r.Client, user.Spec.Name, user.Namespace)
	if err != nil {
		return ctlerrors.NewInvalid(err)
	}

	password, err := apiutil.ResourceValue(r.Client, user.Spec.Password, user.Namespace)
	if err != nil {
		return ctlerrors.NewInvalid(err)
	}

	dbs := make(map[string]*postgres.Client)

	// Connect to database
	if user.Spec.AccessSpecs != nil {
		for _, accessSpec := range *user.Spec.AccessSpecs {
			hostCred, err := apiutil.PgHostCredentialByName(r.Client, user.Namespace, accessSpec.HostCredential)
			if err != nil {
				r.logger.Error(err, "Failed to get host credential from the access spec. Skipping '"+accessSpec.HostCredential+"'")
				return ctlerrors.NewTemporary(err)
			}
			connStr, err := apiutil.GetConnectionStringWithDatabase(hostCred, r.Client, accessSpec.Database)
			if err != nil {
				r.logger.Error(err, "Failed to get connection string from the access spec. Skipping '"+accessSpec.HostCredential+"'")
				return ctlerrors.NewTemporary(err)
			}

			if dbs[connStr] == nil {
				db, err := postgres.NewClient(ctx, r.logger, connStr)
				if err != nil {
					r.logger.Error(err, "Failed to open database connection")
					return ctlerrors.NewTemporary(err)
				}
				dbs[connStr] = db
			}

			db := dbs[connStr]

			if err := db.EnsureUser(username, password); err != nil {
				return ctlerrors.NewTemporary(err)
			}

			switch accessSpec.Permission {
			case api.PermReadOnly:
				if err := db.EnsureReadonlyRoleToUser(accessSpec.Database, username); err != nil {
					return ctlerrors.NewTemporary(err)
				}
			case api.PermReadWrite:
				if err := db.EnsureReadwriteRoleToUser(accessSpec.Database, username); err != nil {
					return ctlerrors.NewTemporary(err)
				}
			}

			// database := accessSpec.Database
			// accessSpec.Permission
		}
	}

	return nil
}

func (r *PgUserReconciler) finalize(user *api.PgUser) error {
	r.logger.Info("Successfully finalized PgUser")
	return nil
}

func (r *PgUserReconciler) handleResult(err error) (ctrl.Result, error) {
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

	if r.user != nil {
		r.user.Status.Phase = phase
		r.user.Status.PhaseUpdated = metav1.Now()
		r.user.Status.Error = errorMessage
	}

	if err := r.Status().Update(context.Background(), r.user); err != nil {
		r.logger.Error(err, "Failed to update the status")
	}

	isRequeue := (phase == api.PhaseFailed)

	return ctrl.Result{Requeue: isRequeue}, err
}
