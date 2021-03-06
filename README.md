# 7 days golang programs from scratch

7天用Go从零实现系列

参考大佬 @geektutu 的代码仓库 [7days-golang
](https://github.com/geektutu/7days-golang)

# 分布式缓存系统GeeCache

**缓存**：第一次请求时将一些耗时操作的结果暂存，以后遇到相同的请求，直接返回暂存的数据。

缓存中最简单的莫过于存储在内存中的**键值对缓存**了。

简单的Map 键值对缓存可能存在诸如内存不够、并发写入冲突、单机性能不够等问题。

设计一个分布式缓存系统，需要考虑**资源控制、淘汰策略、并发、分布式节点通信**等各个方面的问题。而且，针对不同的应用场景，还需要在不同的特性之间权衡，例如，是否需要支持缓存更新？还是假定缓存在淘汰之前是不允许改变的。不同的权衡对应着不同的实现。

GeeCache 基本上模仿了 `groupcache` 的实现，支持特性有：

- 单机缓存和基于 HTTP 的分布式缓存
- 最近最少访问(Least Recently Used, LRU) 缓存策略
- 使用 Go 锁机制防止缓存击穿
- 使用一致性哈希选择节点，实现负载均衡
- 使用 protobuf 优化节点间二进制通信

## LRU缓存淘汰策略

**常用的三种缓存淘汰算法：**

- **FIFO 先进先出**，淘汰缓存中最老(最早添加)的记录。创建一个队列，新增记录添加到队尾，每次内存不够时，淘汰队首。对于很早被添加进来的数据，因为呆的的时间长而被淘汰，会被频繁地添加进缓存，又被淘汰出去，导致缓存命中率降低。
- **LFU 最少使用**，淘汰缓存中访问频率最低的记录。维护一个按照访问次数排序的队列，每次访问，访问次数加1，队列重新排序，淘汰时选择访问次数最少的即可。缺点：维护每个记录的访问次数，对内存的消耗是很高的；LFU 算法受历史数据的影响比较大。
- **LRU 最近最少使用**，相对于仅考虑时间因素的 FIFO 和仅考虑访问频率的 LFU，LRU 算法可以认为是相对平衡的一种淘汰算法。LRU 认为，如果数据最近被访问过，那么将来被访问的概率也会更高。LRU 算法的实现非常简单，维护一个队列，如果某条记录被访问了，则移动到队尾，那么队首则是最近最少访问的数据，淘汰该条记录即可。

**LRU核心数据结构**

![LRU核心数据结构](figure/lru.jpg)
- 绿色的是字典(map)，存储键和值的映射关系。这样根据某个键(key)查找对应的值(value)的复杂是O(1)，在字典中插入一条记录的复杂度也是O(1)。
- 红色的是双向链表(double linked list)实现的队列。将所有的值放到双向链表中，这样，当访问到某个值时，将其移动到队尾的复杂度是O(1)，在队尾新增一条记录以及删除一条记录的复杂度均为O(1)。

创建一个包含字典和双向链表的结构体类型 `Cache` ，方便实现后续的增删查改操作：
- 查找：第一步是从字典中找到对应的双向链表的节点，第二步，将该节点移动到队尾。
- 删除：实际上是缓存淘汰。即移除最近最少访问的节点（队首）
- 新增：如果键存在，则更新对应节点的值，并将该节点移到队尾。
- 修改：不存在则是新增场景，首先队尾添加新节点 &entry{key, value}, 并字典中添加 key 和节点的映射关系。

## 单机并发缓存

多个协程(goroutine)同时读写同一个变量，在并发度较高的情况下，会发生冲突。确保一次只有一个协程(goroutine)可以访问该变量以避免冲突，这称之为互斥，互斥锁可以解决这个问题。

> `sync.Mutex` 是一个互斥锁，可以由不同的协程加锁和解锁。

`sync.Mutex` 是 Go 语言标准库提供的一个互斥锁，当一个协程(goroutine)获得了这个锁的拥有权后，其它请求锁的协程(goroutine) 就会阻塞在 `Lock()` 方法的调用上，直到调用 `Unlock()` 锁被释放。

在 `add` 方法中，判断了 `c.lru` 是否为 nil，如果等于 nil 再创建实例。这种方法称之为**延迟初始化(Lazy Initialization)**，一个对象的延迟初始化意味着*该对象的创建将会延迟至第一次使用该对象时*。主要用于提高性能，并减少程序内存要求

**主体结构Group**

可以认为是一个缓存的命名空间，每个 Group 拥有一个唯一的名称 name。

分布式缓存工作原理：

```shell
                            是
接收 key --> 检查是否被缓存 -----> 返回缓存值 ⑴
                |  否                         是
                |-----> 是否应当从远程节点获取 -----> 与远程节点交互 --> 返回缓存值 ⑵
                            |  否
                            |-----> 调用`回调函数`，获取值并添加到缓存 --> 返回缓存值 ⑶
```

```shell
geecache/
    |--lru/
        |--lru.go  // lru 缓存淘汰策略
    |--byteview.go // 缓存值的抽象与封装
    |--cache.go    // 并发控制
    |--geecache.go // 负责与外部交互，控制缓存存储和获取的主流程
```

**缓存不存在？**

设计了一个回调函数(callback)，在缓存不存在时，调用这个函数，得到源数据。如何从源头获取数据，应该是用户决定的事情，把这件事交给用户好了。

**核心方法Get**
获取缓存，Get 方法实现了上述所说的流程 ⑴ 和 ⑶。
- 流程 ⑴ ：从 mainCache 中查找缓存，如果存在则返回缓存值。
- 流程 ⑶ ：缓存不存在，则调用 load 方法，load 调用 getLocally（分布式场景下会调用 getFromPeer 从其他节点获取），getLocally 调用用户回调函数 g.getter.Get() 获取源数据，并且将源数据添加到缓存 mainCache 中（通过 populateCache 方法）

## HTTP服务端

分布式缓存需要实现节点间通信，建立**基于 HTTP 的通信机制**是比较常见和简单的做法。如果一个节点启动了 HTTP 服务，那么这个节点就可以被其他节点访问。为单机节点搭建 HTTP Server。

不与其他部分耦合，将这部分代码放在新的 http.go 文件中，当前的代码结构如下：
```shell
geecache/
    |--lru/
        |--lru.go  // lru 缓存淘汰策略
    |--byteview.go // 缓存值的抽象与封装
    |--cache.go    // 并发控制
    |--geecache.go // 负责与外部交互，控制缓存存储和获取的主流程
	|--http.go     // 提供被其他节点访问的能力(基于http)
```

## 一致性哈希

主要是为了解决：

- 当一个节点接收到请求时，它自身没有缓存数据，使用一致性哈希确定从哪个节点获取数据（or数据源）。对于给定的 key，每一次都选择同一个节点
- 解决缓存雪崩。缓存雪崩：缓存在同一时刻全部失效，造成瞬时DB请求量大、压力骤增，引起雪崩。常因为缓存服务器宕机，或缓存设置了相同的过期时间引起。普通的哈希解决了缓存性能的问题，但是没有考虑节点数量变化的场景。


**算法原理**：

一致性哈希算法将 key 映射到 2^32 的空间中，将这个数字首尾相连，形成一个环。

- 计算节点/机器(通常使用节点的名称、编号和 IP 地址)的哈希值，放置在环上。
- 计算 key 的哈希值，放置在环上，**顺时针寻找到的第一个节点**，就是应选取的节点/机器。

![](figure/add_peer.jpg)

> 一致性哈希算法，在新增/删除节点时，**只需要重新定位该节点附近的一小部分数据，而不需要重新定位所有的节点**，这就解决了上述的问题。

**虚拟节点**

引入了虚拟节点的概念，一个真实节点对应多个虚拟节点。

假设 1 个真实节点对应 3 个虚拟节点，那么 peer1 对应的虚拟节点是 peer1-1、 peer1-2、 peer1-3（通常以添加编号的方式实现），其余节点也以相同的方式操作。

- 第一步，计算虚拟节点的 Hash 值，放置在环上。
- 第二步，计算 key 的 Hash 值，在环上顺时针寻找到应选取的虚拟节点，例如是 peer2-1，那么就对应真实节点 peer2。

虚拟节点扩充了节点的数量，**解决了节点较少的情况下数据容易倾斜的问题**。而且代价非常小，只需要增加一个字典(map)维护真实节点与虚拟节点的映射关系即可。

## 分布式节点

**分布式缓存工作原理：**

```shell
                            是
接收 key --> 检查是否被缓存 -----> 返回缓存值 ⑴
                |  否                         是
                |-----> 是否应当从远程节点获取 -----> 与远程节点交互 --> 返回缓存值 ⑵
                            |  否
                            |-----> 调用`回调函数`，获取值并添加到缓存 --> 返回缓存值 ⑶
```

进一步细化流程 ⑵：

```shell
使用一致性哈希选择节点        是                                    是
    |-----> 是否是远程节点 -----> HTTP 客户端访问远程节点 --> 成功？-----> 服务端返回返回值
                    |  否                                    ↓  否
                    |----------------------------> 回退到本地节点处理。
```

## 防止缓存击穿

**缓存的几个重要概念：**

- **缓存雪崩**：缓存在同一时刻全部失效，造成瞬时DB请求量大、压力骤增，引起雪崩。缓存雪崩通常因为缓存服务器宕机、缓存的 key 设置了相同的过期时间等引起。

- **缓存击穿**：一个存在的key，在缓存过期的一刻，同时有大量的请求，这些请求都会击穿到 DB ，造成瞬时DB请求量大、压力骤增。

- **缓存穿透**：查询一个不存在的数据，因为不存在则不会写到缓存中，所以每次都会去请求 DB，如果瞬间流量过大，穿透到 DB，导致宕机。


**仿singleflight库**

直译过来就是单飞，这个库的主要作用就是**将一组相同的请求合并成一个请求，实际上只会去请求一次，然后对所有的请求返回相同的结果。**

在一瞬间有大量请求get(key)，而且key未被缓存或者未被缓存在当前节点 如果不用singleflight，那么这些请求都会发送远端节点或者从本地数据库读取，会造成远端节点或本地数据库压力猛增。使用singleflight，**第一个get(key)请求到来时，singleflight会记录当前key正在被处理，后续的请求只需要等待第一个请求处理完成，取返回值即可。**

并发场景下如果 GeeCache 已经向其他节点/源获取数据了，那么就加锁阻塞其他相同的请求，等待请求结果，防止其他节点/源压力猛增被击穿。


## 使用 Protobuf通信

protobuf 即 `Protocol Buffers`，Google 开发的一种数据描述语言，是一种轻便高效的结构化数据存储格式，与语言、平台无关，**可扩展可序列化**。**protobuf 以二进制方式存储，占用空间小。**

protobuf 广泛地应用于远程过程调用(RPC) 的二进制传输，使用 protobuf 的目的非常简单，为了获得更高的性能。**传输前使用 protobuf 编码，接收方再进行解码，可以显著地降低二进制传输的大小**。另外一方面，protobuf 可非常适合传输结构化数据，便于通信字段的扩展。

使用 protobuf 一般分为以下 2 步：

按照 protobuf 的语法，在 .proto 文件中定义数据结构，并使用 protoc 生成 Go 代码（.proto 文件是跨平台的，还可以生成 C、Java 等其他源码文件）。
在项目代码中引用生成的 Go 代码。



