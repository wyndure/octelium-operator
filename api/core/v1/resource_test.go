package v1

import (
	"encoding/json"
	"testing"

	core "github.com/octelium/octelium/apis/main/corev1"
	"google.golang.org/protobuf/proto"
)

func TestProtobufJSONAndDeepCopy(t *testing.T) {
	input := []byte(`{"apiVersion":"core.octelium.wyndure.com/v1","kind":"OcteliumSecret","metadata":{"name":"sample","labels":{"team":"one"}},"octelium":{"metadata":{"displayName":"Sample"}},"spec":{"data":{"valueBytes":"aGVsbG8="}}}`)
	var resource OcteliumSecret
	if e := json.Unmarshal(input, &resource); e != nil {
		t.Fatal(e)
	}
	if string(resource.Spec.Data.GetValueBytes()) != "hello" {
		t.Fatal("protobuf oneof was not decoded")
	}
	var _ = resource.Spec
	clone := resource.DeepCopy()
	clone.Spec.Data.Type = &core.Secret_Spec_Data_Value{Value: "changed"}
	clone.Labels["team"] = "two"
	clone.Octelium.Metadata.DisplayName = "Other"
	if string(resource.Spec.Data.GetValueBytes()) != "hello" || resource.Labels["team"] != "one" || resource.Octelium.Metadata.DisplayName != "Sample" {
		t.Fatal("deep copy aliased protobuf or metadata state")
	}
	data, e := json.Marshal(&resource)
	if e != nil {
		t.Fatal(e)
	}
	var roundTrip OcteliumSecret
	if e = json.Unmarshal(data, &roundTrip); e != nil {
		t.Fatal(e)
	}
	if !proto.Equal(roundTrip.Spec, resource.Spec) {
		t.Fatal("protobuf JSON round trip lost data")
	}
}

func TestProtobufRejectsUnknownAndConflictingOneofs(t *testing.T) {
	for _, spec := range []string{`{"madeUpField":true}`, `{"data":{"value":"x","valueBytes":"eA=="}}`} {
		var resource OcteliumSecret
		if e := json.Unmarshal([]byte(`{"metadata":{"name":"sample"},"spec":`+spec+`}`), &resource); e == nil {
			t.Fatal("invalid protobuf spec accepted", spec)
		}
	}
}

func TestMetadataConfigMapRoundTrip(t *testing.T) {
	const configMapName = "sample.metadata"
	for _, resource := range Resources() {
		t.Run(resource.UpstreamKind(), func(t *testing.T) {
			input := []byte(`{"metadata":{"name":"sample"},"metadataConfigMap":"sample.metadata","spec":{}}`)
			if err := json.Unmarshal(input, resource); err != nil {
				t.Fatal(err)
			}
			if resource.MetadataConfigMapName() != configMapName {
				t.Fatal("lost ConfigMap name")
			}
			clone := resource.DeepCopyObject().(Resource)
			if clone.MetadataConfigMapName() != configMapName {
				t.Fatal("deep copy lost ConfigMap name")
			}
			encoded, err := json.Marshal(clone)
			if err != nil {
				t.Fatal(err)
			}
			var wire struct {
				MetadataConfigMap string                     `json:"metadataConfigMap"`
				Spec              map[string]json.RawMessage `json:"spec"`
			}
			if err = json.Unmarshal(encoded, &wire); err != nil {
				t.Fatal(err)
			}
			if wire.MetadataConfigMap != configMapName {
				t.Fatal("ConfigMap name missing from top level")
			}
			if _, found := wire.Spec["metadataConfigMap"]; found {
				t.Fatal("operator setting leaked into upstream spec")
			}
			if err = json.Unmarshal([]byte(`{"metadata":{"name":"sample"},"spec":{}}`), resource); err != nil {
				t.Fatal(err)
			}
			if resource.MetadataConfigMapName() != "" {
				t.Fatal("omitted field retained old value")
			}
		})
	}
}
