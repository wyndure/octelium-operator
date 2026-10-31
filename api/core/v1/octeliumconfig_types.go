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
// OcteliumConfig uses the upstream specification without a copied Go schema.
type OcteliumConfig struct {
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
	Spec   *upstream.Config_Spec `json:"spec"`
	Status ResourceStatus        `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:object:generate=false
type OcteliumConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OcteliumConfig `json:"items"`
}

func (o *OcteliumConfig) UpstreamKind() string {
	return "Config"
}

func (o *OcteliumConfig) UpstreamSpec() proto.Message {
	if o.Spec == nil {
		o.Spec = &upstream.Config_Spec{}
	}

	return o.Spec
}

func (o *OcteliumConfig) Extension() *ResourceOptions {
	return &o.Octelium
}

func (o *OcteliumConfig) SyncStatus() *ResourceStatus {
	return &o.Status
}

func (o *OcteliumConfig) MarshalJSON() ([]byte, error) {
	return marshalResource(o.TypeMeta, o.ObjectMeta, o.Octelium, o.OcteliumRef, o.UpstreamSpec(), o.MetadataConfigMap, o.Status)
}

func (o *OcteliumConfig) UnmarshalJSON(b []byte) error {
	o.Spec = &upstream.Config_Spec{}
	return unmarshalResource(b, &o.TypeMeta, &o.ObjectMeta, &o.Octelium, &o.OcteliumRef, o.Spec, &o.MetadataConfigMap, &o.Status)
}

func (o *OcteliumConfig) DeepCopyInto(out *OcteliumConfig) {
	*out = *o

	o.ObjectMeta.DeepCopyInto(&out.ObjectMeta)

	out.Octelium = o.Octelium.DeepCopy()
	out.Status = o.Status.DeepCopy()

	if o.Spec != nil {
		out.Spec = proto.Clone(o.Spec).(*upstream.Config_Spec)
	}
}

func (o *OcteliumConfig) DeepCopy() *OcteliumConfig {
	if o == nil {
		return nil
	}

	out := new(OcteliumConfig)

	o.DeepCopyInto(out)

	return out
}

func (o *OcteliumConfig) DeepCopyObject() runtime.Object {
	return o.DeepCopy()
}

func (o *OcteliumConfigList) DeepCopyInto(out *OcteliumConfigList) {
	*out = *o

	o.ListMeta.DeepCopyInto(&out.ListMeta)

	if o.Items != nil {
		out.Items = make([]OcteliumConfig, len(o.Items))
		for i := range o.Items {
			o.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

func (o *OcteliumConfigList) DeepCopy() *OcteliumConfigList {
	if o == nil {
		return nil
	}
	out := new(OcteliumConfigList)
	o.DeepCopyInto(out)
	return out
}

func (o *OcteliumConfigList) DeepCopyObject() runtime.Object {
	return o.DeepCopy()
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &OcteliumConfig{}, &OcteliumConfigList{})

		return nil
	})
}

func (o *OcteliumConfig) Reference() *connectionapi.OcteliumReference {
	return &o.OcteliumRef
}

func (o *OcteliumConfig) MetadataConfigMapName() string {
	return o.MetadataConfigMap
}
