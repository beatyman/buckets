package bucketsd

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/beatyman/buckets/api/bucketsd/pb"
	"github.com/beatyman/buckets/api/common"
	"github.com/beatyman/buckets/buckets"
	tdb "github.com/beatyman/buckets/threaddb"
	"github.com/ipfs/go-cid"
	ipfsfiles "github.com/ipfs/go-ipfs-files"
	ipld "github.com/ipfs/go-ipld-format"
	logging "github.com/ipfs/go-log/v2"
	dag "github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-unixfs"
	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/textileio/dcrypto"
	"github.com/textileio/go-threads/core/thread"
	"golang.org/x/sync/errgroup"
	"io"
	"io/ioutil"
	"math"
	gopath "path"
	"sync"
	"time"
)
var (
	log = logging.Logger("db")
	// ErrArchivingFeatureDisabled indicates an archive was requested with archiving disabled.
	ErrArchivingFeatureDisabled = errors.New("archiving feature is disabled")

	// ErrStorageQuotaExhausted indicates the requested operation exceeds the storage allowance.
	ErrStorageQuotaExhausted = errors.New("storage quota exhausted")

	// errInvalidNodeType indicates a node with type other than raw of proto was encountered.
	errInvalidNodeType = errors.New("invalid node type")

	// errDBRequired indicates the request requires a thread ID.
	errDBRequired = errors.New("db required")
)
type ctxKey string
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

// makeSeed returns a raw ipld node containing a random seed.
func makeSeed(key []byte) (ipld.Node, error) {
	seed := make([]byte, 32)
	if _, err := rand.Read(seed); err != nil {
		return nil, err
	}
	// Encrypt seed if key is set (treating the seed differently here would complicate bucket reading)
	if key != nil {
		var err error
		seed, err = encryptData(seed, nil, key)
		if err != nil {
			return nil, err
		}
	}
	return dag.NewRawNode(seed), nil
}

// encryptData encrypts data with the new key, decrypting with current key if needed.
func encryptData(data, currentKey, newKey []byte) ([]byte, error) {
	if currentKey != nil {
		var err error
		data, err = decryptData(data, currentKey)
		if err != nil {
			return nil, err
		}
	}
	r, err := dcrypto.NewEncrypter(bytes.NewReader(data), newKey)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(r)
}

// createPristinePath creates an IPFS path which only contains the seed file.
// The returned path will be pinned.
func (s *Service) createPristinePath(
	ctx context.Context,
	seed ipld.Node,
	key []byte,
) (context.Context, path.Resolved, error) {
	// Create the initial bucket directory
	n, err := newDirWithNode(seed, buckets.SeedName, key)
	if err != nil {
		return ctx, nil, err
	}
	if err = s.IPFSClient.Dag().AddMany(ctx, []ipld.Node{n, seed}); err != nil {
		return ctx, nil, err
	}
	pins := []ipld.Node{n}
	if key != nil {
		pins = append(pins, seed)
	}
	ctx, err = s.pinBlocks(ctx, pins)
	if err != nil {
		return ctx, nil, err
	}
	return ctx, path.IpfsPath(n.Cid()), nil
}

// pinBlocks pins blocks, accounting for sum bytes pinned for context.
func (s *Service) pinBlocks(ctx context.Context, nodes []ipld.Node) (context.Context, error) {
	var totalAddedSize int64
	for _, n := range nodes {
		s, err := n.Stat()
		if err != nil {
			return ctx, fmt.Errorf("getting size of node: %v", err)
		}
		totalAddedSize += int64(s.CumulativeSize)
	}

	// Check context owner's storage allowance
	owner, ok := buckets.BucketOwnerFromContext(ctx)
	if ok {
		log.Debugf("pinBlocks: storage: %d used, %d available, %d requested",
			owner.StorageUsed, owner.StorageAvailable, totalAddedSize)
	}
	if ok && totalAddedSize > 0 && totalAddedSize > owner.StorageAvailable {
		return ctx, ErrStorageQuotaExhausted
	}

	if err := s.IPFSClient.Dag().Pinning().AddMany(ctx, nodes); err != nil {
		return ctx, fmt.Errorf("pinning set of nodes: %v", err)
	}
	return s.addPinnedBytes(ctx, totalAddedSize), nil
}

