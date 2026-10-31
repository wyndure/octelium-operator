package connection

import (
	"context"
	"net"
	"testing"
	"time"

	rcore "github.com/octelium/octelium/apis/rsc/rcorev1"
	"github.com/octelium/octelium/cluster/common/spiffec"
	api "github.com/wyndure/octelium-operator/api/v1"
	"google.golang.org/grpc"
)

// This uses the same transport factory as the upstream resource server. Only
// the RPC handler is stubbed so the transport test does not need a database.
func TestInsecureConnectsToUpstreamTransport(t *testing.T) {
	t.Setenv("OCTELIUM_ENABLE_SPIFFE_CSI", "false")
	t.Setenv("SPIFFE_ENDPOINT_SOCKET", "")
	if spiffec.GetSPIFFEEndpointSocket() != "" {
		t.Skip("plaintext upstream transport requires an environment without a mounted SPIFFE socket")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	option, err := spiffec.GetGRPCServerCred(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer(option)
	rcore.RegisterResourceServiceServer(server, testResourceServer{})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)

	target := &api.Octelium{Spec: api.OcteliumSpec{
		Endpoint:    listener.Addr().String(),
		Credentials: api.ConnectionCredentials{Type: api.CredentialsInsecure},
	}}
	remote, closeClient, err := DefaultFactory(ctx, target)
	if err != nil {
		t.Fatal(err)
	}
	defer closeClient()
	if _, err := remote.Get(ctx, "ClusterConfig", "default"); err != nil {
		t.Fatal("plaintext RPC to upstream transport failed", err)
	}
}

func TestSPIFFEDoesNotFallBackToPlaintext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	target := &api.Octelium{Spec: api.OcteliumSpec{
		Endpoint:    "127.0.0.1:8080",
		Credentials: api.ConnectionCredentials{TrustDomain: "test.local", SocketPath: t.TempDir() + "/absent.sock"},
	}}
	if _, _, err := DefaultFactory(ctx, target); err == nil {
		t.Fatal("missing SPIFFE credentials silently accepted")
	}
}

func TestValidateTransport(t *testing.T) {
	for _, test := range []struct {
		name        string
		credentials api.ConnectionCredentials
		valid       bool
	}{
		{"insecure", api.ConnectionCredentials{Type: api.CredentialsInsecure}, true},
		{"spiffe", api.ConnectionCredentials{Type: api.CredentialsSPIFFE, TrustDomain: "test.local"}, true},
		{"default-spiffe", api.ConnectionCredentials{TrustDomain: "test.local"}, true},
		{"default-needs-domain", api.ConnectionCredentials{}, false},
		{"unsupported-tls", api.ConnectionCredentials{Type: "TLS"}, false},
		{"unknown", api.ConnectionCredentials{Type: "Other"}, false},
		{"insecure-domain", api.ConnectionCredentials{Type: api.CredentialsInsecure, TrustDomain: "test.local"}, false},
		{"insecure-socket", api.ConnectionCredentials{Type: api.CredentialsInsecure, SocketPath: "/agent.sock"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			target := &api.Octelium{Spec: api.OcteliumSpec{Endpoint: "server:8080", Credentials: test.credentials}}
			if err := Validate(target); (err == nil) != test.valid {
				t.Fatalf("Validate() = %v, want valid=%v", err, test.valid)
			}
			if !test.valid {
				if _, _, err := DefaultFactory(context.Background(), target); err == nil {
					t.Fatal("factory accepted invalid credentials")
				}
			}
		})
	}
}

func TestSPIFFEConnectsToUpstreamTransport(t *testing.T) {
	socket := startTestWorkloadAPI(t)
	t.Setenv("OCTELIUM_ENABLE_SPIFFE_CSI", "true")
	t.Setenv("SPIFFE_ENDPOINT_SOCKET", socket)
	t.Setenv("OCTELIUM_SPIFFE_TRUST_DOMAIN", "test.local")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	option, err := spiffec.GetGRPCServerCred(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer(option)
	rcore.RegisterResourceServiceServer(server, testResourceServer{})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)

	target := &api.Octelium{Spec: api.OcteliumSpec{
		Endpoint:    listener.Addr().String(),
		Credentials: api.ConnectionCredentials{TrustDomain: "test.local", SocketPath: socket},
	}}
	remote, closeClient, err := DefaultFactory(ctx, target)
	if err != nil {
		t.Fatal(err)
	}
	defer closeClient()
	if _, err := remote.Get(ctx, "ClusterConfig", "default"); err != nil {
		t.Fatal("SPIFFE RPC to upstream transport failed", err)
	}

	target.Spec.Credentials.TrustDomain = "other.local"
	if _, _, err := DefaultFactory(ctx, target); err == nil {
		t.Fatal("mismatched workload trust domain accepted")
	}

	target.Spec.Credentials = api.ConnectionCredentials{Type: api.CredentialsInsecure}
	plain, closePlain, err := DefaultFactory(ctx, target)
	if err != nil {
		t.Fatal(err)
	}
	defer closePlain()
	short, cancelShort := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancelShort()
	if _, err := plain.Get(short, "ClusterConfig", "default"); err == nil {
		t.Fatal("SPIFFE server accepted plaintext")
	}
}
