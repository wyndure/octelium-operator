package connection

import (
	"context"
	"testing"

	coreapi "github.com/wyndure/octelium-operator/api/core/v1"
	api "github.com/wyndure/octelium-operator/api/v1"
	"github.com/wyndure/octelium-operator/internal/backend"
	kcore "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type noRPC struct{ backend.Interface }
type spyReader struct {
	client.Reader
	secrets int
}

func (s *spyReader) Get(ctx context.Context, key client.ObjectKey, obj client.Object, options ...client.GetOption) error {
	if _, ok := obj.(*kcore.Secret); ok {
		s.secrets++
	}
	return s.Reader.Get(ctx, key, obj, options...)
}

func fixture(t *testing.T) (*Resolver, client.Client, *api.Octelium, *int, *int) {
	t.Helper()
	scheme := runtime.NewScheme()
	_ = api.AddToScheme(scheme)
	_ = coreapi.AddToScheme(scheme)
	_ = kcore.AddToScheme(scheme)
	target := &api.Octelium{Name: "one", Namespace: testOwner, UID: types.UID("connection-one"), Spec: api.OcteliumSpec{Endpoint: "server:8080", Credentials: api.ConnectionCredentials{Type: "SPIFFE", TrustDomain: "test.local"}}}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(target, &kcore.Namespace{Name: testOwner}, &kcore.Namespace{Name: "tenant", Labels: map[string]string{"allowed": "yes"}}).Build()
	calls, closed := new(int), new(int)
	r := &Resolver{Reader: &spyReader{Reader: c}, Factory: func(context.Context, *api.Octelium) (backend.Interface, func(), error) {
		*calls++
		return &noRPC{}, func() { *closed++ }, nil
	}}
	t.Cleanup(r.Close)
	return r, c, target, calls, closed
}

func resource(ns string) *coreapi.OcteliumNamespace {
	return &coreapi.OcteliumNamespace{Name: "sample", Namespace: ns, OcteliumRef: api.OcteliumReference{Name: "one", Namespace: testOwner}}
}

func TestSameNamespaceDefaultAndDeniedBeforeConnection(t *testing.T) {
	r, c, target, calls, _ := fixture(t)
	ctx := context.Background()
	allowed, e := r.Resolve(ctx, resource(testOwner))
	if e != nil {
		t.Fatal(e)
	}
	allowed.Release()
	target.Spec.Credentials = api.ConnectionCredentials{Type: api.CredentialsInsecure}
	if e = c.Update(ctx, target); e != nil {
		t.Fatal(e)
	}
	if _, e = r.Resolve(ctx, resource("tenant")); e == nil {
		t.Fatal("cross-namespace reference allowed by default")
	}
	if r.Reader.(*spyReader).secrets != 0 || *calls != 1 {
		t.Fatal("unauthorized request read credentials or opened a connection")
	}
}

func TestSelectorAllKindPolicyAndRevocation(t *testing.T) {
	r, c, target, _, _ := fixture(t)
	ctx := context.Background()
	target.Spec.AllowedResources = &api.AllowedResources{Namespaces: &api.ResourceNamespaces{From: "Selector", Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"allowed": "yes"}}}}
	if e := c.Update(ctx, target); e != nil {
		t.Fatal(e)
	}
	lease, e := r.Resolve(ctx, resource("tenant"))
	if e != nil {
		t.Fatal(e)
	}
	lease.Release()
	var ns kcore.Namespace
	_ = c.Get(ctx, client.ObjectKey{Name: "tenant"}, &ns)
	ns.Labels = nil
	_ = c.Update(ctx, &ns)
	if _, e = r.Resolve(ctx, resource("tenant")); e == nil {
		t.Fatal("namespace selector revocation did not take effect")
	}
	_ = c.Get(ctx, client.ObjectKeyFromObject(target), target)
	target.Spec.AllowedResources = &api.AllowedResources{Namespaces: &api.ResourceNamespaces{From: "All"}, Kinds: []api.ResourceKind{{Group: coreapi.GroupVersion.Group, Kind: "OcteliumSecret"}}}
	_ = c.Update(ctx, target)
	if _, e = r.Resolve(ctx, resource("tenant")); e == nil {
		t.Fatal("kind restriction ignored")
	}
	target.Spec.AllowedResources.Kinds = nil
	_ = c.Update(ctx, target)
	lease, e = r.Resolve(ctx, resource("tenant"))
	if e != nil {
		t.Fatal(e)
	}
	lease.Release()
}

func TestConnectionChangesRetireCachedConnectionAfterActiveLease(t *testing.T) {
	r, c, target, calls, closed := fixture(t)
	ctx := context.Background()
	first, e := r.Resolve(ctx, resource(testOwner))
	if e != nil {
		t.Fatal(e)
	}
	second, e := r.Resolve(ctx, resource(testOwner))
	if e != nil {
		t.Fatal(e)
	}
	second.Release()
	if *calls != 1 {
		t.Fatal("connection was not cached")
	}
	target.Spec.Endpoint = "replacement:8080"
	_ = c.Update(ctx, target)
	rotated, e := r.Resolve(ctx, resource(testOwner))
	if e != nil {
		t.Fatal(e)
	}
	rotated.Release()
	if *calls != 2 || *closed != 0 {
		t.Fatal("endpoint change interrupted an active lease or reused the old connection")
	}
	first.Release()
	if *closed != 1 {
		t.Fatal("retired connection was not closed")
	}
	target.Spec.Credentials = api.ConnectionCredentials{Type: api.CredentialsInsecure}
	_ = c.Update(ctx, target)
	changed, e := r.Resolve(ctx, resource(testOwner))
	if e != nil {
		t.Fatal(e)
	}
	changed.Release()
	if *calls != 3 || *closed != 2 {
		t.Fatal("transport change reused the previous connection")
	}
	if r.Reader.(*spyReader).secrets != 0 {
		t.Fatal("connection attempted to read certificate Secrets")
	}
}

func TestReferenceGVKAndNamespaceDefaults(t *testing.T) {
	r, _, _, _, _ := fixture(t)
	obj := resource(testOwner)
	obj.OcteliumRef.Namespace = ""
	lease, e := r.Resolve(context.Background(), obj)
	if e != nil {
		t.Fatal(e)
	}
	lease.Release()
	obj.OcteliumRef.Kind = "Other"
	if _, e = r.Resolve(context.Background(), obj); e == nil {
		t.Fatal("unsupported reference kind accepted")
	}
}

const testOwner = "owner"
