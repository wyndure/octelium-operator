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
// OcteliumPolicy uses the upstream specification without a copied Go schema.
type OcteliumPolicy struct {
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
	Spec   *upstream.Policy_Spec `json:"spec"`
	Status ResourceStatus        `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=false
type OcteliumPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OcteliumPolicy `json:"items"`
}

func (o *OcteliumPolicy) UpstreamKind() string {
	return "Policy"
}

func (o *OcteliumPolicy) UpstreamSpec() proto.Message {
	if o.Spec == nil {
		o.Spec = &upstream.Policy_Spec{}
	}

	return o.Spec
}

func (o *OcteliumPolicy) Extension() *ResourceOptions {
	return &o.Octelium
}

func (o *OcteliumPolicy) SyncStatus() *ResourceStatus {
	return &o.Status
}

func (o *OcteliumPolicy) MarshalJSON() ([]byte, error) {
	return marshalResource(o.TypeMeta, o.ObjectMeta, o.Octelium, o.OcteliumRef, o.UpstreamSpec(), o.MetadataConfigMap, o.Status)
}

func (o *OcteliumPolicy) UnmarshalJSON(b []byte) error {
	o.Spec = &upstream.Policy_Spec{}

	return unmarshalResource(b, &o.TypeMeta, &o.ObjectMeta, &o.Octelium, &o.OcteliumRef, o.Spec, &o.MetadataConfigMap, &o.Status)
}

func (o *OcteliumPolicy) DeepCopyInto(out *OcteliumPolicy) {
	*out = *o

	o.ObjectMeta.DeepCopyInto(&out.ObjectMeta)

	out.Octelium = o.Octelium.DeepCopy()
	out.Status = o.Status.DeepCopy()

	if o.Spec != nil {
		out.Spec = proto.Clone(o.Spec).(*upstream.Policy_Spec)
	}
}

func (o *OcteliumPolicy) DeepCopy() *OcteliumPolicy {
	if o == nil {
		return nil
	}

	out := new(OcteliumPolicy)

	o.DeepCopyInto(out)

	return out
}

func (o *OcteliumPolicy) DeepCopyObject() runtime.Object {
	return o.DeepCopy()
}

func (o *OcteliumPolicyList) DeepCopyInto(out *OcteliumPolicyList) {
	*out = *o

	o.ListMeta.DeepCopyInto(&out.ListMeta)

	if o.Items != nil {
		out.Items = make([]OcteliumPolicy, len(o.Items))
		for i := range o.Items {
			o.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

func (o *OcteliumPolicyList) DeepCopy() *OcteliumPolicyList {
	if o == nil {
		return nil
	}

	out := new(OcteliumPolicyList)

	o.DeepCopyInto(out)

	return out
}

func (o *OcteliumPolicyList) DeepCopyObject() runtime.Object {
	return o.DeepCopy()
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &OcteliumPolicy{}, &OcteliumPolicyList{})

		return nil
	})
}

func (o *OcteliumPolicy) Reference() *connectionapi.OcteliumReference {
	return &o.OcteliumRef
}

func (o *OcteliumPolicy) MetadataConfigMapName() string {
	return o.MetadataConfigMap
}