// addPinnedBytes adds the provided delta to a running total for context.
func (s *Service) addPinnedBytes(ctx context.Context, delta int64) context.Context {
	total, _ := ctx.Value(ctxKey("pinnedBytes")).(int64)
	ctx = context.WithValue(ctx, ctxKey("pinnedBytes"), total+delta)
	owner, ok := buckets.BucketOwnerFromContext(ctx)
	if ok {
		owner.StorageUsed += delta
		// int64(math.MaxInt64) indicates that the user has no current cap so don't deduct
		if owner.StorageAvailable < int64(math.MaxInt64) {
			owner.StorageAvailable -= delta
		}
		owner.StorageDelta += delta
		ctx = buckets.NewBucketOwnerContext(ctx, owner)
	}
	return ctx
}

// getPinnedBytes returns the total pinned bytes for context.
func (s *Service) getPinnedBytes(ctx context.Context) int64 {
	pinned, _ := ctx.Value(ctxKey("pinnedBytes")).(int64)
	return pinned
}
// dagSize returns the cummulative size of root. If root is nil, it returns 0.
func (s *Service) dagSize(ctx context.Context, root path.Path) (int64, error) {
	if root == nil {
		return 0, nil
	}
	stat, err := s.IPFSClient.Object().Stat(ctx, root)
	if err != nil {
		return 0, fmt.Errorf("getting dag size: %v", err)
	}
	return int64(stat.CumulativeSize), nil
}
// createBootstrapedPath creates an IPFS path which is the bootCid UnixFS DAG,
// with tdb.SeedName seed file added to the root of the DAG. The returned path will
// be pinned.
func (s *Service) createBootstrappedPath(
	ctx context.Context,
	destPath string,
	seed ipld.Node,
	bootCid cid.Cid,
	linkKey,
	fileKey []byte,
) (context.Context, path.Resolved, error) {
	pth := path.IpfsPath(bootCid)
	bootSize, err := s.dagSize(ctx, pth)
	if err != nil {
		return ctx, nil, fmt.Errorf("resolving boot cid node: %v", err)
	}

	// Check context owner's storage allowance
	owner, ok := buckets.BucketOwnerFromContext(ctx)
	if ok {
		log.Debugf("createBootstrappedPath: storage: %d used, %d available, %d requested",
			owner.StorageUsed, owner.StorageAvailable, bootSize)
	}
	if ok && bootSize > 0 && bootSize > owner.StorageAvailable {
		return ctx, nil, ErrStorageQuotaExhausted
	}

	// Here we have to walk and possibly encrypt the boot path dag
	n, nodes, err := s.newDirFromExistingPath(ctx, pth, destPath, linkKey, fileKey, seed, buckets.SeedName)
	if err != nil {
		return ctx, nil, err
	}
	if err = s.IPFSClient.Dag().AddMany(ctx, nodes); err != nil {
		return ctx, nil, err
	}
	var pins []ipld.Node
	if linkKey != nil {
		pins = nodes
	} else {
		pins = []ipld.Node{n}
	}
	ctx, err = s.pinBlocks(ctx, pins)
	if err != nil {
		return ctx, nil, err
	}
	return ctx, path.IpfsPath(n.Cid()), nil
}

// newDirWithNode returns a new proto node directory wrapping the node,
// which is encrypted if key is not nil.
func newDirWithNode(n ipld.Node, name string, key []byte) (ipld.Node, error) {
	dir := unixfs.EmptyDirNode()
	dir.SetCidBuilder(dag.V1CidPrefix())
	if err := dir.AddNodeLink(name, n); err != nil {
		return nil, err
	}
	return encryptNode(dir, key)
}

// encryptNode returns the encrypted version of node if key is not nil.
func encryptNode(n *dag.ProtoNode, key []byte) (*dag.ProtoNode, error) {
	if key == nil {
		return n, nil
	}
	cipher, err := encryptData(n.RawData(), nil, key)
	if err != nil {
		return nil, err
	}
	en := dag.NodeWithData(unixfs.FilePBData(cipher, uint64(len(cipher))))
	en.SetCidBuilder(dag.V1CidPrefix())
	return en, nil
}
// decryptData decrypts data with key.
func decryptData(data, key []byte) ([]byte, error) {
	r, err := dcrypto.NewDecrypter(bytes.NewReader(data), key)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return ioutil.ReadAll(r)
}

