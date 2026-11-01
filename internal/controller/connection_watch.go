package controller

import (
	"context"

	api "github.com/wyndure/octelium-operator/api/core/v1"
	connectionapi "github.com/wyndure/octelium-operator/api/v1"
	kcore "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ResourceReconciler) WatchConnections(b *builder.Builder, prototype api.Resource) *builder.Builder {
	mapEvent := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, changed client.Object) []reconcile.Request {
		return r.resourcesForChange(ctx, prototype, changed)
	})

	return b.Owns(&kcore.ConfigMap{}).Watches(&connectionapi.Octelium{}, mapEvent).Watches(&kcore.Secret{}, mapEvent).Watches(&kcore.Namespace{}, mapEvent)
}

func (r *ResourceReconciler) resourcesForChange(ctx context.Context, prototype api.Resource, changed client.Object) []reconcile.Request {
	objects, err := r.resources(ctx, prototype)
	if err != nil {
		return nil
	}

	var requests []reconcile.Request
	for _, obj := range objects {
		if r.matchesChange(obj, changed) {
			requests = append(requests, ctrl.Request{Namespace: obj.GetNamespace(), Name: obj.GetName()})
		}
	}
	return requests
}

func (r *ResourceReconciler) resources(ctx context.Context, prototype api.Resource) ([]api.Resource, error) {
	listObject, err := r.Scheme().New(api.GroupVersion.WithKind("Octelium" + prototype.UpstreamKind() + "List"))
	if err != nil {
		return nil, err
	}

	list := listObject.(client.ObjectList)
	if err = r.List(ctx, list); err != nil {
		return nil, err
	}

	objects, err := meta.ExtractList(list.(runtime.Object))
	if err != nil {
		return nil, err
	}
	resources := make([]api.Resource, 0, len(objects))
	for _, object := range objects {
		resources = append(resources, object.(api.Resource))
	}
	return resources, nil
}

func (r *ResourceReconciler) matchesChange(obj api.Resource, changed client.Object) bool {
	ref := obj.Reference()
	namespace := ref.Namespace
	if namespace == "" {
		namespace = obj.GetNamespace()
	}

	switch change := changed.(type) {
	case *connectionapi.Octelium:
		return namespace == change.Namespace && ref.Name == change.Name
	case *kcore.Namespace:
		return obj.GetNamespace() == change.Name
	case *kcore.Secret:
		return secretChangeMatches(obj, change)
	}
	return false
}

func secretChangeMatches(obj api.Resource, changed *kcore.Secret) bool {
	data := obj.Extension().DataFromSecret
	if data == nil {
		return false
	}

	sourceNamespace := data.Namespace
	if sourceNamespace == "" {
		sourceNamespace = obj.GetNamespace()
	}
	return sourceNamespace == changed.Namespace && data.Name == changed.Name
}
