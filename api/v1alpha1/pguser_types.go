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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PgUserSpec defines the desired state of PgUser
type PgUserSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Name string `json:"name"`
	// +optional
	Password *string `json:"password,omitempty"`
	// +optional
	// +listType=atomic
	Read *[]ReadAccessSpec `json:"read,omitempty"`
	// +optional
	// +listType=atomic
	Write *[]WriteAccessSpec `json:"write,omitempty"`
}

// ReadAccessSpec defines a read access request specification.
type ReadAccessSpec struct {
	Host ResourceVar `json:"host"`
	// +optional
	AllDatabases *bool `json:"allDatabases,omitempty"`
	// +optional
	Database ResourceVar `json:"database"`
	// +optional
	Schema ResourceVar `json:"schema"`
	Reason string      `json:"reason"`
	// +optional
	Start *metav1.Time `json:"start,omitempty"`
	// +optional
	Stop *metav1.Time `json:"stop,omitempty"`
}

// WriteAccessSpec defines a write access request specification.
type WriteAccessSpec struct {
	ReadAccessSpec `json:",inline"`
	// +optional
	Extended bool `json:"extended"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="PhaseUpdated",type="string",JSONPath=".status.phaseUpdated"
// +kubebuilder:printcolumn:name="Error",type="string",JSONPath=".status.error"

// PgUser is the Schema for the pgusers API
type PgUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PgUserSpec `json:"spec,omitempty"`
	Status Status     `json:"status,omitempty"`
}

func (u *PgUser) GetStatus() Status {
	return u.Status
}

func (u *PgUser) SetStatus(s Status) {
	u.Status.PhaseUpdated = s.PhaseUpdated
	u.Status.Phase = s.Phase
	u.Status.Error = s.Error
}

//+kubebuilder:object:root=true

// PgUserList contains a list of PgUser
type PgUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PgUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PgUser{}, &PgUserList{})
}
