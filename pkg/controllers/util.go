package controllers

import (
	"fmt"
	"github.com/sokdak/dns-ingress/api/v1alpha1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/flowcontrol"
	"time"
)

func GenerateReconcileInformationLabelKeySet(nsn types.NamespacedName) []string {
	return []string{"namespace", nsn.Namespace, "name", nsn.Name}
}

func GenerateReconcileInformationLabelKeySetByIngress(ingress *v1.Ingress) []string {
	return []string{"namespace", ingress.Namespace, "name", ingress.Name}
}

func GenerateReconcileInformationLabelKeySetByDomain(domain *v1alpha1.Domain) []string {
	return []string{"namespace", domain.Namespace, "name", domain.Name}
}

func GetNextBackoffDuration(backoff *flowcontrol.Backoff, req types.NamespacedName, funcName string) time.Duration {
	backoffKey := fmt.Sprintf("%s/%s/%s", req.Namespace, req.Name, funcName)
	backoff.Next(backoffKey, time.Now())
	return backoff.Get(backoffKey)
}

func ResetBackoff(backoff *flowcontrol.Backoff, req types.NamespacedName, funcName string) {
	backoffKey := fmt.Sprintf("%s/%s/%s", req.Namespace, req.Name, funcName)
	backoff.Reset(backoffKey)
}
