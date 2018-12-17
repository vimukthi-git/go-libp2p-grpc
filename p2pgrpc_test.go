package p2pgrpc

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-peer"

	mrand "math/rand"

	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"

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
	h, err := makeBasicHost(38001, true, 0)
	assert.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	grpcProto := NewGRPCProtocol(ctx, h)
	grpcProto.RegisterProtocolSuffix("")
	_, ok := grpcProto.streamChs[Protocol]
	assert.True(t, ok)
	go func() {
		err = grpcProto.Serve()
	}()
	time.Sleep(1 * time.Second)
	assert.Nil(t, err)

	conn, err := grpcProto.Dial(ctx, h.ID(), "", grpc.WithInsecure())
	assert.Nil(t, err)
	assert.Equal(t, connectivity.Idle, conn.GetState())
	cancel()
}

func TestNewGRPCProtocolCustom(t *testing.T) {
	h, err := makeBasicHost(38001, true, 0)
	assert.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	customSuffix := "0x123456789"
	grpcProto := NewGRPCProtocol(ctx, h)
	grpcProto.RegisterProtocolSuffix(customSuffix)
	expectedProtocol := protocol.ID(fmt.Sprintf("%s/%s", Protocol, customSuffix))
	_, ok := grpcProto.streamChs[expectedProtocol]
	assert.True(t, ok)
	go func() {
		err = grpcProto.Serve()
	}()
	time.Sleep(1 * time.Second)
	assert.Nil(t, err)

	conn, err := grpcProto.Dial(ctx, h.ID(), customSuffix, grpc.WithInsecure())
	assert.Nil(t, err)
	assert.Equal(t, connectivity.Idle, conn.GetState())
	cancel()
}

func TestNewGRPCProtocolManyCustom(t *testing.T) {
	h, err := makeBasicHost(38001, true, 0)
	assert.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	customSuffix1 := "0x123456781"
	customSuffix2 := "0x123456782"
	customSuffix3 := "0x123456783"
	grpcProto := NewGRPCProtocol(ctx, h)
	grpcProto.RegisterProtocolSuffix(customSuffix1)
	grpcProto.RegisterProtocolSuffix(customSuffix2)
	grpcProto.RegisterProtocolSuffix(customSuffix3)
	expectedProtocol := protocol.ID(fmt.Sprintf("%s/%s", Protocol, customSuffix1))
	_, ok := grpcProto.streamChs[expectedProtocol]
	assert.True(t, ok)
	go func() {
		err = grpcProto.Serve()
	}()
	time.Sleep(1 * time.Second)
	assert.Nil(t, err)

	conn, err := grpcProto.Dial(ctx, h.ID(), customSuffix1, grpc.WithInsecure())
	assert.Nil(t, err)
	conn2, err := grpcProto.Dial(ctx, h.ID(), customSuffix2, grpc.WithInsecure())
	assert.Nil(t, err)
	assert.Equal(t, connectivity.Idle, conn.GetState())
	assert.Equal(t, connectivity.Idle, conn2.GetState())
	cancel()
}

// MakeBasicHost creates a LibP2P host with a random peer ID listening on the
// given multiaddress. It will use secio if secio is true.
func makeBasicHost(listenPort int, secio bool, randseed int64) (host.Host, error) {
	// If the seed is zero, use real cryptographic randomness. Otherwise, use a
	// deterministic randomness source to make generated keys stay the same
	// across multiple runs
	var r io.Reader
	if randseed == 0 {
		r = rand.Reader
	} else {
		r = mrand.New(mrand.NewSource(randseed))
	}

	// Generate a key pair for this host. We will use it at least
	// to obtain a valid host ID.
	priv, pub, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, err
	}

	// Obtain Peer ID from public key
	pid, err := peer.IDFromPublicKey(pub)
	if err != nil {
		return nil, err
	}

	// Create a peerstore
	ps := pstore.NewPeerstore()

	// If using secio, we add the keys to the peerstore
	// for this peer ID.
	if secio {
		ps.AddPrivKey(pid, priv)
		ps.AddPubKey(pid, pub)
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)),
		libp2p.Identity(priv),
	}

	basicHost, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		return nil, err
	}

	// Build host multiaddress
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", basicHost.ID().Pretty()))

	// Now we can build a full multiaddress to reach this host
	// by encapsulating both addresses:
	fullAddr := basicHost.Addrs()[0].Encapsulate(hostAddr)
	log.Printf("I am %s\n", fullAddr)
	if secio {
		log.Printf("Now run \"./echo -l %d -d %s -secio\" on a different terminal\n", listenPort+1, fullAddr)
	} else {
		log.Printf("Now run \"./echo -l %d -d %s\" on a different terminal\n", listenPort+1, fullAddr)
	}

	return basicHost, nil
}
