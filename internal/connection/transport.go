package connection

import (
	"context"

	api "github.com/wyndure/octelium-operator/api/v1"
	"github.com/wyndure/octelium-operator/internal/backend"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

func DefaultFactory(ctx context.Context, target *api.Octelium) (backend.Interface, func(), error) {
	if err := Validate(target); err != nil {
		return nil, nil, err
	}

	var transport credentials.TransportCredentials
	closeSource := func() {}
	if target.Spec.Credentials.Type == api.CredentialsInsecure {
		transport = insecure.NewCredentials()
	} else {
		config, closeSPIFFE, err := spiffeTransport(ctx, target)
		if err != nil {
			return nil, nil, err
		}
		transport = credentials.NewTLS(config)
		closeSource = closeSPIFFE
	}

	conn, err := grpc.NewClient(target.Spec.Endpoint, grpc.WithTransportCredentials(transport))
	if err != nil {
		closeSource()
		return nil, nil, err
	}

	return backend.New(remoteClient{conn: conn}), func() { _ = conn.Close(); closeSource() }, nil
}
