package main

import (
	"context"
	"errors"
	"github.com/beatyman/buckets/api/bucketsd"
	bpb "github.com/beatyman/buckets/api/bucketsd/pb"
	"github.com/beatyman/buckets/api/common"
	tdb "github.com/beatyman/buckets/threaddb"
	"github.com/beatyman/buckets/util"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	logging "github.com/ipfs/go-log/v2"
	caopts "github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/libp2p/go-libp2p-core/connmgr"
	ma "github.com/multiformats/go-multiaddr"
	dbapi "github.com/textileio/go-threads/api"
	threads "github.com/textileio/go-threads/api/client"
	dbpb "github.com/textileio/go-threads/api/pb"
	"github.com/textileio/go-threads/broadcast"
	tc "github.com/textileio/go-threads/common"
	kt "github.com/textileio/go-threads/db/keytransform"
	netapi "github.com/textileio/go-threads/net/api"
	netclient "github.com/textileio/go-threads/net/api/client"
	netpb "github.com/textileio/go-threads/net/api/pb"
	nutil "github.com/textileio/go-threads/net/util"
	tutil "github.com/textileio/go-threads/util"
	userPb "github.com/textileio/powergate/v2/api/gen/powergate/user/v1"
	"google.golang.org/grpc"
	"net"
	"time"
)

var log = logging.Logger("db")

type Textile struct {
	tn                 tc.NetBoostrapper
	ts                 kt.TxnDatastoreExtended
	th                 *threads.Client
	thn                *netclient.Client
	bucks              *tdb.Buckets
	mail               *tdb.Mail
	server             *grpc.Server
	internalHubSession string
	emailSessionBus    *broadcast.Broadcaster
	conf               Config
}

type Config struct {
	Hub       bool
	Debug     bool
	buckLocks *nutil.SemaphorePool
	// Addresses
	AddrAPI         ma.Multiaddr
	AddrThreadsHost ma.Multiaddr
	AddrGatewayHost ma.Multiaddr
	AddrGatewayURL  string
	AddrIPFSAPI     ma.Multiaddr
	// Buckets
	MaxBucketArchiveRepFactor int
	MaxBucketArchiveSize      int64
	MinBucketArchiveSize      int64
	// Threads
	MaxNumberThreadsPerOwner int
	ThreadsConnManager       connmgr.ConnManager
}

func NewTextile(ctx context.Context, conf Config) (*Textile, error) {
	logging.SetAllLoggers(logging.LevelError)
	t := &Textile{
		conf:               conf,
		internalHubSession: util.MakeToken(32),
	}
	// Configure clients
	ipfsapi, err := httpapi.NewApi(conf.AddrIPFSAPI)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	// Don't allow using textile as a gateway to non-bucket files
	ic, err := ipfsapi.WithOptions(caopts.Api.FetchBlocks(false))
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Configure threads
	netOptions := []tc.NetOption{
		tc.WithNetHostAddr(conf.AddrThreadsHost),
		tc.WithNoNetPulling(true),
		tc.WithNoExchangeEdgesMigration(true),
		tc.WithNetDebug(conf.Debug),
	}
	netOptions = append(netOptions, tc.WithNetBadgerPersistence("./test"))
	if conf.ThreadsConnManager != nil {
		netOptions = append(netOptions, tc.WithConnectionManager(conf.ThreadsConnManager))
	}
	t.tn, err = tc.DefaultNetwork(netOptions...)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Configure gRPC server
	target, err := tutil.TCPAddrFromMultiAddr(conf.AddrAPI)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	// Start threads clients
	t.th, err = threads.NewClient(target, grpc.WithInsecure(), grpc.WithPerRPCCredentials(common.Credentials{}))
	if err != nil {
		log.Error(err)
		return nil, err
	}
	t.thn, err = netclient.NewClient(target, grpc.WithInsecure(), grpc.WithPerRPCCredentials(common.Credentials{}))
	if err != nil {
		log.Error(err)
		return nil, err
	}
	t.bucks, err = tdb.NewBuckets(t.th)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	t.mail, err = tdb.NewMail(t.th)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Configure gRPC services
	t.ts, err = tutil.NewBadgerDatastore("./test1", "eventstore", false)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	ts, err := dbapi.NewService(t.ts, t.tn, dbapi.Config{
		Debug: conf.Debug,
	})
	if err != nil {
		log.Error(err)
		return nil, err
	}
	ns, err := netapi.NewService(t.tn, netapi.Config{
		Debug: conf.Debug,
	})
	if err != nil {
		log.Error(err)
		return nil, err
	}

	bs := &bucketsd.Service{
		Buckets:                   t.bucks,
		GatewayURL:                conf.AddrGatewayURL,
		IPFSClient:                ic,
		MaxBucketArchiveRepFactor: conf.MaxBucketArchiveRepFactor,
		MaxBucketArchiveSize:      conf.MaxBucketArchiveSize,
		MinBucketArchiveSize:      conf.MinBucketArchiveSize,
	}
	// Start serving
	var grpcopts []grpc.ServerOption
	t.server = grpc.NewServer(grpcopts...)
	listener, err := net.Listen("tcp", target)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	go func() {
		//thread api
		dbpb.RegisterAPIServer(t.server, ts)
		netpb.RegisterAPIServer(t.server, ns)
		//用户api
		userPb.RegisterUserServiceServer(t.server, &userPb.UnimplementedUserServiceServer{})
		//bucket api
		bpb.RegisterAPIServiceServer(t.server, bs)

		if err := t.server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Fatalf("serve error: %v", err)
		}
		if err := ts.Close(); err != nil {
			log.Fatalf("error closing thread service: %v", err)
		}
	}()
	log.Info("---------------------------------------")
	return t, nil
}

func (t *Textile) Bootstrap() {
	t.tn.Bootstrap(tutil.DefaultBoostrapPeers())
}

func (t *Textile) Close() error {
	if t.emailSessionBus != nil {
		t.emailSessionBus.Discard()
	}
	stopped := make(chan struct{})
	go func() {
		t.server.GracefulStop()
		close(stopped)
	}()
	timer := time.NewTimer(10 * time.Second)
	select {
	case <-timer.C:
		t.server.Stop()
	case <-stopped:
		timer.Stop()
	}
	if err := t.th.Close(); err != nil {
		return err
	}
	if err := t.tn.Close(); err != nil {
		return err
	}
	if err := t.ts.Close(); err != nil {
		return err
	}
	return nil
}

// AddrFromStr returns a multiaddress from the string.
func AddrFromStr(str string) ma.Multiaddr {
	addr, err := ma.NewMultiaddr(str)
	if err != nil {
		log.Fatal(err)
	}
	return addr
}

func main() {
	c := logging.Config{
		Format: logging.ColorizedOutput,
		Stderr: true,
		Level:  logging.LevelError,
	}
	logging.SetupLogging(c)
	ccCore, err := NewTextile(context.Background(), Config{
		AddrIPFSAPI:     ma.StringCast("/ip4/127.0.0.1/tcp/5001"),
		AddrThreadsHost: ma.StringCast("/ip4/127.0.0.1/tcp/4000"),
		AddrAPI:         ma.StringCast("/ip4/0.0.0.0/tcp/7006"),
	})
	if err != nil {
		log.Fatal(err)
	}
	ccCore.Bootstrap()
	select {}
}
