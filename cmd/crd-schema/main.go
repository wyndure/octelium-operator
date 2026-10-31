// crd-schema derives CRD validation from upstream protobuf descriptors.
// It is run by make manifests; no Octelium specification is maintained here.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	upstreammeta "github.com/octelium/octelium/apis/main/metav1"
	api "github.com/wyndure/octelium-operator/api/core/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
	"sigs.k8s.io/yaml"
)

const (
	schemaType      = "type"
	preserveUnknown = "x-kubernetes-preserve-unknown-fields"
	schemaString    = "string"
	schemaObject    = "object"
	schemaInteger   = "integer"
)

type schema = map[string]any

func message(md protoreflect.MessageDescriptor, depth int) schema {
	name := string(md.FullName())

	switch name {
	case "google.protobuf.Timestamp":
		return schema{schemaType: schemaString, "format": "date-time"}
	case "google.protobuf.Duration":
		return schema{schemaType: schemaString}
	case "google.protobuf.Struct", "google.protobuf.Any":
		return schema{schemaType: schemaObject, preserveUnknown: true}
	case "google.protobuf.Value":
		return schema{preserveUnknown: true}
	case "google.protobuf.ListValue":
		return schema{schemaType: "array", "items": schema{preserveUnknown: true}}
	}

	if depth >= 5 {
		return schema{schemaType: schemaObject, preserveUnknown: true}
	}

	props := schema{}
	for i := 0; i < md.Fields().Len(); i++ {
		f := md.Fields().Get(i)

		props[f.JSONName()] = field(f, depth+1)
	}

	if len(props) == 0 {
		return schema{schemaType: schemaObject}
	}

	return schema{schemaType: schemaObject, "properties": props}
}

func field(f protoreflect.FieldDescriptor, depth int) schema {
	if f.IsMap() {
		return schema{schemaType: schemaObject, "additionalProperties": value(f.MapValue(), depth)}
	}

	if f.IsList() {
		return schema{schemaType: "array", "items": value(f, depth)}
	}

	return value(f, depth)
}

func value(f protoreflect.FieldDescriptor, depth int) schema {
	switch f.Kind() {
	case protoreflect.MessageKind:
		return message(f.Message(), depth)
	case protoreflect.BoolKind:
		return schema{schemaType: "boolean"}
	case protoreflect.StringKind, protoreflect.BytesKind, protoreflect.EnumKind:
		return schema{schemaType: schemaString}
	case protoreflect.Int64Kind, protoreflect.Uint64Kind, protoreflect.Sint64Kind,
		protoreflect.Fixed64Kind, protoreflect.Sfixed64Kind:
		return schema{schemaType: schemaInteger}
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return schema{schemaType: "number"}
	default:
		return schema{schemaType: schemaInteger}
	}
}

func resourceKind(upstreamKind string) string {
	return "Octelium" + upstreamKind
}

func resourcePath(upstreamKind string) string {
	kind := resourceKind(upstreamKind)

	plural := strings.ToLower(kind) + "s"
	if strings.HasSuffix(kind, "Policy") {
		plural = strings.TrimSuffix(strings.ToLower(kind), "y") + "ies"
	}

	return filepath.Join("config/crd/bases", "core.octelium.wyndure.com_"+plural+".yaml")
}

func metadataSchema() schema {
	metadata := message((&upstreammeta.Metadata{}).ProtoReflect().Descriptor(), 0)

	fields := metadata["properties"].(map[string]any)
	for _, native := range []string{"name", "labels", "annotations"} {
		delete(fields, native)
	}

	return metadata
}

func readSchema(path string) schema {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	var doc schema
	if err := yaml.Unmarshal(b, &doc); err != nil {
		panic(err)
	}

	return doc
}

func writeSchema(path string, doc schema) {
	b, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}

	b, err = yaml.JSONToYAML(b)
	if err != nil {
		panic(err)
	}

	const header = "# Generated from Kubebuilder markers and upstream protobuf descriptors. DO NOT EDIT.\n"
	if err := os.WriteFile(path, append([]byte(header), b...), 0644); err != nil {
		panic(err)
	}
}

func updateSchema(doc schema, spec protoreflect.MessageDescriptor) {
	metadata := metadataSchema()

	versions := doc["spec"].(map[string]any)["versions"].([]any)

	for _, version := range versions {
		v := version.(map[string]any)

		openAPI := v["schema"].(map[string]any)["openAPIV3Schema"].(map[string]any)

		props := openAPI["properties"].(map[string]any)
		props["spec"] = message(spec, 0)

		octelium := props["octelium"].(map[string]any)["properties"].(map[string]any)
		octelium["metadata"] = metadata
	}
}

func processResource(r api.Resource) string {
	kind := resourceKind(r.UpstreamKind())
	path := resourcePath(r.UpstreamKind())

	resourceSchema := readSchema(path)

	updateSchema(resourceSchema, r.UpstreamSpec().ProtoReflect().Descriptor())

	writeSchema(path, resourceSchema)

	return kind
}

func main() {
	for _, r := range api.Resources() {
		kind := processResource(r)

		fmt.Println("Generated upstream schema for", kind)
	}
}
