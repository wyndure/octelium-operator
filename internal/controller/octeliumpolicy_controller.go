package controller

import (
	"context"

	api "github.com/wyndure/octelium-operator/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// OcteliumPolicyReconciler registers the typed CR with the shared translator.
type OcteliumPolicyReconciler struct {
	*ResourceReconciler
}

func (r *OcteliumPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.ReconcileResource(ctx, req, &api.OcteliumPolicy{})
}

func (r *OcteliumPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).Named("octeliumpolicy").For(&api.OcteliumPolicy{})

	return r.WatchConnections(builder, &api.OcteliumPolicy{}).Complete(r)
}
