package backend

import (
	"context"
	"fmt"
	"reflect"

	core "github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/admin"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
)

// Create performs a create-only operation. A concurrent creator is observed,
// never updated, so Provision cannot become an upsert through a race.
func (b *Backend) Create(ctx context.Context, in umetav1.ResourceObjectI) (umetav1.ResourceObjectI, error) {
	if in.GetKind() == "ClusterConfig" {
		return b.Get(ctx, in.GetKind(), in.GetMetadata().Name)
	}
	var out umetav1.ResourceObjectI
	var err error
	if in.GetMetadata().IsSystem && b.Remote != nil {
		out, err = b.createSystem(ctx, in)
	} else {
		method := reflect.ValueOf(b.client()).MethodByName("Create" + in.GetKind())
		if !method.IsValid() {
			return nil, fmt.Errorf("%s has no create operation", in.GetKind())
		}
		values := method.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(in)})
		if e := values[1].Interface(); e != nil {
			err = e.(error)
		} else {
			out = values[0].Interface().(umetav1.ResourceObjectI)
		}
	}
	if grpcerr.AlreadyExists(err) {
		return b.Get(ctx, in.GetKind(), in.GetMetadata().Name)
	}
	return out, err
}

func (b *Backend) createSystem(ctx context.Context, in umetav1.ResourceObjectI) (umetav1.ResourceObjectI, error) {
	if err := apivalidation.ValidateCommon(in, &apivalidation.ValidateCommonOpts{ValidateMetadataOpts: apivalidation.ValidateMetadataOpts{RequireName: true, ParentsMax: 1}}); err != nil {
		return nil, err
	}
	if service, ok := in.(*core.Service); ok {
		return admin.NewServer(&admin.Opts{OcteliumC: b.Remote, IsEmbedded: true}).DoCreateService(ctx, service, true)
	}
	return b.callInternal(ctx, "Create"+in.GetKind(), in)
}
