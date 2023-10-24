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
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ConditionTypeProviderChanged    capiv1beta1.ConditionType = "ProviderChanged"
	ConditionTypeZoneChanged        capiv1beta1.ConditionType = "ZoneChanged"
	ConditionTypeProviderLoaded     capiv1beta1.ConditionType = "ProviderLoaded"
	ConditionTypeZoneInfoLoaded     capiv1beta1.ConditionType = "ZoneInfoLoaded"
	ConditionTypeRecordSetCreated   capiv1beta1.ConditionType = "RecordSetCreated"
	ConditionTypeRecordSetRetrieved capiv1beta1.ConditionType = "RecordSetRetrieved"
	ConditionTypeRecordSetUpdated   capiv1beta1.ConditionType = "RecordSetUpdated"
	ConditionTypeRecordSetReady     capiv1beta1.ConditionType = "Ready"

	ConditionReasonServiceAPIFailed = "ServiceAPIRequestFailed"
	ConditionReasonProviderNotFound = "ProviderNotFound"
	ConditionReasonZoneNotFound     = "ZoneNotFound"
)

// DomainSpec defines the desired state of Domain
type DomainSpec struct {
	Provider string   `json:"provider"`
	Type     string   `json:"type"`
	Name     string   `json:"name"`
	Zone     string   `json:"zone"`
	Records  []string `json:"records"`
	TTL      int      `json:"ttl"`
}

// DomainStatus defines the observed state of Domain
type DomainStatus struct {
	Provider    string                 `json:"provider,omitempty"`
	FQDN        string                 `json:"fqdn,omitempty"`
	IngressName string                 `json:"ingressName,omitempty"`
	Conditions  capiv1beta1.Conditions `json:"conditions,omitempty"`
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
	Type    string   `json:"type,omitempty"`
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

func (d *Domain) GetConditions() capiv1beta1.Conditions {
	return d.Status.Conditions
}

func (d *Domain) SetConditions(conds capiv1beta1.Conditions) {
	d.Status.Conditions = conds
}

func (d *Domain) Update(ctx context.Context, client client.Client, applier func(*Domain)) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		// get
		tmp := &Domain{}
		nsn := types.NamespacedName{Name: d.Name, Namespace: d.Namespace}
		if err := client.Get(ctx, nsn, tmp); err != nil {
			return err
		}

		// apply
		applier(d)

		// update
		if err := client.Update(ctx, tmp); err != nil {
			return err
		}

		return nil
	})
}

func (d *Domain) StatusUpdate(ctx context.Context, client client.Client, applier func(*Domain)) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		// get
		tmp := &Domain{}
		nsn := types.NamespacedName{Name: d.Name, Namespace: d.Namespace}
		if err := client.Get(ctx, nsn, tmp); err != nil {
			return err
		}

		// apply
		applier(d)

		// update
		if err := client.Status().Update(ctx, tmp); err != nil {
			return err
		}

		return nil
	})
}
