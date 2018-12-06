package p2pgrpc

import (
	"context"
	"fmt"

	// "net"

	host "github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"
	protocol "github.com/libp2p/go-libp2p-protocol"
	"google.golang.org/grpc"
)

// Protocol is the GRPC-over-libp2p protocol.
const Protocol protocol.ID = "/grpc/0.0.1"

// GRPCProtocol is the GRPC-transported protocol handler.
type GRPCProtocol struct {
	ctx            context.Context
	host           host.Host
	streamProtocol protocol.ID
	grpcServer     *grpc.Server
	streamCh       chan inet.Stream
}

// NewGRPCProtocol attaches the GRPC protocol to a host.
func NewGRPCProtocol(ctx context.Context, host host.Host, streamSuffix string) *GRPCProtocol {
	grpcServer := grpc.NewServer()

	grpcProtocol := &GRPCProtocol{
		ctx:            ctx,
		host:           host,
		streamProtocol: formProtocol(streamSuffix),
		grpcServer:     grpcServer,
		streamCh:       make(chan inet.Stream),
	}

	host.SetStreamHandler(grpcProtocol.streamProtocol, grpcProtocol.HandleStream)

	return grpcProtocol
}

// GetGRPCServer returns the grpc server.
func (p *GRPCProtocol) GetGRPCServer() *grpc.Server {
	return p.grpcServer
}

// Serve initiates the grpc server to accept new connections
// Note: should be called after registering the server and handlers
func (p *GRPCProtocol) Serve() error {
	return p.grpcServer.Serve(newGrpcListener(p))
}

// HandleStream handles an incoming stream.
func (p *GRPCProtocol) HandleStream(stream inet.Stream) {
	select {
	case <-p.ctx.Done():
		return
	case p.streamCh <- stream:
	}
}

// Appends suffix to grpc Protocol
func formProtocol(suffix string) protocol.ID {
	useProtocol := Protocol
	if suffix != "" {
		useProtocol = protocol.ID(fmt.Sprintf("%s/%s", useProtocol, suffix))
	}
	return useProtocol
}
