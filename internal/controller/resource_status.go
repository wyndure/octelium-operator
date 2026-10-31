package controller

import (
	"context"

	api "github.com/wyndure/octelium-operator/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ResourceReconciler) failed(ctx context.Context, obj api.Resource, cause error) (ctrl.Result, error) {
	before := obj.DeepCopyObject().(client.Object)

	apimeta.SetStatusCondition(&obj.SyncStatus().Conditions, metav1.Condition{Type: conditionReady, Status: metav1.ConditionFalse, Reason: "SynchronizationFailed", Message: "Synchronization failed; inspect operator logs", ObservedGeneration: obj.GetGeneration()})

	if e := r.Status().Patch(ctx, obj, client.MergeFrom(before)); e != nil {
		return ctrl.Result{}, e
	}

	return ctrl.Result{}, cause
}

const conditionReady = "Ready"
