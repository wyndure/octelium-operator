package backend

import (
	"context"
	"fmt"
	"reflect"

	rmeta "github.com/octelium/octelium/apis/rsc/rmetav1"
	apicommon "github.com/octelium/octelium/cluster/apiserver/apiserver/common"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// System provisioning uses the upstream resource-server API rather than the
// public administrative API, which intentionally rejects system resources.
func (b *Backend) getInternal(ctx context.Context, kind, name string) (umetav1.ResourceObjectI, error) {
	if kind == "Service" {
		name = vutils.GetServiceFullNameFromName(name)
	}

	return b.callInternal(ctx, "Get"+kind, &rmeta.GetOptions{Name: name})
}

func (b *Backend) callInternal(ctx context.Context, operation string, request any) (umetav1.ResourceObjectI, error) {
	method := reflect.ValueOf(b.Remote.CoreC()).MethodByName(operation)
	if !method.IsValid() {
		return nil, fmt.Errorf("unsupported internal operation %s", operation)
	}

	out := method.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(request)})
	if err := out[1].Interface(); err != nil {
		return nil, err.(error)
	}

	return out[0].Interface().(umetav1.ResourceObjectI), nil
}

func (b *Backend) applySystem(ctx context.Context, in umetav1.ResourceObjectI) (umetav1.ResourceObjectI, error) {
	err := apivalidation.ValidateCommon(in, &apivalidation.ValidateCommonOpts{
		ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{RequireName: true, ParentsMax: 1},
	})
	if err != nil {
		return nil, err
	}

	current, err := b.getInternal(ctx, in.GetKind(), in.GetMetadata().Name)
	if grpcerr.IsNotFound(err) {
		if in.GetKind() == "ClusterConfig" {
			return nil, err // The initializer owns initial ClusterConfig provisioning.
		}

		return b.Create(ctx, in)
	}
	if err != nil {
		return nil, err
	}

	out := proto.Clone(current).(umetav1.ResourceObjectI)

	copyResourceField(out, in, "spec")
	copyResourceField(out, in, "data")

	apicommon.MetadataUpdate(out.GetMetadata(), in.GetMetadata())

	md, desired := out.GetMetadata(), in.GetMetadata()

	md.IsSystem = desired.IsSystem
	md.IsSystemHidden = desired.IsSystemHidden
	md.IsUserHidden = desired.IsUserHidden
	md.SystemLabels = cloneLabels(desired.SystemLabels)

	if proto.Equal(out, current) {
		return current, nil
	}

	return b.callInternal(ctx, "Update"+in.GetKind(), out)
}

func copyResourceField(out, in umetav1.ResourceObjectI, name protoreflect.Name) {
	source := in.ProtoReflect()

	field := source.Descriptor().Fields().ByName(name)
	if field != nil && source.Has(field) {
		out.ProtoReflect().Set(field, protoreflect.ValueOfMessage(proto.Clone(source.Get(field).Message().Interface()).ProtoReflect()))
	}
}

func cloneLabels(labels map[string]string) map[string]string {
	if labels == nil {
		return nil
	}

	out := make(map[string]string, len(labels))
	for key, value := range labels {
		out[key] = value
	}

	return out
}