// newDirFromExistingPath returns a new dir based on path.
// If keys are not nil, this method recursively walks the path, encrypting files and directories.
// If add is not nil, it will be included in the resulting (possibly encrypted) node under a link named addName.
// This method returns the root node and a list of all new nodes (which also includes the root).
func (s *Service) newDirFromExistingPath(
	ctx context.Context,
	pth path.Path,
	destPath string,
	linkKey,
	fileKey []byte,
	add ipld.Node,
	addName string,
) (ipld.Node, []ipld.Node, error) {
	rn, err := s.IPFSClient.ResolveNode(ctx, pth)
	if err != nil {
		return nil, nil, err
	}
	top, ok := rn.(*dag.ProtoNode)
	if !ok {
		return nil, nil, dag.ErrNotProtobuf
	}
	if linkKey == nil && fileKey == nil {
		nodes := []ipld.Node{top}
		if add != nil {
			if err := top.AddNodeLink(addName, add); err != nil {
				return nil, nil, err
			}
			nodes = append(nodes, add)
		}
		return top, nodes, nil
	} else if linkKey == nil || fileKey == nil {
		return nil, nil, fmt.Errorf("invalid link or file key")
	}

	// Walk the node, encrypting the leaves and directories
	var addNode *namedNode
	if add != nil {
		addNode = &namedNode{
			name: addName,
			node: add,
		}
	}
	nmap, err := s.encryptDag(
		ctx,
		top,
		destPath,
		linkKey,
		nil,
		nil,
		fileKey,
		addNode,
	)
	if err != nil {
		return nil, nil, err
	}

	// Collect new nodes
	nodes := make([]ipld.Node, len(nmap))
	i := 0
	for _, tn := range nmap {
		nodes[i] = tn.node
		i++
	}
	return nmap[top.Cid()].node, nodes, nil
}

type namedNode struct {
	name string
	path string
	node ipld.Node
	cid  cid.Cid
}

type namedNodes struct {
	sync.RWMutex
	m map[cid.Cid]*namedNode
}

func newNamedNodes() *namedNodes {
	return &namedNodes{
		m: make(map[cid.Cid]*namedNode),
	}
}

func (nn *namedNodes) Get(c cid.Cid) *namedNode {
	nn.RLock()
	defer nn.RUnlock()
	return nn.m[c]
}

