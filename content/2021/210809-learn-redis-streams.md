---
title: '[译] Redis Streams 介绍'
description: '以前一直说不要用Redis做消息队列，这回它专门为这个功能做了一套数据结构'
pubDate: '2021-08-09'
updateDate: '2021-08-09'
tags: ['中间件', '技术分享会']
---

译者注：这样看来确实省事，省很多代码。

第一个返回值是一个ID，代表着扫描的起始点。这里可能会有点奇怪的是，当返回`0-0`的时候，表示的是游标走到了列表底部，然后又重新从头开始走了，并不是走到底就停止了（因此可能会返回比传入的startID更小的消息）。

```text
> XAUTOCLAIM mystream mygroup Lora 3600000 1526569498055-0 COUNT 1
1) 0-0
2) 1) 1526569506935-0
   2) 1) "message"
      2) "strawberry"
```

## 投递次数

通过 XPENDING 查得的消息的投递次数（译者注：此前翻译为读取次数），会在 XCLAIM 或 XREADGROUP 命令时递增。

所以，如果消费者中出现了异常，那么这个数字会超过1，这个是很普遍的。

同时，在一些特殊情况下，例如消息本身会引发消费者的BUG，那么这条消息可能会被反复的重新投递。这样，这个消息的投递次数就可能会变得很大。一个明智的策略，是当投递次数达到一定值的时候，将它转移到一个专门的处理队列中去，并且通知系统管理员。

同时，这也基本上是 Redis Streams 实现 「死信`dead letter`」的方法。

## Streams的可观测属性

前面我们介绍了 XPENDING 可以用来监测消息。

然后我们还提供了 XINFO 命令来监测 Stream 和 Group 。

```text
> XINFO STREAM mystream
 1) length
 2) (integer) 13
 3) radix-tree-keys
 4) (integer) 1
 5) radix-tree-nodes
 6) (integer) 2
 7) groups
 8) (integer) 2
 9) first-entry
10) 1) 1526569495631-0
    2) 1) "message"
       2) "apple"
11) last-entry
12) 1) 1526569544280-0
    2) 1) "message"
       2) "banana"
```

```go
func learnXInfoStream(rdb *redis.Client) {
	res, err := rdb.XInfoStream(ctx, "lewin_001").Result()
	// &{4 1 2 1 1628599302664-0 {1628599302658-0 map[title:lewin_XADD ts:1628599302486495000]} {1628599302664-0 map[title:lewin_XADD ts:1628599302497030000]}}
	// 其实这里的输出，多一个LastGeneratedID
}
```

```text
> XINFO GROUPS mystream
1) 1) name
   2) "mygroup"
   3) consumers
   4) (integer) 2
   5) pending
   6) (integer) 2
   7) last-delivered-id
   8) "1588152489012-0"
```

```go
func learnXInfoGroups(rdb *redis.Client) {
	res, err := rdb.XInfoGroups(ctx, "lewin_001").Result()
	// [{lewin_001_g1 3 2 1628599302660-0}]
}
type XInfoGroup struct {
	Name            string
	Consumers       int64
	Pending         int64
	LastDeliveredID string
}
```

## 与Kafka的区别

（略）

## 数量限制

可以限制一个Stream的最大长度。

不过稍微有点诡异的是，并不是对Stream本身做限制，而是在执行 XADD 命令的时候做限制。添加 MAXLEN 参数，会将最老的消息丢掉。

```text
> XADD mystream MAXLEN 2 * value 1
1526654998691-0
> XADD mystream MAXLEN 2 * value 2
1526654999635-0
> XADD mystream MAXLEN 2 * value 3
1526655000369-0
> XLEN mystream
(integer) 2
> XRANGE mystream - +
1) 1) 1526654999635-0
   2) 1) "value"
      2) "2"
2) 1) 1526655000369-0
   2) 1) "value"
      2) "3"
```

但是，要知道，Stream是以一种树形结构储存的，因此要在一个巨大的树上删掉某一个节点，可能会造成一定的阻塞。所以，不建议每次插入的时候都添加 MAXLEN 参数。

另一种偷懒的思路，是定时清理。这也会造成一些问题，最好也别用。

我们的解决方案是，参数可以增加一个`~`符号，意思是，我并不要求Stream的长度精确地小于某个值，例如1000，可以稍微大一些，1010，1030，但是会保证最少都有1000。通过这种形式，Stream可以自己选择合适的时机来删除节点。这也应该是我们所需要的特性。

