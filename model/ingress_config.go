// Package model contains common data structures between the controller and pomerium config reconciler
package model

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// TLSCustomCASecret replaces https://pomerium.io/reference/#tls-custom-certificate-authority
	TLSCustomCASecret = "tls_custom_ca_secret"
	// TLSClientSecret replaces https://pomerium.io/reference/#tls-client-certificate
	TLSClientSecret = "tls_client_secret"
	// TLSDownstreamClientCASecret replaces https://pomerium.io/reference/#tls-downstream-client-certificate-authority
	TLSDownstreamClientCASecret = "tls_downstream_client_ca_secret"
	// SecureUpstream indicate that service communication should happen over HTTPS
	SecureUpstream = "secure_upstream"
)

// IngressConfig represents ingress and all other required resources
type IngressConfig struct {
	AnnotationPrefix string
	*networkingv1.Ingress
	Secrets  map[types.NamespacedName]*corev1.Secret
	Services map[types.NamespacedName]*corev1.Service
}

func (ic *IngressConfig) IsSecureUpstream() bool {
	return strings.ToLower(ic.Ingress.Annotations[fmt.Sprintf("%s/secure_upstream", ic.AnnotationPrefix)]) == "true"
}

// GetServicePortByName returns service named port
func (ic *IngressConfig) GetServicePortByName(name types.NamespacedName, port string) (int32, error) {
	svc, ok := ic.Services[name]
	if !ok {
		return 0, fmt.Errorf("service %s was not pre-fetched, this is a bug", name.String())
	}

	for _, servicePort := range svc.Spec.Ports {
		if servicePort.Name == port {
			return servicePort.Port, nil
		}
	}

	return 0, fmt.Errorf("could not find port %s on service %s", port, name.String())
}

// TLSCert represents a parsed TLS secret
type TLSCert struct {
	Key  []byte
	Cert []byte
}

// ParseTLSCerts decodes K8s TLS secret
func (ic *IngressConfig) ParseTLSCerts() ([]*TLSCert, error) {
	certs := make([]*TLSCert, 0, len(ic.Ingress.Spec.TLS))

	for _, tls := range ic.Ingress.Spec.TLS {
		secret := ic.Secrets[types.NamespacedName{Namespace: ic.Ingress.Namespace, Name: tls.SecretName}]
		if secret == nil {
			return nil, fmt.Errorf("secret=%s, but the secret wasn't fetched. this is a bug", tls.SecretName)
		}
		if secret.Type != corev1.SecretTypeTLS {
			return nil, fmt.Errorf("secret=%s, expected type %s, got %s", tls.SecretName, corev1.SecretTypeTLS, secret.Type)
		}
		certs = append(certs, &TLSCert{
			Key:  secret.Data[corev1.TLSPrivateKeyKey],
			Cert: secret.Data[corev1.TLSCertKey],
		})
	}

	return certs, nil
}
