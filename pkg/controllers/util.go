package controllers

import (
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
)

func GenerateReconcileInformationLabelKeySet(nsn types.NamespacedName) []string {
	return []string{"namespace", nsn.Namespace, "name", nsn.Name}
}

func GenerateReconcileInformationLabelKeySetByIngress(ingress *v1.Ingress) []string {
	return []string{"namespace", ingress.Namespace, "name", ingress.Name}
}
