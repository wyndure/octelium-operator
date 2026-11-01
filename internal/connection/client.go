package connection

import (
	"context"

	core "github.com/octelium/octelium/apis/main/corev1"
	rcache "github.com/octelium/octelium/apis/rsc/rcachev1"
	rcore "github.com/octelium/octelium/apis/rsc/rcorev1"
	rlock "github.com/octelium/octelium/apis/rsc/rlockv1"
	rmeta "github.com/octelium/octelium/apis/rsc/rmetav1"
	rpubsub "github.com/octelium/octelium/apis/rsc/rpubsubv1"
	rrate "github.com/octelium/octelium/apis/rsc/rratelimitv1"
	rvector "github.com/octelium/octelium/apis/rsc/rvectorv1"
	"github.com/octelium/octelium/cluster/common/octeliumc"
	"google.golang.org/grpc"
)

// remoteClient is only a transport adapter for the upstream internal client interface.
// All resource operations still use the original clients and resource-management implementation.
type remoteClient struct{ conn *grpc.ClientConn }

func (c remoteClient) CoreC() rcore.ResourceServiceClient {
	return rcore.NewResourceServiceClient(c.conn)
}

func (c remoteClient) CacheC() rcache.MainServiceClient {
	return rcache.NewMainServiceClient(c.conn)
}

func (c remoteClient) RateLimitC() rrate.MainServiceClient {
	return rrate.NewMainServiceClient(c.conn)
}

func (c remoteClient) LockC() rlock.MainServiceClient {
	return rlock.NewMainServiceClient(c.conn)
}

func (c remoteClient) VectorC() rvector.MainServiceClient {
	return rvector.NewMainServiceClient(c.conn)
}

func (c remoteClient) PubSubC() rpubsub.MainServiceClient {
	return rpubsub.NewMainServiceClient(c.conn)
}

func (c remoteClient) CoreV1Utils() octeliumc.CoreV1Utils {
	return coreUtils{c.CoreC()}
}

type coreUtils struct {
	core rcore.ResourceServiceClient
}

func (c coreUtils) GetClusterConfig(ctx context.Context) (*core.ClusterConfig, error) {
	return c.core.GetClusterConfig(ctx, &rmeta.GetOptions{Name: "default"})
}
