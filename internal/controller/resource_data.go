package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	upstreamcore "github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	api "github.com/wyndure/octelium-operator/api/core/v1"
	"google.golang.org/protobuf/encoding/protojson"
	kcore "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ResourceReconciler) prepareData(ctx context.Context, obj api.Resource, current, desired umetav1.ResourceObjectI) (string, error) {
	var e error
	dataDigest := ""
	if secret, ok := desired.(*upstreamcore.Secret); ok {
		if ref := obj.Extension().DataFromSecret; ref != nil {
			if obj.Extension().Data != nil {
				return "", fmt.Errorf("choose data or dataFromSecret")
			}

			sourceNS := ref.Namespace
			if sourceNS == "" {
				sourceNS = obj.GetNamespace()
			}

			if sourceNS != obj.GetNamespace() {
				return "", fmt.Errorf("dataFromSecret must be in the managed resource namespace")
			}

			var source kcore.Secret
			if e = r.Get(ctx, client.ObjectKey{Namespace: sourceNS, Name: ref.Name}, &source); e != nil {
				return "", e
			}

			value, found := source.Data[ref.Key]
			if !found {
				return "", fmt.Errorf("secret key %s not found", ref.Key)
			}

			secret.Data = &upstreamcore.Secret_Data{Type: &upstreamcore.Secret_Data_ValueBytes{ValueBytes: value}}
		}

		if secret.Data != nil {
			encoded, marshalErr := protojson.Marshal(secret.Data)
			if marshalErr != nil {
				return "", marshalErr
			}

			digest := sha256.Sum256(encoded)

			dataDigest = hex.EncodeToString(digest[:])
			if current != nil && obj.SyncStatus().DataDigest == dataDigest {
				secret.Data = nil
			}
		}
	}

	return dataDigest, nil
}
