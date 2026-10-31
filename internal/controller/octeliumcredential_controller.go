package controller

import (
	"context"

	api "github.com/wyndure/octelium-operator/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// OcteliumCredentialReconciler registers the typed CR with the shared translator.
type OcteliumCredentialReconciler struct {
	*ResourceReconciler
}

func (r *OcteliumCredentialReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.ReconcileResource(ctx, req, &api.OcteliumCredential{})
}

func (r *OcteliumCredentialReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).Named("octeliumcredential").For(&api.OcteliumCredential{})

	return r.WatchConnections(builder, &api.OcteliumCredential{}).Complete(r)
}
