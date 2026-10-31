package controller

import (
	"context"

	api "github.com/wyndure/octelium-operator/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// OcteliumServiceReconciler registers the typed CR with the shared translator.
type OcteliumServiceReconciler struct {
	*ResourceReconciler
}

func (r *OcteliumServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.ReconcileResource(ctx, req, &api.OcteliumService{})
}

func (r *OcteliumServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).Named("octeliumservice").For(&api.OcteliumService{})

	return r.WatchConnections(builder, &api.OcteliumService{}).Complete(r)
}
