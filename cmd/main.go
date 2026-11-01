package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	api "github.com/wyndure/octelium-operator/api/core/v1"
	connectionapi "github.com/wyndure/octelium-operator/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

func main() {
	options, err := parseOptions(os.Args[1:], os.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		return
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&options.logging)))

	if err = run(ctrl.SetupSignalHandler(), options); err != nil {
		ctrl.Log.WithName("setup").Error(err, "Operator stopped")
		os.Exit(1)
	}
}

func newScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()

	for _, add := range []func(*runtime.Scheme) error{clientgoscheme.AddToScheme, api.AddToScheme, connectionapi.AddToScheme} {
		if err := add(scheme); err != nil {
			return nil, err
		}
	}

	// +kubebuilder:scaffold:scheme
	return scheme, nil
}
