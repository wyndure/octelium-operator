package controller

import (
	"context"
	"time"

	api "github.com/wyndure/octelium-operator/api/v1"
	"github.com/wyndure/octelium-operator/internal/connection"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type OcteliumReconciler struct {
	client.Client
	Connections *connection.Resolver
}

func (r *OcteliumReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var target api.Octelium
	if e := r.Get(ctx, req.NamespacedName, &target); e != nil {
		if client.IgnoreNotFound(e) == nil {
			r.Connections.Forget(req.Namespace, req.Name)
		}
		return ctrl.Result{}, client.IgnoreNotFound(e)
	}
	base := target.DeepCopy()

	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if err := r.checkConnection(timeout, &target); err != nil {
		setConnectionFailure(&target)
	} else {
		setConnectionReady(&target)
	}

	if err := r.patchConnectionStatus(ctx, base, &target); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *OcteliumReconciler) checkConnection(ctx context.Context, target *api.Octelium) error {
	lease, err := r.Connections.Acquire(ctx, target)
	if err != nil {
		return err
	}
	defer lease.Release()

	_, err = lease.Backend.Get(ctx, "ClusterConfig", "default")
	return err
}

func setConnectionReady(target *api.Octelium) {
	message := "SPIFFE-authenticated resource-server connection is ready"
	if target.Spec.Credentials.Type == api.CredentialsInsecure {
		message = "Plaintext resource-server connection is ready (unauthenticated)"
	}
	apimeta.SetStatusCondition(&target.Status.Conditions, metav1.Condition{
		Type:               conditionReady,
		Status:             metav1.ConditionTrue,
		Reason:             "Connected",
		Message:            message,
		ObservedGeneration: target.Generation,
	})
	target.Status.ObservedGeneration = target.Generation
}

func setConnectionFailure(target *api.Octelium) {
	apimeta.SetStatusCondition(&target.Status.Conditions, metav1.Condition{
		Type:               conditionReady,
		Status:             metav1.ConditionFalse,
		Reason:             "ConnectionFailed",
		Message:            "Resource-server endpoint or credentials are unavailable",
		ObservedGeneration: target.Generation,
	})
	target.Status.ObservedGeneration = target.Generation
}

func (r *OcteliumReconciler) patchConnectionStatus(ctx context.Context, base, target *api.Octelium) error {
	return r.Status().Patch(ctx, target, client.MergeFrom(base))
}

func (r *OcteliumReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).Named("octelium").For(&api.Octelium{}).Complete(r)
}
