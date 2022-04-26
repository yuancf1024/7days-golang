package main

/*
$ curl http://localhost:9999/_geecache/scores/Tom
630
$ curl http://localhost:9999/_geecache/scores/kkk
kkk not exist
*/

import (
	"fmt"
	"geecache"
	"log"
	"net/http"
)

// 使用 map 模拟了数据源 db
var db = map[string]string{
	"Tom": "630",
	"Jack": "589",
	"Sam": "567",
}

func main() {
	// 创建一个名为 scores 的 Group
	geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(
		func(key string) ([]byte, error) { // 若缓存为空，回调函数会从 db 中获取数据并返回
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	// 使用 http.ListenAndServe 在 9999 端口启动了 HTTP 服务
	addr := "localhost:9999"
	peers := geecache.NewHTTPPool(addr)
	log.Println("geecache is running at", addr)
	log.Fatal(http.ListenAndServe(addr, peers))
}