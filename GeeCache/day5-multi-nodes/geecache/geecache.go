package geecache

import (
	"fmt"
	"log"
	"sync"
)


// Group是一个缓存命名空间，并将加载的相关数据分散开来  
type Group struct {
	name string // 每个 Group 拥有一个唯一的名称 name
	getter Getter // 缓存未命中时获取源数据的回调(callback)
	mainCache cache // 一开始实现的并发缓存
}

// Getter为一个键加载数据
type Getter interface {
	Get(key string) ([]byte, error)
}

// GetterFunc 通过一个函数实现Getter。
type GetterFunc func(key string) ([]byte, error)

// Get实现了Getter接口功能
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

var (
	mu sync.RWMutex
	groups = make(map[string]*Group)
)

// NewGroup 创建 Group的一个实例, 实例化 Group，并且将 group 存储在全局变量 groups 中
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name : name,
		getter : getter,
		mainCache : cache{cacheBytes: cacheBytes},
	}
	groups[name] = g
	return g
}

// GetGroup返回先前用NewGroup创建的命名组，如果没有这样的组，则返回nil。 
func GetGroup(name string) *Group {
	mu.RLock() // 使用了只读锁 RLock()，因为不涉及任何冲突变量的写操作
	g := groups[name]
	mu.RUnlock()
	return g
}

// Get 方法 从缓存中获取一个键的值
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	// 流程 ⑴ ：从 mainCache 中查找缓存，如果存在则返回缓存值。

	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}

	// 流程 ⑶ ：缓存不存在，则调用 load 方法
	return g.load(key)
}

// load 调用 getLocally（分布式场景下会调用 getFromPeer 从其他节点获取）
func (g *Group) load(key string) (value ByteView, err error) {
	return g.getLocally(key)
}

// getLocally 调用用户回调函数 g.getter.Get() 获取源数据，
// 并且将源数据添加到缓存 mainCache 中（通过 populateCache 方法）
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}