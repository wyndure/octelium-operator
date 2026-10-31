package connection

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	core "github.com/octelium/octelium/apis/main/corev1"
	rcore "github.com/octelium/octelium/apis/rsc/rcorev1"
	rmeta "github.com/octelium/octelium/apis/rsc/rmetav1"
	workload "github.com/spiffe/go-spiffe/v2/proto/spiffe/workload"
	"google.golang.org/grpc"
)

type testResourceServer struct {
	rcore.UnimplementedResourceServiceServer
}

func (testResourceServer) GetClusterConfig(context.Context, *rmeta.GetOptions) (*core.ClusterConfig, error) {
	return &core.ClusterConfig{Kind: "ClusterConfig"}, nil
}

func issueTestCertificate(t *testing.T, ca *x509.Certificate, caKey *ecdsa.PrivateKey, serial int64, id string) ([]byte, []byte) {
	t.Helper()

	key, err := ecdsa.GenerateKey(caKey.Curve, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	uri, err := url.Parse(id)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(serial),
		NotBefore:             ca.NotBefore,
		NotAfter:              ca.NotAfter,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		URIs:                  []*url.URL{uri},
	}
	certificate, err := x509.CreateCertificate(rand.Reader, template, ca, &key.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	private, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: private})
}

func newTestCA(t *testing.T) (*x509.Certificate, *ecdsa.PrivateKey, []byte) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test CA"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return certificate, key, der
}

// testWorkloadAPI supplies an SVID over the real Workload API protocol.
// Production code still obtains and verifies credentials through go-spiffe.
type testWorkloadAPI struct {
	workload.UnimplementedSpiffeWorkloadAPIServer
	response *workload.X509SVIDResponse
}

func (s testWorkloadAPI) FetchX509SVID(_ *workload.X509SVIDRequest, stream grpc.ServerStreamingServer[workload.X509SVIDResponse]) error {
	if err := stream.Send(s.response); err != nil {
		return err
	}
	<-stream.Context().Done()
	return stream.Context().Err()
}

func startTestWorkloadAPI(t *testing.T) string {
	t.Helper()
	ca, caKey, bundle := newTestCA(t)
	certificate, key := issueTestCertificate(t, ca, caKey, 2, "spiffe://test.local/operator")
	certBlock, _ := pem.Decode(certificate)
	keyBlock, _ := pem.Decode(key)
	server := grpc.NewServer()
	workload.RegisterSpiffeWorkloadAPIServer(server, testWorkloadAPI{response: &workload.X509SVIDResponse{
		Svids: []*workload.X509SVID{{SpiffeId: "spiffe://test.local/operator", X509Svid: certBlock.Bytes, X509SvidKey: keyBlock.Bytes, Bundle: bundle}},
	}})
	socket := filepath.Join(t.TempDir(), "agent.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	return socket
}
