package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const operatorFinalizer = "postgres.jeewangue.com/finalizer"

func setupLogger(ctx context.Context) logr.Logger {
	reqLogger := log.FromContext(ctx)
	requestID, err := uuid.NewRandom()
	if err != nil {
		reqLogger.Error(err, "Failed to pick a request ID. Continuing without")
	}
	return reqLogger.WithValues("requestId", requestID.String())
}
