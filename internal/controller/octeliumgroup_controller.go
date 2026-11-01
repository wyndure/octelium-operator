package controller

import (
	"context"

	api "github.com/wyndure/octelium-operator/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// OcteliumGroupReconciler registers the typed CR with the shared translator.
type OcteliumGroupReconciler struct {
	*ResourceReconciler
}

func (r *OcteliumGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.ReconcileResource(ctx, req, &api.OcteliumGroup{})
}

func (r *OcteliumGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).Named("octeliumgroup").For(&api.OcteliumGroup{})

	return r.WatchConnections(builder, &api.OcteliumGroup{}).Complete(r)
}
