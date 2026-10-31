package backend

import (
	"context"
	"fmt"

	core "github.com/octelium/octelium/apis/main/corev1"
	meta "github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/client/common/rscdiff"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/admin"
	"github.com/octelium/octelium/cluster/common/octeliumc"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"google.golang.org/protobuf/proto"
)

// Interface allows controller testing without replacing the production resource implementation.
type Interface interface {
	Get(context.Context, string, string) (umetav1.ResourceObjectI, error)
	Apply(context.Context, umetav1.ResourceObjectI) (umetav1.ResourceObjectI, error)
	Delete(context.Context, string, string) error
}
type Backend struct {
	Client any
	Remote octeliumc.ClientInterface
}

func New(remote octeliumc.ClientInterface) *Backend {
	return &Backend{Remote: remote}
}

func (b *Backend) client() any {
	if b.Client != nil {
		return b.Client
	}
	return &recorded{Server: admin.NewServer(&admin.Opts{OcteliumC: b.Remote, IsEmbedded: true}), remote: b.Remote}
}

func (b *Backend) Get(ctx context.Context, kind, name string) (umetav1.ResourceObjectI, error) {
	if b.Remote != nil {
		return b.getInternal(ctx, kind, name)
	}
	return get(ctx, b.client(), kind, name)
}

func (b *Backend) Apply(ctx context.Context, in umetav1.ResourceObjectI) (umetav1.ResourceObjectI, error) {
	c := b.client()

	kind := in.GetKind()
	name := in.GetMetadata().Name

	if kind == ucorev1.KindClusterConfig {
		desired := in.(*core.ClusterConfig)

		current, e := b.Get(ctx, kind, name)
		if e != nil {
			return nil, e
		}

		cur := current.(*core.ClusterConfig)

		desired.Metadata.Uid = cur.Metadata.Uid
		desired.Metadata.ResourceVersion = cur.Metadata.ResourceVersion

		// Preserve runtime status: the public CLI updates the specification and user metadata only.
		desired.Status = cur.Status
		if pbutils.IsEqual(desired.Spec, cur.Spec) && pbutils.IsEqual(editableMetadata(desired), editableMetadata(cur)) {
			return cur, nil
		}

		return c.(interface {
			UpdateClusterConfig(context.Context, *core.ClusterConfig) (*core.ClusterConfig, error)
		}).UpdateClusterConfig(ctx, desired)
	}

	if in.GetMetadata().IsSystem && b.Remote != nil {
		return b.applySystem(ctx, in)
	}

	ctl, e := rscdiff.NewDiffCtl(
		ucorev1.API,
		kind,
		c,
		func() (umetav1.ResourceObjectI, error) {
			return ucorev1.NewObject(kind)
		},
		func() (proto.Message, error) {
			return ucorev1.NewObjectListOptions(kind)
		},
		[]umetav1.ResourceObjectI{in},
		false,
		false,
	)

	if e != nil {
		return nil, e
	}

	if e = ctl.Load(ctx); e != nil {
		return nil, e
	}

	if e = ctl.Apply(ctx, &rscdiff.DiffCtlResponse{}); e != nil {
		return nil, e
	}

	if a, ok := c.(*recorded); ok && a.writeErr != nil {
		return nil, a.writeErr
	}

	return get(ctx, c, kind, name)
}

func editableMetadata(in umetav1.ResourceObjectI) *meta.Metadata {
	m := in.GetMetadata()

	return &meta.Metadata{Name: m.Name, DisplayName: m.DisplayName, Description: m.Description, PicURL: m.PicURL, Labels: m.Labels, Annotations: m.Annotations, Tags: m.Tags}
}

func (b *Backend) Delete(ctx context.Context, kind, uid string) error {
	if kind == ucorev1.KindClusterConfig {
		return fmt.Errorf("ClusterConfig cannot be deleted; use Retain")
	}

	return deleteResource(ctx, b.client(), kind, uid)
}
