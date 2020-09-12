```json lw-blog-meta
{"title":"Rabbitmq 入门教程","date":"2020-09-06","brev":"最常用的消息队列组件之一。","tags":["中间件"],"path":"blog/2020/200906-Rabbitmq入门.md"}
```


## 前言

本文翻译自[Rabbitmq官博 - 教程（Go语言）](https://www.rabbitmq.com/tutorials/tutorial-one-go.html)，并且根据我自己的理解做了适当的调整。

原文介绍的非常基础、详尽。但是本文假定你已经有了一定的分布式队列知识，不再赘述那些基础的内容。

## 1. 安装

推荐使用Docker运行，在学习时推荐使用带管理器的版本。

```shell-session
$ docker pull rabbitmq:management
```

启动容器时，记得暴露5672端口，然后再顺便挂一个数据卷：

```shell-session
$ docker run --hostname my-rabbit --name rabbit -v rabbitmq:/var/lib/rabbitmq -p 5672:5672 -p 15672:15672 -dit rabbitmq:management
```

## 2. "Hello World!"

我们使用Go语言的SDK：

```shell-session
$ go get github.com/streadway/amqp
```

然后既然是分布式队列，那么肯定至少有两个进程（或角色），一个负责发布消息（`Publish`），一个负责接收并执行（`Consume`）。

而在发布消息之前，我们要先建立与 Rabbitmq 的连接，建立一个消息队列（`Queue`），然后再向这个队列上发消息：

```go
func send2() {
	// 1. 建立连接
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Println(err)
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		log.Println(err)
	}
	defer ch.Close()

	// 2. 声明队列
	q, err := ch.QueueDeclare(
		"hello", // 队列名称
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Println(err)
	}

	// 3. 发布消息
	err = ch.Publish(
		"",
		q.Name, // 要发往的队列名称
		false,
		false,
		amqp.Publishing{ // 消息结构体
			ContentType: "text/plain",
			Body:        []byte("Hello World!" + fmt.Sprint(time.Now().String())),
		})
	if err != nil {
		log.Println(err)
	}
}
```

然后我们可以在管理页面（http://localhost:15672/#/queues）上看到名叫 hello 的队列上已经有了一个消息。

接下来我们启动另一个进程，监听这个队列，并在收到消息时打印出来：

```go
func recv2() {
	// 1. 偷懒地建立连接
	ch := initChannel()
	// 1+ 这里也可以声明队列，因为我们可能会让接收方比发送方先运行
	// 2. 监听一个队列
	msgs, err := ch.Consume(
		"hello", // 刚才设定的队列名称
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Println(err)
	}
	// 3. 循环处理队列消息
	for d := range msgs {
		log.Printf("Received a message: %s", d.Body)
	}
}
```

这样我们就能看到如下输出：

```shell-session
$ go run recieve.go
2020/09/08 11:22:25 Received a message: Hello World!2020-09-08 11:22:19.886728 +0800 CST m=+0.012350206
```

## 3. 耗时的任务

正常情况下，肯定是比较耗时的任务我们才会用异步任务的方式来去做。我们改造一下上面的代码，用 sleep 来模拟耗时的操作：

```go
func send3() {
	ch := initChannel()
	for i := 1; i <= 5; i++ {  // 连续发5个任务
		ch.Publish(
			"",
			"3-sleep",
			false,
			false,
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte("sleep 2 " + "序号: " + strconv.Itoa(i)), // 指定睡眠任务执行时间
			})
	}
}
```

```go
func recv3() {
	ch := initChannel()
	ch.QueueDeclare(
		"3-sleep", // 队列名称
		false,
		false,
		false,
		false,
		nil,
	)
	msgs, _ := ch.Consume(
		"3-sleep", // 刚才设定的队列名称
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	for msg := range msgs {
		var taskTimeStr = string(msg.Body[6:7])
		taskTime, _ := strconv.Atoi(taskTimeStr)
		log.Println("接受任务：", string(msg.Body))
		time.Sleep(time.Duration(taskTime) * time.Second)
		log.Println("完成任务")
	}
}
```

执行后我们可以看到，5个任务被顺序执行了。

这时如果我们启动两个消费者（接收方），会发生什么？——两个消费者会轮流接收消息，即一个执行135，另一个执行2和4.

是的，当存在多个消费者时，是按顺序给每个接受者轮流派发消息的（即`Round-robin`模式）。

### 4. 当消费者挂了

现实情况的业务任务往往是很复杂的，有可能会执行失败，甚至有可能让整个消费者进程挂掉。

那么我们如何保证当一个消费者挂掉时，它手头的消息能自动地转发给其他消费者去处理？

Rabbitmq 的做法是「消息签收`Message acknowledgment`」。即，每个消息被发到消费者那里去时，并不会从 Rabbitmq 上删除这个消息，而是要等到消费者“确认完成”之后，才会被标记为可删除。（为什么前面的代码没有这个机制？因为某个参数设置了自动签收）

我们改造一下消费者的代码：

```go
func recv3a() {
	ch := initChannel()
	msgs, _ := ch.Consume(
		"3-sleep",
		"",
		false,  // 这里禁止自动签收
		false,
		false,
		false,
		nil,
	)
	for msg := range msgs {
		var taskTimeStr = string(msg.Body[6:7])
		taskTime, _ := strconv.Atoi(taskTimeStr)
		log.Println("接受任务：", string(msg.Body))
		time.Sleep(time.Duration(taskTime) * time.Second)
		log.Println("完成任务")
		msg.Ack(false) // 消息签收。注意参数是false
	}
}
```

运行它，然后在它执行某个任务的时候用 CTRL+C 强制结束它。再次启动它，会发现它会重新执行上次未完成的任务。不会有消息被丢失。

值得一提的是，Rabbitmq 监听的是消费者的“连接”是否断开，如果连接断开则认为消费者挂了，然后才会把未签收的消息转发给其他消费者。

> 注意！一定不要忘记签收，否则会资源泄露，并且未签收的消息将会被一直重复执行。

## 5. 当 Rabbitmq 挂了

这也是有可能的。如果没有被显式指定的话，消息和队列都是存放在内存中的——这意味着，挂掉之后就没了。

因此我们需要将消息和队列写入硬盘。

```go
func send5() {
	ch := initChannel()
	ch.QueueDeclare(
		"5-durable",
		true, // 这里指定队列持久化
		false,
		false,
		false,
		nil,
	)
	for i := 1; i <= 5; i++ {
		ch.Publish(
			"",
			"5-durable",
			false,
			false,
			amqp.Publishing{
				DeliveryMode: amqp.Persistent, // 这里指定消息持久化
				ContentType:  "text/plain",
				Body:         []byte("sleep 2 " + "序号: " + strconv.Itoa(i)),
			})
	}
}
```

```go
func recv5() {
	ch := initChannel()
	msgs, _ := ch.Consume(
		"5-durable",
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	for msg := range msgs {
		var taskTimeStr = string(msg.Body[6:7])
		taskTime, _ := strconv.Atoi(taskTimeStr)
		log.Println("接受任务：", string(msg.Body))
		time.Sleep(time.Duration(taskTime) * time.Second)
		log.Println("完成任务")
		msg.Ack(false)
	}
}
```

然后我们先发布消息，然后重启Rabbitmq进程，然后再启动消费者进程：

```shell-session
$ go run send.go
$ docker restart rabbit
$ go run recv.go
```

重启Rabbitmq之后，我们可以在管理页面（http://localhost:15672/#/queues）上确认，之前创建的非持久化队列都消失了，剩下的只有刚刚创建的名叫 "5-durable" 的持久化队列存在着，且它的消息数量为5个。

运行消费者进程，这5个消息被顺利地处理掉。

> 这里要注意的是，一条消息在被发布后、在被写入硬盘之前，还是有可能被丢失掉的。因此，消息发布者也可能需要[类似签收的机制](https://www.rabbitmq.com/confirms.html)

## 6. 公平的任务分配

现实中的业务任务，执行时间也会有长有短。

这里补充一个小知识，每个消费者在监听队列后，分配到的所有的消息都会先接收过来，再慢慢执行。

假如现在有两个消费者，但是所有的奇数号任务都很耗时而偶数号任务都很简单，那么1号消费者手上的任务就会堆积。因此上面的轮流分配制度就不合适了。

解决的办法是：给每个消费者限定未签收的消息数量（比如设定为1）。如果超过这个数量，就不再给这个消费者发送新的消息。（注意，在实现上是给每个 `Channel` 做限制，而不是给消费者进程，一个进程可以开启多个 Channel ）

```go
func send6() {
	ch := initChannel()
	for i := 1; i <= 10; i++ { // 发10个任务
		ch.Publish(
			"",
			"5-durable",
			false,
			false,
			amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				ContentType:  "text/plain",
				Body:         []byte(fmt.Sprintf("sleep %d 序号: %d", rand.Intn(10), i)), // 随机时间
			})
	}
}
```

```go
func recv6() {
	ch := initChannel()
	ch.Qos(
		1, // 限制数量
		0,
		false,
	)
	msgs, _ := ch.Consume(
		"5-durable",
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	for msg := range msgs {
		var taskTimeStr = string(msg.Body[6:7])
		taskTime, _ := strconv.Atoi(taskTimeStr)
		log.Println("接受任务：", string(msg.Body))
		time.Sleep(time.Duration(taskTime) * time.Second)
		log.Println("完成任务")
		msg.Ack(false)
	}
}
```

同时启动两个消费者，然后运行生产者去发布消息。然后就可以观察到两个消费者按照执行速度依次领取下一个消息，没有任务被堆积。

## 7. 发布/订阅 模式

现在我们来试一试，一条消息会同时发送给多个消费者的情况。这种情况被称为「发布/订阅模式 `publish/subscribe`」

前面我们介绍了`队列 Queue`。但事实上，发布者只能向一个`交换器 Exchanges`发送他的消息，至于发送到哪个队列（甚至是否发送到队列）他是不知道的。交换器的作用就是接受消息，然后决定如何分配这个消息。

> 在前面的教程中，我们并没有声明交换器却能直接声明队列，此时队列是放在一个默认（或称无名）交换器上的，并且通过 `routing-key` 来精确地寻找到相应名字的队列。

交换器有四种类型：`direct`, `topic`, `headers` 和 `fanout`。我们现在只关心最后一种。`fanout`类型简单地将所有消息广播到它名下的所有队列中去。

接下来我们模拟开发一个日志系统。发送端发送日志文本，接收端可以有多个。

我们在发送端上声明交换器 ，然后向这个交换器发送模拟日志消息：

```go
func send7() {
	ch := initChannel()
	ch.ExchangeDeclare(
		"logs",
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	for range time.Tick(time.Second) { // 每秒发送一条消息
		ch.Publish(
			"logs", // 注意这里指定了exchange 并清空了routing-key
			"",
			false,
			false,
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(fmt.Sprintf("【%s】 一些日志内容……", time.Now().String())),
			})
	}
}
```

在这种需求下，接收端并不在乎它上线之前的那些未读消息，因此它连接到 Rabbitmq 时，最好给它分配一个全新的（空的）队列。用队列名称来做个事情是最合适的，而且最好让 Rabbitmq 来分配一个随机的队列名称。另外，当一个接收端下线时，队列也最好要能自动销毁，避免浪费资源。

为了做到上面的需求，我们只需要在创建队列时给它传入一个空字符串作为名称就可以了。 Rabbitmq 会给它生成一个类似 “amq.gen-JzTY20BRgKO-HjmUJj0wLg” 这样的名字。

创建队列之后，还要额外将这个队列绑定到交换器上：

```go
func recv7() {
	ch := initChannel()
	q, _ := ch.QueueDeclare(  // 声明一个随机名称的队列
		"",
		false,
		false,
		true, // 注意要设置exclusive
		false,
		nil,
	)
	ch.QueueBind(  // 声明队列的时候没有指定交换器，必须要额外显式地绑定
		q.Name,
		"",
		"logs", // 我们指定的交换器
		false,
		nil,
	)
	msgs, _ := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	for msg := range msgs {
		log.Println("收到日志：", string(msg.Body))
	}
}
```

运行结果：

1. 先运行发送端代码，然后我们可以在web管理页面看到已经建立了名为“logs”的交换器，并且此时每秒接收消息数量为1
2. 然后启动一个接收端，可以观察到接收端从下一条日志消息开始不断地接收日志消息
3. 再启动另一个接收端，可以观察到在它启动之前的日志是没有传过来的

## 8. 路由

接下来我们尝试一下，在上面广播的基础上，让某个接收端只订阅一部分消息。例如，在日志系统中，只订阅错误级别的日志。

为了实现这个功能，我们只需要在将队列绑定到交换器的时候，指定 `routing-key` 参数就可以了（为了避免混淆，我们这里将其称为绑定关键字`biding-key`）。

但是我们前面声明的`fanout`类型的交换器，并不支持这个关键字参数。所以接下来我们要换成`direct`类型的交换器，它会将消息发送到与绑定关键字完全相同的队列上（如果没有匹配的队列，消息则被丢弃）。

![direct类型交换器](https://www.rabbitmq.com/img/tutorials/direct-exchange.png)

接下来我们需要分别在发送端和接收端上指定关键字：

```go
func send8() {
	ch := initChannel()
	ch.ExchangeDeclare(
		"logs_direct",
		"direct", // 改变交换器类型
		true,
		false,
		false,
		false,
		nil,
	)
	keyMap := map[int]string{0: "black", 1: "green", 2: "orange"}
	for range time.Tick(time.Second) { // 每秒发送一条消息
		key := keyMap[rand.Intn(3)] // 随机关键字
		body := fmt.Sprintf("【%s】 一些日志内容……", time.Now().String())
		fmt.Println(key, body)
		ch.Publish(
			"logs_direct",
			key,
			false,
			false,
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(body),
			})
	}
}
```

```go
func recv8() {
	ch := initChannel()
	q, _ := ch.QueueDeclare(
		"",
		false,
		false,
		true,
		false,
		nil,
	)
	for _, key := range os.Args[1:] { // 从命令行参数中读取关键字，可以绑定多个关键字
		ch.QueueBind(
			q.Name,
			key,
			"logs_direct",
			false,
			nil,
		)
	}
	msgs, _ := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	for msg := range msgs {
		log.Println("收到日志：", string(msg.Body))
	}
}
```

```shell-session
$ go run send.go
$ go run recv.go black
$ go run recv.go orange green
```

运行结果：

1. 先运行发送端代码，然后我们可以在web管理页面看到，“logs_direct”的交换器上每秒接收消息数量为1，但是发出的消息为0
2. 然后启动一个接收端，可以观察到它偶尔会弹出消息
3. 再启动另一个接收端，可以观察到，两个接收端根据关键字分配了不同的消息

## 9. 话题

接下来我们要基于多个维度条件来订阅消息。

我们要使用`topic`类型的交换器。发到这种交换器上的消息的关键字，必须是`.`号连接的字符串，例如`info.server01.app01`。队列的绑定关键字也必须是这种格式。最多255字节。

同时还支持通配符：

- `*` 可以替代1个单词
- `#` 可以替代0个或多个单词

接下来的示例代码中，我们定义关键字的格式为`<facility>.<severity>`（即用来描述日志的设备、等级），然后尝试订阅任意的话题关键字，例如`server1.info`, `*.info` 和 `#` 等等。

```go
func send9() {
	ch := initChannel()
	ch.ExchangeDeclare(
		"logs_topic",
		"topic", // 改变交换器类型
		true,
		false,
		false,
		false,
		nil,
	)
	facilityMap := map[int]string{0: "server0", 1: "server1", 2: "server2"}
	severityMap := map[int]string{0: "error", 1: "warning", 2: "info"}
	for range time.Tick(time.Millisecond * 100) { // 加快速度每0.1秒发送一条消息
		key := facilityMap[rand.Intn(3)] + "." + severityMap[rand.Intn(3)] // 随机关键字
		body := fmt.Sprintf("【%s】[%s] 一些日志内容……", time.Now().String(), key)
		fmt.Println(body)
		ch.Publish(
			"logs_topic",
			key,
			false,
			false,
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(body),
			})
	}
}
```

```go
func recv9() {
	ch := initChannel()
	q, _ := ch.QueueDeclare(
		"",
		false,
		false,
		true,
		false,
		nil,
	)
	for _, key := range os.Args[1:]{ // 从命令行参数中读取关键字，可以绑定多个关键字
		ch.QueueBind(
			q.Name,
			key,
			"logs_topic",
			false,
			nil,
		)
	}
	msgs, _ := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	for msg := range msgs {
		log.Println("收到日志：", string(msg.Body))
	}
}
```

然后就开始玩耍吧：

```shell-session
$ go run send.go
$ go run recv.go #
$ go run recv.go "server0.*"
$ go run recv.go server0.info server0.error
```

## 10. 远程过程调用

好，暂时收一收订阅模式的思维，回到前面说的队列模式。

在常见的业务场景中，我们可能需要从一台电脑上调用另一台电脑的函数进行计算并等待它的结果，这种模式被称为「远程过程调用`Remote Procedure Call`」。

接下来我们用 Rabbitmq 来做个 RPC 系统。我们让执行端去计算一个愚蠢的斐波那契数列`Fibonacci numbers`算法，来模拟耗时的操作。

> 注意，RPC看起来挺酷的，但是风险很大，谨慎使用！（看起来一点也不酷，挺蠢的……）

![rpc模式](https://www.rabbitmq.com/img/tutorials/python-six.png)

我们的RPC系统的运行逻辑如下：

- 客户端启动时，创建一个匿名的回调队列。
- 需要RPC请求时，客户端发送一个消息，消息中包含`reply_to`（对应回调队列）和`correlation_id`（对应当前请求）。
- 请求消息被发往`rpc_queue`。
- 服务端正在监听`rpc_queue`，收到请求后，执行请求并将结果发回回调队列中。
- 客户端监听回调队列，收到回复后，检查`correlation_id`字段。

我们先写一个愚蠢的、耗时的斐波那契函数：

```go
func fib(n int) int {
	if n == 0 {
		return 0
	} else if n == 1 {
		return 1
	} else {
		return fib(n-1) + fib(n-2)
	}
}
```

然后分别写服务端、客户端：

```go
func recv10() {
	ch := initChannel()
	q, _ := ch.QueueDeclare( // 在接收端定义这个rpc请求队列，不需要特别的参数
		"rpc_queue",
		false,
		false,
		false,
		false,
		nil,
	)
	msgs, _ := ch.Consume(
		q.Name,
		"",
		false, // 不要自动签收
		false,
		false,
		false,
		nil,
	)
	for msg := range msgs {
		log.Println("收到请求：", string(msg.Body))
		n, _ := strconv.Atoi(string(msg.Body)) // 忽略异常处理，异常时为0
		resp := fib(n)
		ch.Publish( // 把结果发回回调队列
			"",
			msg.ReplyTo, // 回调队列
			false,
			false,
			amqp.Publishing{
				ContentType:   "text/plain",
				CorrelationId: msg.CorrelationId,
				Body:          []byte(strconv.Itoa(resp)),
			})
		msg.Ack(false) // 不要忘记签收
	}
}
```

```go
func send10() {
	ch := initChannel()
	n, _ := strconv.Atoi(os.Args[1]) // 读取斐波那契函数参数，忽略异常
	q, _ := ch.QueueDeclare( // 声明一个回调队列
		"",
		false,
		false,
		true,
		false,
		nil,
	)
	msgs, _ := ch.Consume( // 在发送请求之前，先监听回调队列
		q.Name, // queue
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	corrId := uuid.New().String() // 生成一个随机id
	ch.Publish(
		"",
		"rpc_queue",
		false,
		false,
		amqp.Publishing{
			ContentType:   "text/plain",
			CorrelationId: corrId, // 指定任务id
			ReplyTo:       q.Name, // 指定回调队列
			Body:          []byte(strconv.Itoa(n)),
		})
	for msg := range msgs { // 处理回调消息
		if corrId == msg.CorrelationId {
			log.Println("收到回调：", string(msg.Body))
			break
		}
	}
}
```

然后我们先运行服务端、再多次运行客户端，就可以观察RPC的运行了！

> 注意！！由于这个斐波那契函数过于愚蠢，请不要一下子将数字设得太大。我这里设到41的时候就已经需要1秒以上的计算时间了。  

> 如果设到一个很大的值，会导致服务端长时间无响应；如果服务端挂掉，这个请求又会被传到另一个服务端，导致另一个服务端也挂掉，连锁反应。这种情况是RPC模式中典型的难点之一了。

## 11. 关于 AMQP 协议

Rabbitmq 使用的是 `AMQP 0-9-1`协议来定义消息。这个协议给消息定义了14个属性，大多数是很少用的，我们常用的有：

- `persistent`： 消息持久性
- `content_type`： 描述消息所使用的的序列化协议。比如典型的有`application/json`。（但这个并不是强制的，只是作为一种提示）
- `reply_to`： 回调队列
- `correlation_id`： 消息id，一般用于RPC

## 12. 发布者确认

> 官方没有提供 golang 的代码演示，因此我自己研究了一下。如有错误请指正。

```go
func send12() {
	ch := initChannel()
	confirms := ch.NotifyPublish(make(chan amqp.Confirmation)) // 监听发布确认结果
	ch.Confirm(false)                                          // 对当前Channel开启监听发布确认
	for i := 1; i <= 10; i++ {
		ch.Publish(
			"",
			"5-durable",
			false,
			false,
			amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				ContentType:  "text/plain",
				Body:         []byte(fmt.Sprintf("sleep %d 序号: %d", rand.Intn(10), i)),
			})
		log.Println("确认一条消息", i, <-confirms) // 阻塞等待发布确认结果
	}
}
```

关于「发布确认`Publishing Confirmation`」这个环节，最大的痛点在于，Rabbitmq 并没有提供一个同步版本的 Publish 方法。它的确认信息都是异步发回的（类似于又建立了一个回调队列）。

> 注：查阅资料发现，Rabbitmq 似乎也有同步的操作，即事务，但是太耗性能。

所以如果我们的业务中需要保证消息被稳固地传到 Rabbitmq 了，我们可能需要为每个请求都新建一个 Channel 对象（因为确认信息会分配到相应的Channel），或者将 Channel 对象用 `sync.Pool` 管理起来。

用 Pool 管理的隐患在于，如果 Rabbitmq 返回了错误消息，那么这个 Channel 就会被关闭，因此后一个使用的地方可能会被前一个使用的结果所影响。目前初步看了一下 Channel 的实现，好像主要都是一些内部的数据结构的建立，并不涉及到底层的连接，因此每次都创建一个新的对象好像也不是不可以接受。

但这还是挺别扭的。有点像用Redis做消息队列的那种感觉——人家设计之初并没有考虑这个东西，强行搞的话就搞得很难受。

或许，消息队列这种东西就不适合用来保证可靠性。它应该还是只能用来处理一些偶尔丢失也无所谓的数据，比如日志消息、比如一个纯粹的触发动作消息（可以建立补偿机制来复原）。真正的数据还是应该由数据库组件来负责。

## 13. 高可用

TODO： 留个坑

## 总结

我们做开发，总归还是要面向异常编程的。消息队列组件目前看来主要的异常点就是两个：一是发布环节的，二是消费环节的。所以主要需要注意的点在于，一是重要的消息要结合数据库和日志来保证可靠性，二是消费端要建立合理的回滚机制来保证幂等。

总体来说，Rabbitmq 给我们提供了充分的队列控制功能。我认为它的队列功能已经足够强大，有基本的SDK就可以了，完全不需要借助 Celery 这类框架的帮助。（Flag就立在这了，期待日后打脸）
