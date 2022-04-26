package geecache

import (
	"fmt"
	"log"
	"net/http"
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