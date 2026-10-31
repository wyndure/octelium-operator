package controller

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	upstreammeta "github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	api "github.com/wyndure/octelium-operator/api/core/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"k8s.io/apimachinery/pkg/util/validation"
)

func desiredResource(obj api.Resource, current umetav1.ResourceObjectI) (umetav1.ResourceObjectI, error) {
	desired, e := ucorev1.NewObject(obj.UpstreamKind())
	if e != nil {
		return nil, e
	}

	md := &upstreammeta.Metadata{}
	if obj.Extension().Metadata != nil {
		md = proto.Clone(obj.Extension().Metadata).(*upstreammeta.Metadata)
	}

	md.Name = obj.GetName()
	md.Labels = cloneMap(obj.GetLabels())
	md.Annotations = publicAnnotations(obj.GetAnnotations())

	if current != nil {
		merged := mergeFields(editableFields(md), editableFields(current.GetMetadata()), obj.SyncStatus().LastSyncedMetadata)
		if e = applyFields(md, merged); e != nil {
			return nil, e
		}
	}

	if e = validateNativeMetadata(md); e != nil {
		return nil, e
	}

	spec, e := protojson.Marshal(obj.UpstreamSpec())
	if e != nil {
		return nil, e
	}

	metadata, e := protojson.Marshal(md)
	if e != nil {
		return nil, e
	}

	wire := map[string]json.RawMessage{"apiVersion": json.RawMessage(`"core/v1"`), "kind": json.RawMessage(fmt.Sprintf("%q", obj.UpstreamKind())), "metadata": metadata, "spec": spec}
	if current == nil && obj.Extension().InitialStatus != nil {
		wire["status"] = obj.Extension().InitialStatus.Raw
	}

	if obj.Extension().Data != nil {
		wire["data"] = obj.Extension().Data.Raw
	}

	b, e := json.Marshal(wire)
	if e != nil {
		return nil, e
	}

	if e = protojson.Unmarshal(b, desired); e != nil {
		return nil, e
	}

	return desired, nil
}

func publicAnnotations(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		if !strings.HasPrefix(k, internalPrefix) && k != "kubectl.kubernetes.io/last-applied-configuration" {
			out[k] = v
		}
	}

	return out
}

func cloneMap(in map[string]string) map[string]string {
	out := map[string]string{}

	maps.Copy(out, in)

	return out
}

// Editable fields are flattened so concurrent changes to unrelated metadata keys survive.
func editableFields(md *upstreammeta.Metadata) map[string]string {
	out := map[string]string{}
	if md == nil {
		return out
	}

	for k, v := range md.Labels {
		out["label/"+k] = v
	}

	for k, v := range publicAnnotations(md.Annotations) {
		out["annotation/"+k] = v
	}

	for k, v := range map[string]string{"displayName": md.DisplayName, "description": md.Description, "picURL": md.PicURL} {
		if v != "" {
			out[k] = v
		}
	}

	if len(md.Tags) > 0 {
		b, _ := json.Marshal(md.Tags)
		out["tags"] = string(b)
	}

	return out
}

func mergeFields(local, remote, base map[string]string) map[string]string {
	out := cloneMap(remote)

	keys := map[string]bool{}
	for k := range local {
		keys[k] = true
	}

	for k := range base {
		keys[k] = true
	}

	for k := range keys {
		v, has := local[k]
		old, was := base[k]
		if has != was || v != old {
			if has {
				out[k] = v
			} else {
				delete(out, k)
			}
		}
	}

	return out
}

func applyFields(md *upstreammeta.Metadata, fields map[string]string) error {
	md.Labels = map[string]string{}
	md.Annotations = map[string]string{}
	md.DisplayName = ""
	md.Description = ""
	md.PicURL = ""
	md.Tags = nil

	for k, v := range fields {
		switch {
		case strings.HasPrefix(k, "label/"):
			md.Labels[strings.TrimPrefix(k, "label/")] = v
		case strings.HasPrefix(k, "annotation/"):
			md.Annotations[strings.TrimPrefix(k, "annotation/")] = v
		case k == "displayName":
			md.DisplayName = v
		case k == "description":
			md.Description = v
		case k == "picURL":
			md.PicURL = v
		case k == "tags":
			if e := json.Unmarshal([]byte(v), &md.Tags); e != nil {
				return e
			}
		}
	}

	return nil
}

func validateNativeMetadata(md *upstreammeta.Metadata) error {
	for k, v := range md.Labels {
		if len(validation.IsQualifiedName(k)) > 0 || len(validation.IsValidLabelValue(v)) > 0 {
			return fmt.Errorf("octelium label %q cannot be represented as Kubernetes metadata", k)
		}
	}

	for k := range md.Annotations {
		if len(validation.IsQualifiedName(k)) > 0 {
			return fmt.Errorf("octelium annotation %q cannot be represented as Kubernetes metadata", k)
		}
	}

	return nil
}

func mirrorMetadata(obj api.Resource, md *upstreammeta.Metadata) error {
	if e := validateNativeMetadata(md); e != nil {
		return e
	}

	obj.SetLabels(cloneMap(md.Labels))

	annotations := cloneMap(md.Annotations)

	for k, v := range obj.GetAnnotations() {
		if strings.HasPrefix(k, internalPrefix) || k == "kubectl.kubernetes.io/last-applied-configuration" {
			annotations[k] = v
		}
	}

	obj.SetAnnotations(annotations)

	external := proto.Clone(md).(*upstreammeta.Metadata)

	external.Name = ""
	external.Labels = nil
	external.Annotations = nil

	obj.Extension().Metadata = external

	return nil
}
