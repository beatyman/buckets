package main

import (
	"context"
	"fmt"
	"github.com/beatyman/buckets/api/bucketsd/client"
	"github.com/beatyman/buckets/api/common"
	"github.com/beatyman/buckets/buckets/local"
	"github.com/beatyman/buckets/cmd"
	"github.com/cheggaaa/pb/v3"
	logging "github.com/ipfs/go-log/v2"
	"github.com/textileio/go-threads/core/thread"
	"google.golang.org/grpc"
	"os"
	"runtime"
	"time"
	tc "github.com/textileio/go-threads/api/client"
)

var bucks *local.Buckets
var log = logging.Logger("db")

func main() {
	cli, err := client.NewClient("127.0.0.1:7006", grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	threadID, _ := thread.Decode("bafk5otulz5hsgszgtpfe7qmpuxqqz4wht4zrhtzj25sguf3fqgrjsni")
	ctx = common.NewThreadIDContext(ctx, threadID)
	resp, err := cli.Create(context.WithValue(context.Background(), "tkey", "bafk5otulz5hsgszgtpfe7qmpuxqqz4wht4zrhtzj25sguf3fqgrjsni"), client.WithName("test"))
	if err != nil {
		log.Fatal("111111111111111111111", err)
	}
	fmt.Printf("%+v\n", threadID)
	fmt.Printf("%+v\n", resp)

	threadCli, err := tc.NewClient("127.0.0.1:7006", grpc.WithInsecure(),grpc.WithPerRPCCredentials(thread.Credentials{}))
	if err != nil {
		log.Fatal(err)
	}
	config := local.DefaultConfConfig()
	cmdClient := cmd.Clients{
		Buckets: cli,
		Threads: threadCli,
	}
	bucks = local.NewBuckets(&cmdClient, config)

	ID,_:=thread.Decode( "bafk5otulz5hsgszgtpfe7qmpuxqqz4wht4zrhtzj25sguf3fqgrjsni")
	buck, err := bucks.GetLocalBucket(ctx, local.Config{Path:  "F:\\workerspace\\ipfs_merge\\buckets", Key: "bgbcg6rt6ao3xix4vpwlmmhiu3eee67ymqwzyowmbqjnvxyctsuo4ukttm22cd7hl5eu53nymhtvio2tvv4ci2hxjpn3lezlhlvfwniq", Thread:ID})
	if err != nil {
		log.Fatal("111111111111111111111", err)
	}
	var events chan local.Event
	events = make(chan local.Event)
	defer close(events)
	go handleEvents(events)
	fmt.Println("start  push")
	roots, err := buck.PushLocal(
		ctx,
		local.WithForce(true),
		local.WithEvents(events),
	)
	if err != nil {
		log.Fatal("111111111111111111111", err)
	}
	fmt.Printf("%+v\n", roots.Remote)

}

func handleEvents(events chan local.Event) {
	var bar *pb.ProgressBar
	if runtime.GOOS != "windows" {
		bar = pb.New(0)
		bar.Set(pb.Bytes, true)
		tmp := `{{string . "prefix"}}{{counters . }} {{bar . "[" "=" ">" "-" "]"}} {{percent . }} {{etime . }}{{string . "suffix"}}`
		bar.SetTemplate(pb.ProgressBarTemplate(tmp))
	}

	clear := func() {
		if bar != nil {
			_, _ = fmt.Fprintf(os.Stderr, "\033[2K\r")
		}
	}

	for e := range events {
		switch e.Type {
		case local.EventProgress:
			if bar == nil {
				continue
			}
			bar.SetTotal(e.Size)
			bar.SetCurrent(e.Complete)
			if !bar.IsStarted() {
				bar.Start()
			}
			bar.Write()
		case local.EventFileComplete:
			clear()
			_, _ = fmt.Fprintf(os.Stdout,
				"+ %s %s %s\n",
				e.Cid,
				e.Path,
				formatBytes(e.Size, false),
			)
			if bar != nil && bar.IsStarted() {
				bar.Write()
			}
		case local.EventFileRemoved:
			clear()
			_, _ = fmt.Fprintf(os.Stdout, "- %s\n", e.Path)
			if bar != nil && bar.IsStarted() {
				bar.Write()
			}
		}
	}
}

// Copied from https://github.com/cheggaaa/pb/blob/master/v3/util.go
const (
	_KiB = 1024
	_MiB = 1048576
	_GiB = 1073741824
	_TiB = 1099511627776

	_kB = 1e3
	_MB = 1e6
	_GB = 1e9
	_TB = 1e12
)

// Copied from https://github.com/cheggaaa/pb/blob/master/v3/util.go
func formatBytes(i int64, useSIPrefix bool) (result string) {
	if !useSIPrefix {
		switch {
		case i >= _TiB:
			result = fmt.Sprintf("%.02f TiB", float64(i)/_TiB)
		case i >= _GiB:
			result = fmt.Sprintf("%.02f GiB", float64(i)/_GiB)
		case i >= _MiB:
			result = fmt.Sprintf("%.02f MiB", float64(i)/_MiB)
		case i >= _KiB:
			result = fmt.Sprintf("%.02f KiB", float64(i)/_KiB)
		default:
			result = fmt.Sprintf("%d B", i)
		}
	} else {
		switch {
		case i >= _TB:
			result = fmt.Sprintf("%.02f TB", float64(i)/_TB)
		case i >= _GB:
			result = fmt.Sprintf("%.02f GB", float64(i)/_GB)
		case i >= _MB:
			result = fmt.Sprintf("%.02f MB", float64(i)/_MB)
		case i >= _kB:
			result = fmt.Sprintf("%.02f kB", float64(i)/_kB)
		default:
			result = fmt.Sprintf("%d B", i)
		}
	}
	return
}
