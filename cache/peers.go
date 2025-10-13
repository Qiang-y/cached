package cache

type PeerPicker interface {
	PeerPicker(key string) (peerGetter PeerGetter, ok bool)
}

// PeerGetter 对应一个通信客户端
type PeerGetter interface {
	Get(group string, key string) ([]byte, error)
}
