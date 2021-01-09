# 分布式缓存学习日志
[![Build Status](https://www.travis-ci.com/noraincode/gocache.svg?branch=master)](https://www.travis-ci.com/noraincode/gocache)
[![codecov](https://codecov.io/gh/noraincode/gocache/branch/master/graph/badge.svg?token=ILDX3TNROB)](https://codecov.io/gh/noraincode/gocache)

from: https://github.com/geektutu/7days-golang
## 1. 分布式缓存

缓存中最简单的莫过于存储在内存中的键值对缓存了. 说到键值对, 很容易想到的是字典 (dict) 类型, Go 语言中称之为 map. 直接创建一个 map, 每次有新数据就往 map 中插入, 这样做有什么问题呢?

### 1.1 内存不够了怎么办？

那就随机删掉几条数据好了。随机删掉好呢？还是按照时间顺序好呢？或者是有没有其他更好的淘汰策略呢？不同数据的访问频率是不一样的，优先删除访问频率低的数据是不是更好呢？数据的访问频率可能随着时间变化，那优先删除最近最少访问的数据可能是一个更好的选择。我们需要实现一个合理的淘汰策略。

### 1.2 并发写入冲突了怎么办？

对缓存的访问，一般不可能是串行的。map 是没有并发保护的，应对并发的场景，修改操作(包括新增，更新和删除)需要加锁。

### 1.3 单机性能不够怎么办？

单台计算机的资源是有限的，计算、存储等都是有限的。随着业务量和访问量的增加，单台机器很容易遇到瓶颈。如果利用多台计算机的资源，并行处理提高性能就要缓存应用能够支持分布式，这称为水平扩展(scale horizontally)。与水平扩展相对应的是垂直扩展(scale vertically)，即通过增加单个节点的计算、存储、带宽等，来提高系统的性能，硬件的成本和性能并非呈线性关系，大部分情况下，分布式系统是一个更优的选择。

## 2. FIFO/LFU/LRU 算法

### 2.1 FIFO(First In First Out)

先进先出，也就是淘汰缓存中最老(最早添加)的记录。FIFO 认为，最早添加的记录，其不再被使用的可能性比刚添加的可能性大。这种算法的实现也非常简单，创建一个队列，新增记录添加到队尾，每次内存不够时，淘汰队首。但是很多场景下，部分记录虽然是最早添加但也最常被访问，而不得不因为呆的时间太长而被淘汰。这类数据会被频繁地添加进缓存，又被淘汰出去，导致缓存命中率降低。

### 2.2 LFU(Least Frequently Used)

最少使用，也就是淘汰缓存中访问频率最低的记录。LFU 认为，如果数据过去被访问多次，那么将来被访问的频率也更高。LFU 的实现需要维护一个按照访问次数排序的队列，每次访问，访问次数加1，队列重新排序，淘汰时选择访问次数最少的即可。LFU 算法的命中率是比较高的，但缺点也非常明显，维护每个记录的访问次数，对内存的消耗是很高的；另外，如果数据的访问模式发生变化，LFU 需要较长的时间去适应，也就是说 LFU 算法受历史数据的影响比较大。例如某个数据历史上访问次数奇高，但在某个时间点之后几乎不再被访问，但因为历史访问次数过高，而迟迟不能被淘汰。

### 2.3 LRU(Least Recently Used)

最近最少使用，相对于仅考虑时间因素的 FIFO 和仅考虑访问频率的 LFU，LRU 算法可以认为是相对平衡的一种淘汰算法。LRU 认为，如果数据最近被访问过，那么将来被访问的概率也会更高。LRU 算法的实现非常简单，维护一个队列，如果某条记录被访问了，则移动到队尾，那么队首则是最近最少访问的数据，淘汰该条记录即可。

## 3. LRU 算法实现
### 核心数据结构
code: `/lru/lru.go`

字典(map)，存储键和值的映射关系

根据某个键(key)查找对应的值(value)的复杂是O(1)，在字典中插入一条记录的复杂度也是O(1). 

将所有的值放到双向链表中，这样，当访问到某个值时，将其移动到队尾的复杂度是O(1)，在队尾新增一条记录以及删除一条记录的复杂度均为O(1)。

## 4. 单机并发缓存
### 4.1 sync.Mutex
多个协程(goroutine)同时读写同一个变量，在并发度较高的情况下，会发生冲突。确保一次只有一个协程(goroutine)可以访问该变量以避免冲突，这称之为互斥，互斥锁可以解决这个问题。
> sync.Mutex 是一个互斥锁，可以由不同的协程加锁和解锁。

sync.Mutex 是 Go 语言标准库提供的一个互斥锁，当一个协程(goroutine)获得了这个锁的拥有权后，其它请求锁的协程(goroutine) 就会阻塞在 Lock() 方法的调用上，直到调用 Unlock() 锁被释放。

使用 sync.Mutex 封装 LRU 的几个方法，使之支持并发的读写。在这之前，抽象一个只读数据结构 ByteView 用来表示缓存值，是 gocache 主要的数据结构之一。

code `/byteview.go`

### 4.2 支持并发读写
实例化 lru，封装 get 和 set 方法，并添加互斥锁 mu

code: `/cache.go`

### 4.3 主体结构 Group
Group 是 gocache 最核心的数据结构，负责与用户的交互，并且控制缓存值存储和获取的流程

```                            是
接收 key --> 检查是否被缓存 -----> 返回缓存值 ⑴
                |  否                         是
                |-----> 是否应当从远程节点获取 -----> 与远程节点交互 --> 返回缓存值 ⑵
                            |  否
                            |-----> 调用`回调函数`，获取值并添加到缓存 --> 返回缓存值 ⑶
```

代码结构:

```
gocache/
    |--lru/
        |--lru.go  // lru 缓存淘汰策略
    |--byteview.go // 缓存值的抽象与封装
    |--cache.go    // 并发控制
    |--gocache.go // 负责与外部交互，控制缓存存储和获取的主流程
```

#### 4.3.1 回调 Getter
如果缓存不存在，应从数据源（文件，数据库等）获取数据并添加到缓存中

是否应该支持多种数据源的配置? 不应该

- 数据源的种类太多，没办法一一实现
- 扩展性不好, 如何从源头获取数据，应该是用户决定的事情，我们就把这件事交给用户好了

**Solution**: 设计一个回调函数(callback)，在缓存不存在时，调用这个函数，得到源数据

`/gocache.go`
- 定义接口 Getter 和 回调函数 Get(key string)([]byte, error)，参数是 key，返回值是 []byte。
- 定义函数类型 GetterFunc，并实现 Getter 接口的 Get 方法。
- 函数类型实现某一个接口，称之为接口型函数，方便使用者在调用时既能够传入函数作为参数，也能够传入实现了该接口的结构体作为参数。

#### 4.3.2 Group 的定义
- 一个 Group 可以认为是一个缓存的命名空间，每个 Group 拥有一个唯一的名称 name。比如可以创建三个 Group，缓存学生的成绩命名为 scores，缓存学生信息的命名为 info，缓存学生课程的命名为 courses。
- 第二个属性是 getter Getter，即缓存未命中时获取源数据的回调(callback)。
- 第三个属性是 mainCache cache，即一开始实现的并发缓存。
- 构建函数 NewGroup 用来实例化 Group，并且将 group 存储在全局变量 groups 中。
- GetGroup 用来特定名称的 Group，这里使用了只读锁 RLock()，因为不涉及任何冲突变量的写操作。

#### 4.3.3 Group 的 Get 方法
- Get 方法实现了上述所说的流程 ⑴ 和 ⑶。
- 流程 ⑴ ：从 mainCache 中查找缓存，如果存在则返回缓存值。
- 流程 ⑶ ：缓存不存在，则调用 load 方法，load 调用 getLocally（分布式场景下会调用 getFromPeer 从其他节点获取），getLocally 调用用户回调函数 g.getter.Get() 获取源数据，并且将源数据添加到缓存 mainCache 中（通过 populateCache 方法）