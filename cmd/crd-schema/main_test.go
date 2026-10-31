package main

import (
	"reflect"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestMessageOmitsEmptyProperties(t *testing.T) {
	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name:   proto.String("empty.proto"),
		Syntax: proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: proto.String("Empty")},
			{
				Name: proto.String("Container"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("empty"),
						Number:   proto.Int32(1),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".Empty"),
					},
					{
						Name:     proto.String("entries"),
						Number:   proto.Int32(2),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".Empty"),
					},
				},
			},
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	empty := schema{schemaType: schemaObject}
	if got := message(file.Messages().ByName("Empty"), 0); !reflect.DeepEqual(got, empty) {
		t.Fatalf("empty message schema = %#v, want %#v", got, empty)
	}

	want := schema{
		schemaType: schemaObject,
		"properties": schema{
			"empty": empty,
			"entries": schema{
				schemaType: "array",
				"items":    empty,
			},
		},
	}
	if got := message(file.Messages().ByName("Container"), 0); !reflect.DeepEqual(got, want) {
		t.Fatalf("container schema = %#v, want %#v", got, want)
	}
}
