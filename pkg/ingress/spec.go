package ingress

import (
	networkingv1 "k8s.io/api/networking/v1"
)

type IngressController interface{
	Name() string
	Render() (string, error)
}

type NginxRenderer struct {
	ingress *networkingv1.Ingress
}
