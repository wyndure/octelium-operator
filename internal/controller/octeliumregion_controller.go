package controller

import (
	"context"

	api "github.com/wyndure/octelium-operator/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// OcteliumRegionReconciler registers the typed CR with the shared translator.
type OcteliumRegionReconciler struct {
	*ResourceReconciler
}

func (r *OcteliumRegionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.ReconcileResource(ctx, req, &api.OcteliumRegion{})
}

func (r *OcteliumRegionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).Named("octeliumregion").For(&api.OcteliumRegion{})

	return r.WatchConnections(builder, &api.OcteliumRegion{}).Complete(r)
}
