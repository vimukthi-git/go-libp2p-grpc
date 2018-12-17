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
	grpcServer     *grpc.Server
	streamChs      map[protocol.ID]chan inet.Stream
}

// NewGRPCProtocol attaches the GRPC protocol to a host.
func NewGRPCProtocol(ctx context.Context, host host.Host) *GRPCProtocol {
	grpcServer := grpc.NewServer()

	grpcProtocol := &GRPCProtocol{
		ctx:            ctx,
		host:           host,
		grpcServer:     grpcServer,
		streamChs:      make(map[protocol.ID]chan inet.Stream),
	}

	return grpcProtocol
}

// RegisterProtocol registers a protocol to be handled by the grpc server instance
func (p *GRPCProtocol) RegisterProtocolSuffix(suf string) {
	protoID := formProtocol(suf)
	p.streamChs[protoID] = make(chan inet.Stream)
	p.host.SetStreamHandler(protoID, p.HandleStream)
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
	case p.streamChs[stream.Protocol()] <- stream:
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
