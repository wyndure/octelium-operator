package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	api "github.com/wyndure/octelium-operator/api/core/v1"
	connectionapi "github.com/wyndure/octelium-operator/api/v1"
	"github.com/wyndure/octelium-operator/internal/backend"
	"github.com/wyndure/octelium-operator/internal/connection"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const finalizer = "operator.octelium.wyndure.com/resource"

type ResourceReconciler struct {
	client.Client
	backend     backend.Interface
	Connections *connection.Resolver
	Interval    time.Duration
}

func (r *ResourceReconciler) ReconcileResource(ctx context.Context, req ctrl.Request, obj api.Resource) (ctrl.Result, error) {
	if e := r.Get(ctx, req.NamespacedName, obj); e != nil {
		return ctrl.Result{}, client.IgnoreNotFound(e)
	}

	lease, e := r.Connections.Resolve(ctx, obj)
	if e != nil {
		return r.failed(ctx, obj, e)
	}

	defer lease.Release()
	if stored := obj.SyncStatus().ConnectionUID; stored != "" && stored != string(lease.Connection.UID) {
		return r.failed(ctx, obj, fmt.Errorf("octeliumRef cannot change an already bound connection identity"))
	}

	scoped := *r

	scoped.backend = lease.Backend

	return scoped.reconcileAuthorized(ctx, req, obj, lease.Connection)
}

func (r *ResourceReconciler) reconcileAuthorized(ctx context.Context, req ctrl.Request, obj api.Resource, target *connectionapi.Octelium) (ctrl.Result, error) {
	interval := r.Interval
	if interval <= 0 {
		interval = 30 * time.Second
	}

	if !obj.GetDeletionTimestamp().IsZero() {
		return r.finalizeResource(ctx, obj)
	}

	if !controllerutil.ContainsFinalizer(obj, finalizer) {
		base := obj.DeepCopyObject().(client.Object)
		controllerutil.AddFinalizer(obj, finalizer)
		return ctrl.Result{RequeueAfter: time.Millisecond}, r.Patch(ctx, obj, client.MergeFrom(base))
	}

	name := resourceName(obj)
	if bound := obj.SyncStatus().ExternalName; bound != "" && bound != name {
		return r.failed(ctx, obj, fmt.Errorf("octelium.metadata.name cannot change an already bound resource; recreate the CR to select a new name"))
	}
	current, e := r.backend.Get(ctx, obj.UpstreamKind(), name)
	if e != nil && !grpcerr.IsNotFound(e) {
		return r.failed(ctx, obj, e)
	}

	if grpcerr.IsNotFound(e) {
		current = nil
	}

	if current != nil && obj.GetReconciliationPolicy() != api.PolicyProvision && obj.SyncStatus().ExternalUID != "" && obj.SyncStatus().ExternalUID != current.GetMetadata().Uid {
		return r.failed(ctx, obj, fmt.Errorf("external resource identity changed; refusing to overwrite a replacement"))
	}

	if obj.SyncStatus().ExternalUID != "" && current == nil && (obj.SyncStatus().ExternalName == "" || obj.SyncStatus().ConnectionEndpoint != target.Spec.Endpoint) {
		return r.failed(ctx, obj, fmt.Errorf("selected name or endpoint does not contain the bound resource"))
	}

	if obj.GetReconciliationPolicy() == api.PolicyProvision && current != nil {
		return r.observeResource(ctx, req, obj, target, "", current, interval)
	}

	desired, e := desiredResource(obj, current)
	if e != nil {
		return r.failed(ctx, obj, e)
	}

	dataDigest, e := r.prepareData(ctx, obj, current, desired)
	if e != nil {
		return r.failed(ctx, obj, e)
	}

	var observed umetav1.ResourceObjectI
	if obj.GetReconciliationPolicy() == api.PolicyProvision {
		observed, e = r.backend.Create(ctx, desired)
	} else {
		observed, e = r.backend.Apply(ctx, desired)
	}
	if e != nil {
		return r.failed(ctx, obj, e)
	}

	return r.observeResource(ctx, req, obj, target, dataDigest, observed, interval)
}

func (r *ResourceReconciler) observeResource(ctx context.Context, req ctrl.Request, obj api.Resource, target *connectionapi.Octelium, dataDigest string, observed umetav1.ResourceObjectI, interval time.Duration) (ctrl.Result, error) {

	if e := r.publishMetadata(ctx, obj, observed.GetMetadata()); e != nil {
		return r.failed(ctx, obj, e)
	}

	if e := r.synchronizeMetadata(ctx, req, obj, observed.GetMetadata()); e != nil {
		return ctrl.Result{}, e
	}

	if e := r.updateSynchronizedStatus(ctx, obj, target, dataDigest, observed); e != nil {
		return ctrl.Result{}, e
	}

	return ctrl.Result{RequeueAfter: interval}, nil
}
