package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResourceVar represents a value or reference to a value.
type ResourceVar struct {
	// Defaults to "".
	// +optional
	Value string `json:"value,omitempty"`
	// Source to read the value from.
	// +optional
	ValueFrom *ResourceVarSource `json:"valueFrom,omitempty"`
}

// ResourceVarSource represents a source for the value of a ResourceVar
type ResourceVarSource struct {
	// Selects a key of a secret in the custom resource's namespace
	// +optional
	SecretKeyRef *KeySelector `json:"secretKeyRef,omitempty"`

	// Selects a key of a config map in the custom resource's namespace
	// +optional
	ConfigMapKeyRef *KeySelector `json:"configMapKeyRef,omitempty"`
}

// KeySelector selects a key of a Secret or ConfigMap.
type KeySelector struct {
	// The name of the secret or config map in the namespace to select from.
	Name string `json:"name,omitempty"`
	// The key of the secret or config map to select from.  Must be a valid key.
	Key string `json:"key"`
}

// Status defines the observed state of object.
type Status struct {
	// Represents the observations of a foo's current state.
	// Known .status.conditions.type are: "Available", "Progressing", and "Degraded"
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	PhaseUpdated metav1.Time `json:"phaseUpdated"`
	Phase        Phase       `json:"phase"`
	Error        string      `json:"error,omitempty"`
}

// Phase represents the current phase of the object.
type Phase string

const (
	// PhasePending indicates that the controller will reconcile soon.
	PhasePending Phase = "Pending"
	// PhaseAvailable indicates that the controller has reconciled the object
	// and that it is available.
	PhaseAvailable Phase = "Available"
	// PhaseInvalid indicates that the controller was unable to
	// reconcile the object as the specification of it is invalid and should be
	// fixed. It will not be attempted again before the resource is updated.
	PhaseInvalid Phase = "Invalid"
	// PhaseFailed indicates that the controller was unable to
	// reconcile the object and will retry.
	PhaseFailed Phase = "Failed"
)
