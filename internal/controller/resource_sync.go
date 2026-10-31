package controller

import (
	"context"
	"reflect"

	upstreammeta "github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	api "github.com/wyndure/octelium-operator/api/core/v1"
	connectionapi "github.com/wyndure/octelium-operator/api/v1"
	"google.golang.org/protobuf/encoding/protojson"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ResourceReconciler) synchronizeMetadata(ctx context.Context, req ctrl.Request, obj api.Resource, metadata *upstreammeta.Metadata) error {
	// The operator owns desired spec; metadata is three-way synchronized in both directions.
	base := obj.DeepCopyObject().(client.Object)
	if err := mirrorMetadata(obj, metadata); err != nil {
		return r.failedError(ctx, obj, err)
	}

	if reflect.DeepEqual(base, obj) {
		return nil
	}

	if err := r.Patch(ctx, obj, client.MergeFrom(base)); err != nil {
		return err
	}
	// Refresh the resourceVersion/generation before writing the status subresource.
	return r.Get(ctx, req.NamespacedName, obj)
}

func (r *ResourceReconciler) updateSynchronizedStatus(ctx context.Context, obj api.Resource, target *connectionapi.Octelium, dataDigest string, observed umetav1.ResourceObjectI) error {
	before := obj.DeepCopyObject().(client.Object)
	status := obj.SyncStatus()

	status.ConnectionUID = string(target.UID)
	status.ConnectionKey = target.Namespace + "/" + target.Name
	status.ConnectionEndpoint = target.Spec.Endpoint

	if dataDigest != "" {
		status.DataDigest = dataDigest
	}

	status.ExternalUID = observed.GetMetadata().Uid
	status.ObservedGeneration = obj.GetGeneration()
	status.LastSyncedMetadata = editableFields(observed.GetMetadata())

	if field := observed.ProtoReflect().Descriptor().Fields().ByName("status"); field != nil {
		data, err := protojson.Marshal(observed.ProtoReflect().Get(field).Message().Interface())
		if err != nil {
			return r.failedError(ctx, obj, err)
		}

		status.UpstreamStatus = &runtime.RawExtension{Raw: data}
	}
	apimeta.SetStatusCondition(&status.Conditions, metav1.Condition{
		Type:               conditionReady,
		Status:             metav1.ConditionTrue,
		Reason:             "Synchronized",
		Message:            "Resource synchronized with Octelium",
		ObservedGeneration: obj.GetGeneration(),
	})

	if reflect.DeepEqual(before, obj) {
		return nil
	}

	return r.Status().Patch(ctx, obj, client.MergeFrom(before))
}

func (r *ResourceReconciler) failedError(ctx context.Context, obj api.Resource, cause error) error {
	if _, err := r.failed(ctx, obj, cause); err != nil {
		return err
	}

	return cause
}
