package controller

import (
	"encoding/json"
	"testing"

	upstreammeta "github.com/octelium/octelium/apis/main/metav1"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestMetadataDataContainsEveryField(t *testing.T) {
	md := &upstreammeta.Metadata{Name: "default", Uid: "example-uid", Labels: map[string]string{"team": "platform"}, Tags: []string{"first", "second"}, IsSystem: true}

	data, err := metadataData(md)
	if err != nil {
		t.Fatal(err)
	}

	raw, err := (protojson.MarshalOptions{EmitUnpopulated: true}).Marshal(md)
	if err != nil {
		t.Fatal(err)
	}

	var want map[string]json.RawMessage
	if err = json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}

	if len(data) != len(want) {
		t.Fatal("metadata fields missing")
	}

	// Decode ConfigMap values back to protobuf JSON to ensure no field was lost.
	for key, expected := range want {
		value, ok := data[key]
		if !ok {
			t.Fatalf("missing %s", key)
		}

		var text string

		if len(expected) > 0 && expected[0] == '"' {
			if err = json.Unmarshal(expected, &text); err != nil {
				t.Fatal(err)
			}

			if value != text {
				t.Fatalf("string field %s differs", key)
			}
		} else if value != string(expected) {
			t.Fatalf("JSON field %s differs", key)
		}
	}

	if data["isSystem"] != "true" || data["name"] != "default" || data["uid"] != "example-uid" {
		t.Fatal("scalar metadata changed")
	}
}
