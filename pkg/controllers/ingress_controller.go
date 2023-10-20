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
	"go.uber.org/multierr"
	v1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
)

// IngressReconciler reconciles a Domain object
type IngressReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	DefaultDNSProvider     string
	DefaultIngressEndpoint string
	DefaultDomainZone      string
}

//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingress,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingress/status,verbs=get;update;patch

func (r *IngressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("start reconcile", GenerateReconcileInformationLabelKeySet(req.NamespacedName))
	defer l.Info("end reconcile", GenerateReconcileInformationLabelKeySet(req.NamespacedName))

	// get ingress object
	ingressObj := &v1.Ingress{}
	if err := r.Client.Get(ctx, req.NamespacedName, ingressObj); err != nil {
		// if ingress not found, kube-gc will delete all related domain records
		if k8serrors.IsNotFound(err) {
			l.Info("ignoring since ingress object has been deleted", GenerateReconcileInformationLabelKeySet(req.NamespacedName))
			return ctrl.Result{}, nil
		}
	}
	l.Info("got ingress rules", "vhosts", len(ingressObj.Spec.Rules),
		GenerateReconcileInformationLabelKeySet(req.NamespacedName))

	// get domain resource by ingress labels
	domainObjList := &v1alpha1.DomainList{}
	objListOpts := []client.ListOption{
		client.MatchingLabels{LabelKeyDomainMappedIngressName: ingressObj.Name},
		client.InNamespace(ingressObj.Namespace),
	}
	if err := r.Client.List(ctx, domainObjList, objListOpts...); err != nil {
		return ctrl.Result{}, fmt.Errorf("can't list domain objects: %w", err)
	}
	l.Info("listed actual domains", "count", len(domainObjList.Items),
		GenerateReconcileInformationLabelKeySet(req.NamespacedName))

	// generating canonical host map with rules
	actualHosts := map[string]*v1alpha1.Domain{}
	for _, domain := range domainObjList.Items {
		hostKey := fmt.Sprintf("%s.%s", domain.Spec.Name, domain.Spec.Zone)
		actualHosts[hostKey] = domain.DeepCopy()
	}

	// multierr for add/delete operations
	errs := multierr.Combine(nil)

	// sync for new vhost and existing entries
	rulesHosts := map[string]bool{}
	for _, rule := range ingressObj.Spec.Rules {
		rulesHosts[rule.Host] = true

		domainObj, ok := actualHosts[rule.Host]
		if !ok {
			// if not exist, create a new domain resource
			err := r.handleDomainCreation(ctx, ingressObj, rule.Host)
			errs = multierr.Append(errs, err)
			continue
		}

		// if exists, update the domain resource
		err := r.handleDomainUpdate(ctx, ingressObj, domainObj)
		errs = multierr.Append(errs, err)
	}

	// sync for dangling entries
	willRemovedCanonicalHosts := make([]string, 0)
	for k := range actualHosts {
		if !rulesHosts[k] {
			willRemovedCanonicalHosts = append(willRemovedCanonicalHosts, k)
		}
	}

	// if it has entries to be removed, iterates and delete it
	if len(willRemovedCanonicalHosts) > 0 {
		for _, host := range willRemovedCanonicalHosts {
			if err := r.Delete(ctx, actualHosts[host]); err != nil {
				// if already deleted, continue iterating
				if k8serrors.IsNotFound(err) {
					continue
				}
				l.Error(err, "occurred error while deleting domain resource",
					"vhost", host,
					GenerateReconcileInformationLabelKeySet(req.NamespacedName))
				errs = multierr.Append(errs, err)
			}
			// delete key from actualHosts map
			delete(actualHosts, host)
			l.Info("deleted dangling domain",
				"vhost", host,
				"name", actualHosts[host].Name,
				"namespace", actualHosts[host].Namespace,
				GenerateReconcileInformationLabelKeySet(req.NamespacedName))
		}
	}

	// if reconcile has error, retry the reconcile again
	if len(multierr.Errors(errs)) > 0 {
		return ctrl.Result{}, errs
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Ingress{}).
		Owns(&v1alpha1.Domain{}).
		Complete(r)
}

