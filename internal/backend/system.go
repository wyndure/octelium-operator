package backend

import (
	"context"
	"fmt"
	"reflect"

	core "github.com/octelium/octelium/apis/main/corev1"
	rmeta "github.com/octelium/octelium/apis/rsc/rmetav1"
	apicommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/cluster/genesis/genesis/genesisutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"google.golang.org/protobuf/proto"
)

// System resources are provisioning objects, excluded by the public administrative API.
// Reuse Genesis's upsert helpers and the existing internal client for those objects.
func (b *Backend) getInternal(ctx context.Context, kind, name string) (umetav1.ResourceObjectI, error) {
	if kind == "Service" {
		name = vutils.GetServiceFullNameFromName(name)
	}

	method := reflect.ValueOf(b.Remote.CoreC()).MethodByName("Get" + kind)
	if !method.IsValid() {
		return nil, fmt.Errorf("unsupported internal kind %s", kind)
	}

	out := method.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(&rmeta.GetOptions{Name: name})})
	if e := out[1].Interface(); e != nil {
		return nil, e.(error)
	}

	return out[0].Interface().(umetav1.ResourceObjectI), nil
}

func (b *Backend) applySystem(ctx context.Context, in umetav1.ResourceObjectI) (umetav1.ResourceObjectI, error) {
	kind := in.GetKind()

	name := in.GetMetadata().Name

	cur, err := b.getInternal(ctx, kind, name)
	if err != nil && !grpcerr.IsNotFound(err) {
		return nil, err
	}

	switch desired := in.(type) {
	case *core.Namespace:
		if err == nil && proto.Equal(desired.Spec, cur.(*core.Namespace).Spec) && proto.Equal(editableMetadata(in), editableMetadata(cur)) {
			return cur, nil
		}

		if err = genesisutils.CreateOrUpdateNamespace(ctx, b.Remote, desired); err != nil {
			return nil, err
		}
	case *core.Secret:
		if err != nil {
			return b.Remote.CoreC().CreateSecret(ctx, desired)
		}

		current := cur.(*core.Secret)
		if desired.Data == nil {
			desired.Data = current.Data
		}

		if proto.Equal(desired.Spec, current.Spec) && proto.Equal(desired.Data, current.Data) && proto.Equal(editableMetadata(in), editableMetadata(cur)) {
			return current, nil
		}

		out := proto.Clone(current).(*core.Secret)

		out.Spec = desired.Spec
		out.Data = desired.Data

		apicommon.MetadataUpdate(out.Metadata, desired.Metadata)

		return b.Remote.CoreC().UpdateSecret(ctx, out)
	default:
		return nil, fmt.Errorf("system provisioning is supported for Namespace and Secret, not %s", kind)
	}

	return b.getInternal(ctx, kind, name)
}
