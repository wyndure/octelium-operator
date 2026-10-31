package controller

import (
	"context"

	api "github.com/wyndure/octelium-operator/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// OcteliumConfigReconciler registers the typed CR with the shared translator.
type OcteliumConfigReconciler struct {
	*ResourceReconciler
}

func (r *OcteliumConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.ReconcileResource(ctx, req, &api.OcteliumConfig{})
}

func (r *OcteliumConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).Named("octeliumconfig").For(&api.OcteliumConfig{})

	return r.WatchConnections(builder, &api.OcteliumConfig{}).Complete(r)
}