func (r *IngressReconciler) handleDomainCreation(ctx context.Context, ingress *v1.Ingress, vhost string) error {
	l := log.FromContext(ctx)

	// get provider and ingress endpoint from annotation
	provider, ok := ingress.Annotations[AnnotationKeyIngressDnsProvider]
	if !ok {
		provider = r.DefaultDNSProvider
	}

	ingressEp, ok := ingress.Annotations[AnnotationKeyIngressEndpoint]
	if !ok {
		ingressEp = r.DefaultIngressEndpoint
	}

	domainZone, ok := ingress.Annotations[AnnotationKeyDomainZone]
	if !ok {
		domainZone = r.DefaultDomainZone
	}

	// split the vhost into name and zone
	name := strings.Split(vhost, domainZone)[0]

	// prototyping object
	newDomain := &v1alpha1.Domain{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", ingress.Name, common.GenerateMD5Hash(vhost)),
			Namespace: ingress.Namespace,
			Labels:    map[string]string{LabelKeyDomainMappedIngressName: ingress.Name},
		},
		Spec: v1alpha1.DomainSpec{
			Provider: provider,
			Name:     name,
			Zone:     domainZone,
			Records:  []string{ingressEp},
		},
	}

	// set controller reference
	_ = controllerutil.SetControllerReference(ingress, newDomain, r.Scheme)

	// create object
	if err := r.Client.Create(ctx, newDomain); err != nil {
		return fmt.Errorf("can't create domain: %w", err)
	}

	l.Info("created domain resource",
		"vhost", vhost, "provider", provider, "ingressEp", ingressEp, "zone", domainZone,
		GenerateReconcileInformationLabelKeySetByIngress(ingress))
	return nil
}

func (r *IngressReconciler) handleDomainUpdate(ctx context.Context, ingress *v1.Ingress, domain *v1alpha1.Domain) error {
	l := log.FromContext(ctx)

	// get provider and ingress endpoint from annotation
	provider, ok := ingress.Annotations[AnnotationKeyIngressDnsProvider]
	if !ok {
		provider = r.DefaultDNSProvider
	}

	ingressEp, ok := ingress.Annotations[AnnotationKeyIngressEndpoint]
	if !ok {
		ingressEp = r.DefaultIngressEndpoint
	}

	domainZone, ok := ingress.Annotations[AnnotationKeyDomainZone]
	if !ok {
		domainZone = r.DefaultDomainZone
	}

	// update object with RetryOnConflict
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		// get object
		tmpDomainObj := &v1alpha1.Domain{}
		tmpDomainNamespacedName := types.NamespacedName{Namespace: domain.Namespace, Name: domain.Name}
		if err := r.Client.Get(ctx, tmpDomainNamespacedName, tmpDomainObj); err != nil {
			return fmt.Errorf("can't RetryOnConflict; can't get object: %w", err)
		}

		// force apply
		modifiedTmpDomainObj := tmpDomainObj.DeepCopy()
		if modifiedTmpDomainObj.Spec.Provider != provider {
			modifiedTmpDomainObj.Spec.Provider = provider
		}

		if len(modifiedTmpDomainObj.Spec.Records) > 0 && modifiedTmpDomainObj.Spec.Records[0] != ingressEp {
			modifiedTmpDomainObj.Spec.Records = []string{ingressEp}
		}

		if modifiedTmpDomainObj.Spec.Zone != domainZone {
			modifiedTmpDomainObj.Spec.Zone = domainZone
		}

		// update
		if !reflect.DeepEqual(modifiedTmpDomainObj, tmpDomainObj) {
			if err := r.Client.Update(ctx, domain); err != nil {
				return fmt.Errorf("can't RetryOnConflict; can't update object: %w", err)
			}
			l.Info("updated domain resource",
				"provider", fmt.Sprintf("%s -> %s", tmpDomainObj.Spec.Provider, modifiedTmpDomainObj.Spec.Provider),
				"ingressEp", fmt.Sprintf("%s -> %s", tmpDomainObj.Spec.Records[0], modifiedTmpDomainObj.Spec.Records[0]),
				"zone", fmt.Sprintf("%s -> %s", tmpDomainObj.Spec.Zone, modifiedTmpDomainObj.Spec.Zone),
				GenerateReconcileInformationLabelKeySetByIngress(ingress))
		}

		return nil
	})
}
