package v1

import (
	upstream "github.com/octelium/octelium/apis/main/corev1"
	connectionapi "github.com/wyndure/octelium-operator/api/v1"
	"google.golang.org/protobuf/proto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=false
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type=='Ready')].status"
// OcteliumNamespace uses the upstream specification without a copied Go schema.
type OcteliumNamespace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// MetadataConfigMap names a ConfigMap in this CR's namespace for upstream metadata.
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	MetadataConfigMap string                          `json:"metadataConfigMap,omitempty"`
	OcteliumRef       connectionapi.OcteliumReference `json:"octeliumRef"`
	Octelium          ResourceOptions                 `json:"octelium,omitempty"`
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	Spec   *upstream.Namespace_Spec `json:"spec"`
	Status ResourceStatus           `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=false
type OcteliumNamespaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OcteliumNamespace `json:"items"`
}

func (o *OcteliumNamespace) UpstreamKind() string {
	return "Namespace"
}

func (o *OcteliumNamespace) UpstreamSpec() proto.Message {
	if o.Spec == nil {
		o.Spec = &upstream.Namespace_Spec{}
	}

	return o.Spec
}

func (o *OcteliumNamespace) Extension() *ResourceOptions {
	return &o.Octelium
}

func (o *OcteliumNamespace) SyncStatus() *ResourceStatus {
	return &o.Status
}

func (o *OcteliumNamespace) MarshalJSON() ([]byte, error) {
	return marshalResource(o.TypeMeta, o.ObjectMeta, o.Octelium, o.OcteliumRef, o.UpstreamSpec(), o.MetadataConfigMap, o.Status)
}

func (o *OcteliumNamespace) UnmarshalJSON(b []byte) error {
	o.Spec = &upstream.Namespace_Spec{}

	return unmarshalResource(b, &o.TypeMeta, &o.ObjectMeta, &o.Octelium, &o.OcteliumRef, o.Spec, &o.MetadataConfigMap, &o.Status)
}

func (o *OcteliumNamespace) DeepCopyInto(out *OcteliumNamespace) {
	*out = *o

	o.ObjectMeta.DeepCopyInto(&out.ObjectMeta)

	out.Octelium = o.Octelium.DeepCopy()
	out.Status = o.Status.DeepCopy()

	if o.Spec != nil {
		out.Spec = proto.Clone(o.Spec).(*upstream.Namespace_Spec)
	}
}

func (o *OcteliumNamespace) DeepCopy() *OcteliumNamespace {
	if o == nil {
		return nil
	}

	out := new(OcteliumNamespace)

	o.DeepCopyInto(out)

	return out
}

func (o *OcteliumNamespace) DeepCopyObject() runtime.Object {
	return o.DeepCopy()
}

func (o *OcteliumNamespaceList) DeepCopyInto(out *OcteliumNamespaceList) {
	*out = *o

	o.ListMeta.DeepCopyInto(&out.ListMeta)

	if o.Items != nil {
		out.Items = make([]OcteliumNamespace, len(o.Items))
		for i := range o.Items {
			o.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

func (o *OcteliumNamespaceList) DeepCopy() *OcteliumNamespaceList {
	if o == nil {
		return nil
	}

	out := new(OcteliumNamespaceList)

	o.DeepCopyInto(out)

	return out
}

func (o *OcteliumNamespaceList) DeepCopyObject() runtime.Object {
	return o.DeepCopy()
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &OcteliumNamespace{}, &OcteliumNamespaceList{})

		return nil
	})
}

func (o *OcteliumNamespace) Reference() *connectionapi.OcteliumReference {
	return &o.OcteliumRef
}

func (o *OcteliumNamespace) MetadataConfigMapName() string {
	return o.MetadataConfigMap
}