func (nn *namedNodes) Store(c cid.Cid, n *namedNode) {
	nn.Lock()
	defer nn.Unlock()
	nn.m[c] = n
}
// decryptNode returns a decrypted version of node and whether or not it is a directory.
func decryptNode(cn ipld.Node, key []byte) (ipld.Node, bool, error) {
	switch cn := cn.(type) {
	case *dag.RawNode:
		return cn, false, nil // All raw nodes will be leaves
	case *dag.ProtoNode:
		if key == nil {
			return cn, false, nil // Could be a joint, but it's not important to know in the public case
		}
		fn, err := unixfs.FSNodeFromBytes(cn.Data())
		if err != nil {
			return nil, false, err
		}
		if fn.Data() == nil {
			return cn, false, nil // This node is a raw file wrapper
		}
		plain, err := decryptData(fn.Data(), key)
		if err != nil {
			return nil, false, err
		}
		n, err := dag.DecodeProtobuf(plain)
		if err != nil {
			return dag.NewRawNode(plain), false, nil
		}
		n.SetCidBuilder(dag.V1CidPrefix())
		return n, true, nil
	default:
		return nil, false, errInvalidNodeType
	}
}
// encryptDag creates an encrypted version of root that includes all child nodes.
// Leaf nodes are encrypted and linked to parents, which are then encrypted and
// linked to their parents, and so on up to root.
// add will be added to the encrypted root node if not nil.
// This method returns a map of all nodes keyed by their _original_ plaintext cid,
// and a list of the root's direct links.
func (s *Service) encryptDag(
	ctx context.Context,
	root ipld.Node,
	destPath string,
	linkKey []byte,
	currentFileKeys,
	newFileKeys map[string][]byte,
	newFileKey []byte,
	add *namedNode,
) (map[cid.Cid]*namedNode, error) {
	// Step 1: Create a preordered list of joint and leaf nodes
	var stack, joints []*namedNode
	var cur *namedNode
	jmap := make(map[cid.Cid]*namedNode)
	lmap := make(map[cid.Cid]*namedNode)
	ds := s.IPFSClient.Dag()

	stack = append(stack, &namedNode{node: root, path: destPath})
	for len(stack) > 0 {
		n := len(stack) - 1
		cur = stack[n]
		stack = stack[:n]

		if _, ok := jmap[cur.node.Cid()]; ok {
			continue
		}
		if _, ok := lmap[cur.node.Cid()]; ok {
			continue
		}

	types:
		switch cur.node.(type) {
		case *dag.RawNode:
			lmap[cur.node.Cid()] = cur
		case *dag.ProtoNode:
			// Add links to the stack
			cur.cid = cur.node.Cid()
			if currentFileKeys != nil {
				var err error
				cur.node, _, err = decryptNode(cur.node, linkKey)
				if err != nil {
					return nil, err
				}
			}
			for _, l := range cur.node.Links() {
				if l.Name == "" {
					// We have discovered a raw file node wrapper
					// Use the original cur node because file node wrappers aren't encrypted
					lmap[cur.cid] = cur
					break types
				}
				ln, err := l.GetNode(ctx, ds)
				if err != nil {
					return nil, err
				}
				stack = append(stack, &namedNode{
					name: l.Name,
					path: gopath.Join(cur.path, l.Name),
					node: ln,
				})
			}
			joints = append(joints, cur)
			jmap[cur.cid] = cur
		default:
			return nil, errInvalidNodeType
		}
	}

	// Step 2: Encrypt all leaf nodes in parallel
	nmap := newNamedNodes()
	eg, gctx := errgroup.WithContext(ctx)
	for _, l := range lmap {
		l := l
		cfk := getFileKey(nil, currentFileKeys, l.path)
		nfk := getFileKey(newFileKey, newFileKeys, l.path)
		if nfk == nil {
			// This shouldn't happen
			return nil, fmt.Errorf("new file key not found for path %s", l.path)
		}
		eg.Go(func() error {
			if gctx.Err() != nil {
				return nil
			}
			var cn ipld.Node
			switch l.node.(type) {
			case *dag.RawNode:
				data, err := encryptData(l.node.RawData(), cfk, nfk)
				if err != nil {
					return err
				}
				cn = dag.NewRawNode(data)
			case *dag.ProtoNode:
				var err error
				cn, err = s.encryptFileNode(gctx, l.node, cfk, nfk)
				if err != nil {
					return err
				}
			}
			nmap.Store(l.node.Cid(), &namedNode{
				name: l.name,
				node: cn,
			})
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	// Step 3: Encrypt joint nodes in reverse, walking up to root
	// Note: In the case where we're re-encrypting a dag, joints will already be decrypted.
	for i := len(joints) - 1; i >= 0; i-- {
		j := joints[i]
		jn := j.node.(*dag.ProtoNode)
		dir := unixfs.EmptyDirNode()
		dir.SetCidBuilder(dag.V1CidPrefix())
		for _, l := range jn.Links() {
			ln := nmap.Get(l.Cid)
			if ln == nil {
				return nil, fmt.Errorf("link node not found")
			}
			if err := dir.AddNodeLink(ln.name, ln.node); err != nil {
				return nil, err
			}
		}
		if i == 0 && add != nil {
			if err := dir.AddNodeLink(add.name, add.node); err != nil {
				return nil, err
			}
			nmap.Store(add.node.Cid(), add)
		}
		cn, err := encryptNode(dir, linkKey)
		if err != nil {
			return nil, err
		}
		nmap.Store(j.cid, &namedNode{
			name: j.name,
			node: cn,
		})
	}
	return nmap.m, nil
}

func getFileKey(key []byte, pathKeys map[string][]byte, pth string) []byte {
	if pathKeys == nil {
		return key
	}
	k, ok := pathKeys[pth]
	if ok {
		return k
	}
	return key
}

// encryptFileNode encrypts node with the new key, decrypting with current key if needed.
func (s *Service) encryptFileNode(ctx context.Context, n ipld.Node, currentKey, newKey []byte) (ipld.Node, error) {
	fn, err := s.IPFSClient.Unixfs().Get(ctx, path.IpfsPath(n.Cid()))
	if err != nil {
		return nil, err
	}
	defer fn.Close()
	file := ipfsfiles.ToFile(fn)
	if file == nil {
		return nil, fmt.Errorf("node is a directory")
	}
	var r1 io.Reader
	if currentKey != nil {
		r1, err = dcrypto.NewDecrypter(file, currentKey)
		if err != nil {
			return nil, err
		}
	} else {
		r1 = file
	}
	r2, err := dcrypto.NewEncrypter(r1, newKey)
	if err != nil {
		return nil, err
	}
	pth, err := s.IPFSClient.Unixfs().Add(
		ctx,
		ipfsfiles.NewReaderFile(r2),
		options.Unixfs.CidVersion(1),
		options.Unixfs.Pin(false),
	)
	if err != nil {
		return nil, err
	}
	return s.IPFSClient.ResolveNode(ctx, pth)
}
func (s Service) Create(ctx context.Context, request *pb.CreateRequest) (*pb.CreateResponse, error) {
	log.Debugf("received create request")

	threadID ,_:= thread.Decode("bafk5otulz5hsgszgtpfe7qmpuxqqz4wht4zrhtzj25sguf3fqgrjsni")
	//threadID := thread.NewIDV1(thread.Raw, 32)
	ctx = common.NewThreadIDContext(ctx, threadID)

	dbID, ok := common.ThreadIDFromContext(ctx)
	log.Info(dbID,"============================", ctx.Value(ctxKey("threadID")))
	if !ok {
		return nil, errDBRequired
	}
	dbToken, _ := thread.TokenFromContext(ctx)

	var bootCid cid.Cid
	if request.BootstrapCid != "" {
		var err error
		bootCid, err = cid.Decode(request.BootstrapCid)
		if err != nil {
			return nil, fmt.Errorf("invalid bootstrap cid: %v", err)
		}
	}

	// If not created with --unfreeze, just do the normal case.
	ctx, buck, seed, err := s.createBucket(ctx, dbID, dbToken, request.Name, request.Private, bootCid)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	var seedData []byte
	if buck.IsPrivate() {
		fileKey, err := buck.GetFileEncryptionKeyForPath("")
		if err != nil {
			return nil, err
		}
		seedData, err = decryptData(seed.RawData(), fileKey)
		if err != nil {
			return nil, err
		}
	} else {
		seedData = seed.RawData()
	}

	root, err := getPbRoot(dbID, buck)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	log.Info("==========================")
	return &pb.CreateResponse{
		Root:    root,
		Seed:    seedData,
		SeedCid: seed.Cid().String(),
		Pinned:  s.getPinnedBytes(ctx),
	}, nil
}
// createBucket returns a new bucket and seed node.
func (s *Service) createBucket(
	ctx context.Context,
	dbID thread.ID,
	dbToken thread.Token,
	name string,
	private bool,
	bootCid cid.Cid,
) (nctx context.Context, buck *tdb.Bucket, seed ipld.Node, err error) {
	var owner thread.PubKey
	if dbToken.Defined() {
		owner, err = dbToken.PubKey()
		if err != nil {
			return ctx, nil, nil, fmt.Errorf("creating bucket: invalid token public key")
		}
	}

	// Create bucket keys if private
	var linkKey, fileKey []byte
	if private {
		var err error
		linkKey, err = dcrypto.NewKey()
		if err != nil {
			return ctx, nil, nil, err
		}
		fileKey, err = dcrypto.NewKey()
		if err != nil {
			return ctx, nil, nil, err
		}
	}

	// Make a random seed, which ensures a bucket's uniqueness
	seed, err = makeSeed(fileKey)
	if err != nil {
		return
	}

	// Create the bucket directory
	var buckPath path.Resolved
	if bootCid.Defined() {
		ctx, buckPath, err = s.createBootstrappedPath(ctx, "", seed, bootCid, linkKey, fileKey)
		if err != nil {
			return ctx, nil, nil, fmt.Errorf("creating prepared bucket: %v", err)
		}
	} else {
		ctx, buckPath, err = s.createPristinePath(ctx, seed, linkKey)
		if err != nil {
			return ctx, nil, nil, fmt.Errorf("creating pristine bucket: %v", err)
		}
	}

	// Create top-level metadata
	now := time.Now()
	md := map[string]tdb.Metadata{
		"": tdb.NewDefaultMetadata(owner, fileKey, now),
		buckets.SeedName: {
			Roles:     make(map[string]buckets.Role),
			UpdatedAt: now.UnixNano(),
		},
	}
	// Create the bucket using the IPNS key as instance ID
	buck, err = s.Buckets.New(
		ctx,
		dbID,
		"key",
		buckPath,
		now,
		owner,
		md,
		tdb.WithNewBucketName(name),
		tdb.WithNewBucketKey(linkKey),
		tdb.WithNewBucketToken(dbToken))
	if err != nil {
		return
	}

	// Finally, publish the new bucket's address to the name system
	//go s.IPNSManager.Publish(buckPath, buck.Key)
	return ctx, buck, seed, nil
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
func getPbRoot(dbID thread.ID, buck *tdb.Bucket) (*pb.Root, error) {
	var pmdOld *pb.Metadata
	md, ok := buck.Metadata[""]
	if ok {
		var err error
		pmdOld, err = metadataToPb(md)
		if err != nil {
			return nil, err
		}
	}
	pmd := make(map[string]*pb.Metadata)
	for p, md := range buck.Metadata {
		m, err := metadataToPb(md)
		if err != nil {
			return nil, err
		}
		pmd[p] = m
	}
	return &pb.Root{
		Thread:       dbID.String(),
		Key:          buck.Key,
		Owner:        buck.Owner,
		Name:         buck.Name,
		Version:      int32(buck.Version),
		LinkKey:      buck.LinkKey,
		Path:         buck.Path,
		Metadata:     pmdOld, // @todo: For v3, remove this.
		PathMetadata: pmd,
		Archives:     archivesToPb(buck.Archives),
		CreatedAt:    buck.CreatedAt,
		UpdatedAt:    buck.UpdatedAt,
	}, nil
}

func metadataToPb(md tdb.Metadata) (*pb.Metadata, error) {
	roles := make(map[string]pb.PathAccessRole)
	for k, r := range md.Roles {
		var pr pb.PathAccessRole
		switch r {
		case buckets.None:
			pr = pb.PathAccessRole_PATH_ACCESS_ROLE_UNSPECIFIED
		case buckets.Reader:
			pr = pb.PathAccessRole_PATH_ACCESS_ROLE_READER
		case buckets.Writer:
			pr = pb.PathAccessRole_PATH_ACCESS_ROLE_WRITER
		case buckets.Admin:
			pr = pb.PathAccessRole_PATH_ACCESS_ROLE_ADMIN
		default:
			return nil, fmt.Errorf("unknown path access role %d", r)
		}
		roles[k] = pr
	}
	return &pb.Metadata{
		Key:       md.Key,
		Roles:     roles,
		UpdatedAt: md.UpdatedAt,
	}, nil
}

func archivesToPb(archives tdb.Archives) *pb.Archives {
	pba := &pb.Archives{
		Current: &pb.Archive{
			Cid:      archives.Current.Cid,
			DealInfo: dealsToPb(archives.Current.Deals),
		},
		History: make([]*pb.Archive, len(archives.History)),
	}
	for i, a := range archives.History {
		pba.History[i] = &pb.Archive{
			Cid:      a.Cid,
			DealInfo: dealsToPb(a.Deals),
		}
	}
	return pba
}

func dealsToPb(deals []tdb.Deal) []*pb.DealInfo {
	pbd := make([]*pb.DealInfo, len(deals))
	for i, d := range deals {
		pbd[i] = &pb.DealInfo{
			ProposalCid: d.ProposalCid,
			Miner:       d.Miner,
		}
	}
	return pbd
}
