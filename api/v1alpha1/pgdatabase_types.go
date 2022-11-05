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

// PgDatabaseSpec defines the desired state of PgDatabase
type PgDatabaseSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Host string `json:"host,omitempty"`
	Name string `json:"name,omitempty"`
}

// PgDatabaseStatus defines the observed state of PgDatabase
type PgDatabaseStatus struct {
	Status `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="PhaseUpdated",type="string",JSONPath=".status.phaseUpdated"
// +kubebuilder:printcolumn:name="Error",type="string",JSONPath=".status.error"

// PgDatabase is the Schema for the pgdatabases API
type PgDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PgDatabaseSpec   `json:"spec,omitempty"`
	Status PgDatabaseStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PgDatabaseList contains a list of PgDatabase
type PgDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PgDatabase `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PgDatabase{}, &PgDatabaseList{})
}
