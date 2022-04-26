package geecache

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

const defaultBasePath = "/_geecache/"

// HTTPPool为一个HTTP对等体池实现了PeerPicker。  
type HTTPPool struct {
	// 对等体的基本URL，例如: “https://example.net:8000”
	self string // 用来记录自己的地址, 包括主机名/IP 和端口
	basePath string // 作为节点间通讯地址的前缀，默认是 /_geecache/
}

// NewHTTPPool初始化对等体的HTTP池。  
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self: self,
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
		http.Error(w, "no such group: " + groupname, http.StatusNotFound)
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

var _ PeerPicker = (*HTTPPool)(nil)

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

