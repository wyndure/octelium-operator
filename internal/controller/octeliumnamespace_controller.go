package controller

import (
	"context"

	api "github.com/wyndure/octelium-operator/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// OcteliumNamespaceReconciler registers the typed CR with the shared translator.
type OcteliumNamespaceReconciler struct {
	*ResourceReconciler
}

func (r *OcteliumNamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.ReconcileResource(ctx, req, &api.OcteliumNamespace{})
}

func (r *OcteliumNamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).Named("octeliumnamespace").For(&api.OcteliumNamespace{})

	return r.WatchConnections(builder, &api.OcteliumNamespace{}).Complete(r)
}
