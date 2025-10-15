package cache

import (
	"github.com/Qiang-y/cached/cache/cachepb"
)

type PeerPicker interface {
	PeerPicker(key string) (peerGetter PeerGetter, ok bool)
}

// PeerGetter 对应一个通信客户端
type PeerGetter interface {
	Get(req *cachepb.Request, res *cachepb.Response) error
}
