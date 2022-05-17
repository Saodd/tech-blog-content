```yaml lw-blog-meta
title: "Redis知识点总结"
date: "2021-05-17"
brev: "整理一遍之后，确实有不少收获，八股不仅仅是八股"
tags: ["中间件"]
```

## 背景

虽然我干着全栈的工作，但是终究是要选择一个领域作为侧重点，才能收获最高的“投产比”。所以，毕竟我也是后端出身，终究还是回到后端去深耕吧。

那么，作为一个后端程序员，redis是肯定逃不了的。之前一直给自己找借口“公司的业务就是不够复杂，就是用不到那么多特性呀”，这种鸵鸟式的行为还是就此而止吧。

学！

## Redis的AP模型

之前在[讲etcd](../2021/210116-etcd-guide.md) 的时候介绍过CAP理论，Redis实现的是其中的AP，也就是只保证可用性而放弃了一致性。

AP与CP的关键区别在于：Redis在执行更新之后，会立即响应客户端，而不会等待确认其他节点（包括cluster, slave, 硬盘等角色）同步成功，因此可能产生不一致的现象（包括数据丢失）。

## Redis数据结构

（[参考](http://redisdoc.com/sorted_set/index.html) ）主要数据结构有：字符串、哈希表、列表、集合、有序集合，HyperLogLog，Bitmap

1. 字符串：简单的键值储存
2. 哈希表：把一批键值存放在一起，起一个隔离作用。用途：可以存放对象
3. 列表（list）：实际上是双向链表而不是数组，支持任意位置插入弹出。
4. 集合：略
5. 有序集合（sorted set）：最然叫做集合但是每个键都是有值的（形似哈希表），但是键会根据值的大小进行排序，底层实现可能是"跳表"。用途：排行榜
6. HyperLogLog：用于估计数量的集合，只能估计数量而不能查询精确的值，特点是牺牲了精度节省了空间。用途：巨量数据的统计，例如网站日活
7. Bitmap：用0/1二进制数据，比普通的集合更加高效。用途：中等量级的数量统计
8. Stream：新的数据结构，专门用来做队列

关于各个数据结构的底层结构，可以提一下：

1. 哈希表用的是拉链法（即冲突的值串在一起），读写复杂度是O(1)
2. List是双向链表（链表+数组结合以提升性能[ziplist](https://redis.com/ebook/part-2-core-concepts/01chapter-9-reducing-memory-use/9-1-short-structures/9-1-1-the-ziplist-representation/) ），查询复杂度O(n)，但一般用法在头尾增删是O(1)
3. 有序集合是跳表，即在普通链表的基础上增加logN级别的索引以达到logN的复杂度，类似算法中的分治法的思路

## Redis持久化

参考：[Redis 持久化详解及配置](https://zhuanlan.zhihu.com/p/98497789)

RDB模式：即全量备份。当满足条件时，redis会fork一个子进程，把数据库中的内容全部保存并替换本地文件。下次启动时会从文件中恢复数据。

参考启动命令：`redis-server --save 15 1 --loglevel debug`，意思是每15秒至少1次写入即保存一次。

AOF模式：即增量备份，写入动作会增量写入磁盘，写入是异步的，可以配置周期或者改成同步。下次启动时会从文件中恢复。

参考启动命令：`redis-server --appendonly yes --appendfsync everysec`

## Redis主从复制

为什么需要主从复制？

1. 提升可靠性，一个节点挂了还能用其他节点
2. 提升并发性能，读写分离

### 主从复制的启动方式

这里启动三个Redis实例，名称以及主从关系为：redis1 -> redis2 -> redis3，分别映射到本机的30001, 30002, 30003三个端口上，参考启动命令：

```shell
docker run --rm --name redis1 -p 30001:6379 -it redis redis-server
docker run --rm --name redis2 -p 30002:6379 -it redis redis-server --slaveof 10.0.6.239 30001
docker run --rm --name redis3 -p 30003:6379 -it redis redis-server --slaveof 10.0.6.239 30002
```

slave刚刚连接master的时候，会执行一次全量复制(`psync -1`)，之后会维护一个`offset`的值，后续只做基于偏移量的部分同步(`psync [offset]`)。主从之间还会发送`PING`和`ACK`这类消息。 参考：[深入学习Redis（3）：主从复制](https://www.cnblogs.com/kismetv/p/9236731.html)

## Redis哨兵模式

核心原理：Redis本身只有主从复制，没有在宕机情况下的主从切换机制。所以要依赖一套哨兵`sentinel`集群来做这个监控和切换的动作。

注意：`sentinel`实质上也是一个redis，是一个特殊的redis实例。

所以，一套典型的高可用Redis集群包括：三个Redis实例（一主两从），和三个Sentinel实例（地位均等）。

### Redis-Sentinel启动方式

由于一些坑（后面我会介绍），这里我把6个实例全部运行在同一个docker容器里：

```shell
docker run --rm --name rrrrr111 -v /xxx/sentinel.conf:/sentinel.conf -it redis bash

> cp /sentinel.conf /sentinel1.conf
> cp /sentinel.conf /sentinel2.conf
> cp /sentinel.conf /sentinel3.conf
```

> 注意看上面的命令，我把一个`sentinel.conf`文件挂载进去，然后`cp`了三份，原因是`sentinel`实例启动的时候，会直接读写这个配置文件，因此它不能直接复用同一个文件，必须拷贝三份（三份文件初始内容相同）！！

然后我们用`exec`命令在这个容器里启动三个redis实例，一主二从：

```shell
docker exec -it rrrrr111 redis-server --port 6001
docker exec -it rrrrr111 redis-server --port 6002 --slaveof 127.0.0.1 6001
docker exec -it rrrrr111 redis-server --port 6003 --slaveof 127.0.0.1 6001
```

> 注意看上面的命令，redis实例之间全部通过`127.0.0.1`进行本地通信，这个是由于`sentinel`后面需要直接与各个redis进行通信，因此如果在不同的docker里，会得到错误的ip，因此用`127.0.0.1`是最稳妥的办法。（至于在生产中，肯定要将这六个实例分别启动在至少三台机器上，至于如何配置ip，我暂时忽略，等用到时再说吧）

接着启动三个sentinel实例：

```shell
docker exec -it rrrrr111 redis-sentinel /sentinel1.conf --port 7001
docker exec -it rrrrr111 redis-sentinel /sentinel2.conf --port 7002
docker exec -it rrrrr111 redis-sentinel /sentinel3.conf --port 7003
```

最后再提一下，可以通过`redis-cli`来查看内部的数据运行状况：

```shell
docker exec -it rrrrr111 redis-cli -p 6001  # 连接master
docker exec -it rrrrr111 redis-cli -p 7001  # 连接第一个sentinel
```

### 如果杀掉master

最先有反应的是两个`slave`（因为它们的主从复制连接断开了），他俩会输出这样的日志：

```text
79:S 16 May 2022 11:09:17.513 # Connection with master lost.
79:S 16 May 2022 11:09:17.514 * Caching the disconnected master state.
79:S 16 May 2022 11:09:18.155 * Connecting to MASTER 127.0.0.1:6001
79:S 16 May 2022 11:09:18.155 * MASTER <-> REPLICA sync started
79:S 16 May 2022 11:09:18.155 # Error condition on socket for SYNC: Connection refused
```

随后三个`sentinel`也会逐个随机地产生反应（默认配置是10秒检测一次，要等到10秒的时间点到达），它们会输出这样的日志：

```text
16:X 16 May 2022 11:09:22.656 # +sdown master rrrrr111 127.0.0.1 6001
16:X 16 May 2022 11:09:22.697 # +new-epoch 1
16:X 16 May 2022 11:09:22.708 # +vote-for-leader fd0f529f7a34c44ee8da9a8ca9e3096f4e278e00 1
16:X 16 May 2022 11:09:22.715 # +odown master rrrrr111 127.0.0.1 6001 #quorum 3/2
16:X 16 May 2022 11:09:22.715 # Next failover delay: I will not start a failover before Mon May 16 11:09:43 2022
16:X 16 May 2022 11:09:23.951 # +config-update-from sentinel fd0f529f7a34c44ee8da9a8ca9e3096f4e278e00 127.0.0.1 7002 @ rrrrr111 127.0.0.1 6001
16:X 16 May 2022 11:09:23.951 # +switch-master rrrrr111 127.0.0.1 6001 127.0.0.1 6002
16:X 16 May 2022 11:09:23.951 * +slave slave 127.0.0.1:6003 127.0.0.1 6003 @ rrrrr111 127.0.0.1 6002
16:X 16 May 2022 11:09:23.952 * +slave slave 127.0.0.1:6001 127.0.0.1 6001 @ rrrrr111 127.0.0.1 6002
```

上面日志的意思是：

- 失去与master(6001)的连接（此时是"主观下线"，即还不确定是否真的下线了）
- 当超过半数的哨兵认为主观下线后，就认为是"客观下线"
- 哨兵们投票选出一个`leader`，最后选定了是端口号为7002这个实例（此时从7002的日志里可以看到"成功当选"的日志）
- 7002带头选拔一个`slave`，经过两轮墨迹，最后选定了6002
- 通知6002转换为`master`

此时可以看到6002的日志显示转换为master的标志：

```text
79:M 16 May 2022 11:09:22.939 * MASTER MODE enabled (括号内省略......)
```

接着可以用`redis-cli`测试一下，确实可以在新的master:6002节点上做写入动作了。

此时如果复活6001，它会先以`master`身份启动（因为启动命令中没有`--slaveof`），然后`sentinel`发现了它并要求它转换为`slave`，然后6001就成为了6002的`slave`。

### 客户端如何确定master

大概是这样的过程：

- 客户端从预设的哨兵地址列表中（在本例中是7001,7002,7003）轮询，找到一个可用的哨兵节点
- 向哨兵询问当前的`master`，并向这个master节点进行确认
- 订阅(`sub`)哨兵的某个`Pub`，这样当主从切换的时候可以第一时间得知

## Redis集群模式

与主从相对的模式，就是横向拓展，即分片。术语是`cluster`。

这里有个概念叫做『槽`slot`』，cluster中的的key会经过哈希然后对16384取模，意思是将所有key划分在16384个区域（槽）内了，每个master实例分摊其中一部分槽。

### Redis-Cluster启动方式

一个基本配置是：3主3从一共6个节点，数据集被分为3个部分分别储存，每个部分有1主1从保证高可用。

与哨兵模式稍有不同，所有6个节点都是以相同的方式启动：（[参考](https://segmentfault.com/a/1190000022808576) ）

```shell
docker run --rm --name rrrrr -it redis bash
docker exec -it rrrrr redis-server --port 7001 --cluster-enabled yes --cluster-config-file /nodes_7001.conf
docker exec -it rrrrr redis-server --port 7002 --cluster-enabled yes --cluster-config-file /nodes_7002.conf
docker exec -it rrrrr redis-server --port 7003 --cluster-enabled yes --cluster-config-file /nodes_7003.conf
docker exec -it rrrrr redis-server --port 7004 --cluster-enabled yes --cluster-config-file /nodes_7004.conf
docker exec -it rrrrr redis-server --port 7005 --cluster-enabled yes --cluster-config-file /nodes_7005.conf
docker exec -it rrrrr redis-server --port 7006 --cluster-enabled yes --cluster-config-file /nodes_7006.conf
```

通过上面的命令，在同一个容器内的7001~7006端口上分别启动了redis服务，每个服务使用一个单独的配置文件（为了保证快速故障恢复）

然后有趣的是，此时它们互相之间还没有联系，接下来要通过`redis-cli`去将它们组织成cluster，命令：

```shell
redis-cli --cluster create 127.0.0.1:7001 127.0.0.1:7002 127.0.0.1:7003 127.0.0.1:7004 127.0.0.1:7005 127.0.0.1:7006 --cluster-replicas 1
```

上面的命令，分别罗列了6个实例的ip+端口地址，最后指定的`--cluster-replicas 1`意思是每个master配一个slave

接下来日志会输出一些内容，意思是计划将7001,7002,7003作为master，7004,7005,7006作为slave，每个master划分的槽的范围，并询问用户是否确认。

确认后，6个redis实例在cli的指挥下变成了3主3从并且互相关联。

> 所以，Sentinel在启动的时候是需要指定master的，而且client也需要知道sentinel的地址；而Cluster在启动时不需要关心其他节点，启动之后再由redis-cli组建集群，client那边连接任意一个redis节点都可。

### Cluster模式下的一些操作

首先，使用cli连接redis实例的时候，要指定`-c`来启动cluster模式：

```text
redis-cli -p 7001 -c
```

然后做基本的读写操作：

```text
127.0.0.1:7001> set 1 1111
-> Redirected to slot [9842] located at 127.0.0.1:7002
OK
127.0.0.1:7002> get 1
"1111"
```

在上面的命令中，`1`这个key被划归到`9842`这个`slot`中，然后依据集群的分配规则，应当指向7002节点，因此`redis-cli`自动向7002节点建立了连接，随后进行读写。

然后我们试着杀死7002节点。（注意，此时7002是master，它的slave是7005）

7002死后，7005的主从复制连接立刻发现它死了，因此7005不断轮询重连，并输出重连失败的日志。

当达到某个时间阈值（默认20秒？）后，7005会自动升级为master。此时如果再对集群进行读写操作，会自动重定向到7005节点上。

随后重启7002，它启动后会自动切换为7005的slave 。

> 也就是说，cluster模式附带了哨兵模式的高可用能力。

## Redis作队列

看到 [这篇文章](https://www.51cto.com/article/659208.html) 讲得不错，大家可以参考一下。主要四个方面：

1. 用List实现队列
2. 用Pub/Sub实现队列
3. 用最新的Stream
4. 其他正规的队列中间件

用`List`实现应该是最简单的实现了，常见的队列框架`Celery`或者`dramatiq`都使用它作为底层依赖（不过通过配置也可以使用`Stream`）

它最大的缺点就是，消息读取是一个`pop`操作，会把原数据删除。这导致一不能重复消费，二不能正确处理消费者崩溃的情况。

在框架中的实现，是依赖了`lua`脚本在redis上做原子操作，给消费者维护一个临时队列，以此达到”追踪每个消费者当前消费状态“的目的。虽然能解决问题，这也导致这部分的架构实现比较复杂，基本不太可能自行debug，只能祈祷框架的实现质量够高……（这里点名批评`Celery`）

然后是`Pub/Sub`的实现。

它的优点：可以支持多重消费，并且原生支持话题路由。

但它的致命缺陷是，它没有底层的数据结构，它实质上只是一个"临时的消息转发路由"。这导致一消费者必须在线才能获得消息（历史消息丢弃），二是可用性太低（宕机即丢失所有数据），三在队列积压的时候更容易丢数据（超出缓冲区容量）

因此`Stream`作为一个全新的、专业做队列功能的数据结构而诞生了。它能覆盖我们对一个消息队列中间件的所有期待：

1. 可以重复消费，也可以集群轮流消费（通过`XGroup`实现）
2. 所有消息保留（类似链表的新的数据结构类型），且能够持久化（与其他数据一样的RDB或AOF）
3. 可靠的异常恢复机制（`ACK`命令，以及可以记录重试次数）
4. 容量上限高（可以定义最大长度，超过长度则丢弃）
5. 运维友好（多种监控命令，虽然不够完善，但勉强够用吧）
6. 甚至可以读写分离（但是写命令都要master节点，效率不算高）

OK，这样看起来，`Stream`似乎是完美了？

不，最后的隐患，存在于Redis本身的特性上：因为Redis是`AP模型`，所以它并不保证一致性，即不保证在所有异常情况下都能数据不丢失。（这一点不再重复讲了~）

总结：如果业务相对简单，并且允许偶尔消息丢失，用 Redis Stream 是个不错的选择。至于List和Pub/Sub就请让它们消逝在历史长河中吧。
