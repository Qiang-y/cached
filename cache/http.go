package cache

import (
	"fmt"
	"github.com/Qiang-y/cached/cache/consistenthash"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/api/"
	defaultReplicas = 50
)

// HTTPPool implements PeerPicker for a pool of HTTP peers.
type HTTPPool struct {
	self        string
	basePath    string
	mu          sync.Mutex
	peers       *consistenthash.Map    // 一致性hash的map，选择存有不同key的节点
	httpGetters map[string]*httpGetter // 存放访问其余节点的http客户端
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// Set 设置更新池的节点列表
func (p *HTTPPool) Set(peers ...string) {
	if len(peers) == 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	p.peers = consistenthash.NewMap(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

// PeerPicker 根据key选择一个对等节点
func (p *HTTPPool) PeerPicker(key string) (peerGetter PeerGetter, ok bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	peer := p.peers.Get(key)
	if peer == "" || peer == p.self {
		return nil, false
	}
	p.Log("Pick peer %s", peer)
	return p.httpGetters[peer], true
}

// 编译时检查，检查HTTPPool实现了PeerPicker接口
var _ PeerPicker = (*HTTPPool)(nil)

func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "nil group "+groupName, http.StatusNotFound)
		return
	}

	value, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(value.ByteSlice())
}

// HTTP 客户端
type httpGetter struct {
	baseURL string // 表示要访问的远程节点地址
}

func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server return code: %v", resp.StatusCode)
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	return bytes, nil
}
