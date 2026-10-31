package controller

import (
	"context"

	api "github.com/wyndure/octelium-operator/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// OcteliumClusterConfigReconciler registers the typed CR with the shared translator.
type OcteliumClusterConfigReconciler struct {
	*ResourceReconciler
}

func (r *OcteliumClusterConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.ReconcileResource(ctx, req, &api.OcteliumClusterConfig{})
}

func (r *OcteliumClusterConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).Named("octeliumclusterconfig").For(&api.OcteliumClusterConfig{})

	return r.WatchConnections(builder, &api.OcteliumClusterConfig{}).Complete(r)
}
