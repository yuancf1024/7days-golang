package main

import (
	"fmt"
	"sync"
	"time"
)

var m sync.Mutex
var set = make(map[int]bool, 0)

func printOnce(num int) {
	// 用互斥锁的Lock()和Unlock() 方法将冲突的部分包裹起来：
	m.Lock()
	// 释放锁还有另外一种写法
	// defer m.Unlock()
	if _, exist := set[num]; !exist {
		fmt.Println(num)
	}
	set[num] = true
	m.Unlock()
}

func main() {
	for i := 0; i < 100; i++ {
		go printOnce(100)
	}
	time.Sleep(time.Second)
}