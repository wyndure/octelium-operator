package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"

	upstreammeta "github.com/octelium/octelium/apis/main/metav1"
	api "github.com/wyndure/octelium-operator/api/core/v1"
	"google.golang.org/protobuf/encoding/protojson"
	kcore "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// publishMetadata exports the server's complete metadata, including read-only fields.
func (r *ResourceReconciler) publishMetadata(ctx context.Context, obj api.Resource, metadata *upstreammeta.Metadata) error {
	name := obj.MetadataConfigMapName()
	if name == "" {
		return nil
	}

	if errors := validation.IsDNS1123Subdomain(name); len(errors) != 0 {
		return fmt.Errorf("invalid metadataConfigMap name: %v", errors)
	}

	if metadata == nil {
		return fmt.Errorf("upstream returned no resource metadata")
	}

	data, err := metadataData(metadata)
	if err != nil {
		return err
	}

	cm := &kcore.ConfigMap{}

	key := client.ObjectKey{Namespace: obj.GetNamespace(), Name: name}

	err = r.Get(ctx, key, cm)
	if apierrors.IsNotFound(err) {
		cm = &kcore.ConfigMap{Namespace: key.Namespace, Name: key.Name, Data: data}
		if err = controllerutil.SetControllerReference(obj, cm, r.Scheme()); err != nil {
			return err
		}

		return r.Create(ctx, cm)
	}

	if err != nil {
		return err
	}

	if !metav1.IsControlledBy(cm, obj) {
		return fmt.Errorf("metadata ConfigMap %s is not owned by this resource", key)
	}

	_, err = controllerutil.CreateOrPatch(ctx, r.Client, cm, func() error {
		if !metav1.IsControlledBy(cm, obj) {
			return fmt.Errorf("metadata ConfigMap %s is not owned by this resource", key)
		}

		if cm.Data == nil {
			cm.Data = map[string]string{}
		}

		maps.Copy(cm.Data, data)

		return nil
	})

	return err
}

func metadataData(metadata *upstreammeta.Metadata) (map[string]string, error) {
	encoded, err := (protojson.MarshalOptions{EmitUnpopulated: true}).Marshal(metadata)
	if err != nil {
		return nil, err
	}

	var fields map[string]json.RawMessage

	if err = json.Unmarshal(encoded, &fields); err != nil {
		return nil, err
	}

	data := make(map[string]string, len(fields))

	for key, raw := range fields {
		var value string

		if len(raw) > 0 && raw[0] == '"' {
			if err = json.Unmarshal(raw, &value); err != nil {
				return nil, err
			}
		} else {
			value = string(raw)
		}

		data[key] = value
	}

	return data, nil
}
