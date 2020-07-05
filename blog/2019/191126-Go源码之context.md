```json lw-blog-meta
{"Title":"Go源码：context标准库","Date":"2019-11-26","Brev":"在读 net/http 包之前，先来看一下 context 这个包。它主要是针对请求来进行资源管理，提供了跨Go程的资源组织功能。希望通过学习这个包，深入了解一下 chan 的使用。","Tags":["Golang","源码"]}
```



## context包 简介

看一下[官方定义](https://golang.org/pkg/context/)

> Package context defines the Context type, which carries deadlines, cancellation signals, and other request-scoped values across API boundaries and between processes.  
> Incoming requests to a server should create a Context, and outgoing calls to servers should accept a Context. The chain of function calls between them must propagate the Context, optionally replacing it with a derived Context created using WithCancel, WithDeadline, WithTimeout, or WithValue. When a Context is canceled, all Contexts derived from it are also canceled.

包上下文定义了上下文类型，它跨API边界和进程之间传递截止日期、取消信号和其他请求范围的值。

对服务器的传入请求应该创建上下文，而对服务器的传出调用应该接受上下文。它们之间的函数调用链必须传播上下文，可以选择将其替换为使用WithCancel、WithDeadline、WithTimeout或WithValue创建的派生上下文。当一个上下文被取消时，它派生的所有上下文也被取消。

## Demo 示例

代码拷贝自[MojoTech的博客](https://mojotv.cn/2018/12/26/what-is-context-in-go):

```go
func someHandler() {
    // 创建继承Background的子节点Context
    ctx, cancel := context.WithCancel(context.Background())
    go doSth(ctx)

    //模拟程序运行 - Sleep 5秒
    time.Sleep(5 * time.Second)
    cancel()
}

//每1秒work一下，同时会判断ctx是否被取消，如果是就退出
func doSth(ctx context.Context) {
    var i = 1
    for {
        time.Sleep(1 * time.Second)
        select {
        case <-ctx.Done():
            fmt.Println("done")
            return
        default:
            fmt.Printf("work %d seconds: \n", i)
        }
        i++
    }
}

func main() {
    fmt.Println("start...")
    someHandler()
    fmt.Println("end.")
}
```

核心语句是`select {case <-ctx.Done() }`，只要调用了`Cancel()`（或者被其他条件比如Deadline触发Cancel），这个chan就是可读的，下面就写一些撤销资源的语句吧。

有个语法细节强调一下，`select`语句中，既可以识别普通的chan数据（比如`someChan <- 1`），也会识别`close()`信号；前者传递来的数据只能被抽取一次，而后者会通知所有的等待读取者。

在实际运用中，我们以`context.Background()`作为根节点，然后调用`WithCancel()`之类的方法，派生出子节点。父节点可以主动调用Cancel，所有的子孙节点都随之取消。

## 核心：Context接口

看名字也知道它是这个包里最核心的内容了，它是一个接口，定义了几个方法：

```go
type Context interface {
	// 如果返回 ok==false 则表示没有设置Deadline时间
	Deadline() (deadline time.Time, ok bool)

	Done() <-chan struct{}

    // 如果ctx没有被取消，返回nil；如果是，则返回相应的原因（错误类型）
	Err() error

	// 用于储存一些键值对。要注意使用类型断言。
	Value(key interface{}) interface{}
}
```

有个语法细节强调一下，`<-chan struct{}`是一个只读的chan，数据类型是`struct{}`是一个空结构体。在Go的世界中，如果仅仅只需要传递信号而不需要附带额外的数据，使用空结构体是最佳的。
更多有关空结构体的介绍，可以参考[一只IT小小鸟的博客](https://blog.csdn.net/qq_34777600/article/details/87195673)

### 常量：Background 和 TODO

这两个值我们往往用作根节点，他们被定义为常量：

```go
var (
	background = new(emptyCtx)
	todo       = new(emptyCtx)
)
```

他们的类型是`emptyCtx`，这个类是符合`Context`接口的，只不过所有的方法都是空白的，啥也不做。

## 1. 方法一：WithCancel

```go
func WithCancel(parent Context) (ctx Context, cancel CancelFunc) {
	c := newCancelCtx(parent)
	propagateCancel(parent, &c)
	return &c, func() { c.cancel(true, Canceled) }
}
```

```go
func newCancelCtx(parent Context) cancelCtx {
	return cancelCtx{Context: parent}
}
```

这个方法会将parent复制一份，返回一个新的`cancelCtx`对象（继承自parent）和它对应的`cancel`方法。如果调用这个cancel方法，就会将Done返回的chan关闭掉（会影响子孙，不影响父对象）。

### 1.1 类：cancelCtx

```go
type cancelCtx struct {
	Context

	mu       sync.Mutex            // protects following fields
	done     chan struct{}         // created lazily, closed by first cancel call
	children map[canceler]struct{} // set to nil by the first cancel call
	err      error                 // set to non-nil by the first cancel call
}
```

这个类非常简单，啥也没有。唯一有点意外的是带了一把锁，不过仔细想想也是应该的。

### 1.2 绑定父对象

这里用的是注册的机制，即，每次生成一个子对象时，去找到父对象中的一个注册表，将子对象加进去。这样，父对象cancel的时候就可以顺利地传播到所有子孙。

```go
func propagateCancel(parent Context, child canceler) {
	if parent.Done() == nil {
		return // parent is never canceled
	}
	if p, ok := parentCancelCtx(parent); ok {
		p.mu.Lock()
		if p.err != nil {
			// parent has already been canceled
			child.cancel(false, p.err)
		} else {
			if p.children == nil {
				p.children = make(map[canceler]struct{})
			}
			p.children[child] = struct{}{}
		}
		p.mu.Unlock()
	} else {
		go func() {
			select {
			case <-parent.Done():
				child.cancel(false, parent.Err())
			case <-child.Done():
			}
		}()
	}
}
```

上面`parentCancelCtx`这个方法，是检查父对象是否能够被cancel。如果能被取消，那就简单地注册一下；如果不能取消，例如`Background`，那么这个子对象就自立门户，直接监听父对象的Done。

### 1.3 取消

好的，现在已经能够形成一个cancel链条了，只要上游cancel了，下游都会跟着生效。那么调用cancel时发生了什么？

上面那个函数还有个邪门的地方，看它的签名参数`child canceler`，子对象本身并不是声明为`cancelCtx`类，而是声明为`canceler`这个接口。

```go
type canceler interface {
	cancel(removeFromParent bool, err error)
	Done() <-chan struct{}
}
```

那我们回去找一下`cancelCtx`类的`cancel`方法：

```go
func (c *cancelCtx) cancel(removeFromParent bool, err error) {
	if err == nil {
		panic("context: internal error: missing cancel error")
	}
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return // already canceled
	}
	c.err = err
	if c.done == nil {
		c.done = closedchan
	} else {
		close(c.done)
	}
	for child := range c.children {
		// NOTE: acquiring the child's lock while holding parent's lock.
		child.cancel(false, err)
	}
	c.children = nil
	c.mu.Unlock()

	if removeFromParent {
		removeChild(c.Context, c)
	}
}
```

从上面可以看出，调用cancel时，做了四件事：第一，设置自己的err；第二，关闭自己的done；第三，调用所有子孙的cancel；第四，从父对象的注册表中除掉自己的名字。

那么到此为止，`context`包的主要的逻辑其实已经梳理清楚了。还有一些方法都是些锦上添花的功能。我们接下来看一下。

## 2. 方法二：WithTimeout

它是`WithDeadline`的快捷方式：

```go
func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc) {
	return WithDeadline(parent, time.Now().Add(timeout))
}
```

## 3. 方法三：WithDeadline

顾名思义，设置一个死线，过了这个时间后自动触发Cancel。要注意的一个点是，想象一下，父对象最大时间1秒，而子对象最大时间3秒，会发生什么？也就是说，parent可能也有死线，这时候就要判定一下，以二者更早的那一个为准。

```go
func WithDeadline(parent Context, d time.Time) (Context, CancelFunc) {
	if cur, ok := parent.Deadline(); ok && cur.Before(d) {
		// The current deadline is already sooner than the new one.
		return WithCancel(parent)
	}
	c := &timerCtx{
		cancelCtx: newCancelCtx(parent),
		deadline:  d,
	}
	propagateCancel(parent, c)
	dur := time.Until(d)
	if dur <= 0 {
		c.cancel(true, DeadlineExceeded) // deadline has already passed
		return c, func() { c.cancel(false, Canceled) }
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err == nil {
		c.timer = time.AfterFunc(dur, func() {
			c.cancel(true, DeadlineExceeded)
		})
	}
	return c, func() { c.cancel(true, Canceled) }
}
```

这个函数做了这几件事：

1. 判断parent是否有Deadline，如果有而且还比当前规定的还短，那就以parent为准。
2. 以parent为模板，创建一个新的`timerCtx`对象。注意与前面的`cancelCtx`类是不同的，他们两个都是**类**，都是实现了`Context`**接口**的类。
3. 把新对象注册到parent中去。
4. 生成一个`time.Timer`来计时，到时间后执行cancel。

### 3.1 类：timerCtx

```go
type timerCtx struct {
	cancelCtx
	timer *time.Timer // Under cancelCtx.mu.
	deadline time.Time
}
```

这个类与`cancelCtx`类并没有太大区别，多了两个时间相关的私有变量。相应地，它的`Cancel()`方法也有一些改变：

```go
func (c *timerCtx) cancel(removeFromParent bool, err error) {
	c.cancelCtx.cancel(false, err)
	if removeFromParent {
		// Remove this timerCtx from its parent cancelCtx's children.
		removeChild(c.cancelCtx.Context, c)
	}
	c.mu.Lock()
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.mu.Unlock()
}
```

它的Cancel方法，首先调用`cancelCtx.Cancel()`，然后再把`Timer`也停止掉。

## 4. 方法四：WithValue

也是顾名思义，给上下文环境附带一个Value（键值对）。设想一个典型的场景，比如某个HTTP请求，我们要打日志，是不是会考虑给这个请求做一个ID号以便后期追踪？这个ID号就可以放在这里面。

意外的是，文档中强调：`key`最好不要用built-in的数据类型，以免不同的包之间产生冲突。用户应该自己定义一个类型来作为key。可以考虑`struct{}`，因为不需要分配内存。

```go
func WithValue(parent Context, key, val interface{}) Context {
	if key == nil {
		panic("nil key")
	}
	if !reflectlite.TypeOf(key).Comparable() {
		panic("key is not comparable")
	}
	return &valueCtx{parent, key, val}
}
```

### 4.1 类：valueCtx

```go
type valueCtx struct {
	Context
	key, val interface{}
}
```

非常简单的类，仅仅增加了键值对。那么如何获取这个值？用这个：

```go
func (c *valueCtx) Value(key interface{}) interface{} {
	if c.key == key {
		return c.val
	}
	return c.Context.Value(key)
}
```

一个细节要提醒的是，`struct{}{}==strct{}{}`是成立的。

## 我的总结

`context`包是一个相对较新的包，提供的是上下文的资源管理功能。设计目标是针对请求的资源管理。设计思路是建立一个树形的派生关系，通过管理上游节点来实现对所有下游子孙节点的统一管理。

请求，既可以是作为服务端接收的请求，也可以是作为客户端发出的请求。这就是`net/http`包的功能了。

到这里忽然想起来，`gin`框架中的请求处理函数，签名是`type HandlerFunc func(*Context)`，虽然这个`gin.Context`跟Go标准库中的`context.Context`是不同的东西，但是应该设计思路是类似的。等看完标准库，再去看看`gin`源码吧。
