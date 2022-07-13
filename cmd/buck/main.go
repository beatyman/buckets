package main

import (
	"context"
	"github.com/beatyman/buckets/api/bucketsd/client"
	logging "github.com/ipfs/go-log/v2"
	"google.golang.org/grpc"
)

var log = logging.Logger("db")
func main() {
	cli,err:=client.NewClient("127.0.0.1:7006",grpc.WithInsecure())
	if err!=nil{
		log.Fatal(err)
	}
	resp,err:=cli.Create(context.TODO())
	if err!=nil{
		log.Fatal(err)
	}
	log.Infof("%+v",resp)
}
