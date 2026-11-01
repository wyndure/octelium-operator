package controller

import (
	"strings"
	"testing"

	api "github.com/wyndure/octelium-operator/api/v1"
)

func TestReadyConditionDescribesTransport(t *testing.T) {
	for _, mode := range []string{api.CredentialsSPIFFE, api.CredentialsInsecure} {
		target := &api.Octelium{Spec: api.OcteliumSpec{Credentials: api.ConnectionCredentials{Type: mode}}}
		setConnectionReady(target)
		message := target.Status.Conditions[0].Message
		if mode == api.CredentialsInsecure && !strings.Contains(message, "(unauthenticated)") {
			t.Fatalf("plaintext connection misreported: %s", message)
		}
		if mode == api.CredentialsSPIFFE && !strings.Contains(message, "SPIFFE-authenticated") {
			t.Fatalf("SPIFFE connection misreported: %s", message)
		}
	}
}
