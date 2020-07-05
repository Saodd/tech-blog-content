```json lw-blog-meta
{"Title":"Golang之Redis库选择","Date":"2020-03-08","Brev":"之前自己一直用的是 go-redis，用的挺好。但是看到公司项目用的是 redigo，感觉不好用，一方面对连接池的支持好像有问题，另一方面语法规则也很奇怪。所以决定把两个库的源代码都看一下做个对比。","Tags":["Golang","Redis","源码"]}
```



## 概况

Redis的go语言客户端并不算多。Github上千星的只有两个：

- [go-redis/go-redis](https://github.com/go-redis/redis)
- [gomodule/redigo](https://github.com/gomodule/redigo)

所以基本上只在他们之间二选一。

> 另外值得一提的是，还有一个10K-star的项目[CodisLabs/codis](https://github.com/CodisLabs/codis)是[豌豆荚](wandoujia.com)开源的，用Go实现的Redis数据库（不是客户端）。虽然星星很多，我也没有仔细看，但是粗略感觉这个思路似乎不太好？对于这种极致性能的应用，我猜还是C语言比go更靠谱一些。

## 非功能特性对比

`go-redis`选用的是`BSD`开源许可，截止目前有172个release、113贡献者、600+个issue、1400+个commit，使用了`go mod`并且最新版本已经迭代到了v7，设置了travis自动化脚本，单元测试覆盖率90+%。一切看起来都非常完美。

`redigo`选用的是`Apache`开源许可，截止目前有2个release、47个贡献者、300+个issue、200+个commit，使用的是老旧的`go vet`，没有使用任何先进的自动化工具。

结论：在非功能层面上，`go-redis`完胜，而`redigo`看起来就是一个维护非常不上心的半吊子三方库。

> 另外还想吐槽的一点是，`redigo`选用的组织名是`gomodule`，而这个过于夸张的名字下面只有一个像样的库，这种行为我觉得有占便宜的嫌疑，我个人是看不惯的。

## 功能特性： go-redis

我们先从最基本的`Client`对象的构造函数开始：

```go
// NewClient returns a client to the Redis Server specified by Options.
func NewClient(opt *Options) *Client {
	opt.init()

	c := Client{
		baseClient: newBaseClient(opt, newConnPool(opt)),
		ctx:        context.Background(),
	}
	c.cmdable = c.Process

	return &c
}
```

从上面可以了解到：首先，Client的底层连接默认都是连接池模式，连接池的确是Redis的常规用法；其次，它支持`context`，这允许我们实现更多的控制功能。

然后是`Options`对象，它储存着所有Redis有关的配置信息。它有一个`init()`方法，这个方法会将一些零值变量初始化为默认值。默认值中值得一提的是，默认连接池大小是CPU数的10倍，默认地址是`localhost:6379`。

`Client`作为一个对外暴露的对象，它的定义是这样的：

```go
type Client struct {
	*baseClient
	cmdable
	hooks
	ctx context.Context
}
```

从上面可以了解到，这是一种类似于继承的方式（在Go术语中叫`组合composition`或者`内嵌embedding`），表现形式就是可以通过子类对象本身直接调用父类成员和方法。其中，`baseClient`这个类只包含`opt`和`connPool`，以及一个`onClose`的函数。

`Client`对象可以并发使用，完全可以在应用中把它做一个全局变量，充分利用它的并发特性。

它包含很多方法可供调用，包括最基本的`Do()`方法，以及`Get()`, `Set()`, `HGet()`等快捷方式。所有的快捷方式都封装在`cmdable`这个类中，这个类是`Client`的父类。

所有的方法都通过`Client.hooks.process`这个方法来实现，然后它又通过`Client.baseClient.process`方法来实现。`hooks`是一个`struct`，这个类在子类`Client`中的表现形式是内嵌结构体（而非内嵌指针），所以在初始化`Client`对象的时候也会初始化一个`hooks`内嵌对象。`hooks`的作用是设置一些前置后者后置的额外函数任务。

`baseClient.process`这个方法中，先实现了一个请求失败重试的循环；然后用`baseClient.withConn`方法中实现一个类似python风格的上下文管理器功能，从`baseClient.connPool`中取出一个`Conn`后再放回。`connPool`用一个锁来保护并发。获取`Conn`后，如果还未被使用过（未被初始化过），则访问`baseClient.opt`来进行初始化；初始化后通过`Conn.WithWriter`和`Conn.WithReader`来实际操作底层的数据流。

在使用`Conn`连接时，都有`context`来进行超时管理。

每一个任务都被封装到一个`Cmder`对象中（类似于`http.Request`），它是一个接口，支持多种任务类型。例如在调用`Client.Get()`来发起任务时，会生成一个`StringCmd`。任务的结果和错误也同样被封装到`Cmder`对象中，最后我们常用`Cmder.Result()`方法来解封这个对象，获取执行的结果。

整体看下来，框架结构设计非常合理，至少初看一遍找不到任何槽点。而且用到的语法都很先进，让我收获很大。

## 功能特性： redigo

在官方文档上没有给出demo用法，所以姑且按我们的使用方式来。

也是从连接函数开始，这里首先调用的是`DialURL`函数。它会把传入的URL字符串和其他配置项做一些分析，转化为自身可以识别的形式。然后调用`Dial`函数，它负责建立tcp连接（以及tls握手），连接成功后封装为一个可操作的对象`Client`返回给包外的调用方。注意，这里的底层只有一个tcp连接，没有连接池。

如果要使用连接池模式，要使用`NewPool`方法。注意，这个方法被标记为**Deprecated**，在源码中可以看到各种蛋疼的东西，并发逻辑比较复杂，大致看上去感觉非常Errprone。但与此同时并没有其他的连接池对象可用了。连接池中，所有连接放在一个`idleList`里，组织形式是链表，从头部取出，用完后放回尾部。

每个底层`Client`连接都是通过`Dial`函数创建的，先封装在一个`poolConn`中，这个对象保存了创建时间用于控制连接的最大生存时间，以及保存了在链表中的前后邻居的指针可以直接在前后插入。

然后`poolConn`再封装进一个`activeConn`对象中，这个对象才是我们外部操作的对象。这个对象封装了一些操作接口，负责把应用功能传递到底层的连接中去执行。还有一个重要的`Close`方法，它是连接池与普通连接的最大区别，它不关闭底层的连接，而是将其放回池子里。

## 主观感受与结论

默认开启连接池模式的`go-redis`显然更加易用，各种接口函数设计得也相对合理。总体体验良好。

不推荐使用`redigo`。
