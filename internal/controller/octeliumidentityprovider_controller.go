package controller

import (
	"context"

	api "github.com/wyndure/octelium-operator/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// OcteliumIdentityProviderReconciler registers the typed CR with the shared translator.
type OcteliumIdentityProviderReconciler struct {
	*ResourceReconciler
}

func (r *OcteliumIdentityProviderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.ReconcileResource(ctx, req, &api.OcteliumIdentityProvider{})
}

func (r *OcteliumIdentityProviderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).Named("octeliumidentityprovider").For(&api.OcteliumIdentityProvider{})

	return r.WatchConnections(builder, &api.OcteliumIdentityProvider{}).Complete(r)
}
