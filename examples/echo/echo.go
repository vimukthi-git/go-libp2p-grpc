package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/centrifuge/go-libp2p-grpc"
	"github.com/centrifuge/go-libp2p-grpc/examples/echo/echosvc"
	golog "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	gologging "github.com/whyrusleeping/go-logging"

	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/grpc"
)

// Echoer implements the EchoService.
type Echoer struct {
	PeerID peer.ID
}

// Echo asks a node to respond with a message.
func (e *Echoer) Echo(ctx context.Context, req *echosvc.EchoRequest) (*echosvc.EchoReply, error) {
	return &echosvc.EchoReply{
		Message: req.GetMessage(),
		PeerId:  e.PeerID.Pretty(),
	}, nil
}

func main() {
	// LibP2P code uses golog to log messages. They log with different
	// string IDs (i.e. "swarm"). We can control the verbosity level for
	// all loggers with:
	golog.SetAllLoggers(gologging.INFO) // Change to DEBUG for extra info

	// Parse options from the command line
	listenF := flag.Int("l", 0, "wait for incoming connections")
	target := flag.String("d", "", "target peer to dial")
	echoMsg := flag.String("m", "Hello, world", "message to echo")
	secio := flag.Bool("secio", false, "enable secio")
	seed := flag.Int64("seed", 0, "set random seed for id generation")
	flag.Parse()

	if *listenF == 0 {
		log.Fatal("Please provide a port to bind on with -l")
	}

	// Make a host that listens on the given multiaddress
	ha, err := MakeBasicHost(*listenF, *secio, *seed)
	if err != nil {
		log.Fatal(err)
	}

	// Set the grpc protocol handler on it
	grpcProto := p2pgrpc.NewGRPCProtocol(context.Background(), ha)

	// Register our echoer GRPC service.
	echosvc.RegisterEchoServiceServer(grpcProto.GetGRPCServer(), &Echoer{PeerID: ha.ID()})

	// start accepting the connections
	go func() {
		log.Fatalln(grpcProto.Serve())
	}()

	if *target == "" {
		log.Println("listening for connections")
		select {} // hang forever
	}
	/**** This is where the listener code ends ****/

	// The following code extracts target's the peer ID from the
	// given multiaddress
	ipfsaddr, err := ma.NewMultiaddr(*target)
	if err != nil {
		log.Fatalln(err)
	}

	pid, err := ipfsaddr.ValueForProtocol(ma.P_IPFS)
	if err != nil {
		log.Fatalln(err)
	}

	peerid, err := peer.IDB58Decode(pid)
	if err != nil {
		log.Fatalln(err)
	}

	// Decapsulate the /ipfs/<peerID> part from the target
	// /ip4/<a.b.c.d>/ipfs/<peer> becomes /ip4/<a.b.c.d>
	targetPeerAddr, _ := ma.NewMultiaddr(
		fmt.Sprintf("/ipfs/%s", peer.IDB58Encode(peerid)))
	targetAddr := ipfsaddr.Decapsulate(targetPeerAddr)

	// We have a peer ID and a targetAddr so we add it to the peerstore
	// so LibP2P knows how to contact it
	ha.Peerstore().AddAddr(peerid, targetAddr, pstore.PermanentAddrTTL)

	// make a new stream from host B to host A
	log.Println("dialing via grpc")
	grpcConn, err := grpcProto.Dial(context.Background(), peerid, "", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalln(err)
	}

	// create our service client
	echoClient := echosvc.NewEchoServiceClient(grpcConn)
	echoReply, err := echoClient.Echo(context.Background(), &echosvc.EchoRequest{Message: *echoMsg})
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("read reply:")
	err = (&jsonpb.Marshaler{EmitDefaults: true, Indent: "\t"}).
		Marshal(os.Stdout, echoReply)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println()
}

// doEcho reads a line of data a stream and writes it back
func doEcho(s net.Stream) error {
	buf := bufio.NewReader(s)
	str, err := buf.ReadString('\n')
	if err != nil {
		return err
	}

	log.Printf("read: %s\n", str)
	_, err = s.Write([]byte(str))
	return err
}
