package main

import (
	"fmt"
	"time"

	"github.com/wyndure/octelium-operator/internal/connection"
	"github.com/wyndure/octelium-operator/internal/controller"
	ctrl "sigs.k8s.io/controller-runtime"
)

func registerControllers(manager ctrl.Manager, connections *connection.Resolver, interval time.Duration) error {
	shared := &controller.ResourceReconciler{Client: manager.GetClient(), Connections: connections, Interval: interval}

	controllers := []struct {
		name       string
		reconciler interface{ SetupWithManager(ctrl.Manager) error }
	}{
		{"Octelium", &controller.OcteliumReconciler{Client: manager.GetClient(), Connections: connections}},
		{"ClusterConfig", &controller.OcteliumClusterConfigReconciler{ResourceReconciler: shared}},
		{"Config", &controller.OcteliumConfigReconciler{ResourceReconciler: shared}},
		{"Policy", &controller.OcteliumPolicyReconciler{ResourceReconciler: shared}},
		{"IdentityProvider", &controller.OcteliumIdentityProviderReconciler{ResourceReconciler: shared}},
		{"Namespace", &controller.OcteliumNamespaceReconciler{ResourceReconciler: shared}},
		{"Group", &controller.OcteliumGroupReconciler{ResourceReconciler: shared}},
		{"User", &controller.OcteliumUserReconciler{ResourceReconciler: shared}},
		{"Service", &controller.OcteliumServiceReconciler{ResourceReconciler: shared}},
		{"Credential", &controller.OcteliumCredentialReconciler{ResourceReconciler: shared}},
		{"Secret", &controller.OcteliumSecretReconciler{ResourceReconciler: shared}},
		{"Region", &controller.OcteliumRegionReconciler{ResourceReconciler: shared}},
	}

	for _, entry := range controllers {
		if err := entry.reconciler.SetupWithManager(manager); err != nil {
			return fmt.Errorf("register %s controller: %w", entry.name, err)
		}
	}

	// +kubebuilder:scaffold:builder
	return nil
}