```text
XADD mystream MAXLEN ~ 1000 * ... ...
```

我们还提供一个专门的修剪命令 XTRIM ，用法同上：

```text
> XTRIM mystream MAXLEN 10
> XTRIM mystream MAXLEN ~ 10
```

同时 XTRIM 还可以实现另一种修剪策略。我们可以指定 MINID ，这样就会删掉所有小于这个ID的消息。

开发人员应当知晓不同策略的优缺点，并选择合适的。以后可能还会加入其它修剪策略。

## 特殊ID

`-` `+` `0` `$` `*` `~` 这几个符号，前面都介绍过了。不要搞混哦。

## 持久化、集群、消息安全性

Stream 和其他 Redis数据结构 一样，都是通过 AOF或者RDB 的形式进行异步的复制和持久化。

值得一提的是，Group 的状态也同样会被保存。（放心吧）

但是开发人员也要知道，AOF和RDB都是有缺陷的，注意：

- 如果是AOF，而且你的消息还很重要，那一定要选择强力的同步策略。
- 因为是异步保存，所以可能会丢失掉一些东西。
- 也许你可以考虑使用 WAIT 命令，来强制等待数据传播到Redis从库上。但是这并不是万能的，在哨兵或者集群模式的故障恢复过程中，只能做到「尽可能的`best effort`」恢复，因此某些情况下选拔为主库的从库可能仍然缺失某些数据。

## 从中间删掉一个消息

Stream也支持！虽然对于一个 append-only 的数据结构来说，这个行为非常奇怪，但有时确实会有用。

```text
> XRANGE mystream - + COUNT 2
1) 1) 1526654999635-0
   2) 1) "value"
      2) "2"
2) 1) 1526655000369-0
   2) 1) "value"
      2) "3"
> XDEL mystream 1526654999635-0
(integer) 1
> XRANGE mystream - + COUNT 2
1) 1) 1526655000369-0
   2) 1) "value"
      2) "3"
```

但是，（由于树的特性），内存可能并没有被立即回收，可能要等到上方的大节点清空之后才会。所以，不要滥用这个功能。

## 空Stream

与其他数据结构不同的是，当其他数据结构被清空到0长度的时候，它本身的键也会被删除。而Stream允许被保留为0长度。

原因是 Stream 是会被 Group 绑定的，我们不希望Group中的数据会因为Stream被清空而被清理掉。

目前，即使Stream没有绑定的Group也不会被清理。不过这个也许以后会改变。

## 性能：消费消息的延迟

范围查询的效率，不弱于ZSET（有序集合）。

插入效率极高，在平均性能的机器上借助pipeline可以达到每秒一百万条的插入效率。

但是当我们考虑，一条消息从 XADD 开始，直到被 XREADGROUP 投递到消费者的过程中，性能究竟如何？

## 是如何阻塞消费者的？

- 被阻塞的客户端，引用是存放在hash表里的一个列表里。因此，对于一个特定的key，我们可以知道所有在等待它的客户端。
- 当写入事件（XADD）发生时，目标key会被放入一个 「就绪列表`ready keys`」 中，稍后再处理。这里注意，由于事件循环的特性，可能在它被处理之前，还有其他的写入事件发生。
- 在返回事件循环之前，才会处理 就绪列表 。此时会扫描所有在等待的客户端，符合条件的客户端就会收到新到来的消息。在Stream中，就是阻塞的消费者会收到消息。

所以，生产者发布的消息的响应，几乎会与 消费者收到的消息 ，同时到达。

这种模型叫做「基于推的`push based`」，因此延迟会非常的可预测。

## 测试数据

10k 压力下 99.9% 的请求延迟 <= 2毫秒。

测试场景比较保守。在真实场景下可能会更快。

## 译者小结

总体来说，文档非常详尽，基本上是入门级别的。即使我在翻译的时候已经做了大量的简略，可依然写了这么这么长。

把各个主要命令，以及数据结构的特性，都说清楚了。非常的棒。

不过目前总体来说，用 Redis Streams 来做消息队列，存在一些问题：

1. 没有自动重新投递，要主动去检查并且夺取。
2. 命令很简单直接，但是配套库不够 high-level 。

但基本上算不上大问题，都是多自己写点代码就可以解决的事情，不算很麻烦。足够满足我们目前项目的需要。

还有一点就是Redis的持久化稍微有些不够强健，对于重要数据来说，还是别用它比较好。不过这个我们并不关心，哈~

总体体验不错，至少比RocketMQ强太多太多了。
