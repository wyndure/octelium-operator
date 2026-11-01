package backend

import (
	"context"
	"net"
	"strings"
	"testing"

	core "github.com/octelium/octelium/apis/main/corev1"
	meta "github.com/octelium/octelium/apis/main/metav1"
	rcore "github.com/octelium/octelium/apis/rsc/rcorev1"
	rmeta "github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/octeliumc"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	api "github.com/wyndure/octelium-operator/api/core/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type internalClient struct {
	octeliumc.ClientInterface
	core rcore.ResourceServiceClient
}

func (c internalClient) CoreC() rcore.ResourceServiceClient { return c.core }

type internalMemory struct {
	objects                   map[string]umetav1.ResourceObjectI
	creates, updates, deletes int
}

func (m *internalMemory) intercept(_ context.Context, request any, info *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (any, error) {
	method := info.FullMethod[strings.LastIndex(info.FullMethod, "/")+1:]
	if kind, ok := strings.CutPrefix(method, "Get"); ok {
		options := request.(*rmeta.GetOptions)
		name := options.Name
		if options.Uid != "" {
			for key, object := range m.objects {
				if strings.HasPrefix(key, kind+"/") && object.GetMetadata().Uid == options.Uid {
					return proto.Clone(object), nil
				}
			}
		}
		if object := m.objects[kind+"/"+name]; object != nil {
			return proto.Clone(object), nil
		}
		return nil, status.Error(codes.NotFound, "missing")
	}
	if kind, ok := strings.CutPrefix(method, "Delete"); ok {
		uid := request.(*rmeta.DeleteOptions).Uid
		for key, object := range m.objects {
			if strings.HasPrefix(key, kind+"/") && object.GetMetadata().Uid == uid {
				delete(m.objects, key)
				m.deletes++
				return &rmeta.OperationResult{}, nil
			}
		}
		return nil, status.Error(codes.NotFound, "missing")
	}
	in := request.(umetav1.ResourceObjectI)
	kind := strings.TrimPrefix(strings.TrimPrefix(method, "Create"), "Update")
	key := kind + "/" + in.GetMetadata().Name
	if strings.HasPrefix(method, "Create") {
		m.creates++
		if m.objects[key] != nil {
			return nil, status.Error(codes.AlreadyExists, "concurrent creation")
		}
	} else if strings.HasPrefix(method, "Update") {
		m.updates++
	}
	out := proto.Clone(in).(umetav1.ResourceObjectI)
	out.ProtoReflect().Set(out.ProtoReflect().Descriptor().Fields().ByName("kind"), protoreflect.ValueOfString(kind))
	out.GetMetadata().Uid = "ae675c45-e1dc-479c-96ec-13e73ddbb40b"
	out.GetMetadata().ResourceVersion = "v1"
	m.objects[key] = out
	return out, nil
}
func systemFixture(t *testing.T) (*Backend, *internalMemory) {
	t.Helper()
	m := &internalMemory{objects: map[string]umetav1.ResourceObjectI{}}
	m.objects["Namespace/default"] = &core.Namespace{Kind: "Namespace", Metadata: &meta.Metadata{Name: "default", Uid: "6a555459-b538-4936-b342-c48c57ad63f7"}, Spec: &core.Namespace_Spec{}, Status: &core.Namespace_Status{}}
	m.objects["Region/default"] = &core.Region{Kind: "Region", Metadata: &meta.Metadata{Name: "default", Uid: "6a555459-b538-4936-b342-c48c57ad63f7"}, Spec: &core.Region_Spec{}, Status: &core.Region_Status{}}
	srv := grpc.NewServer(grpc.UnaryInterceptor(m.intercept))
	rcore.RegisterResourceServiceServer(srv, &rcore.UnimplementedResourceServiceServer{})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve(listener) }()
	t.Cleanup(srv.Stop)
	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return New(internalClient{core: rcore.NewResourceServiceClient(conn)}), m
}

func TestSystemProvisioningEveryKind(t *testing.T) {
	for _, resource := range api.Resources() {
		t.Run(resource.UpstreamKind(), func(t *testing.T) {
			b, m := systemFixture(t)
			object, err := ucorev1.NewObject(resource.UpstreamKind())
			if err != nil {
				t.Fatal(err)
			}
			// Populate protobuf fields through the upstream JSON codec.
			if err := protojson.Unmarshal([]byte(`{"kind":"`+resource.UpstreamKind()+`","metadata":{"name":"managed","isSystem":true,"description":"initial"},"spec":{}}`), object); err != nil {
				t.Fatal(err)
			}
			if svc, ok := object.(*core.Service); ok {
				svc.Metadata.Name = "managed.default"
				svc.Spec.Mode = core.Service_Spec_WEB
				svc.Spec.Port = 8080
				svc.Status = &core.Service_Status{ManagedService: &core.Service_Status_ManagedService{Image: "example/managed:1"}}
			}
			if resource.UpstreamKind() == "ClusterConfig" {
				if _, err = b.Apply(t.Context(), object); status.Code(err) != codes.NotFound {
					t.Fatalf("missing bootstrap = %v", err)
				}
				m.objects["ClusterConfig/managed"] = proto.Clone(object).(umetav1.ResourceObjectI)
			}
			out, err := b.Apply(t.Context(), object)
			if err != nil {
				t.Fatal(err)
			}
			creates := m.creates
			if _, err = b.Apply(t.Context(), object); err != nil {
				t.Fatal(err)
			}
			if m.creates != creates || m.updates != 0 {
				t.Fatalf("not idempotent: creates=%d updates=%d", m.creates, m.updates)
			}
			if svc, ok := out.(*core.Service); ok {
				if svc.Status.NamespaceRef == nil || svc.Status.RegionRef == nil || svc.Status.ManagedService.Image != "example/managed:1" {
					t.Fatal("upstream Service initialization was lost")
				}
			}
			object.GetMetadata().Description = "changed"
			if _, err = b.Apply(t.Context(), object); err != nil {
				t.Fatal(err)
			}
			if m.updates != 1 {
				t.Fatal("system update was not issued")
			}
			if !m.objects[object.GetKind()+"/"+object.GetMetadata().Name].GetMetadata().IsSystem {
				t.Fatal("system flag was lost")
			}
			if resource.UpstreamKind() != "ClusterConfig" {
				stored := m.objects[object.GetKind()+"/"+object.GetMetadata().Name]
				if err := b.Delete(t.Context(), object.GetKind(), stored.GetMetadata().Uid); err != nil {
					t.Fatal(err)
				}
				if m.deletes != 1 || m.objects[object.GetKind()+"/"+object.GetMetadata().Name] != nil {
					t.Fatal("system resource was not deleted")
				}
			}

		})
	}
}

func TestCreateOnlyObservesAlreadyExistingResource(t *testing.T) {
	for _, system := range []bool{false, true} {
		b, m := systemFixture(t)
		existing := &core.Namespace{Kind: "Namespace", Metadata: &meta.Metadata{Name: "managed", Description: "external", IsSystem: system}, Spec: &core.Namespace_Spec{}}
		m.objects["Namespace/managed"] = existing
		desired := proto.Clone(existing).(*core.Namespace)
		desired.Metadata.Description = "must-not-update"
		out, err := b.Create(t.Context(), desired)
		if err != nil {
			t.Fatal(err)
		}
		if out.GetMetadata().Description != "external" || m.updates != 0 {
			t.Fatal("create-only path updated an existing resource")
		}
	}
}
