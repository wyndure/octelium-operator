// Package connection resolves namespaced references before contacting Octelium.
package connection

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"sync"

	coreapi "github.com/wyndure/octelium-operator/api/core/v1"
	api "github.com/wyndure/octelium-operator/api/v1"
	"github.com/wyndure/octelium-operator/internal/backend"
	kcore "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const APIVersion = "octelium.wyndure.com/v1"
const Kind = "Octelium"

type Factory func(context.Context, *api.Octelium) (backend.Interface, func(), error)

type session struct {
	signature string
	backend   backend.Interface
	close     func()
	refs      int
	retired   bool
}

type Resolver struct {
	Reader   client.Reader
	Factory  Factory
	mu       sync.Mutex
	sessions map[string]*session
}

type Lease struct {
	Backend    backend.Interface
	Connection *api.Octelium
	release    func()
}

func (l *Lease) Release() {
	if l.release != nil {
		l.release()

		l.release = nil
	}
}

// Resolve checks both the target-side grant and optional kind restrictions before opening a connection.
func (r *Resolver) Resolve(ctx context.Context, obj coreapi.Resource) (*Lease, error) {
	ref, err := normalizedReference(obj)
	if err != nil {
		return nil, err
	}

	target, err := r.loadTarget(ctx, ref)
	if err != nil {
		return nil, err
	}
	if err := r.authorize(ctx, obj, target); err != nil {
		return nil, err
	}

	return r.Acquire(ctx, target)
}

func normalizedReference(obj coreapi.Resource) (api.OcteliumReference, error) {
	ref := *obj.Reference()
	if ref.APIVersion == "" {
		ref.APIVersion = APIVersion
	}
	if ref.Kind == "" {
		ref.Kind = Kind
	}
	if ref.APIVersion != APIVersion || ref.Kind != Kind || ref.Name == "" {
		return api.OcteliumReference{}, fmt.Errorf("octeliumRef must identify %s %s with a name", APIVersion, Kind)
	}
	if obj.GetNamespace() == "" {
		return api.OcteliumReference{}, fmt.Errorf("managed resource must have a namespace")
	}
	if ref.Namespace == "" {
		ref.Namespace = obj.GetNamespace()
	}
	return ref, nil
}

func (r *Resolver) loadTarget(ctx context.Context, ref api.OcteliumReference) (*api.Octelium, error) {
	var target api.Octelium
	if err := r.Reader.Get(ctx, client.ObjectKey{Namespace: ref.Namespace, Name: ref.Name}, &target); err != nil {
		return nil, err
	}
	if !target.DeletionTimestamp.IsZero() {
		return nil, fmt.Errorf("referenced Octelium is being deleted")
	}
	return &target, nil
}

func (r *Resolver) authorize(ctx context.Context, obj coreapi.Resource, target *api.Octelium) error {
	policy := target.Spec.AllowedResources

	from := "Same"

	if policy != nil && len(policy.Kinds) > 0 {
		permitted := false
		for _, kind := range policy.Kinds {
			if kind.Group == coreapi.GroupVersion.Group && kind.Kind == "Octelium"+obj.UpstreamKind() {
				permitted = true
				break
			}
		}

		if !permitted {
			return fmt.Errorf("octelium allowedResources denies this resource kind")
		}
	}

	var namespaces *api.ResourceNamespaces
	if policy != nil {
		namespaces = policy.Namespaces
	}

	if namespaces != nil && namespaces.From != "" {
		from = namespaces.From
	}

	switch from {
	case "Same":
		if obj.GetNamespace() == target.Namespace {
			return nil
		}
	case "All":
		return nil
	case "Selector":
		if namespaces.Selector == nil {
			return fmt.Errorf("allowedResources Selector requires selector")
		}

		selector, e := metav1.LabelSelectorAsSelector(namespaces.Selector)
		if e != nil {
			return e
		}

		var ns kcore.Namespace
		if e = r.Reader.Get(ctx, client.ObjectKey{Name: obj.GetNamespace()}, &ns); e != nil {
			return e
		}

		if selector.Matches(labels.Set(ns.Labels)) {
			return nil
		}
	default:
		return fmt.Errorf("invalid allowedResources namespace policy %q", from)
	}

	return fmt.Errorf("octelium allowedResources denies namespace %q", obj.GetNamespace())
}

