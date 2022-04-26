package geecache

import (
	"testing"
	"reflect"
	"fmt"
	"log"
)

// 用一个 map 模拟耗时的数据库
var db = map[string]string {
	"Tom": "630",
	"Jack": "589",
	"Sam": "567",
}

// 写一个测试用例来保证回调函数能够正常工作
func TestGetter(t *testing.T) {
	// 借助 GetterFunc 的类型转换，将一个匿名回调函数转换成了接口 f Getter。
	var f Getter = GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})

	expect := []byte("key")
	// 调用该接口的方法 f.Get(key string)，实际上就是在调用匿名回调函数
	if v, _ := f.Get("key"); !reflect.DeepEqual(v, expect) {
		t.Errorf("callback failed")
	}
}

// 创建 group 实例，并测试 Get 方法
func TestGet(t *testing.T) {
	loadCounts := make(map[string]int, len(db)) // 使用 loadCounts 统计某个键调用回调函数的次数
	gee := NewGroup("scores", 2<<10, GetterFunc( // 1）在缓存为空的情况下，能够通过回调函数获取到源数据
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				if _, ok := loadCounts[key]; !ok { // key 还没有调用过回调函数
					loadCounts[key] = 0
				}
				loadCounts[key]++
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
	
	for k, v := range db { // gee.Get(k) 在缓存已经存在的情况下，直接从缓存中获取
		if view, err := gee.Get(k); err != nil || view.String() != v {
			t.Fatal("failed to get value of Tom")
		}
		if _, err := gee.Get(k); err != nil || loadCounts[k] > 1 { // 如果次数大于1，则表示调用了多次回调函数，没有缓存。
			t.Fatalf("cache %s miss", k)
		}
	}

	if view, err := gee.Get("unknown"); err == nil {
		t.Fatalf("the value of unkonw should be empty, but %s got", view)
	}
}

// func TestGetGroup(t *testing.T) {
// 	groupName := "scores"
// 	NewGroup(groupName, 2<<10, GetterFunc(
// 		func(key string) (bytes []byte, err error) { return }))
// 	if group := GetGroup(groupName); group == nil || group.name != groupName {
// 		t.Fatalf("group %s not exist", groupName)
// 	}

// 	if group := GetGroup(groupName + "111"); group != nil {
// 		t.Fatalf("expect nil, but %s got", group.name)
// 	}
// }