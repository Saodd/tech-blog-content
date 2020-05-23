---
title: 'etcd入门教程 与 分布式理论'
description: '以分布式锁为例子，简单分析一下etcd的用法和原理。'
pubDate: '2021-01-16'
updateDate: '2021-01-16'
tags: ['中间件', '技术分享会', '源码']
---

注意这里有个巨坑，在官方的release下面的Demo中写的`put`是`v3`的API！！而`v2`的API是`set`！！

这官方镜像里的`etcdctl`居然默认用`v2`版本的API你敢信？？而且`v2`设置的key-value，跟`v3`是在不同的空间的，你敢信？？（虽然好像有道理，但是我感觉不能原谅:(

```shell
$ ETCDCTL_API=3 etcdctl get some_key
some_key
some_value2

$ etcdctl get some_key
aa
```

所以我们接下来要主动设定一个环境变量，或者在使用SDK时只用同一个版本的。

## Go SDK

```shell
go get go.etcd.io/etcd/clientv3@release-3.4
```

> 注意，在[Client官方文档](https://github.com/etcd-io/etcd/tree/master/client/v3) 中写的安装命令是错的，最后在[issue-12484](https://github.com/etcd-io/etcd/issues/12484#issuecomment-740192667) 中找到上述临时解决方案。

虽然它的文档做的实在是漏洞百出，但是基本的Demo还是有的，我们直接去release的源码里去找Demo，例如 [release-3.4/clientv3](https://github.com/etcd-io/etcd/tree/release-3.4/clientv3)

### Key-Value 用法示例

- [example_kv_test](https://github.com/etcd-io/etcd/blob/release-3.4/clientv3/example_kv_test.go)

抄一段最简单的例子吧：

```go
func main() {
    cli, err := clientv3.New(clientv3.Config{
        Endpoints:   []string{"10.0.6.239:12379", "10.0.6.239:22379", "10.0.6.239:32379"},
        DialTimeout: 2 * time.Second,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer cli.Close()

    // 设一个值
    _, err = cli.Put(context.TODO(), "some_key", "some_value_20210116")
    if err != nil {
        log.Fatal(err)
    }

    // 取一个值
    resp, err := cli.Get(context.Background(), "some_key")
    if err != nil {
        log.Fatal(err)
    }
    for _, ev := range resp.Kvs {
        fmt.Printf("%s : %s\n", ev.Key, ev.Value)
    }
    // 输出: some_key : some_value_20210116
}
```

这里比较反直觉的是，Get 命令居然返回的是一个列表。原因是etcd并没有单独的Get接口，它只有一个叫`Range`的接口，因为可以提供很多选项并且进行前缀查询，所以才返回一个`repeated`类型。

### lease 用法示例

- [example_lease_test](https://github.com/etcd-io/etcd/blob/release-3.4/clientv3/example_lease_test.go)

### lock 用法示例

```go
func main() {
    // cli := ...

    session, _ := concurrency.NewSession(cli)
    defer session.Close()

    lk := concurrency.NewMutex(session, "/locked/resource/1")

    fmt.Println("排队取锁")
    ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
    if err := lk.Lock(ctx); err != nil {
        log.Fatal(err)
    }
    fmt.Println("带锁工作ing……")
    time.Sleep(10 * time.Second)
    if err := lk.Unlock(context.Background()); err != nil {
        log.Fatal(err)
    }
    fmt.Println("释放锁")
}
```

## Go SDK Lock源码剖析

### 准备：基本概念

鸣谢： [基于 etcd 实现分布式锁 - 徐亚松](https://segmentfault.com/a/1190000021603215) 。我在阅读`clientv3`源码时，一开始对很多概念不了解，导致源码也无法理解。在参考这篇文章后有了大体的思路，再结合源码去理解就很容易了。

所以先解释几个概念：

- `Lease 租约`：可以理解为是Redis中的带过期时间的键，或者理解为是一种 session 。在etcd中，可以给一个KV绑定一个Lease，当Lease到期时KV也被删除。Lease可以续期（KeepAlive）。
- `Revision 版本号`：重要！每个KV都有一个全局唯一的版本号！etcd的每次**事务**都会让版本号+1 。
- `Prefix 前缀`：可以把前缀当目录用。get一个前缀的时候，可以取出下面的所有子键。

我们用前面的 Golang代码 来获取、释放锁，然后用`etcdctl`观察一下对应的 KV 的情况：

```shell
# 1. 获取锁之前，revision==114，没有子键
$ etcdctl get /locks/resource/1 --prefix -w json
{"header":{"cluster_id":9946116703747541375,"member_id":6533212518309537241,"revision":114,"raft_term":278}}

# 2. 执行Golang代码，获取锁并占住，此时revision==115了，并且增加了一个新的key
$ etcdctl get /locks/resource/1 --prefix -w json
{
    "header":{"cluster_id":9946116703747541375,"member_id":6533212518309537241,"revision":115,"raft_term":278},
    "kvs":[{"key":"L2xvY2tzL3Jlc291cmNlLzEvMjVkOTc3MGFlZjZkZjlmMg==","create_revision":115,"mod_revision":115,"version":1,"lease":2727341938199886322}],
    "count":1
}
$ etcdctl get /locks/resource/1 --prefix
/locks/resource/1/25d9770aef6df9f2

# 3. 此时还可以观察到有一个lease被创建了，并且与上面那个KV中的lease对应
$ etcdctl lease list
found 1 leases
25d9770aef6df9f2

# 4. Golang代码把锁释放了，此时revision==116了，并且子键被删除
$ etcdctl get /locks/resource/1 --prefix -w json
{"header":{"cluster_id":9946116703747541375,"member_id":6533212518309537241,"revision":116,"raft_term":278}}
```

### 步骤一：创建租约

根据上面的介绍，Lease 是有过期时间的，并且是用来绑定在KV上的。那么在SDK中是如何实现的呢：

```go
func main() {
    // ...
    session, _ := concurrency.NewSession(cli)
    defer session.Close()
    // ...
}
```

就是上面这两行代码，在SDK中，把Lease转化为了session的概念，更接近传统数据库的术语，也许让人更好理解吧（并不）。

session的工作很简单，创建时获取一个租约并且维持，结束时释放租约。

```go
// 下面代码经过简化
func NewSession(client *v3.Client, opts ...SessionOption) (*Session, error) {
    // Grant 获取租约
    resp, _ := client.Grant(ops.ctx, int64(ops.ttl))
    id = resp.ID

    // 启动一个Go程不断地续约
    keepAlive, _ := client.KeepAlive(ctx, id)
    go func() {
        for range keepAlive {  // 它是一个 <-chan
        }
    }()

    return s, nil
}
```

### 步骤二：获取锁

这个锁呢，实质上是一个 `key="{prefix}{UUID}" value=""` 的一个键值对罢了，例如第一个客户端创建`/locks/resource/1/UUID1`，第二个客户端创建`/locks/resource/1/UUID2`。

我们依然用上面的Golang获取锁的代码来看一下，这次我们同时启动两个示例来抢这把锁：

```shell
# 此时一个进程获取到锁，另一个进程在等待
$ etcdctl get /locks/resource/1 --prefix -w json
{
    "header":{"cluster_id":9946116703747541375,"member_id":6533212518309537241,"revision":118,"raft_term":278},
    "kvs":[
        {"key":"L2xvY2tzL3Jlc291cmNlLzEvMjVkOTc3MGFlZjZkZjlmOA==","create_revision":118,"mod_revision":118,"version":1,"lease":2727341938199886328},
        {"key":"L2xvY2tzL3Jlc291cmNlLzEvNmZmMTc3MGFmMTA1ZDg1MQ==","create_revision":117,"mod_revision":117,"version":1,"lease":8066359296474339409}
    ],
    "count":2
}
```

在上面的状态下，`/locks/resource/1`这个前缀下面有两个子键，它们分别是两个Golang进程对应的“锁”（并且分别对应着不同的Lease）。

它们最关键的区别是`create_revision`不同。因此每次要获取锁的流程：

1. 申请Lease
2. 用Lease创建一个KV
3. 检查"我"创建的KV是否是**当前前缀下最小的revision**
4. 如果是，则说明"我"拿到了锁，可以执行被锁住的代码了；
5. 如果不是，则表明锁在"别人"那里，那就watch "前面一个人的"KV

提醒：在下面的代码中，`Op`代表一次"数据库操作"，`Txn`代表一个"事务"。一个`Txn`可以装进很多`Op`来形成原子性操作。（从下面代码也可以看出，etcd的事务具有足够的自定义能力和表达能力，而且与Redis+lua相比有更好的原生支持）

```go
// 下面代码经过简化
func (m *Mutex) Lock(ctx context.Context) error {
    // 1. 此时session已经申请了租约，用租约ID作为UUID拼接一个key
    m.myKey = fmt.Sprintf("%s%x", m.pfx, s.Lease())

    // 2. 制作一系列的Op作为事务提交
    // 2.1 检查自己的UUID之前有没有用过
    cmp := v3.Compare(v3.CreateRevision(m.myKey), "=", 0)
    // 2.2 正常情况是没有用过的，那就put创建这个KV
    put := v3.OpPut(m.myKey, "", v3.WithLease(s.Lease()))
    // 2.3 如果创建过，就取回来
    get := v3.OpGet(m.myKey)
    // 2.4 检查当前**前缀**下的revision最小的那个KV
    getOwner := v3.OpGet(m.pfx, v3.WithFirstCreate()...)
    // 2.5 组装事务并提交
    resp, _ := client.Txn(ctx).If(cmp).Then(put, getOwner).Else(get, getOwner).Commit()

    // 3. 检查revision
    // 3.1 如果我就是最小的，那说明我拿到了锁，返回
    ownerKey := resp.Responses[1].GetResponseRange().Kvs
    if len(ownerKey) == 0 || ownerKey[0].CreateRevision == resp.Header.Revision {
        return nil
    }
    // 3.2 如果我不是最小的，那就阻塞监听等待我上一个revision删除的事件（上一个删除了说明轮到我了）
    hdr, _ := waitDeletes(ctx, client, m.pfx, m.myRev-1)

    return werr
}
```

![Lock返回值截图](../pic/2021/Snipaste_2021-01-17_17-06-13.png)

这样梳理之后，是不是很简单？

那么总结一下etcd的实现与Redis+lua的实现有什么异同点？

- 都用了一个UUID来证明是自己。
- 都用了一个带有过期时间的KV。
- etcd的事务，与Redis的lua脚本是相似的。
- etcd可以watch等待回调，并且无缝衔接；而不用像Redis一样傻傻地轮询。
- etcd中所有竞争者都注册了，也就是可以知道其他竞争者的情况；Redis不行。

所以，虽然为了一致性而牺牲了性能，但是显而易见etcd的功能比Redis强大太多。前者是一个吸取了众多经验教训的现代化服务治理组件，而后者是一个单纯的缓存组件。

### 步骤三：释放锁

```go
func (m *Mutex) Unlock(ctx context.Context) error {
    client := m.s.Client()
    if _, err := client.Delete(ctx, m.myKey); err != nil {
        return err
    }
    m.myKey = "\x00"
    m.myRev = -1
    return nil
}
```

太简单了，一个Delete命令删除自己的KV就可以了。

### 步骤四：释放租约

```go
// 下面代码经过简化
func (s *Session) Close() error {
    // Revoke 释放租约
    _, err := s.client.Revoke(ctx, s.id)
    return err
}
```

太简单了，一个Revoke命令就搞定了。

## Python SDK

项目地址 [kragniz/python-etcd3](https://github.com/kragniz/python-etcd3)

```python
import etcd3
import time

if __name__ == '__main__':
    client = etcd3.client(host="10.0.6.239", port=12379)
    v, _ = client.get("some_key")
    print(v)  # b'some_value_20210116'

    with client.lock("resource/1") as lock:
        print(lock.key)
        time.sleep(10)
```

这里面有坑：

- 只能指定一个server
- 会给key加一个前缀"/locks/"，不可修改。
- 不给key加UUID后缀，而是把UUID作为value存进去。
- KeepAlive 的逻辑我没看，但我估计够呛，恐怕实现不了。
- 没有静态类型，太难写了，难道要对着gRPC来撸吗？

简而言之：「大人，时代变了。」

## gRPC 直连

etcdv3 最为强大的另一个优势是，它通过 gRPC 来提供服务。这不仅仅提升了性能，同时也保证了跨语言的兼容性。

下面我来直接使用 gRPC 裸连 etcd cluster，做一个查询操作：

```go
package main

import (
    "context"
    pb "go.etcd.io/etcd/etcdserver/etcdserverpb"
    "google.golang.org/grpc"
    "log"
    "time"
)

func main() {
    conn, err := grpc.Dial("10.0.6.239:12379", grpc.WithInsecure())
    if err != nil {
        log.Fatal("连接失败: ", err)
    }
    defer conn.Close()

    client := pb.NewKVClient(conn)
    req := &pb.RangeRequest{
        Key: []byte("some_key"),
    }
    ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
    resp, err := client.Range(ctx, req)
    if err != nil {
        log.Fatalln(err)
    }
    for _, kv := range resp.Kvs {
        log.Println(string(kv.Key), string(kv.Value))
    }
}

// 输出:
// 2021/01/17 19:07:16 some_key some_value_20210116
```

我写这段代码大概用了3分钟，并且一次性调通。静态语言的优势完全体现，对着proto文件、跟着IDE提示 瞎几把写，就能写出正确的代码，再也不用求爷爷(Google)告奶奶(Stackoverflow)了！

## 服务注册与发现

除了锁之外，当然也可以做这类事情。

我大概想象了一下，无非就是利用Lease机制，当一个服务上线时去拿一个Lease然后Put一个KV，然后一直KeepAlive呗。抛砖引玉，我想象中的机制应该是：

- 服务上线时，把自己注册到某个KV服务中
  - 注册是有时效的，在etcd中可以通过Lease实现
  - 服务自己不断地KeepAlive
- 客户端从KV服务中获取可用的服务端列表
  - 在etcd中可以通过前缀查询来实现
  - 客户端可以同时连接多个服务端，用RoundRobin等方式（负载均衡）提高可用性
- 服务下线时，主动把自己在KV服务中的注册信息清除
  - 意外下线（宕机），注册信息可能还会残留一个Lease的时间，到期自动清除
  - 可以有专门的HealthCheck服务去缩短这个残留时间

去Google简单了解了一下，再想想，感觉好像也没有两句话那么简单。

今天写累了，下周再说吧（咕咕咕），散会。

## 附录：推荐阅读

- [在微服务架构中是否可用 redis 代替 etcd? - v2ex](https://www.v2ex.com/t/520367)
- [Etcd 最佳实（踩）践（坑） - pandaychen](https://pandaychen.github.io/2019/10/20/ETCD-BEST-PRACTISE/)
- [三年之久的 etcd3 数据不一致 bug 分析 - 王超凡](https://mp.weixin.qq.com/s/VJi1jzTK2G7bH1pi4ND7Yw)

## 总结

整整两天的学习成果就是这么一篇文章了。收获很大很大。

首先是 CAP理论，让我对分布式系统有了新的更清晰的认识。

然后对我触动最大的是上面参考文章的最后一篇，大佬们在对etcd进行debug的时候，动不动就改源码自己定制，这是该夸大佬们艺高人胆大呢，还是该夸Go语言大道至简好读好写呢？反正我试着在脑海里把etcd换成了Redis想象了一下……嗯，我还是写我的Go吧 :)

最后还是要吐槽一下。etcd的文档真的太不健全了，刚开始的时候连Demo都跑不起来，真的很令人暴躁。我就觉得吧，整个微服务体系，k8s, protobuf, grpc, etcd 它们的文档以及各种下载资源都有各种各样的问题，真的好难折腾。只能安慰自己，这已经是现在互联网领域最先进的技术了，在我接下来的学习过程中，缺少文档恐怕将会是常态。要冷静、冷静、冷静……
