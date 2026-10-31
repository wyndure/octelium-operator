package connection

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	api "github.com/wyndure/octelium-operator/api/v1"
)

func spiffeTransport(ctx context.Context, target *api.Octelium) (*tls.Config, func(), error) {
	credentials := target.Spec.Credentials
	trustDomain, err := spiffeid.TrustDomainFromString(credentials.TrustDomain)
	if err != nil {
		return nil, nil, err
	}

	socket := credentials.SocketPath
	if socket == "" {
		socket = "/run/spire/sockets/spire-agent.sock"
	}
	if len(socket) < len("unix://") || socket[:len("unix://")] != "unix://" {
		socket = "unix://" + socket
	}

	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	source, err := workloadapi.NewX509Source(timeout, workloadapi.WithClientOptions(workloadapi.WithAddr(socket)))
	if err != nil {
		return nil, nil, fmt.Errorf("SPIFFE workload credentials unavailable: %w", err)
	}

	svid, err := source.GetX509SVID()
	if err != nil {
		_ = source.Close()
		return nil, nil, err
	}
	if svid.ID.TrustDomain() != trustDomain {
		_ = source.Close()
		return nil, nil, fmt.Errorf("workload SVID trust domain does not match connection credentials")
	}

	return tlsconfig.MTLSClientConfig(source, source, tlsconfig.AuthorizeMemberOf(trustDomain)), func() {
		_ = source.Close()
	}, nil
}
