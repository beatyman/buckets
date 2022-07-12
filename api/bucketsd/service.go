package bucketsd

import (
	"context"
	"github.com/beatyman/buckets/api/bucketsd/pb"
	tdb "github.com/beatyman/buckets/threaddb"
	iface "github.com/ipfs/interface-go-ipfs-core"
)

// Service is a gRPC service for buckets.
type Service struct {
	Buckets                   *tdb.Buckets
	GatewayURL                string
	GatewayBucketsHost        string
	IPFSClient                iface.CoreAPI
	MaxBucketArchiveSize      int64
	MinBucketArchiveSize      int64
	MaxBucketArchiveRepFactor int
}

func (s Service) List(ctx context.Context, request *pb.ListRequest) (*pb.ListResponse, error) {
	panic("implement me")
}

func (s Service) Create(ctx context.Context, request *pb.CreateRequest) (*pb.CreateResponse, error) {
	panic("implement me")
}

func (s Service) Root(ctx context.Context, request *pb.RootRequest) (*pb.RootResponse, error) {
	panic("implement me")
}

func (s Service) Links(ctx context.Context, request *pb.LinksRequest) (*pb.LinksResponse, error) {
	panic("implement me")
}

func (s Service) ListPath(ctx context.Context, request *pb.ListPathRequest) (*pb.ListPathResponse, error) {
	panic("implement me")
}

func (s Service) ListIpfsPath(ctx context.Context, request *pb.ListIpfsPathRequest) (*pb.ListIpfsPathResponse, error) {
	panic("implement me")
}

func (s Service) PushPath(server pb.APIService_PushPathServer) error {
	panic("implement me")
}

func (s Service) PushPaths(server pb.APIService_PushPathsServer) error {
	panic("implement me")
}

func (s Service) PullPath(request *pb.PullPathRequest, server pb.APIService_PullPathServer) error {
	panic("implement me")
}

func (s Service) PullIpfsPath(request *pb.PullIpfsPathRequest, server pb.APIService_PullIpfsPathServer) error {
	panic("implement me")
}

func (s Service) SetPath(ctx context.Context, request *pb.SetPathRequest) (*pb.SetPathResponse, error) {
	panic("implement me")
}

func (s Service) MovePath(ctx context.Context, request *pb.MovePathRequest) (*pb.MovePathResponse, error) {
	panic("implement me")
}

func (s Service) Remove(ctx context.Context, request *pb.RemoveRequest) (*pb.RemoveResponse, error) {
	panic("implement me")
}

func (s Service) RemovePath(ctx context.Context, request *pb.RemovePathRequest) (*pb.RemovePathResponse, error) {
	panic("implement me")
}

func (s Service) PushPathAccessRoles(ctx context.Context, request *pb.PushPathAccessRolesRequest) (*pb.PushPathAccessRolesResponse, error) {
	panic("implement me")
}

func (s Service) PullPathAccessRoles(ctx context.Context, request *pb.PullPathAccessRolesRequest) (*pb.PullPathAccessRolesResponse, error) {
	panic("implement me")
}

func (s Service) DefaultArchiveConfig(ctx context.Context, request *pb.DefaultArchiveConfigRequest) (*pb.DefaultArchiveConfigResponse, error) {
	panic("implement me")
}

func (s Service) SetDefaultArchiveConfig(ctx context.Context, request *pb.SetDefaultArchiveConfigRequest) (*pb.SetDefaultArchiveConfigResponse, error) {
	panic("implement me")
}

func (s Service) Archive(ctx context.Context, request *pb.ArchiveRequest) (*pb.ArchiveResponse, error) {
	panic("implement me")
}

func (s Service) Archives(ctx context.Context, request *pb.ArchivesRequest) (*pb.ArchivesResponse, error) {
	panic("implement me")
}

func (s Service) ArchiveWatch(request *pb.ArchiveWatchRequest, server pb.APIService_ArchiveWatchServer) error {
	panic("implement me")
}
