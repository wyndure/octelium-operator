package backend

import (
	"context"
	"testing"

	core "github.com/octelium/octelium/apis/main/corev1"
	meta "github.com/octelium/octelium/apis/main/metav1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type cliMemory struct {
	core.UnimplementedMainServiceServer
	objects          map[string]*core.Namespace
	creates, updates int
}

func (c *cliMemory) ListNamespace(context.Context, *core.ListNamespaceOptions) (*core.NamespaceList, error) {
	out := &core.NamespaceList{}
	for _, o := range c.objects {
		out.Items = append(out.Items, proto.Clone(o).(*core.Namespace))
	}
	return out, nil
}

func (c *cliMemory) GetNamespace(_ context.Context, in *meta.GetOptions) (*core.Namespace, error) {
	o := c.objects[in.Name]
	if o == nil {
		return nil, status.Error(codes.NotFound, "missing")
	}
	return proto.Clone(o).(*core.Namespace), nil
}

func (c *cliMemory) CreateNamespace(_ context.Context, in *core.Namespace) (*core.Namespace, error) {
	c.creates++
	o := proto.Clone(in).(*core.Namespace)
	o.Metadata.Uid = "external"
	c.objects[o.Metadata.Name] = o
	return o, nil
}

func (c *cliMemory) UpdateNamespace(_ context.Context, in *core.Namespace) (*core.Namespace, error) {
	c.updates++
	o := proto.Clone(in).(*core.Namespace)
	c.objects[o.Metadata.Name] = o
	return o, nil
}

func (c *cliMemory) GetClusterConfig(context.Context, *core.GetClusterConfigRequest) (*core.ClusterConfig, error) {
	return nil, status.Error(codes.NotFound, "not provisioned")
}

func TestUpstreamDiffAppliesOneResourceWithoutPruning(t *testing.T) {
	c := &cliMemory{objects: map[string]*core.Namespace{"unrelated": {Kind: "Namespace", Metadata: &meta.Metadata{Name: "unrelated"}, Spec: &core.Namespace_Spec{}}}}
	b := &Backend{Client: c}
	in := &core.Namespace{ApiVersion: "core/v1", Kind: "Namespace", Metadata: &meta.Metadata{Name: "managed"}, Spec: &core.Namespace_Spec{}}
	if _, e := b.Apply(context.Background(), in); e != nil {
		t.Fatal(e)
	}
	if _, e := b.Apply(context.Background(), in); e != nil {
		t.Fatal(e)
	}
	if c.creates != 1 || c.updates != 0 || c.objects["unrelated"] == nil {
		t.Fatal("diff was not idempotent or pruned an unrelated object")
	}
	in.Metadata.Description = "updated"
	if _, e := b.Apply(context.Background(), in); e != nil {
		t.Fatal(e)
	}
	if c.updates != 1 {
		t.Fatal("upstream diff did not issue update")
	}
}

func TestMissingClusterConfigIsNeverCreated(t *testing.T) {
	b := &Backend{Client: &cliMemory{}}
	_, e := b.Apply(context.Background(), &core.ClusterConfig{Kind: "ClusterConfig", Metadata: &meta.Metadata{Name: "default"}, Spec: &core.ClusterConfig_Spec{}})
	if status.Code(e) != codes.NotFound {
		t.Fatal("expected missing bootstrap state to be returned", e)
	}
}
