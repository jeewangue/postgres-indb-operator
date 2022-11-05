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
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	api "github.com/jeewangue/postgres-indb-operator/api/v1alpha1"
	ctlerrors "github.com/jeewangue/postgres-indb-operator/internal/errors"
	"github.com/jeewangue/postgres-indb-operator/internal/kube"
	"github.com/jeewangue/postgres-indb-operator/internal/postgres"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	postgresv1alpha1 "github.com/jeewangue/postgres-indb-operator/api/v1alpha1"
)

// PgHostCredentialReconciler reconciles a PgHostCredential object
type PgHostCredentialReconciler struct {
	client.Client
	Scheme *runtime.Scheme
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
	reqLogger := log.FromContext(ctx)

	requestID, err := uuid.NewRandom()
	if err != nil {
		reqLogger.Error(err, "Failed to pick a request ID. Continuing without")
	}
	reqLogger = reqLogger.WithValues("requestId", requestID.String())
	reqLogger.Info("Reconciling PgHostCredential")

	status, err := r.reconcile(ctx, reqLogger, req)
	if err != nil {
		reqLogger.Error(err, "Failed to reconcile PgHostCredential object")
	}
	status.Persist(ctx, err)

	reqLogger.Info("Result of reconcilation", "status", status, "err", err)
	if status.IsRequeue() {
		return ctrl.Result{Requeue: true}, err
	} else {
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *PgHostCredentialReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&postgresv1alpha1.PgHostCredential{}).
		Complete(r)
}

func (r *PgHostCredentialReconciler) reconcile(ctx context.Context, reqLogger logr.Logger, request reconcile.Request) (*Status, error) {
	// Fetch the PgHostCredential instance
	cred := &api.PgHostCredential{}
	err := r.Client.Get(ctx, request.NamespacedName, cred)
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

	// PgHostCredential instance created or updated
	reqLogger = reqLogger.WithValues("hostCredential", cred.Name)
	reqLogger.Info("Reconciling found PgHostCredential resource")

	status := NewStatus(reqLogger, r.Client, cred)

	var host, user, password string
	{
		if host, err = kube.ResourceValue(r.Client, cred.Spec.Host, cred.Namespace); err != nil {
			return status, ctlerrors.NewInvalid(err)
		}
		if user, err = kube.ResourceValue(r.Client, cred.Spec.User, cred.Namespace); err != nil {
			return status, ctlerrors.NewInvalid(err)
		}
		if password, err = kube.ResourceValue(r.Client, cred.Spec.Password, cred.Namespace); err != nil {
			return status, ctlerrors.NewInvalid(err)
		}
	}

	connStr := postgres.ConnectionString{
		Host:     host,
		User:     user,
		Password: password,
		Database: "postgres",
		Params:   cred.Spec.Params,
	}

	conn, err := pgx.Connect(context.Background(), connStr.Raw())
	if err != nil {
		return status, ctlerrors.NewTemporary(err)
	}
	defer conn.Close(ctx)

	if err := conn.Ping(ctx); err != nil {
		return status, ctlerrors.NewTemporary(err)
	}

	return status, nil
}
