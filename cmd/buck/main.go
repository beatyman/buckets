package main

import (
	"context"
	"fmt"
	"github.com/beatyman/buckets/api/bucketsd/client"
	"github.com/beatyman/buckets/api/common"
	logging "github.com/ipfs/go-log/v2"
	"github.com/textileio/go-threads/core/thread"
	"google.golang.org/grpc"
	"time"
)

var log = logging.Logger("db")
func main() {
	cli,err:=client.NewClient("127.0.0.1:7006",grpc.WithInsecure())
	if err!=nil{
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	threadID ,_:= thread.Decode("bafk5otulz5hsgszgtpfe7qmpuxqqz4wht4zrhtzj25sguf3fqgrjsni")
	ctx = common.NewThreadIDContext(ctx, threadID)
	resp,err:=cli.Create(context.WithValue(context.Background(),"tkey","bafk5otulz5hsgszgtpfe7qmpuxqqz4wht4zrhtzj25sguf3fqgrjsni"),client.WithName("test"))
	if err!=nil{
		log.Fatal("111111111111111111111",err)
	}
	fmt.Printf("%+v\n",threadID)
	fmt.Printf("%+v\n",resp)
}
