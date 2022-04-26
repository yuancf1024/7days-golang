package geecache

import (
	"fmt"
	"geecache/consistenthash"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_geecache/"
	defaultReplicas = 50
)

// HTTPPool为一个HTTP对等体池实现了PeerPicker。
type HTTPPool struct {
	// peer的基本URL，例如: “https://example.net:8000”
	self        string                 // 用来记录自己的地址, 包括主机名/IP 和端口
	basePath    string                 // 作为节点间通讯地址的前缀，默认是 /_geecache/
	mu          sync.Mutex             // guards peers and httpGetters
	peers       *consistenthash.Map    // 新增成员变量 peers，类型是一致性哈希算法的 Map，用来根据具体的 key 选择节点
	httpGetters map[string]*httpGetter // keyed by e.g. "http://10.0.0.2:8008"
	// 新增成员变量 httpGetters，映射远程节点与对应的 httpGetter
	// 每一个远程节点对应一个 httpGetter，因为 httpGetter 与远程节点的地址 baseURL 有关。
}

// NewHTTPPool初始化对等体的HTTP池。
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Log 打印服务端名字
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP 处理所有http请求
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 首先判断访问路径的前缀是否是 basePath，不是返回错误。
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)

	// 约定访问路径格式
	// /<basepath>/<groupname>/<key> required
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupname := parts[0]
	key := parts[1]

	group := GetGroup(groupname) // 通过 groupname 得到 group 实例
	if group == nil {
		http.Error(w, "no such group: "+groupname, http.StatusNotFound)
		return
	}

	view, err := group.Get(key) // 使用 group.Get(key) 获取缓存数据
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice()) // 使用 w.Write() 将缓存值作为 httpResponse 的 body 返回
}

// Set updates the pool's list of peers
// Set() 方法实例化了一致性哈希算法，并且添加了传入的节点
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers { // 并为每一个节点创建了一个 HTTP 客户端 httpGetter
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

// PickPeer picks a peer according to key
// PickPeer() 包装了一致性哈希算法的 Get() 方法，根据具体的 key，
// 选择节点，返回节点对应的 HTTP 客户端。
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil) // 确保这个类型实现了这个接口 如果没有实现会报错的

// 首先创建具体的 HTTP 客户端类 httpGetter，实现 PeerGetter 接口。
type httpGetter struct {
	baseURL string
}

func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL, // baseURL 表示将要访问的远程节点的地址，例如 http://example.com/_geecache/
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
	res, err := http.Get(u) // 使用 http.Get() 方式获取返回值，并转换为 []bytes 类型
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	return bytes, nil
}

var _ PeerGetter = (*httpGetter)(nil)
