package controller

import (
	"context"

	"github.com/octelium/octelium/pkg/grpcerr"
	api "github.com/wyndure/octelium-operator/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *ResourceReconciler) finalizeResource(ctx context.Context, obj api.Resource) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(obj, finalizer) {
		return ctrl.Result{}, nil
	}
	// ClusterConfig has no upstream deletion operation and is always retained.
	if obj.Extension().DeletionPolicy != "Retain" && (obj.Extension().Metadata == nil || !obj.Extension().Metadata.IsSystem) && obj.UpstreamKind() != "ClusterConfig" && obj.SyncStatus().ExternalUID != "" {
		current, e := r.backend.Get(ctx, obj.UpstreamKind(), obj.GetName())
		if e != nil && !grpcerr.IsNotFound(e) {
			return r.failed(ctx, obj, e)
		}
		// Never delete a replacement resource with the same name and a different identity.
		if e == nil && current.GetMetadata().Uid == obj.SyncStatus().ExternalUID {
			if e = r.backend.Delete(ctx, obj.UpstreamKind(), obj.SyncStatus().ExternalUID); e != nil && !grpcerr.IsNotFound(e) {
				return r.failed(ctx, obj, e)
			}
		}
	}

	base := obj.DeepCopyObject().(client.Object)

	controllerutil.RemoveFinalizer(obj, finalizer)

	return ctrl.Result{}, r.Patch(ctx, obj, client.MergeFrom(base))
}
