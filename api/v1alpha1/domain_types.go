/*
Copyright 2023 sokdakino.

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
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// DomainSpec defines the desired state of Domain
type DomainSpec struct {
	Provider string   `json:"provider"`
	Name     string   `json:"name"`
	Zone     string   `json:"zone"`
	Records  []string `json:"records"`
	TTL      int      `json:"ttl"`
}

// DomainStatus defines the observed state of Domain
type DomainStatus struct {
	Provider    string                  `json:"provider,omitempty"`
	FQDN        string                  `json:"fqdn,omitempty"`
	IngressName string                  `json:"ingressName,omitempty"`
	Conditions  []capiv1beta1.Condition `json:"conditions,omitempty"`
	//+optional
	Zone *ZoneStatus `json:"zone,omitempty"`
	//+optional
	Record *RecordStatus `json:"record,omitempty"`
}

type ZoneStatus struct {
	Name string `json:"name,omitempty"`
	Id   string `json:"id,omitempty"`
	//+optional
	Activated *bool `json:"activated,omitempty"`
}

type RecordStatus struct {
	Name    string   `json:"name,omitempty"`
	Id      string   `json:"id,omitempty"`
	Records []string `json:"records,omitempty"`
	TTL     *int     `json:"ttl,omitempty"`
	//+optional
	Activated *bool `json:"activated,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="provider",type=string,JSONPath=".spec.provider"
//+kubebuilder:printcolumn:name="virtualhost",type=string,JSONPath="$.spec.name.concat('.', $.spec.zone)"
//+kubebuilder:printcolumn:name="activated",type=bool,JSONPath=".status.activated"

// Domain is the Schema for the domains API
type Domain struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DomainSpec   `json:"spec,omitempty"`
	Status DomainStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DomainList contains a list of Domain
type DomainList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Domain `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Domain{}, &DomainList{})
}
