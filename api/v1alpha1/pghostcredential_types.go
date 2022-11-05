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

// PgHostCredentialSpec defines the desired state of PgHostCredential
type PgHostCredentialSpec struct {
	// Host is the hostname of the Postgres instance.
	Host ResourceVar `json:"host,omitempty"`

	// User is the admin user for the Postgres instance. It will be used by
	// postgres-indb-operator to manage resources on the host.
	User ResourceVar `json:"user,omitempty"`

	// Password is the admin user password for the Postgres instance. It will
	// be used by postgres-indb-operator to manage resources on the host.
	Password ResourceVar `json:"password,omitempty"`

	// Params is the space-separated list of parameters (e.g.,
	// `"sslmode=require"`)
	Params string `json:"params,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="PhaseUpdated",type="string",JSONPath=".status.phaseUpdated"
// +kubebuilder:printcolumn:name="Error",type="string",JSONPath=".status.error"

// PgHostCredential is the Schema for the pghostcredentials API
type PgHostCredential struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PgHostCredentialSpec `json:"spec,omitempty"`
	Status Status               `json:"status,omitempty"`
}

func (u *PgHostCredential) GetStatus() Status {
	return u.Status
}

func (u *PgHostCredential) SetStatus(s Status) {
	u.Status.PhaseUpdated = s.PhaseUpdated
	u.Status.Phase = s.Phase
	u.Status.Error = s.Error
}

//+kubebuilder:object:root=true

// PgHostCredentialList contains a list of PgHostCredential
type PgHostCredentialList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PgHostCredential `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PgHostCredential{}, &PgHostCredentialList{})
}
