package controller

import (
	"reflect"
	"testing"

	core "github.com/octelium/octelium/apis/main/corev1"
	meta "github.com/octelium/octelium/apis/main/metav1"
	api "github.com/wyndure/octelium-operator/api/core/v1"
)

func TestMetadataIsIndependentOfKubernetes(t *testing.T) {
	obj := &api.OcteliumNamespace{
		Name:        "kubernetes-name",
		Labels:      map[string]string{"app.kubernetes.io/name": "gitops"},
		Annotations: map[string]string{"argocd.argoproj.io/tracking-id": "not-an-octelium-annotation"},
		Spec:        &core.Namespace_Spec{},
		Octelium:    api.ResourceOptions{Metadata: &meta.Metadata{Name: "upstream-name", Labels: map[string]string{"team": "upstream"}, Annotations: map[string]string{"note": "upstream"}}},
	}
	desired, err := desiredResource(obj, nil)
	if err != nil {
		t.Fatal(err)
	}
	md := desired.GetMetadata()
	if md.Name != "upstream-name" || !reflect.DeepEqual(md.Labels, obj.Octelium.Metadata.Labels) || !reflect.DeepEqual(md.Annotations, obj.Octelium.Metadata.Annotations) {
		t.Fatalf("unexpected upstream metadata: %v", md)
	}
	before := obj.DeepCopy()
	md.Labels["remote"] = "updated"
	if err := mirrorMetadata(obj, md); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(obj.Labels, before.Labels) || !reflect.DeepEqual(obj.Annotations, before.Annotations) || obj.Name != before.Name {
		t.Fatal("Kubernetes metadata was changed by upstream synchronization")
	}
	if obj.Octelium.Metadata.Name != "upstream-name" || obj.Octelium.Metadata.Labels["remote"] != "updated" {
		t.Fatal("upstream metadata was not mirrored")
	}
	obj.Octelium.Metadata.Name = ""
	if resourceName(obj) != obj.Name {
		t.Fatal("empty name did not use the Kubernetes name")
	}
	if err := mirrorMetadata(obj, md); err != nil {
		t.Fatal(err)
	}
	if obj.Octelium.Metadata.Name != "" {
		t.Fatal("observed upstream name silently enabled a name override")
	}
}