func Validate(target *api.Octelium) error {
	host, port, e := net.SplitHostPort(target.Spec.Endpoint)
	if e != nil || host == "" {
		return fmt.Errorf("endpoint must be a DNS name/IP and port without a URL scheme")
	}

	n, e := strconv.Atoi(port)
	if e != nil || n < 1 || n > 65535 {
		return fmt.Errorf("endpoint port is invalid")
	}

	c := target.Spec.Credentials

	switch c.Type {
	case "", "SPIFFE":
		if c.TrustDomain == "" {
			return fmt.Errorf("SPIFFE credentials require trustDomain")
		}

	case api.CredentialsInsecure:
		if c.TrustDomain != "" || c.SocketPath != "" {
			return fmt.Errorf("insecure transport does not use trustDomain or socketPath")
		}
	default:
		return fmt.Errorf("unsupported credential type %q", c.Type)
	}

	return nil
}

func (r *Resolver) Acquire(ctx context.Context, target *api.Octelium) (*Lease, error) {
	if e := Validate(target); e != nil {
		return nil, e
	}

	key := target.Namespace + "/" + target.Name
	signature, err := connectionSignature(target)
	if err != nil {
		return nil, err
	}

	if lease := r.cachedLease(target, key, signature); lease != nil {
		return lease, nil
	}

	factory := r.Factory
	if factory == nil {
		factory = DefaultFactory
	}

	backend, closeConnection, err := factory(ctx, target)
	if err != nil {
		return nil, err
	}

	current := &session{signature: signature, backend: backend, close: closeConnection, refs: 1}
	return r.storeSession(target, key, current), nil
}

func connectionSignature(target *api.Octelium) (string, error) {
	encoded, err := json.Marshal(target.Spec)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%x", target.UID, sha256.Sum256(encoded)), nil
}

func (r *Resolver) cachedLease(target *api.Octelium, key, signature string) *Lease {
	r.mu.Lock()
	defer r.mu.Unlock()

	if session := r.sessions[key]; session != nil && session.signature == signature {
		session.refs++
		return r.lease(target, session)
	}
	return nil
}

func (r *Resolver) storeSession(target *api.Octelium, key string, current *session) *Lease {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.sessions == nil {
		r.sessions = map[string]*session{}
	}

	old := r.sessions[key]
	if old != nil && old.signature == current.signature {
		old.refs++
		current.close()
		return r.lease(target, old)
	}

	r.sessions[key] = current
	if old != nil {
		old.retired = true

		if old.refs == 0 {
			old.close()
		}
	}

	return r.lease(target, current)
}

func (r *Resolver) lease(target *api.Octelium, s *session) *Lease {
	return &Lease{Backend: s.backend, Connection: target.DeepCopy(), release: func() {
		r.mu.Lock()

		defer r.mu.Unlock()

		s.refs--
		if s.retired && s.refs == 0 {
			s.close()
		}
	}}
}

func (r *Resolver) Close() {
	r.mu.Lock()

	defer r.mu.Unlock()

	for key, s := range r.sessions {
		s.retired = true
		if s.refs == 0 {
			s.close()
		}

		delete(r.sessions, key)
	}
}

func (r *Resolver) Forget(namespace, name string) {
	r.mu.Lock()

	defer r.mu.Unlock()

	key := namespace + "/" + name
	if s := r.sessions[key]; s != nil {
		s.retired = true
		if s.refs == 0 {
			s.close()
		}

		delete(r.sessions, key)
	}
}
