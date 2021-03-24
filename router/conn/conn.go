package conn

import (
	"context"
	"fmt"

	"github.com/luffy050596/vulcan-pkg-app/metrics"
	"github.com/luffy050596/vulcan-pkg-app/router/balancer"
	"github.com/luffy050596/vulcan-pkg-app/router/routetable"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type Conn struct {
	grpc.ClientConnInterface
}

func NewConn(serviceName string, balancerType balancer.BalancerType, logger log.Logger, rt routetable.RouteTable, r registry.Discovery) (*Conn, error) {
	if balancerType == balancer.BalancerTypeMaster && !balancer.MasterBalancerRegistered.Load() {
		balancer.RegisterMasterBalancer(rt)
	}
	if balancerType == balancer.BalancerTypeReader && !balancer.ReaderBalancerRegistered.Load() {
		balancer.RegisterReaderBalancer(rt)
	}

	conn, err := kgrpc.DialInsecure(
		context.Background(),
		kgrpc.WithEndpoint(fmt.Sprintf("discovery:///%s", serviceName)),
		kgrpc.WithDiscovery(r),
		kgrpc.WithNodeFilter(balancer.NewFilter()),
		kgrpc.WithOptions(
			grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"loadBalancingConfig": [{"%s":{}}]}`, string(balancerType))),
		),
		kgrpc.WithMiddleware(
			recovery.Recovery(),
			metadata.Client(),
			tracing.Client(),
			metrics.Middleware(),
			logging.Client(logger),
		),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "create grpc connection failed. app=%s", serviceName)
	}
	return &Conn{ClientConnInterface: conn}, nil
}
