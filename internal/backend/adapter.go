package backend

import (
	"context"
	"fmt"
	"reflect"

	core "github.com/octelium/octelium/apis/main/corev1"
	meta "github.com/octelium/octelium/apis/main/metav1"
	rmeta "github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/admin"
	"github.com/octelium/octelium/cluster/common/octeliumc"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"google.golang.org/protobuf/proto"
)

// recorded delegates to the same upstream admin methods called by octeliumctl.
// It captures validation errors that the interactive diff helper otherwise prints and skips.
type recorded struct {
	*admin.Server

	remote octeliumc.ClientInterface

	writeErr error
}

func (a *recorded) CreateConfig(ctx context.Context, in *core.Config) (*core.Config, error) {
	out, e := a.Server.CreateConfig(ctx, in)
	if e != nil {
		a.writeErr = e
	}

	return out, e
}

func (a *recorded) UpdateConfig(ctx context.Context, in *core.Config) (*core.Config, error) {
	out, e := a.Server.UpdateConfig(ctx, in)
	if e != nil {
		a.writeErr = e
	}

	return out, e
}

func (a *recorded) CreatePolicy(ctx context.Context, in *core.Policy) (*core.Policy, error) {
	out, e := a.Server.CreatePolicy(ctx, in)
	if e != nil {
		a.writeErr = e
	}

	return out, e
}

func (a *recorded) UpdatePolicy(ctx context.Context, in *core.Policy) (*core.Policy, error) {
	out, e := a.Server.UpdatePolicy(ctx, in)
	if e != nil {
		a.writeErr = e
	}

	return out, e
}

func (a *recorded) CreateIdentityProvider(ctx context.Context, in *core.IdentityProvider) (*core.IdentityProvider, error) {
	out, e := a.Server.CreateIdentityProvider(ctx, in)
	if e != nil {
		a.writeErr = e
	}

	return out, e
}

func (a *recorded) UpdateIdentityProvider(ctx context.Context, in *core.IdentityProvider) (*core.IdentityProvider, error) {
	out, e := a.Server.UpdateIdentityProvider(ctx, in)
	if e != nil {
		a.writeErr = e
	}

	return out, e
}

func (a *recorded) CreateNamespace(ctx context.Context, in *core.Namespace) (*core.Namespace, error) {
	out, e := a.Server.CreateNamespace(ctx, in)
	if e != nil {
		a.writeErr = e
	}

	return out, e
}

func (a *recorded) UpdateNamespace(ctx context.Context, in *core.Namespace) (*core.Namespace, error) {
	out, e := a.Server.UpdateNamespace(ctx, in)
	if e != nil {
		a.writeErr = e
	}

	return out, e
}

func (a *recorded) CreateGroup(ctx context.Context, in *core.Group) (*core.Group, error) {
	out, e := a.Server.CreateGroup(ctx, in)
	if e != nil {
		a.writeErr = e
	}

	return out, e
}

func (a *recorded) UpdateGroup(ctx context.Context, in *core.Group) (*core.Group, error) {
	out, e := a.Server.UpdateGroup(ctx, in)
	if e != nil {
		a.writeErr = e
	}

	return out, e
}

func (a *recorded) CreateUser(ctx context.Context, in *core.User) (*core.User, error) {
	out, e := a.Server.CreateUser(ctx, in)
	if e != nil {
		a.writeErr = e
	}

	return out, e
}

func (a *recorded) UpdateUser(ctx context.Context, in *core.User) (*core.User, error) {
	out, e := a.Server.UpdateUser(ctx, in)
	if e != nil {
		a.writeErr = e
	}

	return out, e
}

func (a *recorded) CreateService(ctx context.Context, in *core.Service) (*core.Service, error) {
	out, e := a.Server.CreateService(ctx, in)
	if e != nil {
		a.writeErr = e
	}

	return out, e
}

