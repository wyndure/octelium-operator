package main

import (
	"flag"
	"fmt"
	"io"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type options struct {
	kubeconfigPath string
	metricsAddress string
	probeAddress   string
	leaderElection bool
	syncInterval   time.Duration
	logging        zap.Options
}

func parseOptions(args []string, output io.Writer) (options, error) {
	var result options

	flags := flag.NewFlagSet("octelium-operator", flag.ContinueOnError)

	flags.SetOutput(output)

	flags.StringVar(&result.kubeconfigPath, "kubeconfig", "", "Path to Kubernetes client configuration")
	flags.StringVar(&result.metricsAddress, "metrics-bind-address", "0", "Metrics address; 0 disables metrics")
	flags.StringVar(&result.probeAddress, "health-probe-bind-address", ":8081", "Probe address")
	flags.BoolVar(&result.leaderElection, "leader-elect", false, "Enable leader election")
	flags.DurationVar(&result.syncInterval, "sync-interval", 30*time.Second, "External drift resynchronization interval")

	result.logging.BindFlags(flags)

	if err := flags.Parse(args); err != nil {
		return result, err
	}

	if flags.NArg() != 0 {
		return result, fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}

	if result.syncInterval <= 0 {
		return result, fmt.Errorf("sync-interval must be positive")
	}

	return result, nil
}
