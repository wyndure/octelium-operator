package controller

import (
	"os"
	"path/filepath"
	"testing"

	"sigs.k8s.io/yaml"
)

func TestEveryCRDIsNamespacedAndReferencesAreRequired(t *testing.T) {
	files, e := filepath.Glob("../../config/crd/bases/*.yaml")
	if e != nil {
		t.Fatal(e)
	}
	for _, file := range files {
		data, e := os.ReadFile(file)
		if e != nil {
			t.Fatal(e)
		}
		var crd map[string]any
		if e = yaml.Unmarshal(data, &crd); e != nil {
			t.Fatal(e)
		}
		spec := crd["spec"].(map[string]any)
		if spec["scope"] != "Namespaced" {
			t.Fatal("cluster-scoped CRD remains", file)
		}
		if spec["group"] == "octelium.wyndure.com" {
			continue
		}
		for _, version := range spec["versions"].([]any) {
			schema := version.(map[string]any)["schema"].(map[string]any)["openAPIV3Schema"].(map[string]any)
			found := false
			for _, name := range schema["required"].([]any) {
				if name == "octeliumRef" {
					found = true
				}
			}
			if !found {
				t.Fatal("reference is not required", file)
			}
		}
	}
}