func (a *recorded) UpdateService(ctx context.Context, in *core.Service) (*core.Service, error) {
	out, e := a.Server.UpdateService(ctx, in)
	if e != nil {
		a.writeErr = e
	}

	return out, e
}

func (a *recorded) CreateCredential(ctx context.Context, in *core.Credential) (*core.Credential, error) {
	out, e := a.Server.CreateCredential(ctx, in)
	if e != nil {
		a.writeErr = e
	}

	return out, e
}

func (a *recorded) UpdateCredential(ctx context.Context, in *core.Credential) (*core.Credential, error) {
	out, e := a.Server.UpdateCredential(ctx, in)
	if e != nil {
		a.writeErr = e
	}

	return out, e
}

func (a *recorded) CreateSecret(ctx context.Context, in *core.Secret) (*core.Secret, error) {
	out, e := a.Server.CreateSecret(ctx, in)
	if e != nil {
		a.writeErr = e
	}

	return out, e
}

func (a *recorded) UpdateSecret(ctx context.Context, in *core.Secret) (*core.Secret, error) {
	if in.Data == nil {
		cur, e := a.remote.CoreC().GetSecret(ctx, &rmeta.GetOptions{Name: in.Metadata.Name})
		if e != nil {
			a.writeErr = e

			return nil, e
		}

		in.Data = cur.Data
	}

	out, e := a.Server.UpdateSecret(ctx, in)
	if e != nil {
		a.writeErr = e
	}

	return out, e
}

// Region bootstrap is internal; the public CLI intentionally has no Region create command.
func (a *recorded) CreateRegion(ctx context.Context, in *core.Region) (*core.Region, error) {
	out, e := a.remote.CoreC().CreateRegion(ctx, in)
	if e != nil {
		a.writeErr = e
	}

	return out, e
}

func (a *recorded) UpdateRegion(ctx context.Context, in *core.Region) (*core.Region, error) {
	cur, e := a.remote.CoreC().GetRegion(ctx, &rmeta.GetOptions{Name: in.Metadata.Name})
	if e != nil {
		a.writeErr = e

		return nil, e
	}

	out := proto.Clone(cur).(*core.Region)

	out.Spec = in.Spec

	md := proto.Clone(in.Metadata).(*meta.Metadata)

	md.Uid = cur.Metadata.Uid
	md.ResourceVersion = cur.Metadata.ResourceVersion
	md.CreatedAt = cur.Metadata.CreatedAt

	out.Metadata = md

	out, e = a.remote.CoreC().UpdateRegion(ctx, out)
	if e != nil {
		a.writeErr = e
	}

	return out, e
}

func (a *recorded) DeleteRegion(ctx context.Context, in *meta.DeleteOptions) (*meta.OperationResult, error) {
	_, e := a.remote.CoreC().DeleteRegion(ctx, &rmeta.DeleteOptions{Uid: in.Uid})

	return &meta.OperationResult{}, e
}

func get(ctx context.Context, c any, kind, name string) (umetav1.ResourceObjectI, error) {
	var req any = &meta.GetOptions{Name: name}
	if kind == ucorev1.KindClusterConfig {
		req = &core.GetClusterConfigRequest{}
	}

	method := reflect.ValueOf(c).MethodByName("Get" + kind)
	if !method.IsValid() {
		return nil, fmt.Errorf("unsupported kind %s", kind)
	}

	out := method.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(req)})
	if e := out[1].Interface(); e != nil {
		return nil, e.(error)
	}

	return out[0].Interface().(umetav1.ResourceObjectI), nil
}

func deleteResource(ctx context.Context, c any, kind, uid string) error {
	method := reflect.ValueOf(c).MethodByName("Delete" + kind)
	if !method.IsValid() {
		return fmt.Errorf("%s has no delete operation", kind)
	}

	out := method.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(&meta.DeleteOptions{Uid: uid})})
	if e := out[1].Interface(); e != nil {
		return e.(error)
	}

	return nil
}
