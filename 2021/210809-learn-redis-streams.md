```yaml lw-blog-meta
title: "[译] Redis Streams 介绍"
date: "2021-08-09"
brev: "以前一直说不要用Redis做消息队列，这回它专门为这个功能做了一套数据结构"
tags: ["中间件","技术分享会"]
```

## 背景

项目里要选用一个消息中间件。调研的是 RabbitMQ ，但是最后说，阿里云的Rabbit太贵了。

替代选项一，RocketMQ，好家伙，文档太烂了，玩了一下午愣是连发布消息的流程都走不通。

替代选项二，Redis Streams 。这玩意我一开始是抗拒的，因为Redis它就是个缓存中间件，不应该拿它来做消息队列。奈何竞争对手们都太垃圾了，硬着头皮上吧。

其实我个人心里头还有个阳春白雪 etcd ，但是这玩意肯定比RabbitMQ还更不受世俗待见，也就想一想罢了，提都不敢提。

本文翻译自 Redis官网的 [Introduction to Redis Streams](https://redis.io/topics/streams-intro) 并添加了自己的Demo (Golang版本) 。

在美登技术分享会上，我将所有Demo都重写为Node.ts版本了，用法其实很简单就是简单地把参数以字符串的形式罗列上去就行了，不再重复贴出。

## Introduction to Redis Streams

`Stream` 是一种新的数据类型，从Redis5.0 开始引入，它将 日志数据结构 以一种更加抽象的方式做成了模型。然而日志的本质依然不变：类似 日志文件 经常被实现为一个 append-only 模式的文件，Redis Streams 也大概是一个 append-only 的数据结构。至少在概念上，因为是一种存活于内存中的抽象数据结构，Redis Streams 实现了很多强大的操作来克服传统日志文件的不足。

它可以说是Redis中最复杂的数据结构，尽管作为数据结构它本身很简单，但是它得实现很多额外的特性：一个阻塞操作的集合，这允许消费者阻塞等待新的消息；和一个新的概念叫**消费者群组**。

消费者群组`Consumer Groups` 最初是由 Kafka 引入的。Redis用完全不同的方式实现了类似的理念，但是目标是完全一致的：允许一组客户端来协作地共同消费一个Stream中的消息。

> 译者注：这里的Stream对应于其他消息组件中的Queue。

## 基本概念

基本操作并没有很复杂。

因为Streams 是 append-only 的数据结构，因此最基础的写操作 `XADD` ，将一个消息`entry`追加到特定的stream中去。一个`entry`并不是简单的字符串，而是一个或多个key-value对。通过这种方式，stream中的每个消息都已经是结构化的，打个比喻，就像是csv格式的文件。

```text
> XADD mystream * sensor-id 1234 temperature 19.8
1518951480106-0
```

上面的命令调用了`XADD`，把一条数据`sensor-id: 1234, temperature: 19.8`推入了一个stream，这个stream的key叫`mystream`，然后这条消息有一个自动生成的ID也就是命令的返回值`1518951480106-0`。

也就是说，第一个参数是stream名称，第二个参数是ID，后续参数是键值对。

这条命令中的`*`是让服务器自动生成ID，它是自增的。自增ID应该就是你需要的，你不应该有自己指定ID的这种需求。

可以通过`XLEN`命令来查询stream中的消息数量：

```text
> XLEN mystream
(integer) 1
```

关于ID，它由两个部分组成：`毫秒时间戳-顺序号`。由于顺序号是int64，所以理论上一个毫秒内生成的消息数量是没有任何限制的。

也许屏幕前的你会奇怪，为啥要把时间戳作为ID的一部分？一个很自然的理由，这样做的话，就原生支持对消息队列的**时间范围查询**了，使用`XRANGE`命令！

你可以像这样自己指定ID：

```text
> XADD somestream 0-1 field value
0-1
> XADD somestream 0-2 foo bar
0-2
```

不过要记住，ID是自增的，新消息的ID必须大于队列中最后一个消息的ID。

## 练习-1

```go
func learnXADD(rdb *redis.Client) {
	res, err := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "lewin_001",
		Values: map[string]string{
			"title": "lewin_XADD",
			"ts":    time.Now().String(),
		},
	}).Result()
	if err != nil {
		log.Fatalln(err)
	} else {
		log.Println(res)  // 输出： 1628478239161-0
	}
}
```

```go
func learnXLEN(rdb *redis.Client) {
	res, err := rdb.XLen(ctx, "lewin_001").Result()
	if err != nil {
		log.Fatalln(err)
	} else {
		log.Println(res)  // 输出： 1
	}
}
```

## 获取消息

获取消息 会比 发布消息 更复杂得多。

不同于Redis其他的阻塞操作，例如BLPOP只会让一个数据分给一个客户端。

在Stream中，可能会有多种分配模型（译者注：消息队列的多种模型）。例如，`fan out`模式，会让所有客户端都收到同一条消息。

另一种模型，可以将Stream视作一种时间序列存储，我们可能需要按时间范围来查询，或者利用一个游标(cursor)来检查历史消息。

第三种模型，多个消费者共同消费一个Stream，每个消费者只看见其中的一部分消息，这样可以实现扩容。这也是Kafka的消费者群组的概念。

### 范围查询 XRANGE & XREVRANGE

范围查询的时候，我们需要指定两个参数，起始ID和终止ID，范围包括起止点。用`-`和`+`分别代表最小的和最大的ID。

```text
> XRANGE mystream - +
1) 1) 1518951480106-0
   2) 1) "sensor-id"
      2) "1234"
      3) "temperature"
      4) "19.8"
2) 1) 1518951482479-0
   2) 1) "sensor-id"
      2) "9999"
      3) "temperature"
      4) "18.2"
```

```go
func learnXRANGE(rdb *redis.Client) {
	res, err := rdb.XRange(ctx, "lewin_001", "-", "+").Result()
	if err != nil {
		log.Fatalln(err)
	} else {
		log.Println(res) // 输出：[{1628478239161-0 map[title:lewin_XADD ts:2021-08-09 11:03:58.963222 +0800 CST m=+0.000485610]}]
		log.Println(res[0].Values)  // 这个类型是 map[string]interface{}
	}
}
```

返回的每个消息，由两个部分组成：ID和键值对。

也可以自己指定时间范围，注意要用**UNIX毫秒**（13位）。这种情况下，可以省略ID中的"顺序号"的部分，仅指定时间也可以工作。

```text
> XRANGE mystream 1518951480106 1518951480107
1) 1) 1518951480106-0
   2) 1) "sensor-id"
      2) "1234"
      3) "temperature"
      4) "19.8"
```

实际场景中，数据量可能很大，为了安全起见，可以加一个COUNT参数来限制返回的数量：

```text
> XRANGE mystream - + COUNT 2
```

```go
// XRANGEN 增加了COUNT参数
func learnXRANGEN(rdb *redis.Client) {
	res, err := rdb.XRangeN(ctx, "lewin_001", "-", "+", 1).Result()
	log.Println(err)
	log.Println(res)
}
```

如果想要接着上一次 `COUNT 2` 的结果，继续查询两个，该怎么做？——我们需要把上次结果中的最后一个的ID作为下一次查询的起始ID，并且在左边加上小括号`(`：

```text
> XRANGE mystream (1519073279157-0 + COUNT 2
```

因为`XRANGE`的时间复杂度是`logN`，所以这种方式很快。所以XRANGE实际上也就是一种游标(cursor)了，不需要再增加一个XSCAN命令。

> 译者注：似乎是由于它的底层数据结构是跳表，所以能达到logN的复杂度。  
> 然后顺便学一句法语，"the de facto" 意思是「实际上的」，对标英语中的 "the fact" 。

`XREVRANGE`命令用法与上面一致，不过它是倒序的，也就是说常常用它来获取最后1条(n条)消息：

```text
> XREVRANGE mystream + - COUNT 1
```

注意！这个命令的 end参数 在 start参数 的前面！

## 监听新消息 XREAD

它与其他Redis数据结构有如下不同：

1. Stream可以有多个客户端（消费者），默认情况下会发送给每个客户端。这个默认行为与 阻塞列表不同，与 Pub/Sub 相同。
2. Pub/Sub 是一种 即用即弃`fire and forget` 的机制，从来不会储存；阻塞列表 ，是从列表中pop出来。而Stream不同，所有的消息都会添加到stream中永久保存（除非用户显式要求删除）。客户端可以通过记住上一个ID来判断下一个消息是否是新的。
3. Stream提供了更多队列方面的高级功能，接下来慢慢说。

先看最简单的使用方式：

```text
> XREAD COUNT 2 STREAMS mystream 0
1) 1) "mystream"
   2) 1) 1) 1519073278252-0
         2) 1) "foo"
            2) "value_1"
      2) 1) 1519073279157-0
         2) 1) "foo"
            2) "value_2"
```

```go
func learnXREAD(rdb *redis.Client) {
	res, err := rdb.XRead(ctx, &redis.XReadArgs{
		Streams: []string{"lewin_001", "0"},
		Count:   2,
	}).Result()
	log.Println(err)
	log.Println(res)
	// 输出：[{lewin_001 [{1628586038144-0 map[title:lewin_XADD ts:1628586038063701000]} {1628586038147-0 map[title:lewin_XADD ts:1628586038071655000]}]}]
}
```

这是一种**非阻塞**的调用方式，`STREAMS`参数是必须的，`COUNT`参数不是。`STREAMS`参数可以指定多个stream，每个分别指定key（名字）和起始ID（不含）（指定0则为不限制），这样，这条命令会从所有stream中获取ID大于指定值的消息。

在上面的命令中，我们指定了`STREAMS mystream 0`，所以我们会从这个stream中读取ID大于0-0的所有消息。（译者注：这里只能写`0`，不能写`-`）

返回值中包含了 stream key，因为这条命令可以同时读取多个stream 。如果要读取多个，写法是`STREAMS mystream otherstream 0 0`。所以`STREAM`参数必须放在最后一个（使用SDK时一般不用考虑）。

如果仅仅是这样，那 XREAD 跟 XRANGE 也没区别。重点来了，它支持阻塞调用，加一个`BLOCK`参数：

```text
> XREAD BLOCK 0 STREAMS mystream $
```

```go
// 只读取新消息。为了测试我们需要再启动一个进程（或者go程）去发布新消息
func learnXREAD2(rdb *redis.Client) {
	res, err := rdb.XRead(ctx, &redis.XReadArgs{
		Streams: []string{"lewin_001", "$"},
		Block: 0,
	}).Result()
	log.Println(err)
	log.Println(res)
}
```

上面的命令中，指定了`BLOCK 0`，意思是永远阻塞、没有超时限制。只要有大于等于1条消息符合要求，就会立即返回。

然后最后一个ID参数`$`，意思是最大的ID，意思是我们只希望获取从此刻开始的新消息。

注意，在BLOCK模式下，依然可以指定多个stream、指定起始ID、指定COUNT，可以根据需求灵活搭配。

XREAD 只有 COUNT 和 BLOCK 两个选项，所以它是一种非常基础的实现。更高级的功能，我们需要 XREADGROUP .

## 消费者组

更典型的场景，是多个消费者消费一个stream，每个人轮流领取任务分别独立处理。

虽然Redis中的 Consumer Group 与 Kafka中的 在实现上完全不同，不过他们的功能是一样的，所以我们决定继续保留这个术语。

一个消费者组，像是一个虚拟的消费者，同时服务于多个（真实的客户端）消费者。它保证了：

1. 每条消息被分配给不同的消费者，同一条消息不会被发给多人。
2. 在一个组里，每个消费者有且必须提供一个唯一的名字（字符串ID），这样即使消费者掉线了，当下一个自称为这个名字的消费者重新连接以后，可以恢复掉线前的处理状态。不过，这样就要求每个消费者能够提供全局唯一的ID。
3. 在一个组里，有一个「未消费的首条ID `first ID nerver consumed`」（译者注：理解为未读位置指针），因此，当组里的任意消费者请求新消息时，可以提供整个组的最新消息。
4. 消费一条消息，需要一个专门的命令来做显式的ACK。Redis会将ACK认为是这条消息被正确处理了，并将这条消息从这个组里移除掉。
5. 在一个组里，所有`pending`状态的消息都会被追踪，即那些已经被分发给某个消费者但是还未被ACK的消息。凭借这项特性，组里的每个消费者在查询历史消息的时候，只会看见那些被分发给自己的消息（而不会看见别人的）。

画个图来想象一下，消费者组大概长什么样子：

```text
+----------------------------------------+
| consumer_group_name: mygroup           |
| consumer_group_stream: somekey         |
| last_delivered_id: 1292309234234-92    |
|                                        |
| consumers:                             |
|    "consumer-1" with pending messages  |
|       1292309234234-4                  |
|       1292309234232-8                  |
|    "consumer-42" with pending messages |
|       ... (and so forth)               |
+----------------------------------------+
```

理解了这些之后，你应该能想明白很多事情。包括一个stream可以同时被多个组读取，也可以被多个消费者和多个组同时读取。

组，有3个命令：

- XGROUP: 创建、销毁和管理组
- XREADGROUP: 通过组来读取一个stream
- XACK：确认消费

## 创建一个组

```text
> XGROUP CREATE mystream mygroup $
OK
```

```go
func learnXGROUP(rdb *redis.Client)  {
	res, err := rdb.XGroupCreate(ctx, "lewin_001", "lewin_001_g1", "0").Result()
	log.Println(err)
	log.Println(res)
}
```

我们可以在一个已经存在的stream上创建组，最后的起始ID参数格式与XREAD一致。

译者注：这个命令不幂等，不能重复创建，如果组已经存在了则会返回错误。也不能在不存在的stream上创建组。

添加`MKSTREAM`参数，则可以自动创建stream：

```text
> XGROUP CREATE newstream mygroup $ MKSTREAM
OK
```

```go
func learnXGROUPMKSTREAM(rdb *redis.Client)  {
	res, err := rdb.XGroupCreateMkStream(ctx, "lewin_002", "lewin_002_g1", "0").Result()
	log.Println(err)
	log.Println(res)
}
```

然后我们就可以用 XREADGROUP 命令来从这个组里读取消息了。

译者注：消费者组 是需要在Redis中注册的，并且它的生命周期应当只与对应的stream有关，而与消费者无关。而消费者 是不需要注册的，只需要一个"名义"就行。

接下来的例子中，将会给消费者取两个名字，Alice 和 Bob 。

```text
> XREADGROUP GROUP mygroup Alice COUNT 1 STREAMS mystream >
1) 1) "mystream"
   2) 1) 1) 1526569495631-0
         2) 1) "message"
            2) "apple"
```

```go
func learnXReadGroup(rdb *redis.Client)  {
	res, err := rdb.XReadGroup(ctx,&redis.XReadGroupArgs{
		Group:    "lewin_001_g1",
		Consumer: "Alice",
		Streams: []string{"lewin_001", ">"},
		Count:    1,
		Block:    0,
		NoAck:    false,
	}).Result()
	log.Println(err)
	log.Println(res) // 输出：[{lewin_001 [{1628586038144-0 map[title:lewin_XADD ts:1628586038063701000]}]}]
}
```

上面参数中的`>`符号，代表的是 这个组的最新未读消息ID ，同时也是未被分配给其他消费者的最新消息。同时会带来副作用，让整个组的未读指针向前移动。

如果我们不指定`>`，而是指定一个具体的ID，那么返回的将是 属于这个消费者的pending的历史消息 ，而不是新消息。不会影响这个组和组里的其他消费者。

接下来再试一下 XACK 命令，它会将一条消息从一个消费者的历史pending列表中移除：

```text
> XACK mystream mygroup 1526569495631-0
(integer) 1
```

```go
func learnXAck(rdb *redis.Client)  {
	res, err := rdb.XAck(ctx, "lewin_001","lewin_001_g1", "1628586038144-0" ).Result()
	log.Println(err)
	log.Println(res)  // 输出：1
}
```

此时可以随便玩一下。用Alice去查询`0`，用Bob读一个`>`，然后用Bob读一个`0`。

有一些细节可以了解一下：

- 消费者可以自动创建，不需要显式注册。
- 你可以通过 XREADGROUP 同时监听多个stream，但是在此之前，你必须给这些stream分别创建名字相同的Group （即每个Group只能对应一个stream）。这个需求可能永远也不会出现，但是有必要告诉你它在技术上是可行的。
- XREADGROUP 是一个写操作，因此它只能在主库上执行。

在实践中，还要注意，当消费者启动上线时，最好先去消费历史消息，等所有的历史消息处理完毕，`0`返回空列表的时候，再用`>`去读取新消息。

## 不再上线的消费者

因为pending消息是归属于某个消费者的。当某个消费者不再上线了（或者说，没有客户端再以这个消费者的名字去获取消息了），那么这个消费者的pending消息将会永远留在那里无人问津。

提供了一种机制：一个消费者可以「索取`claim`」另一个消费者的pending消息。被索取的消息将会改变归属。

要完整的实现这个操作，第一步我们需要先查询某个组里的pending消息列表，这个命令是 XPENDING 。这是个只读的操作。

```text
> XPENDING mystream mygroup
1) (integer) 2
2) 1526569498055-0
3) 1526569506935-0
4) 1) 1) "Bob"
      2) "2"
```

```go
func learn(rdb *redis.Client)  {
	res, err := rdb.XPending(ctx,"lewin_001", "lewin_001_g1").Result()
	log.Println(err)
	log.Println(res)  // 输出：&{2 1628586038147-0 1628586038149-0 map[Alice:1 Bob:1]}
}
```

返回值有点复杂。第一个是所有pending消息的数量，第二个是最小ID，第三个是最大ID，第四个是一个列表，显示所有消费者以及它们各自的pending消息数量。

这个命令还支持更多的参数：

```text
XPENDING <key> <groupname> [<start-id> <end-id> <count> [<consumer-name>]]
```

```text
> XPENDING mystream mygroup - + 10
1) 1) 1526569498055-0
   2) "Bob"
   3) (integer) 74170458
   4) (integer) 1
2) 1) 1526569506935-0
   2) "Bob"
   3) (integer) 74170458
   4) (integer) 1
```

```go
func learnXPendingExt(rdb *redis.Client)  {
	res, err := rdb.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream:   "lewin_001",
		Group:    "lewin_001_g1",
		Idle:     0,
		Start:    "-",
		End:      "+",
		Count:    10,
		Consumer: "",
	}).Result()
	log.Println(err)
	log.Println(res)  // 输出：[{1628586038147-0 Alice 34m52.584s 3} {1628586038149-0 Bob 34m59.159s 2}]
}
```

此时我们获得的返回值是关于所有消息的详细信息，而不是一个总览信息。每个消息的信息分别包含：消息ID、消费者名字、空闲时间（从上次被读取到现在的时间）、被读取的次数。

译者注：注意，这个读取，实际上是「投递`delivered`」，而且是仅仅在这个组内的投递情况。

如果我们在组以外（例如其他组，或者干脆XRANGE）读取消息，那是不会影响组内pending消息的空闲时间和投递次数的。

然后是 XCLAIM 命令。这个命令很多参数：

```text
XCLAIM <key> <group> <consumer> <min-idle-time> <ID-1> <ID-2> ... <ID-N>
```

参数的主要意思：把所有大于 min-idle-time 的、且列在参数中的消息，都改派给 指定的consumer 。

这个限制最小闲置时间是有用的，可以防止一条消息被重复改派。因为改派之后，闲置时间就重置了，后续的CLAIM命令就不会生效。

```text
> XCLAIM mystream mygroup Alice 3600000 1526569498055-0
1) 1) 1526569498055-0
   2) 1) "message"
      2) "orange"
```

```go
func learnXClaim(rdb *redis.Client)  {
	res, err := rdb.XClaim(ctx,&redis.XClaimArgs{
		Stream:   "lewin_001",
		Group:    "lewin_001_g1",
		Consumer: "Catty",
		MinIdle:  time.Minute*10,
		Messages: []string{"1628586038149-0"},
	}).Result()
	log.Println(err)
	log.Println(res)  // 输出：[{1628586038149-0 map[title:lewin_XADD ts:1628586038074163000]}]
}
```

执行这条命令后，新获得消息的消费者，拥有了对这条消息进行查询和ACK的权力。

这条命令同时还会返回那些索取成功的消息数据。如果你不想关心消息本身，你可以指定一个 JUSTID 参数（略）。如果索取失败，则不会返回对应的消息（返回一个空列表）。

## 自动索取 Automatic claiming

> 从 Redis6.2 版本加入。（吐槽：好家伙，这也太新了，我现在开发机上的版本都才6.0）

前面的 XPENDING 和 XCALIM 已经可以实现需求了。这个新加入的命令则是将他们合在一起，起到简化的作用。

```text
XAUTOCLAIM <key> <group> <consumer> <min-idle-time> <start> [COUNT count] [JUSTID]
```

前面的例子改写一下：

```text
> XAUTOCLAIM mystream mygroup Alice 3600000 0-0 COUNT 1
1) 1526569498055-0
2) 1) 1526569498055-0
   2) 1) "message"
      2) "orange"
```

```go
func learnXAutoClaim(rdb *redis.Client) {
	msgs, start, err := rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   "lewin_001",
		Group:    "lewin_001_g1",
		MinIdle:  time.Second * 30,
		Start:    "0",
		Count:    1,
		Consumer: "Catty",
	}).Result()
	log.Println(err)
	log.Println(msgs)
	log.Println(start)
}
// XPendingExt:  [{1628599302658-0 Alice 1m35.471s 1} {1628599302660-0 Bob 1m28.381s 1}]
// XAutoClaim: [{1628599302658-0 map[title:lewin_XADD ts:1628599302486495000]}]
// XPendingExt:  [{1628599302658-0 Catty 2ms 2} {1628599302660-0 Bob 1m28.386s 1}]
```

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
