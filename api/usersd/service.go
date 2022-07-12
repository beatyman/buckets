package usersd

import (
	"context"
	"github.com/beatyman/buckets/api/usersd/pb"
	tdb "github.com/beatyman/buckets/threaddb"
)

type Service struct {
	Mail            *tdb.Mail
}

func (s Service) GetThread(ctx context.Context, request *pb.GetThreadRequest) (*pb.GetThreadResponse, error) {
	panic("implement me")
}

func (s Service) ListThreads(ctx context.Context, request *pb.ListThreadsRequest) (*pb.ListThreadsResponse, error) {
	panic("implement me")
}

func (s Service) SetupMailbox(ctx context.Context, request *pb.SetupMailboxRequest) (*pb.SetupMailboxResponse, error) {
	panic("implement me")
}

func (s Service) SendMessage(ctx context.Context, request *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	panic("implement me")
}

func (s Service) ListInboxMessages(ctx context.Context, request *pb.ListInboxMessagesRequest) (*pb.ListInboxMessagesResponse, error) {
	panic("implement me")
}

func (s Service) ListSentboxMessages(ctx context.Context, request *pb.ListSentboxMessagesRequest) (*pb.ListSentboxMessagesResponse, error) {
	panic("implement me")
}

func (s Service) ReadInboxMessage(ctx context.Context, request *pb.ReadInboxMessageRequest) (*pb.ReadInboxMessageResponse, error) {
	panic("implement me")
}

func (s Service) DeleteInboxMessage(ctx context.Context, request *pb.DeleteInboxMessageRequest) (*pb.DeleteInboxMessageResponse, error) {
	panic("implement me")
}

func (s Service) DeleteSentboxMessage(ctx context.Context, request *pb.DeleteSentboxMessageRequest) (*pb.DeleteSentboxMessageResponse, error) {
	panic("implement me")
}

func (s Service) GetUsage(ctx context.Context, request *pb.GetUsageRequest) (*pb.GetUsageResponse, error) {
	panic("implement me")
}

func (s Service) ArchivesLs(ctx context.Context, request *pb.ArchivesLsRequest) (*pb.ArchivesLsResponse, error) {
	panic("implement me")
}

func (s Service) ArchivesImport(ctx context.Context, request *pb.ArchivesImportRequest) (*pb.ArchivesImportResponse, error) {
	panic("implement me")
}

func (s Service) ArchiveRetrievalLs(ctx context.Context, request *pb.ArchiveRetrievalLsRequest) (*pb.ArchiveRetrievalLsResponse, error) {
	panic("implement me")
}

func (s Service) ArchiveRetrievalLogs(request *pb.ArchiveRetrievalLogsRequest, server pb.APIService_ArchiveRetrievalLogsServer) error {
	panic("implement me")
}

