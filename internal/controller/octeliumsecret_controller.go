package controller

import (
	"context"

	api "github.com/wyndure/octelium-operator/api/core/v1"

	ctrl "sigs.k8s.io/controller-runtime"
)

// OcteliumSecretReconciler registers the typed CR with the shared translator.
type OcteliumSecretReconciler struct {
	*ResourceReconciler
}

func (r *OcteliumSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.ReconcileResource(ctx, req, &api.OcteliumSecret{})
}

func (r *OcteliumSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).Named("octeliumsecret").For(&api.OcteliumSecret{})

	return r.WatchConnections(builder, &api.OcteliumSecret{}).Complete(r)
}
