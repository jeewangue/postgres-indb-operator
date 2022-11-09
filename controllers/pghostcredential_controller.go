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
	"time"

	"github.com/go-logr/logr"
	"github.com/jackc/pgx/v5"
	api "github.com/jeewangue/postgres-indb-operator/api/v1alpha1"
	apiutil "github.com/jeewangue/postgres-indb-operator/api/v1alpha1/util"
	ctlerrors "github.com/jeewangue/postgres-indb-operator/internal/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// PgHostCredentialReconciler reconciles a PgHostCredential object
type PgHostCredentialReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	logger   logr.Logger
	hostCred *api.PgHostCredential
}

//+kubebuilder:rbac:groups=postgres.jeewangue.com,resources=pghostcredentials,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=postgres.jeewangue.com,resources=pghostcredentials/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=postgres.jeewangue.com,resources=pghostcredentials/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PgHostCredential object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *PgHostCredentialReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = setupLogger(ctx)
	r.logger.Info("Reconciling PgHostCredential")

	result, err := r.handleResult(r.reconcile(ctx, req))
	r.logger.Info("Finished reconciling PgHostCredential")
	return result, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *PgHostCredentialReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.PgHostCredential{}).
		Complete(r)
}

func (r *PgHostCredentialReconciler) reconcile(ctx context.Context, req reconcile.Request) error {
	// Fetch the PgHostCredential instance
	cred := &api.PgHostCredential{}
	{
		err := r.Client.Get(ctx, req.NamespacedName, cred)
		if err != nil {
			if errors.IsNotFound(err) {
				// Request object not found, could have been deleted after reconcile request.
				// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
				// Return and don't requeue
				r.logger.Info("Object not found")
				return ctlerrors.NewInvalid(err)
			}
			// Error reading the object - requeue the request.
			return ctlerrors.NewTemporary(err)
		}

		r.hostCred = cred
	}

	// PgHostCredential instance created or updated
	r.logger = r.logger.WithValues("hostCredential", cred.Name)
	r.logger.Info("Reconciling found PgHostCredential resource")

	connStr, err := apiutil.GetConnectionString(cred, r.Client)
	if err != nil {
		return ctlerrors.NewInvalid(err)
	}

	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return ctlerrors.NewTemporary(err)
	}
	defer conn.Close(ctx)

	if err := conn.Ping(ctx); err != nil {
		return ctlerrors.NewTemporary(err)
	}

	return nil
}

func (r *PgHostCredentialReconciler) handleResult(err error) (ctrl.Result, error) {
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

	if r.hostCred != nil {
		r.hostCred.Status.Phase = phase
		r.hostCred.Status.PhaseUpdated = metav1.Now()
		r.hostCred.Status.Error = errorMessage
	}

	if err := r.Status().Update(context.Background(), r.hostCred); err != nil {
		r.logger.Error(err, "Failed to update the status")
	}

	if phase == api.PhaseInvalid {
		return ctrl.Result{}, err
	} else {
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}
}
