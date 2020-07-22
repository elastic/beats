package client

import (
	"context"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control"
	"net"

	"google.golang.org/grpc"
)

func dialContext(ctx context.Context) (*grpc.ClientConn, error) {
	return grpc.DialContext(ctx, control.Address(), grpc.WithInsecure(), grpc.WithContextDialer(dialer))
}

func dialer(ctx context.Context, addr string) (net.Conn, error) {
	var d net.Dialer
	return d.DialContext(ctx, "unix", addr)
}
