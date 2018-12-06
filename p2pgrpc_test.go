package p2pgrpc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"

	"github.com/libp2p/go-libp2p-protocol"
	"github.com/stretchr/testify/assert"
)

func TestFormProtocol(t *testing.T) {
	streamSuffix := ""
	assert.Equal(t, Protocol, formProtocol(streamSuffix))

	streamSuffix = "0x123456789"
	expectedProtocol := protocol.ID(fmt.Sprintf("%s/%s", Protocol, streamSuffix))
	assert.Equal(t, expectedProtocol, formProtocol(streamSuffix))
}

func TestNewGRPCProtocolDefault(t *testing.T) {
	h, err := MakeBasicHost(38001, true, 0)
	assert.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	grpcProto := NewGRPCProtocol(ctx, h, "")
	assert.Equal(t, Protocol, grpcProto.streamProtocol)
	go func() {
		err = grpcProto.Serve()
	}()
	time.Sleep(1 * time.Second)
	assert.Nil(t, err)

	conn, err := grpcProto.Dial(ctx, h.ID(), grpc.WithInsecure())
	assert.Nil(t, err)
	assert.Equal(t, connectivity.Idle, conn.GetState())
	cancel()
}

func TestNewGRPCProtocolCustom(t *testing.T) {
	h, err := MakeBasicHost(38001, true, 0)
	assert.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	customSuffix := "0x123456789"
	grpcProto := NewGRPCProtocol(ctx, h, customSuffix)
	expectedProtocol := protocol.ID(fmt.Sprintf("%s/%s", Protocol, customSuffix))
	assert.Equal(t, expectedProtocol, grpcProto.streamProtocol)
	go func() {
		err = grpcProto.Serve()
	}()
	time.Sleep(1 * time.Second)
	assert.Nil(t, err)

	conn, err := grpcProto.Dial(ctx, h.ID(), grpc.WithInsecure())
	assert.Nil(t, err)
	assert.Equal(t, connectivity.Idle, conn.GetState())
	cancel()
}
