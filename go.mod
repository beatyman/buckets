module github.com/beatyman/buckets

go 1.16

require (
	github.com/alecthomas/jsonschema v0.0.0-20200530073317-71f438968921
	github.com/gogo/status v1.1.0
	github.com/golang/protobuf v1.5.2
	github.com/gosimple/slug v1.12.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/ipfs/go-blockservice v0.3.0
	github.com/ipfs/go-cid v0.1.0
	github.com/ipfs/go-datastore v0.5.0
	github.com/ipfs/go-ds-flatfs v0.4.5
	github.com/ipfs/go-ipfs-blockstore v1.2.0
	github.com/ipfs/go-ipfs-chunker v0.0.5
	github.com/ipfs/go-ipfs-ds-help v1.1.0
	github.com/ipfs/go-ipfs-exchange-offline v0.2.0
	github.com/ipfs/go-ipfs-http-client v0.4.0
	github.com/ipfs/go-ipld-cbor v0.0.5
	github.com/ipfs/go-ipld-format v0.4.0
	github.com/ipfs/go-log/v2 v2.3.0
	github.com/ipfs/go-merkledag v0.6.0
	github.com/ipfs/go-unixfs v0.3.1
	github.com/ipfs/interface-go-ipfs-core v0.7.0
	github.com/libp2p/go-libp2p-core v0.8.6
	github.com/logrusorgru/aurora v2.0.3+incompatible
	github.com/manifoldco/promptui v0.9.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-multiaddr v0.5.0
	github.com/multiformats/go-multibase v0.0.3
	github.com/multiformats/go-multihash v0.1.0
	github.com/radovskyb/watcher v1.0.7
	github.com/sabhiram/go-gitignore v0.0.0-20210923224102-525f6e181f06
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	github.com/textileio/dcrypto v0.0.1
	github.com/textileio/go-threads v1.1.6-0.20220406044848-cdd032536e1f
	github.com/textileio/powergate/v2 v2.6.2
	github.com/textileio/textile/v2 v2.6.18
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	google.golang.org/grpc v1.39.0
	google.golang.org/protobuf v1.27.1
)

replace github.com/ipfs/go-datastore => github.com/ipfs/go-datastore v0.4.5

replace github.com/ipfs/go-ds-flatfs => github.com/ipfs/go-ds-flatfs v0.4.4

replace github.com/ipfs/go-ipfs-blockstore => github.com/ipfs/go-ipfs-blockstore v1.0.4
