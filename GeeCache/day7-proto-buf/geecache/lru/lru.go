package lru

import "container/list"

// Cache is a LRU cache. It is not safe for concurrent access.
type Cache struct {
	maxBytes int64 // 允许使用的最大内存
	nbytes int64 // 当前已使用的内存
	ll *list.List // Go 语言标准库实现的双向链表list.List
	cache map[string]*list.Element // 键是字符串，值是双向链表中对应节点的指针
	// 可选并在清除entry时执行。
	OnEvicted func(key string, value Value) // 某条记录被移除时的回调函数，可以为 nil
}

// 记录的定义
type entry struct {
	key string
	value Value
}

// Value 使用使用 Len 来计算它需要多少字节
type Value interface { // 实现了 Value 接口的任意类型
	Len() int // 用于返回值所占用的内存大小
}

// New 是 Cache 的构造函数
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes: maxBytes,
		ll: list.New(),
		cache: make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// Get 查找键的值
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele) // 将链表中的节点 ele 移动到队尾（双向链表作为队列，队首队尾是相对的，在这里约定 front 为队尾）
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

// 删除，RemoveOldest实际上是缓存淘汰. 移除最近最少访问的节点（队首）。
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back() // 返回链表最后一个元素(取到队首节点)
	if ele != nil {
		c.ll.Remove(ele) // 删除链表中的元素ele
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key) // 从字典中 c.cache 删除该节点的映射关系
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len()) // 更新当前所用的内存 c.nbytes
		if c.OnEvicted != nil { // 如果回调函数 OnEvicted 不为 nil，则调用回调函数
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// 新增/修改: Add 向缓存中添加一个值。
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok { // 如果键存在，则更新对应节点的值，并将该节点移到队尾
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len()) // 新增的值大小减去原有key对应值的大小
		kv.value = value
	} else { // 不存在则是新增场景，首先队尾添加新节点 &entry{key, value}, 并字典中添加 key 和节点的映射关系。
		ele := c.ll.PushFront(&entry{key, value}) // 将一个值为v的新元素插入链表的第一个位置(队尾)，返回生成的新元素
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len()) // 更新 c.nbytes
	}
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest() // 如果当前内存超过了设定的最大值 c.maxBytes，则移除最少访问的节点。
	}
}

//  Len() 用来获取添加了多少条数据
func (c *Cache) Len() int {
	return c.ll.Len()
}

