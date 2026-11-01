package controller

import (
	"context"
	"strings"

	core "github.com/octelium/octelium/apis/main/corev1"
	meta "github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	api "github.com/wyndure/octelium-operator/api/core/v1"
	connectionapi "github.com/wyndure/octelium-operator/api/v1"
	"github.com/wyndure/octelium-operator/internal/backend"
	"github.com/wyndure/octelium-operator/internal/connection"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	kcore "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type provisionBackend struct {
	memoryBackend
	creates int
	updates int
}

func (m *provisionBackend) Create(ctx context.Context, in umetav1.ResourceObjectI) (umetav1.ResourceObjectI, error) {
	m.creates++
	return m.memoryBackend.Apply(ctx, in)
}
func (m *provisionBackend) Apply(ctx context.Context, in umetav1.ResourceObjectI) (umetav1.ResourceObjectI, error) {
	m.updates++
	return m.memoryBackend.Apply(ctx, in)
}

var _ = Describe("Provision strategy", func() {
	It("creates once, observes drift without writes, refreshes metadata output and honors deletion policy", func() {
		for _, policy := range []string{"Delete", "Retain"} {
			remote := &provisionBackend{}
			r := &ResourceReconciler{Client: k8sClient, Connections: &connection.Resolver{Reader: k8sClient, Factory: func(context.Context, *connectionapi.Octelium) (backend.Interface, func(), error) {
				return remote, func() {}, nil
			}}}
			obj := &api.OcteliumNamespace{Name: "provision-" + strings.ToLower(policy), Namespace: testNamespace, MetadataConfigMap: "provision-metadata-" + strings.ToLower(policy), OcteliumRef: connectionapi.OcteliumReference{Name: testConnectionName}, Spec: &core.Namespace_Spec{Attrs: &structpb.Struct{Fields: map[string]*structpb.Value{"desired": structpb.NewStringValue("original")}}}, Octelium: api.ResourceOptions{ReconciliationPolicy: api.PolicyProvision, DeletionPolicy: policy, Metadata: &meta.Metadata{Description: "initial"}}}
			Expect(k8sClient.Create(ctx, obj)).To(Succeed())
			req := ctrl.Request{Name: obj.Name, Namespace: obj.Namespace}
			reconcile := func() {
				_, err := r.ReconcileResource(ctx, req, &api.OcteliumNamespace{})
				Expect(err).NotTo(HaveOccurred())
			}
			reconcile()
			reconcile()
			reconcile()
			Expect(remote.creates).To(Equal(1))
			Expect(remote.updates).To(BeZero())
			remote.object.(*core.Namespace).Spec.Attrs.Fields["external"] = structpb.NewStringValue("drift")
			remote.object.GetMetadata().Description = "externally-updated"
			Expect(k8sClient.Get(ctx, req.NamespacedName, obj)).To(Succeed())
			obj.Spec.Attrs.Fields["desired"] = structpb.NewStringValue("changed-in-cr")
			// Existing resources must not depend on create-time Secret sources.
			obj.Octelium.DataFromSecret = &api.SecretDataSource{Name: "missing-source", Key: "value"}
			Expect(k8sClient.Update(ctx, obj)).To(Succeed())
			before := proto.Clone(remote.object)
			reconcile()
			Expect(proto.Equal(before, remote.object)).To(BeTrue())
			Expect(remote.creates).To(Equal(1))
			Expect(remote.updates).To(BeZero())
			cm := &kcore.ConfigMap{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: obj.MetadataConfigMap, Namespace: obj.Namespace}, cm)).To(Succeed())
			Expect(cm.Data["description"]).To(Equal("externally-updated"))
			Expect(k8sClient.Get(ctx, req.NamespacedName, obj)).To(Succeed())
			Expect(k8sClient.Delete(ctx, obj)).To(Succeed())
			reconcile()
			if policy == "Delete" {
				Expect(remote.deletes).To(Equal(1))
			} else {
				Expect(remote.deletes).To(BeZero())
			}
		}
	})

	It("defaults reconciliation policy and validates it for every managed kind", func() {
		for _, obj := range api.Resources() {
			obj.SetName("policy-check")
			obj.SetNamespace(testNamespace)
			*obj.Reference() = connectionapi.OcteliumReference{Name: testConnectionName}
			Expect(k8sClient.Create(ctx, obj)).To(Succeed())
			Expect(obj.GetReconciliationPolicy()).To(Equal(api.PolicyReconcile))
			Expect(k8sClient.Delete(ctx, obj)).To(Succeed())
		}
		invalid := &api.OcteliumNamespace{Name: "invalid-policy", Namespace: testNamespace, Octelium: api.ResourceOptions{ReconciliationPolicy: "Ignore"}, OcteliumRef: connectionapi.OcteliumReference{Name: testConnectionName}, Spec: &core.Namespace_Spec{}}
		Expect(k8sClient.Create(ctx, invalid)).NotTo(Succeed())
	})
})
