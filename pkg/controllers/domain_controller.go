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

package controllers

import (
	"context"
	"fmt"
	"github.com/sokdak/dns-ingress/api/v1alpha1"
	"github.com/sokdak/dns-ingress/pkg/common"
	"github.com/sokdak/dns-ingress/pkg/provider"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/flowcontrol"
	"reflect"
	"sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sort"
)

// DomainReconciler reconciles a Domain object
type DomainReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	Backoff           flowcontrol.Backoff
	ProviderClientMap map[string]provider.Client
}

//+kubebuilder:rbac:groups=dns-ingress.io,resources=domains,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dns-ingress.io,resources=domains/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dns-ingress.io,resources=domains/finalizers,verbs=update

func (r *DomainReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("start reconcile", GenerateReconcileInformationLabelKeySet(req.NamespacedName))
	defer l.Info("end reconcile", GenerateReconcileInformationLabelKeySet(req.NamespacedName))

	// get domain object
	domain := &v1alpha1.Domain{}
	if err := r.Client.Get(ctx, req.NamespacedName, domain); err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("can't get domain object: %w", err)
	}

	// check provider available
	service, found := r.ProviderClientMap[domain.Spec.Provider]
	if !found {
		copiedDomain := domain.DeepCopy()
		conditions.MarkFalse(copiedDomain, v1alpha1.ConditionTypeProviderLoaded,
			v1alpha1.ConditionReasonProviderNotFound, v1beta1.ConditionSeverityError,
			"dns provider %s not found on configuration", domain.Spec.Provider)

		if !reflect.DeepEqual(copiedDomain, domain) {
			if err := r.Client.Update(ctx, domain); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// if has deletionTimestamp with finalizer, delete the record
	if domain.DeletionTimestamp != nil && controllerutil.ContainsFinalizer(domain, FinalizerDomain) {
		if domain.Status.Record != nil && len(domain.Status.Record.Id) > 0 {
			rs, err := service.Get(ctx, domain.Status.Record.Id, domain.Status.Zone.Id)
			if err != nil {
				l.Error(err, "Reconciler error")
				return ctrl.Result{RequeueAfter: GetNextBackoffDuration(&r.Backoff, req.NamespacedName, "Delete-Get")}, nil
			}
			ResetBackoff(&r.Backoff, req.NamespacedName, "Delete-Get")

			// if recordset exists then try to delete
			if rs != nil {
				if err := service.Delete(ctx, domain.Status.Record.Id, domain.Status.Zone.Id); err != nil {
					l.Error(err, "Reconciler error")
					return ctrl.Result{RequeueAfter: GetNextBackoffDuration(&r.Backoff, req.NamespacedName, "Delete")}, nil
				}
				ResetBackoff(&r.Backoff, req.NamespacedName, "Delete")
			}
		}

		if controllerutil.RemoveFinalizer(domain, FinalizerDomain) {
			if err := r.Client.Update(ctx, domain); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// if don't have finalizer, add it
	if !controllerutil.ContainsFinalizer(domain, FinalizerDomain) {
		if controllerutil.AddFinalizer(domain, FinalizerDomain) {
			if err := r.Client.Update(ctx, domain); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// if ProviderChanged true, teardown the record and requeue
	if conditions.IsTrue(domain, v1alpha1.ConditionTypeProviderChanged) {
		l.Info("provider change detected",
			GenerateReconcileInformationLabelKeySetByDomain(domain))
		if err := teardownRecordSet(ctx, service, domain); err != nil {
			l.Error(err, "Reconciler error")
			return ctrl.Result{RequeueAfter: GetNextBackoffDuration(&r.Backoff, req.NamespacedName, "ProviderChanged-Delete")}, nil
		}
		ResetBackoff(&r.Backoff, req.NamespacedName, "ProviderChanged-Delete")

		if err := domain.StatusUpdate(ctx, r.Client, func(d *v1alpha1.Domain) {
			conditions.Delete(d, v1alpha1.ConditionTypeProviderChanged)
			d.Status.Provider = ""
			d.Status.FQDN = ""
			d.Status.Zone = nil
			d.Status.Record = nil
		}); err != nil {
			l.Error(err, "Reconciler error")
			return ctrl.Result{RequeueAfter: GetNextBackoffDuration(&r.Backoff, req.NamespacedName, "ProviderChanged-UpdateCR")}, nil
		}
		ResetBackoff(&r.Backoff, req.NamespacedName, "ProviderChanged-UpdateCR")
		return ctrl.Result{Requeue: true}, nil
	}

	// if status.provider is present and spec.provider are not matched, mark ProviderChanged true
	if len(domain.Status.Provider) > 0 && (domain.Spec.Provider != domain.Status.Provider) {
		return ctrl.Result{}, domain.StatusUpdate(ctx, r.Client, func(d *v1alpha1.Domain) {
			conditions.MarkTrue(d, v1alpha1.ConditionTypeProviderChanged)
		})
	}

	// if status.provider is not present, copy provider to status
	if len(domain.Status.Provider) == 0 {
		return ctrl.Result{Requeue: true}, domain.Update(ctx, r.Client, func(d *v1alpha1.Domain) {
			d.Status.Provider = d.Spec.Provider
		})
	}

	// if ZoneChanged true, teardown the record and requeue
	if conditions.IsTrue(domain, v1alpha1.ConditionTypeZoneChanged) {
		l.Info("zone change detected",
			GenerateReconcileInformationLabelKeySetByDomain(domain))
		if err := teardownRecordSet(ctx, service, domain); err != nil {
			l.Error(err, "Reconciler error")
			return ctrl.Result{RequeueAfter: GetNextBackoffDuration(&r.Backoff, req.NamespacedName, "ZoneChanged-Delete")}, nil
		}
		ResetBackoff(&r.Backoff, req.NamespacedName, "ZoneChanged-Delete")

		if err := domain.StatusUpdate(ctx, r.Client, func(d *v1alpha1.Domain) {
			conditions.Delete(d, v1alpha1.ConditionTypeZoneChanged)
			d.Status.Provider = ""
			d.Status.FQDN = ""
			d.Status.Zone = nil
			d.Status.Record = nil
		}); err != nil {
			l.Error(err, "Reconciler error")
			return ctrl.Result{RequeueAfter: GetNextBackoffDuration(&r.Backoff, req.NamespacedName, "ZoneChanged-K8sUpdate")}, nil
		}
		ResetBackoff(&r.Backoff, req.NamespacedName, "ZoneChanged-K8sUpdate")
		return ctrl.Result{Requeue: true}, nil
	}

	// if status.zone is present and spec.zone are not matched, mark ZoneChanged true
	if domain.Status.Zone != nil && (domain.Spec.Zone != domain.Status.Zone.Name) {
		return ctrl.Result{}, domain.StatusUpdate(ctx, r.Client, func(d *v1alpha1.Domain) {
			conditions.MarkTrue(d, v1alpha1.ConditionTypeZoneChanged)
		})
	}

	// if status.zone is not present, load the zone info (mark ZoneInfoLoaded true)
	if domain.Status.Zone == nil {
		z, err := service.GetZone(ctx, domain.Spec.Zone)
		if err != nil {
			if err := domain.StatusUpdate(ctx, r.Client, func(d *v1alpha1.Domain) {
				conditions.MarkFalse(d, v1alpha1.ConditionTypeZoneInfoLoaded,
					v1alpha1.ConditionReasonServiceAPIFailed, v1beta1.ConditionSeverityError,
					"request failed: %s", err.Error())
			}); err != nil {
				l.Error(err, "Reconciler error")
			}
			return ctrl.Result{RequeueAfter: GetNextBackoffDuration(&r.Backoff, req.NamespacedName, "Zone-Get")}, nil
		}
		ResetBackoff(&r.Backoff, req.NamespacedName, "Zone-Get")

		// if zone not found, retry after backoff
		if z == nil {
			if err := domain.StatusUpdate(ctx, r.Client, func(d *v1alpha1.Domain) {
				conditions.MarkFalse(d, v1alpha1.ConditionTypeZoneInfoLoaded,
					v1alpha1.ConditionReasonZoneNotFound, v1beta1.ConditionSeverityError,
					"zone %s is not available", domain.Spec.Zone)
			}); err != nil {
				l.Error(err, "Reconciler error")
			}
			return ctrl.Result{RequeueAfter: GetNextBackoffDuration(&r.Backoff, req.NamespacedName, "Zone-K8sUpdate-NotFound")}, nil
		}
		ResetBackoff(&r.Backoff, req.NamespacedName, "Zone-K8sUpdate-NotFound")

		if err := domain.StatusUpdate(ctx, r.Client, func(d *v1alpha1.Domain) {
			conditions.MarkTrue(d, v1alpha1.ConditionTypeZoneInfoLoaded)
			d.Status.Zone = &v1alpha1.ZoneStatus{
				Name:      z.Name,
				Id:        z.Id,
				Activated: common.BoolPointer(z.Activated),
			}
		}); err != nil {
			l.Error(err, "Reconciler error")
			return ctrl.Result{RequeueAfter: GetNextBackoffDuration(&r.Backoff, req.NamespacedName, "Zone-K8sUpdate")}, nil
		}
		ResetBackoff(&r.Backoff, req.NamespacedName, "Zone-K8sUpdate")
		return ctrl.Result{Requeue: true}, nil
	}

	// if status.record is empty then try to load the record (get or create)
	if domain.Status.Record == nil {
		rs, err := service.GetByName(ctx, domain.Name, domain.Status.Zone.Id)
		if err != nil {
			if err := domain.StatusUpdate(ctx, r.Client, func(d *v1alpha1.Domain) {
				conditions.MarkFalse(d, v1alpha1.ConditionTypeRecordSetRetrieved,
					v1alpha1.ConditionReasonServiceAPIFailed, v1beta1.ConditionSeverityError,
					"request failed: %s", err.Error())
			}); err != nil {
				l.Error(err, "Reconciler error")
			}
			return ctrl.Result{RequeueAfter: GetNextBackoffDuration(&r.Backoff, req.NamespacedName, "Record-Get")}, nil
		}
		ResetBackoff(&r.Backoff, req.NamespacedName, "Record-Get")

		if rs == nil {
			rs, err = service.Create(ctx, domain.Spec.Name, domain.Status.Zone.Id, domain.Spec.Type, domain.Spec.Records, domain.Spec.TTL)
			if err != nil {
				if err := domain.StatusUpdate(ctx, r.Client, func(d *v1alpha1.Domain) {
					conditions.Delete(d, v1alpha1.ConditionTypeRecordSetRetrieved)
					conditions.MarkFalse(d, v1alpha1.ConditionTypeRecordSetCreated,
						v1alpha1.ConditionReasonServiceAPIFailed, v1beta1.ConditionSeverityError,
						"request failed: %s", err.Error())
				}); err != nil {
					l.Error(err, "Reconciler error")
				}
				return ctrl.Result{RequeueAfter: GetNextBackoffDuration(&r.Backoff, req.NamespacedName, "Record-Create")}, nil
			}
			ResetBackoff(&r.Backoff, req.NamespacedName, "Record-Create")
		}

		// sort strings before put in the status
		sort.Strings(rs.Records)
		if err := domain.StatusUpdate(ctx, r.Client, func(d *v1alpha1.Domain) {
			conditions.Delete(d, v1alpha1.ConditionTypeRecordSetRetrieved)
			conditions.MarkTrue(d, v1alpha1.ConditionTypeRecordSetCreated)
			d.Status.Record = &v1alpha1.RecordStatus{
				Name:      rs.Name,
				Id:        rs.Id,
				Type:      rs.Type,
				Records:   rs.Records,
				TTL:       common.IntPointer(rs.TTL),
				Activated: common.BoolPointer(rs.Activated),
			}
			d.Status.FQDN = rs.FQDN
		}); err != nil {
			l.Error(err, "Reconciler error")
			return ctrl.Result{RequeueAfter: GetNextBackoffDuration(&r.Backoff, req.NamespacedName, "Record-K8sUpdate-Create")}, nil
		}
		ResetBackoff(&r.Backoff, req.NamespacedName, "Record-K8sUpdate-Create")
	}

	// if status.record.** and spec.** mismatched then try to update the record (update)
	sort.Strings(domain.Spec.Records)
	if domain.Status.Record.Name != domain.Spec.Name || domain.Status.Record.Type != domain.Spec.Type ||
		!reflect.DeepEqual(domain.Status.Record.Records, domain.Spec.Records) || *domain.Status.Record.TTL != domain.Spec.TTL {
		rs, err := service.Update(ctx, domain.Status.Record.Id, domain.Status.Zone.Id, domain.Spec.Type, domain.Spec.Records, domain.Spec.TTL)
		if err != nil {
			if err := domain.StatusUpdate(ctx, r.Client, func(d *v1alpha1.Domain) {
				conditions.MarkFalse(d, v1alpha1.ConditionTypeRecordSetUpdated,
					v1alpha1.ConditionReasonServiceAPIFailed, v1beta1.ConditionSeverityError,
					"update request failed: %s", err.Error())
			}); err != nil {
				l.Error(err, "Reconciler error")
			}
			return ctrl.Result{RequeueAfter: GetNextBackoffDuration(&r.Backoff, req.NamespacedName, "Record-Update")}, nil
		}
		ResetBackoff(&r.Backoff, req.NamespacedName, "Record-Update")

		// integrity failed, need to evict the entire record status
		if rs == nil {
			if err := domain.StatusUpdate(ctx, r.Client, func(d *v1alpha1.Domain) {
				conditions.Delete(d, v1alpha1.ConditionTypeRecordSetCreated)
				conditions.Delete(d, v1alpha1.ConditionTypeRecordSetUpdated)
				conditions.Delete(d, v1alpha1.ConditionTypeRecordSetRetrieved)
				conditions.Delete(d, v1alpha1.ConditionTypeRecordSetReady)
				d.Status.Record = nil
				d.Status.FQDN = ""
			}); err != nil {
				l.Error(err, "Reconciler error")
			}
			return ctrl.Result{Requeue: true}, nil
		}

		// sort strings before put in the status
		sort.Strings(rs.Records)
		if err := domain.StatusUpdate(ctx, r.Client, func(d *v1alpha1.Domain) {
			conditions.MarkTrue(d, v1alpha1.ConditionTypeRecordSetUpdated)
			d.Status.Record = &v1alpha1.RecordStatus{
				Name:      rs.Name,
				Id:        rs.Id,
				Type:      rs.Type,
				Records:   rs.Records,
				TTL:       common.IntPointer(rs.TTL),
				Activated: common.BoolPointer(rs.Activated),
			}
			d.Status.FQDN = rs.FQDN
		}); err != nil {
			l.Error(err, "Reconciler error")
			return ctrl.Result{RequeueAfter: GetNextBackoffDuration(&r.Backoff, req.NamespacedName, "Record-K8sUpdate-Update")}, nil
		}
		ResetBackoff(&r.Backoff, req.NamespacedName, "Record-K8sUpdate-Update")
	}

	// if RecordSetCreated/RecordSetRetrieved/RecordSetUpdated true, remove all conditions and mark Ready true
	if conditions.IsTrue(domain, v1alpha1.ConditionTypeRecordSetCreated) ||
		conditions.IsTrue(domain, v1alpha1.ConditionTypeRecordSetUpdated) ||
		conditions.IsTrue(domain, v1alpha1.ConditionTypeRecordSetRetrieved) {
		if err := domain.StatusUpdate(ctx, r.Client, func(d *v1alpha1.Domain) {
			conditions.Delete(d, v1alpha1.ConditionTypeRecordSetCreated)
			conditions.Delete(d, v1alpha1.ConditionTypeRecordSetUpdated)
			conditions.Delete(d, v1alpha1.ConditionTypeRecordSetRetrieved)
			conditions.MarkTrue(d, v1alpha1.ConditionTypeRecordSetReady)
		}); err != nil {
			l.Error(err, "Reconciler error")
		}
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DomainReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Domain{}).
		Complete(r)
}

func teardownRecordSet(ctx context.Context, service provider.Client, domain *v1alpha1.Domain) error {
	if domain.Status.Record != nil && len(domain.Status.Record.Id) > 0 {
		if err := service.Delete(ctx, domain.Status.Record.Id, domain.Status.Zone.Id); err != nil {
			return fmt.Errorf("can't teardown recordset: %w", err)
		}
	}
	return nil
}
