package controllers

import (
	"context"

	"github.com/go-logr/logr"
	api "github.com/jeewangue/postgres-indb-operator/api/v1alpha1"
	ctlerrors "github.com/jeewangue/postgres-indb-operator/internal/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const operatorFinalizer = "postgres.jeewangue.com/finalizer"

// ObjectWithStatus extends client.Object with Status
type ObjectWithStatus interface {
	client.Object
	GetStatus() api.Status
	SetStatus(s api.Status)
}

type Status struct {
	logger logr.Logger
	client client.Client
	Object ObjectWithStatus `json:"object,omitempty"`
}

func NewStatus(logger logr.Logger, client client.Client, object ObjectWithStatus) *Status {
	return &Status{
		logger: logger,
		client: client,
		Object: object,
	}
}

// IsRequeue determines to requeue reconcile.
func (s *Status) IsRequeue() bool {
	if s == nil {
		return true
	}

	switch s.Object.GetStatus().Phase {
	case api.PhaseAvailable:
		return false
	case api.PhaseInvalid:
		return false
	case api.PhaseFailed:
		return true
	default:
		return true
	}
}

// Persist writes the status to the object and persists it on
// client. Any errors are logged.
func (s *Status) Persist(ctx context.Context, err error) {
	ok := s.update(err)
	if !ok {
		return
	}
	err = s.client.Status().Update(ctx, s.Object)
	if err != nil {
		s.logger.Error(err, "failed to set status", "status", s)
	}
}

// update updates the object based on its values and returns whether any
// changes were written.
func (s *Status) update(err error) bool {
	var errorMessage string
	var phase api.Phase

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

	phaseEqual := s.Object.GetStatus().Phase == phase
	errorEqual := s.Object.GetStatus().Error == errorMessage

	if phaseEqual && errorEqual {
		return false
	}

	s.Object.SetStatus(api.Status{
		PhaseUpdated: metav1.Now(),
		Phase:        phase,
		Error:        errorMessage,
	})

	return true
}
