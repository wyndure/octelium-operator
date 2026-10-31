package v1

import (
	"encoding/json"
	"maps"

	connectionapi "github.com/wyndure/octelium-operator/api/v1"

	upstreammeta "github.com/octelium/octelium/apis/main/metav1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:object:generate=false
// Resource is implemented by every typed ResourceOptions CR.
type Resource interface {
	client.Object
	Reference() *connectionapi.OcteliumReference
	UpstreamKind() string
	MetadataConfigMapName() string
	UpstreamSpec() proto.Message
	Extension() *ResourceOptions
	SyncStatus() *ResourceStatus
}

// +kubebuilder:object:generate=false
// ResourceOptions holds metadata without a Kubernetes equivalent and create-only bootstrap state.
type ResourceOptions struct {
	// Native name, labels and annotations belong to Kubernetes metadata instead.
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	Metadata *upstreammeta.Metadata `json:"metadata,omitempty"`
	// InitialStatus is only used when creating a resource. Runtime status is observed, never overwritten.
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	InitialStatus  *runtime.RawExtension `json:"initialStatus,omitempty"`
	Data           *runtime.RawExtension `json:"data,omitempty"`
	DataFromSecret *SecretDataSource     `json:"dataFromSecret,omitempty"`
	// DeletionPolicy defaults to Delete; Retain leaves the external resource in place.
	// +kubebuilder:validation:Enum=Delete;Retain
	DeletionPolicy string `json:"deletionPolicy,omitempty"`
}

func (o ResourceOptions) MarshalJSON() ([]byte, error) {
	var md json.RawMessage

	if o.Metadata != nil {
		b, e := protojson.Marshal(o.Metadata)
		if e != nil {
			return nil, e
		}
		md = b
	}

	return json.Marshal(struct {
		Metadata       json.RawMessage       `json:"metadata,omitempty"`
		InitialStatus  *runtime.RawExtension `json:"initialStatus,omitempty"`
		Data           *runtime.RawExtension `json:"data,omitempty"`
		DataFromSecret *SecretDataSource     `json:"dataFromSecret,omitempty"`
		DeletionPolicy string                `json:"deletionPolicy,omitempty"`
	}{md, o.InitialStatus, o.Data, o.DataFromSecret, o.DeletionPolicy})
}

func (o *ResourceOptions) UnmarshalJSON(b []byte) error {
	var w struct {
		Metadata       json.RawMessage       `json:"metadata"`
		InitialStatus  *runtime.RawExtension `json:"initialStatus"`
		Data           *runtime.RawExtension `json:"data"`
		DataFromSecret *SecretDataSource     `json:"dataFromSecret"`
		DeletionPolicy string                `json:"deletionPolicy"`
	}

	if e := json.Unmarshal(b, &w); e != nil {
		return e
	}

	o.Metadata = nil
	if len(w.Metadata) > 0 && string(w.Metadata) != "null" {
		o.Metadata = &upstreammeta.Metadata{}
		if e := protojson.Unmarshal(w.Metadata, o.Metadata); e != nil {
			return e
		}
	}

	o.DataFromSecret = w.DataFromSecret
	o.Data = w.Data
	o.InitialStatus = w.InitialStatus
	o.DeletionPolicy = w.DeletionPolicy

	return nil
}

func (o ResourceOptions) DeepCopy() ResourceOptions {
	out := o

	if o.DataFromSecret != nil {
		ref := *o.DataFromSecret
		out.DataFromSecret = &ref
	}

	if o.Metadata != nil {
		out.Metadata = proto.Clone(o.Metadata).(*upstreammeta.Metadata)
	}

	if o.Data != nil {
		out.Data = o.Data.DeepCopy()
	}

	if o.InitialStatus != nil {
		out.InitialStatus = o.InitialStatus.DeepCopy()
	}

	return out
}

// +kubebuilder:object:generate=false
type ResourceStatus struct {
	DataDigest         string `json:"dataDigest,omitempty"`
	ConnectionUID      string `json:"connectionUID,omitempty"`
	ConnectionKey      string `json:"connectionKey,omitempty"`
	ConnectionEndpoint string `json:"connectionEndpoint,omitempty"`
	ObservedGeneration int64  `json:"observedGeneration,omitempty"`
	ExternalUID        string `json:"externalUID,omitempty"`
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// LastSyncedMetadata is the three-way merge base. It never contains a resource spec or Secret data.
	LastSyncedMetadata map[string]string `json:"lastSyncedMetadata,omitempty"`
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	UpstreamStatus *runtime.RawExtension `json:"upstreamStatus,omitempty"`
}

func (s ResourceStatus) DeepCopy() ResourceStatus {
	out := s

	out.Conditions = append([]metav1.Condition(nil), s.Conditions...)

	if s.LastSyncedMetadata != nil {
		out.LastSyncedMetadata = map[string]string{}

		maps.Copy(out.LastSyncedMetadata, s.LastSyncedMetadata)
	}

	if s.UpstreamStatus != nil {
		out.UpstreamStatus = s.UpstreamStatus.DeepCopy()
	}

	return out
}

func marshalResource(tm metav1.TypeMeta, md metav1.ObjectMeta, o ResourceOptions, ref connectionapi.OcteliumReference, spec proto.Message, configMapName string, status ResourceStatus) ([]byte, error) {
	b, e := protojson.Marshal(spec)
	if e != nil {
		return nil, e
	}

	return json.Marshal(struct {
		metav1.TypeMeta   `json:",inline"`
		Metadata          metav1.ObjectMeta               `json:"metadata"`
		ResourceOptions   ResourceOptions                 `json:"octelium,omitempty"`
		OcteliumRef       connectionapi.OcteliumReference `json:"octeliumRef"`
		MetadataConfigMap string                          `json:"metadataConfigMap,omitempty"`
		Spec              json.RawMessage                 `json:"spec"`
		Status            ResourceStatus                  `json:"status,omitempty"`
	}{tm, md, o, ref, configMapName, b, status})
}

func unmarshalResource(b []byte, tm *metav1.TypeMeta, md *metav1.ObjectMeta, o *ResourceOptions, ref *connectionapi.OcteliumReference, spec proto.Message, configMapName *string, status *ResourceStatus) error {
	var w struct {
		metav1.TypeMeta   `json:",inline"`
		Metadata          metav1.ObjectMeta               `json:"metadata"`
		ResourceOptions   ResourceOptions                 `json:"octelium"`
		OcteliumRef       connectionapi.OcteliumReference `json:"octeliumRef"`
		MetadataConfigMap string                          `json:"metadataConfigMap,omitempty"`
		Spec              json.RawMessage                 `json:"spec"`
		Status            ResourceStatus                  `json:"status"`
	}

	if e := json.Unmarshal(b, &w); e != nil {
		return e
	}

	*tm = w.TypeMeta
	*md = w.Metadata
	*o = w.ResourceOptions
	*ref = w.OcteliumRef
	*status = w.Status
	*configMapName = w.MetadataConfigMap

	if len(w.Spec) == 0 || string(w.Spec) == "null" {
		return nil
	}

	return protojson.Unmarshal(w.Spec, spec)
}

// +kubebuilder:object:generate=false
// SecretDataSource keeps sensitive values in native Kubernetes Secrets.
type SecretDataSource struct {
	// Namespace defaults to the managed resource namespace.
	// +optional
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
	Key       string `json:"key"`
}
