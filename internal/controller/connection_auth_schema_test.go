package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	api "github.com/wyndure/octelium-operator/api/v1"
)

var _ = Describe("Connection transport admission", func() {
	It("admits explicit plaintext without adding SPIFFE fields", func() {
		target := &api.Octelium{Name: "plaintext", Namespace: testNamespace, Spec: api.OcteliumSpec{
			Endpoint: "server:8080", Credentials: api.ConnectionCredentials{Type: api.CredentialsInsecure},
		}}
		Expect(k8sClient.Create(ctx, target)).To(Succeed())
		DeferCleanup(func() { Expect(k8sClient.Delete(ctx, target)).To(Succeed()) })
		Expect(target.Spec.Credentials.Type).To(Equal(api.CredentialsInsecure))
		Expect(target.Spec.Credentials.SocketPath).To(BeEmpty())
		Expect(target.Spec.Credentials.TrustDomain).To(BeEmpty())
	})

	It("retains SPIFFE as the default", func() {
		target := &api.Octelium{Name: "default-auth", Namespace: testNamespace, Spec: api.OcteliumSpec{
			Endpoint: "server:8080", Credentials: api.ConnectionCredentials{TrustDomain: "test.local"},
		}}
		Expect(k8sClient.Create(ctx, target)).To(Succeed())
		DeferCleanup(func() { Expect(k8sClient.Delete(ctx, target)).To(Succeed()) })
		Expect(target.Spec.Credentials.Type).To(Equal(api.CredentialsSPIFFE))
	})

	It("rejects unsupported TLS and conflicting plaintext settings", func() {
		for _, credentials := range []api.ConnectionCredentials{
			{Type: "TLS"},
			{Type: api.CredentialsInsecure, TrustDomain: "test.local"},
			{Type: api.CredentialsInsecure, SocketPath: "/agent.sock"},
			{},
		} {
			target := &api.Octelium{Name: "invalid-auth", Namespace: testNamespace, Spec: api.OcteliumSpec{
				Endpoint: "server:8080", Credentials: credentials,
			}}
			Expect(k8sClient.Create(ctx, target)).NotTo(Succeed())
		}
	})
})
