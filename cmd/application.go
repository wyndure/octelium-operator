package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/wyndure/octelium-operator/internal/connection"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

func run(ctx context.Context, options options) error {
	config, err := loadKubernetesConfig(options.kubeconfigPath)
	if err != nil {
		return fmt.Errorf("load Kubernetes configuration: %w", err)
	}

	manager, err := newManager(config, options)
	if err != nil {
		return err
	}

	connections := &connection.Resolver{Reader: manager.GetAPIReader()}

	defer connections.Close()

	if err = registerControllers(manager, connections, options.syncInterval); err != nil {
		return err
	}

	if err = registerProbes(manager); err != nil {
		return err
	}

	if err = manager.Start(ctx); err != nil {
		return fmt.Errorf("run manager: %w", err)
	}

	return nil
}

func newManager(config *rest.Config, options options) (ctrl.Manager, error) {
	scheme, err := newScheme()
	if err != nil {
		return nil, fmt.Errorf("register Kubernetes API types: %w", err)
	}

	manager, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: options.metricsAddress},
		HealthProbeBindAddress: options.probeAddress,
		LeaderElection:         options.leaderElection,
		LeaderElectionID:       "octelium-operator.octelium.wyndure.com",
	})
	if err != nil {
		return nil, fmt.Errorf("create manager: %w", err)
	}

	return manager, nil
}

func registerProbes(manager ctrl.Manager) error {
	if err := manager.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return fmt.Errorf("register liveness probe: %w", err)
	}

	if err := manager.AddReadyzCheck("readyz", func(request *http.Request) error {
		ctx, cancel := context.WithTimeout(request.Context(), time.Second)

		defer cancel()

		if !manager.GetCache().WaitForCacheSync(ctx) {
			return fmt.Errorf("controller cache has not synchronized")
		}

		return nil
	}); err != nil {
		return fmt.Errorf("register readiness probe: %w", err)
	}

	return nil
}

func loadKubernetesConfig(path string) (*rest.Config, error) {
	if path != "" {
		return clientcmd.BuildConfigFromFlags("", path)
	}

	return ctrl.GetConfig()
}
