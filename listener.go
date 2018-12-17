package p2pgrpc

import (
	"context"
	"io"
	"net"

	manet "github.com/multiformats/go-multiaddr-net"
	inet "github.com/libp2p/go-libp2p-net"
	"reflect"
)

// grpcListener implements the net.Listener interface.
type grpcListener struct {
	*GRPCProtocol
	listenerCtx       context.Context
	listenerCtxCancel context.CancelFunc
}

// newGrpcListener builds a new GRPC listener.
func newGrpcListener(proto *GRPCProtocol) net.Listener {
	l := &grpcListener{
		GRPCProtocol: proto,
	}
	l.listenerCtx, l.listenerCtxCancel = context.WithCancel(proto.ctx)
	return l
}

// Accept waits for and returns the next connection to the listener.
func (l *grpcListener) Accept() (net.Conn, error) {
	cases := make([]reflect.SelectCase, len(l.streamChs) + 1)
	chans := make([]chan inet.Stream, len(l.streamChs))
	i := 0
	for _, ch := range l.streamChs {
		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
		chans[i] = ch
		i++
	}
	cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(l.listenerCtx.Done())}
	chosen, value, _ := reflect.Select(cases)
	// context closed
	if chosen == len(l.streamChs) {
		return nil, io.EOF
	}
	// new protocol stream
	s, ok := value.Interface().(inet.Stream)
	if ok {
		return &streamConn{Stream: s}, nil
	} else {
		// problem
		return nil, io.EOF
	}
}

// Addr returns the listener's network address.
func (l *grpcListener) Addr() net.Addr {
	listenAddrs := l.host.Network().ListenAddresses()
	if len(listenAddrs) > 0 {
		for _, addr := range listenAddrs {
			if na, err := manet.ToNetAddr(addr); err == nil {
				return na
			}
		}
	}
	return fakeLocalAddr()
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (l *grpcListener) Close() error {
	l.listenerCtxCancel()
	return nil
}
