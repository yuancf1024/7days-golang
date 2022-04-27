package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash 映射到 uint32 (0~2^32-1)
// 定义了函数类型 Hash，采取依赖注入的方式，允许用于替换成自定义的 Hash 函数，
// 也方便测试时替换，默认为 crc32.ChecksumIEEE 算法。
type Hash func(data []byte) uint32

// Map 是一致性哈希算法的主数据结构
// Map 包含所有哈希键
type Map struct {
	hash Hash // Hash 函数 hash
	replicas int // 虚拟节点倍数 replicas
	keys []int // Sorted 哈希环 keys
	hashMap map[int]string // 虚拟节点与真实节点的映射表 hashMap, 键是虚拟节点的哈希值，值是真实节点的名称。
}

// New 创建 Map实例
// 构造函数 New() 允许自定义虚拟节点倍数和 Hash 函数
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash: fn,
		hashMap: make(map[int]string),
	}

	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// 添加真实节点/机器的 Add() 方法
func (m *Map) Add(keys ...string) { // Add 函数允许传入 0 或 多个真实节点的名称。
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ { // 对每一个真实节点 key，对应创建 m.replicas 个虚拟节点
			hash := int(m.hash([]byte(strconv.Itoa(i) + key))) // 通过添加编号的方式区分不同虚拟节点。
			// 使用 m.hash() 计算虚拟节点的哈希值
			m.keys = append(m.keys, hash) // 使用 append(m.keys, hash) 添加到环上。
			m.hashMap[hash] = key // 在 hashMap 中增加虚拟节点和真实节点的映射关系
		}
	}
	sort.Ints(m.keys) // 环上的哈希值排序
}

// 实现选择节点的 Get() 方法
// Get获取散列中与所提供的键最近的项
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key))) //  第一步，计算 key 的哈希值
	// 第二步，顺时针找到第一个匹配的虚拟节点的下标 idx, 从 m.keys 中获取到对应的哈希值
	// 二分查找合适的虚拟节点
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	return m.hashMap[m.keys[idx%len(m.keys)]] // 第三步，通过 hashMap 映射得到真实的节点
	// 如果 idx == len(m.keys)，说明应选择 m.keys[0]，
	// 因为 m.keys 是一个环状结构，所以用取余数的方式来处理这种情况。
} 

// 删除只需要删除掉节点对应的虚拟节点和映射关系，
// 至于均摊给其他节点，那是删除之后自然会发生的
// Remove use to remove a key and its virtual keys on the ring and map
func (m *Map) Remove(key string) {
	for i := 0; i < m.replicas; i++ {
		hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
		idx := sort.SearchInts(m.keys, hash)
		m.keys = append(m.keys[:idx], m.keys[idx+1:]...)
		delete(m.hashMap, hash)
	}
}