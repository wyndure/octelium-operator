package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// OcteliumReference identifies a namespaced connection. An omitted namespace means the resource's own namespace.
type OcteliumReference struct {
	// +kubebuilder:default="octelium.wyndure.com/v1"
	// +kubebuilder:validation:Enum="octelium.wyndure.com/v1"
	APIVersion string `json:"apiVersion,omitempty"`
	// +kubebuilder:default=Octelium
	// +kubebuilder:validation:Enum=Octelium
	Kind string `json:"kind,omitempty"`
	// +kubebuilder:validation:MinLength=1
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

type ResourceKind struct {
	// +kubebuilder:validation:MinLength=1
	Group string `json:"group"`
	// +kubebuilder:validation:MinLength=1
	Kind string `json:"kind"`
}

// ResourceNamespaces follows Gateway API namespace selection semantics.
// +kubebuilder:validation:XValidation:rule="self.from != 'Selector' || has(self.selector)",message="Selector requires selector"
// +kubebuilder:validation:XValidation:rule="!has(self.selector) || self.from == 'Selector'",message="selector requires from Selector"
type ResourceNamespaces struct {
	// +kubebuilder:default=Same
	// +kubebuilder:validation:Enum=Same;All;Selector
	From     string                `json:"from,omitempty"`
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
}

type AllowedResources struct {
	// Namespaces defaults to Same. Cross-namespace access must be explicitly granted here by the connection owner.
	Namespaces *ResourceNamespaces `json:"namespaces,omitempty"`
	// Kinds optionally restricts which managed CR kinds may use this connection. Empty permits all supported kinds.
	Kinds []ResourceKind `json:"kinds,omitempty"`
}

// ConnectionCredentials selects SPIFFE mTLS or explicitly unauthenticated plaintext gRPC.
// +kubebuilder:validation:XValidation:rule="self.type != 'SPIFFE' || (has(self.trustDomain) && size(self.trustDomain) > 0)",message="SPIFFE requires trustDomain"
// +kubebuilder:validation:XValidation:rule="self.type != 'Insecure' || ((!has(self.trustDomain) || size(self.trustDomain) == 0) && (!has(self.socketPath) || size(self.socketPath) == 0))",message="Insecure does not use trustDomain or socketPath"
type ConnectionCredentials struct {
	// Type defaults to SPIFFE. Insecure disables both authentication and transport encryption.
	// +kubebuilder:default=SPIFFE
	// +kubebuilder:validation:Enum=SPIFFE;Insecure
	Type        string `json:"type,omitempty"`
	TrustDomain string `json:"trustDomain,omitempty"`
	// SocketPath is the SPIFFE Workload API socket; defaults to /run/spire/sockets/spire-agent.sock in SPIFFE mode.
	SocketPath string `json:"socketPath,omitempty"`
}

type OcteliumSpec struct {
	// Endpoint is a resource-server DNS name/IP and port, without a URL scheme.
	// +kubebuilder:validation:MinLength=1
	Endpoint         string                `json:"endpoint"`
	Credentials      ConnectionCredentials `json:"credentials"`
	AllowedResources *AllowedResources     `json:"allowedResources,omitempty"`
}

type OcteliumStatus struct {
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,path=octeliums
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type=='Ready')].status"
type Octelium struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              OcteliumSpec   `json:"spec"`
	Status            OcteliumStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type OcteliumList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Octelium `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &Octelium{}, &OcteliumList{})

		return nil
	})
}

const (
	CredentialsSPIFFE   = "SPIFFE"
	CredentialsInsecure = "Insecure"
)
