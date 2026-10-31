package controller

import (
	"context"

	api "github.com/wyndure/octelium-operator/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// OcteliumUserReconciler registers the typed CR with the shared translator.
type OcteliumUserReconciler struct {
	*ResourceReconciler
}

func (r *OcteliumUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.ReconcileResource(ctx, req, &api.OcteliumUser{})
}

func (r *OcteliumUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).Named("octeliumuser").For(&api.OcteliumUser{})

	return r.WatchConnections(builder, &api.OcteliumUser{}).Complete(r)
}
