package controller

import (
	"context"
	"fmt"

	connectionapi "github.com/wyndure/octelium-operator/api/v1"
	"github.com/wyndure/octelium-operator/internal/backend"
	"github.com/wyndure/octelium-operator/internal/connection"
	kcore "k8s.io/api/core/v1"

	core "github.com/octelium/octelium/apis/main/corev1"
	meta "github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	api "github.com/wyndure/octelium-operator/api/core/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type memoryBackend struct {
	object  umetav1.ResourceObjectI
	failure error
	deletes int
}

func (m *memoryBackend) Get(_ context.Context, _, _ string) (umetav1.ResourceObjectI, error) {
	if m.object == nil {
		return nil, status.Error(codes.NotFound, "missing")
	}
	return proto.Clone(m.object).(umetav1.ResourceObjectI), nil
}

func (m *memoryBackend) Apply(_ context.Context, in umetav1.ResourceObjectI) (umetav1.ResourceObjectI, error) {
	if m.failure != nil {
		return nil, m.failure
	}
	out := proto.Clone(in).(umetav1.ResourceObjectI)
	out.GetMetadata().Uid = "external-uid"
	out.GetMetadata().ResourceVersion = "external-version"
	m.object = out
	return out, nil
}

func (m *memoryBackend) Delete(_ context.Context, _, uid string) error {
	if uid != "external-uid" {
		return fmt.Errorf("wrong external identity")
	}
	m.deletes++
	m.object = nil
	return nil
}

var _ = Describe("Typed CR translation against a real Kubernetes API", func() {
	It("creates with upstream spec and synchronizes native and Octelium metadata in both directions", func() {
		name := "translation"
		remote := &memoryBackend{}
		r := &ResourceReconciler{Client: k8sClient, Connections: &connection.Resolver{Reader: k8sClient, Factory: func(context.Context, *connectionapi.Octelium) (backend.Interface, func(), error) {
			return remote, func() {}, nil
		}}}
		obj := &api.OcteliumNamespace{Name: name, Labels: map[string]string{"local": "one"}, Spec: &core.Namespace_Spec{}, Octelium: api.ResourceOptions{Metadata: &meta.Metadata{DisplayName: "A namespace"}}}
		obj.SetNamespace(testNamespace)
		*obj.Reference() = connectionapi.OcteliumReference{Name: testConnectionName}
		Expect(k8sClient.Create(ctx, obj)).To(Succeed())
		req := ctrl.Request{Name: name, Namespace: testNamespace}
		_, e := r.ReconcileResource(ctx, req, &api.OcteliumNamespace{})
		Expect(e).NotTo(HaveOccurred())
		_, e = r.ReconcileResource(ctx, req, &api.OcteliumNamespace{})
		Expect(e).NotTo(HaveOccurred())
		Expect(remote.object.(*core.Namespace).Spec).NotTo(BeNil())
		Expect(remote.object.GetMetadata().Labels).To(HaveKeyWithValue("local", "one"))
		Expect(k8sClient.Get(ctx, req.NamespacedName, obj)).To(Succeed())
		Expect(obj.Octelium.Metadata.Uid).To(Equal("external-uid"))
		Expect(apimeta.IsStatusConditionTrue(obj.Status.Conditions, conditionReady)).To(BeTrue())
		remote.object.GetMetadata().Labels["external"] = "two"
		remote.object.GetMetadata().Description = "Changed outside Kubernetes"
		obj.Annotations = map[string]string{"kubectl.kubernetes.io/last-applied-configuration": "private-client-state", "user": "annotation"}
		Expect(k8sClient.Update(ctx, obj)).To(Succeed())
		_, e = r.ReconcileResource(ctx, req, &api.OcteliumNamespace{})
		Expect(e).NotTo(HaveOccurred())
		Expect(k8sClient.Get(ctx, req.NamespacedName, obj)).To(Succeed())
		Expect(obj.Labels).To(HaveKeyWithValue("external", "two"))
		Expect(obj.Octelium.Metadata.Description).To(Equal("Changed outside Kubernetes"))
		Expect(remote.object.GetMetadata().Annotations).NotTo(HaveKey("kubectl.kubernetes.io/last-applied-configuration"))
		delete(obj.Labels, "local")
		Expect(k8sClient.Update(ctx, obj)).To(Succeed())
		_, e = r.ReconcileResource(ctx, req, &api.OcteliumNamespace{})
		Expect(e).NotTo(HaveOccurred())
		Expect(remote.object.GetMetadata().Labels).NotTo(HaveKey("local"))
		Expect(k8sClient.Get(ctx, req.NamespacedName, obj)).To(Succeed())
		Expect(k8sClient.Delete(ctx, obj)).To(Succeed())
		_, e = r.ReconcileResource(ctx, req, &api.OcteliumNamespace{})
		Expect(e).NotTo(HaveOccurred())
		Expect(remote.deletes).To(Equal(1))
	})
	It("publishes metadata to an owned ConfigMap and restores deleted or modified output", func() {
		remote := &memoryBackend{}
		r := &ResourceReconciler{Client: k8sClient, Connections: &connection.Resolver{Reader: k8sClient, Factory: func(context.Context, *connectionapi.Octelium) (backend.Interface, func(), error) {
			return remote, func() {}, nil
		}}}
		obj := &api.OcteliumNamespace{Name: "metadata-export", Namespace: testNamespace,
			MetadataConfigMap: "metadata-export.output", Labels: map[string]string{"team": "platform"},
			OcteliumRef: connectionapi.OcteliumReference{Name: testConnectionName}, Spec: &core.Namespace_Spec{}}
		Expect(k8sClient.Create(ctx, obj)).To(Succeed())
		req := ctrl.Request{Name: obj.Name, Namespace: obj.Namespace}
		reconcile := func() {
			_, err := r.ReconcileResource(ctx, req, &api.OcteliumNamespace{})
			Expect(err).NotTo(HaveOccurred())
		}
		reconcile()
		reconcile()
		cm := &kcore.ConfigMap{}
		cmKey := client.ObjectKey{Name: obj.MetadataConfigMap, Namespace: obj.Namespace}
		Expect(k8sClient.Get(ctx, cmKey, cm)).To(Succeed())
		Expect(cm.Data).To(HaveKeyWithValue("name", obj.Name))
		Expect(cm.Data).To(HaveKeyWithValue("uid", "external-uid"))
		Expect(cm.Data).To(HaveKeyWithValue("resourceVersion", "external-version"))
		Expect(cm.Data).To(HaveKeyWithValue("labels", `{"team":"platform"}`))
		Expect(metav1.IsControlledBy(cm, obj)).To(BeTrue())
		Expect(k8sClient.Get(ctx, req.NamespacedName, obj)).To(Succeed())
		Expect(apimeta.IsStatusConditionTrue(obj.Status.Conditions, conditionReady)).To(BeTrue())
		cm.Data["uid"] = "tampered"
		cm.Data["unrelated"] = "preserved"
		Expect(k8sClient.Update(ctx, cm)).To(Succeed())
		reconcile()
		Expect(k8sClient.Get(ctx, cmKey, cm)).To(Succeed())
		Expect(cm.Data).To(HaveKeyWithValue("uid", "external-uid"))
		Expect(cm.Data).To(HaveKeyWithValue("unrelated", "preserved"))
		Expect(k8sClient.Delete(ctx, cm)).To(Succeed())
		reconcile()
		Expect(k8sClient.Get(ctx, cmKey, cm)).To(Succeed())
		// Removing the option stops publication, including recreation.
		Expect(k8sClient.Get(ctx, req.NamespacedName, obj)).To(Succeed())
		obj.MetadataConfigMap = ""
		Expect(k8sClient.Update(ctx, obj)).To(Succeed())
		Expect(k8sClient.Delete(ctx, cm)).To(Succeed())
		reconcile()
		Expect(apierrors.IsNotFound(k8sClient.Get(ctx, cmKey, cm))).To(BeTrue())
		// Existing output belonging to someone else must never be overwritten.
		foreign := &kcore.ConfigMap{Name: cmKey.Name, Namespace: cmKey.Namespace, Data: map[string]string{"uid": "someone-else"}}
		Expect(k8sClient.Create(ctx, foreign)).To(Succeed())
		Expect(k8sClient.Get(ctx, req.NamespacedName, obj)).To(Succeed())
		obj.MetadataConfigMap = cmKey.Name
		Expect(k8sClient.Update(ctx, obj)).To(Succeed())
		_, err := r.ReconcileResource(ctx, req, &api.OcteliumNamespace{})
		Expect(err).To(MatchError(ContainSubstring("not owned")))
		Expect(k8sClient.Get(ctx, cmKey, cm)).To(Succeed())
		Expect(cm.Data).To(HaveKeyWithValue("uid", "someone-else"))
		Expect(k8sClient.Get(ctx, req.NamespacedName, obj)).To(Succeed())
		Expect(apimeta.IsStatusConditionFalse(obj.Status.Conditions, conditionReady)).To(BeTrue())
	})

	It("defaults Secret data sources to the managed CR namespace", func() {
		source := &kcore.Secret{Name: "local-data-source", Namespace: testNamespace, Data: map[string][]byte{"value": []byte("payload")}}
		Expect(k8sClient.Create(ctx, source)).To(Succeed())
		obj := &api.OcteliumSecret{Name: "secret-data-source", Namespace: testNamespace,
			OcteliumRef: connectionapi.OcteliumReference{Name: testConnectionName}, Spec: &core.Secret_Spec{},
			Octelium: api.ResourceOptions{DataFromSecret: &api.SecretDataSource{Name: source.Name, Key: "value"}}}
		Expect(k8sClient.Create(ctx, obj)).To(Succeed())
		desired := &core.Secret{}
		r := &ResourceReconciler{Client: k8sClient}
		digest, err := r.prepareData(ctx, obj, nil, desired)
		Expect(err).NotTo(HaveOccurred())
		Expect(digest).NotTo(BeEmpty())
		Expect(desired.Data.GetValueBytes()).To(Equal([]byte("payload")))
		obj.Octelium.DataFromSecret.Namespace = "another-namespace"
		_, err = r.prepareData(ctx, obj, nil, desired)
		Expect(err).To(MatchError(ContainSubstring("managed resource namespace")))
	})

	It("reports upstream errors and retains external resources when requested", func() {
		name := "retain-on-delete"
		remote := &memoryBackend{failure: fmt.Errorf("upstream rejected desired spec")}
		r := &ResourceReconciler{Client: k8sClient, Connections: &connection.Resolver{Reader: k8sClient, Factory: func(context.Context, *connectionapi.Octelium) (backend.Interface, func(), error) {
			return remote, func() {}, nil
		}}}
		obj := &api.OcteliumNamespace{Name: name, Spec: &core.Namespace_Spec{}, Octelium: api.ResourceOptions{DeletionPolicy: "Retain"}}
		obj.SetNamespace(testNamespace)
		*obj.Reference() = connectionapi.OcteliumReference{Name: testConnectionName}
		Expect(k8sClient.Create(ctx, obj)).To(Succeed())
		req := ctrl.Request{Name: name, Namespace: testNamespace}
		_, e := r.ReconcileResource(ctx, req, &api.OcteliumNamespace{})
		Expect(e).NotTo(HaveOccurred())
		_, e = r.ReconcileResource(ctx, req, &api.OcteliumNamespace{})
		Expect(e).To(HaveOccurred())
		Expect(k8sClient.Get(ctx, req.NamespacedName, obj)).To(Succeed())
		Expect(apimeta.IsStatusConditionFalse(obj.Status.Conditions, conditionReady)).To(BeTrue())
		remote.failure = nil
		_, e = r.ReconcileResource(ctx, req, &api.OcteliumNamespace{})
		Expect(e).NotTo(HaveOccurred())
		Expect(k8sClient.Get(ctx, req.NamespacedName, obj)).To(Succeed())
		Expect(k8sClient.Delete(ctx, obj)).To(Succeed())
		_, e = r.ReconcileResource(ctx, req, &api.OcteliumNamespace{})
		Expect(e).NotTo(HaveOccurred())
		Expect(remote.deletes).To(BeZero())
		Expect(remote.object).NotTo(BeNil())
	})
	It("refuses to retarget an existing binding to another Octelium instance", func() {
		other := &connectionapi.Octelium{Name: "other", Namespace: testNamespace, Spec: connectionapi.OcteliumSpec{Endpoint: "other.invalid:8080", Credentials: connectionapi.ConnectionCredentials{Type: "SPIFFE", TrustDomain: "test.local"}}}
		Expect(k8sClient.Create(ctx, other)).To(Succeed())
		remote := &memoryBackend{}
		resolver := &connection.Resolver{Reader: k8sClient, Factory: func(context.Context, *connectionapi.Octelium) (backend.Interface, func(), error) {
			return remote, func() {}, nil
		}}
		defer resolver.Close()
		r := &ResourceReconciler{Client: k8sClient, Connections: resolver}
		obj := &api.OcteliumNamespace{Name: "bound", Namespace: testNamespace, OcteliumRef: connectionapi.OcteliumReference{Name: testConnectionName}, Spec: &core.Namespace_Spec{}, Octelium: api.ResourceOptions{DeletionPolicy: "Retain"}}
		Expect(k8sClient.Create(ctx, obj)).To(Succeed())
		req := ctrl.Request{Name: "bound", Namespace: testNamespace}
		_, e := r.ReconcileResource(ctx, req, &api.OcteliumNamespace{})
		Expect(e).NotTo(HaveOccurred())
		_, e = r.ReconcileResource(ctx, req, &api.OcteliumNamespace{})
		Expect(e).NotTo(HaveOccurred())
		Expect(k8sClient.Get(ctx, req.NamespacedName, obj)).To(Succeed())
		original := obj.Status.ConnectionUID
		obj.OcteliumRef.Name = "other"
		Expect(k8sClient.Update(ctx, obj)).To(Succeed())
		_, e = r.ReconcileResource(ctx, req, &api.OcteliumNamespace{})
		Expect(e).To(HaveOccurred())
		Expect(k8sClient.Get(ctx, req.NamespacedName, obj)).To(Succeed())
		Expect(obj.Status.ConnectionUID).To(Equal(original))
		Expect(apimeta.IsStatusConditionFalse(obj.Status.Conditions, conditionReady)).To(BeTrue())
	})

})

const (
	testNamespace      = "tenant"
	testConnectionName = "test"
)
