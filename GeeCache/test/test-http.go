package main

import (
	"log"
	"net/http"
)
// 实现一个服务端，无论接收到什么请求，都返回字符串 “Hello World!”

// 创建任意类型 server，并实现 ServeHTTP 方法。
type server int

func (h *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL.Path)
	w.Write([]byte("Hello World!"))
}

func main() {
	var s server
	// 调用 http.ListenAndServe 在 9999 端口启动 http 服务，
	// 处理请求的对象为 s server。
	http.ListenAndServe("localhost:9999", &s)
}