package controller

import (
	"os"
	"slices"
	"strings"
	"testing"

	api "github.com/wyndure/octelium-operator/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"sigs.k8s.io/yaml"
)

func TestGeneratedRBACCoversEveryWatchedResource(t *testing.T) {
	b, e := os.ReadFile("../../config/rbac/role.yaml")
	if e != nil {
		t.Fatal(e)
	}
	var role rbac.ClusterRole
	if e = yaml.Unmarshal(b, &role); e != nil {
		t.Fatal(e)
	}
	resources := make([]string, 1, 1+2*len(api.Resources()))
	resources[0] = "secrets"
	resources = append(resources, "namespaces", "octeliums", "octeliums/status")
	for _, r := range api.Resources() {
		plural := strings.ToLower("Octelium"+r.UpstreamKind()) + "s"
		if r.UpstreamKind() == "Policy" {
			plural = "octeliumpolicies"
		}
		resources = append(resources, plural, plural+"/status")
	}
	for _, resource := range resources {
		found := false
		for _, rule := range role.Rules {
			if slices.Contains(rule.Resources, resource) && slices.Contains(rule.Verbs, "get") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("generated RBAC lacks %s", resource)
		}
	}
}
