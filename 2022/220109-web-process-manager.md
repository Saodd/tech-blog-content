```yaml lw-blog-meta
title: "做一个服务进程管理工具"
date: "2022-01-09"
brev: "利用前端能力，更方便地启动服务、查看日志"
tags: ["架构"]
```

## 背景

近几年的后端界流行『微服务』的理念，我们项目组从前两年开始的常规重构动作中，也在尝试微服务（加领域驱动开发DDD理念）的实践落地。

从结果看来，有一些好处，但我觉得带来的麻烦是比好处更多一些的（毕竟因为是小厂吧）。一个典型的麻烦，就是现在开发时要启动的服务太多了，算上数据库的话总共有十几个后端服务要启动才能进行日常业务开发，光是每天早上开机然后一个个把服务起起来这个动作就值得吐槽了。

我们内部有少量的讨论，我个人的意见也是，理论上最佳方案应当是：做一个网关组件，通过简单的配置把请求代理到已经启动好的服务上去。

但是在实践中，很显然，很难落地，因为这要求这个网关组件对所有后端服务都有所了解，同时还要侵入每个服务内部去改逻辑让它们把请求发到网关上而不是以前直接写死的其他服务的地址上。这样的业务跨度+业务侵入性，我认为这个方案几乎又是不可行的。

所以回归到关键需求，我目前只想解决『开发时要启动太多服务』这一个痛点就好。

> 关于微服务，目前我认为，它理论上应该是好的，但是前提条件是要做好运维基建以及架构上的规范（而这个东西对于小厂或者很多历史项目来说都是几乎不可能的）。所以在我的视野范围内我认为它没什么意义。  
> 恐怕，我们是把微服务跟其他概念搞混了，我们真正需要的其实只是业务线的拆分，而不是某种刻板的代码组织形式。架构应当服务于业务以及团队实际情况，否则就是空中楼阁。

## 实现原型

所以这个周末动手写了千把行代码，做了一个工具原型：

![](../pic/2022/220109-meizhe-dev-manager.gif)

见上图（请原谅我无情打码），操作流程是先点击『启动所有服务』，然后可以观察到所有服务亮起绿灯，同时左侧主面板上实时输出日志。

这套系统的主要功能是：第一，能观察后端服务运行状态，有能力操作后端服务的启停；第二，接入后端服务的实时日志；第三，在基础功能上提供一些便利功能（高亮和过滤等）。

其实更普通的做法应该是写一些sh脚本来批量启动，至于日志可以借助一些终端复用工具来做分隔。所以我这个工具的优势，是能借助前端能力提供更优秀的展示效果，以及提供更多更灵活的额外能力。

技术栈，后端是`Go` + `gin` + `gorilla`提供web和websocket能力，用子进程管理业务服务进程；前端是`React` + `mobx`，没有其他组件。

技术栈大致与之前 [搭建一个自动部署(CD)系统-优化篇](../2021/210919-dev-a-CD-system-2.md) 是相似的，因此实现思路不再讲了，今天只记录分享一些我在开发过程中的坑：

## 踩坑与反思

### 1.前端ws断线重连

就是要监听ws对象的`close`和`error`事件，最好两个都要监听。

两个都监听，因此也要注意，触发事件的时候的重连动作不要多次重连，否则一个ws断线导致多次重连的话会引发很多问题。

重连的思路是很典型的方案，设置一个时间戳的值然后`useEffect`去观察这个值。

### 2.历史数据恢复与去重

我在前端和后端都保留了历史数据，二者搭配能把历史数据做得更"高可用"。

显然一种方案可以是借助外部储存，后端可以用`redis`前端可以用`indexedDB`，但是我都没有用，我只是简单地在程序内部保留一个数组。

关于这个历史数据的数组的优化稍微提一嘴，后端里我只是简单地当它达到某个长度时，截取掉前面的1/3；在前端涉及到VDOM的话还要记得再做一个key来帮助react优化。

去重主要是在前端做就好，因为后端不会重复。前端我只是简单地以时间戳来判断，收到比当前最新的时间戳更早的消息就全部丢掉。（这样简单地就能让断线重连的场景也能正常工作）

### 3.前端暗系配色

因为大部分IDE都是用的暗色系配色（Jetbrains是`Darcula`，VSCode默认就是黑的），所以为了保持视觉一致，我在前端的页面也做成夜间模式的暗色系配色。

第一次这样搞，感觉还是有一些新鲜的。首先各种灰度颜色都得反过来做一下十六进制减法，例如#333要变成#ccc（我应该没算错吧）；然后其他的彩色，情况更复杂一些，例如green在暗色模式下显示效果就很差，总之就是多多少少都需要一些主动调整。

（设计师还是有必要存在的）

### 4.前端box-sizing

这次没有用任何UI组件库（指`antd`这种的），只用原生样式。

然后本来我写前端已经是很熟练了，都可以~~闭着眼睛~~写很长一段然后再刷新页面查看效果的。可是这次发现有些元素的长宽值很难调整，按我原先的思路写的代码总是会遇到奇奇怪怪的问题。

后来猛然醒悟，原来是没有设置`border-box`啊，虽然可能很多项目里都默认设置上了以前从来不用操心，但是原生的默认值它就不是这个啊。算是一个小小的坑吧。

### 5.前端css属性名字

想写个动画，做X轴位移。

咦，这个css属性叫啥来着…… tran..啥来着？？好像有好几个长得差不多的属性啊……

害，毕竟我不是天天写动画的，确实背不出这个啊。临时搜一下，总共有3个：transform, translate, transition 都要用到。

### 6.前端选择文本

做了一个优化功能，鼠标hover到某条日志文本上的时候，会提示出它所属的服务名称以及时间戳。

一开始偷懒，给每个日志元素都附带了一个`position:absolute`并且`display:none`的标签元素，hover上去的时候它就显示出来，这样就纯css搞定了。

可是，纯css的显示是没问题，如果要选择日志文本并复制的时候，这些额外的标签元素中的文本也被复制到了（即使它absolute显示在鼠标没有经过的地方，但它的xml标签是在选择范围内的），导致复制出去的内容多了一些东西。

所以最后还是得借助js，做一个状态栏来专门显示这些额外信息，而不能与正常的日志混在一起。

### 7.Go使用Interface

这是关于代码抽象程度的思考。

不同功能模块之间的调用，多用`interface`，用得越多，各个模块之间的耦合越少，后续修改维护起来越快。

当然抽象也是有代价的，第一是编码确实会稍微慢一点点，第二用了接口屏蔽掉具体的实现，在阅读代码需要跳转的时候会跳不对地方。

但我个人风格，更倾向于慢工出细活，能做规范的地方多规范一点，多写点接口类型比较好。

### 8.Go组合不是继承

参考阅读： [Question: why is "self" or "this" not considered a best practice for naming your method receiver variable?](https://www.reddit.com/r/golang/comments/3qoo36/question_why_is_self_or_this_not_considered_a/)

在Go里是没有传统的面向对象特性的，具体到代码上就是没有`this`, `self`这些东西。组合也是可以模拟出类似继承的功能的。

直到我今天遇到这样的问题，下面是一个基类和一个派生类：

```go
type Parent struct{}

func (this *Parent) Run() {
	this.Log()
}
func (this *Parent) Log() {
	fmt.Println("我 是 Parent")
}
```

```go
type Child struct{ Parent }

func (this *Child) Log() {
	fmt.Println("我 是 Child")
}
```

如果我们调用`Child.Run()`，最后调用的的依然是`Parent.Log()`。

原理不细说了，我在发现问题的一瞬间就想清楚了前因后果。可是在没亲身掉进去之前 还真没想到会有这样的坑。

对这类继承代码的解决方案，是把继承的部分抽取出来，以接口类型保存下来，这样调用的时候就不会用错了。

关于组合，目前比较新潮的两大技术`React`和`Go`都在强调组合、嫌弃继承，我个人的风格也是接受了它俩的思想的。我觉得对于一些简单的代码（人脑里的栈可以追踪的程度），用面向对象会更符合直觉而且代码结构更清晰，但是如果代码复杂程度上来了，面向对象很快会让你弄不清楚某个属性/方法到底是从哪里来的，这会是弊大于利的。

### 9.websocket不允许并发写

websocket的消息结构是一个一个的包，但是当消息太大的时候，它也是可以拆包的，因此依然要求保证数据按顺序发送。做个比喻，它不是HTTP/2那样的多路复用，而是HTTP/1.1那样做完一个动作才能做下一个。

参考阅读： 协议规范[Base Framing Protocol](https://datatracker.ietf.org/doc/html/rfc6455#section-5.2) 英语不好的同学可以看这个[中文博客](https://www.cnblogs.com/chyingp/p/websocket-deep-in.html)

我快速地看了一眼源码，在`gorilla`的实现里每个conn有个锁，它每次写消息的时候会创建一个`*essageWriter`，这个东西里面做了写缓冲区，当收到大消息包的时候会分段发送，也就是在这个期间内需要一个保护措施。但是拿不到锁会直接panic，有一说一这个还是有点恶心的。目前看来似乎只有这一个库有这种问题，不知道是不是有什么其他的考虑，我没有搜到相关的信息。

其实我使用ws已经比较熟练了，对conn都会做一个client包装一下，不该遇到这种问题。这次是因为想要偷懒，在不同的功能模块里复用同一个conn连接，偷懒的结果是没走原先定义好的chan来传递消息，结果就遇到并发问题。

所以这个问题其实也可以上升到 代码设计原则 上。不该乱打洞的。

### 10.windows信号量与僵尸进程

目前我在原生windows环境下开发（不是WSL），所以这个坑还是坑了我蛮久的。

简而言之，windows的内核API跟Unix是不太一样的（表现在Golang的一些标准库里的接口也是不一样的），同时在编程开发中使用的第三方库也可能区别对待windows和Unix（Python是重灾区），所以在Linux上能顺利杀死的子进程，拉到windows上就不一定了。

在Golang语境下，如果用`cmd.Process.Kill()`温和地通知子进程关闭，那是控制不到孙进程的，很容易产生僵尸进程。

目前的解决方案是 [参考这个](https://stackoverflow.com/a/47059064/12159549) ，用windows提供的`taskkill`强制杀死子进程，这个可以兼顾到孙进程（通过日志可以看到执行结果）。

但是强制杀死子进程又会导致另一个问题，即有可能让某些释放资源的动作有时会被跳过。典型例子是`celery_beat`这个库可能会向一个文件里写入`pid`，正常优雅关闭的情况下是会清理掉的，但是被强制杀死就偶尔会清理不掉。

害，windows还真有点难伺候。

### 11.Python打印输出缓冲

这是个很麻烦的东西。在Python里，如果`print`一个东西，默认情况下它会把输出的内容放在缓冲区里，等到缓冲区满了再flash一次。因此为了确保实时查看日志，必须指定`PYTHONUNBUFFERED=1`环境变量。

为啥要这样设计，我大概能想到，因为Python是单线程的，自然会担心普通的stdout也会造成阻塞，因此试图在这个地方做个优化。

想要优化的想法不错，可是实现太垃圾了。一次简单的`print()`会产生多次输出动作，不仅在Go里很轻易能够感受到、同时也会给日志收集中间件造成很大的麻烦。

在Node.js里也有类似的问题，不过好那么一丝，它是在输出内容有换行符的时候进行截断。

正面示范是Go标准库`fmt`的实现，当试图`Println()`的时候，会在函数层面内建的缓冲区里构建好内容，再一口气通知输出接口。这多好啊，就这么简单一个设计，请问有这么难做吗？

因此我只能在我自己的代码里给Python擦屁股。

### 12.保证休息时间

休息时间是很重要的。我可能有80%的精妙的代码设计都是在 洗澡/吃饭/躺床上失眠 的时候想出来的。

在编码的时候，思维会更局限于当前正在写的位置，并且会更倾向于偷懒，即采用局部最优解而不是全局最优解。只有经过了放松休息，将思维重置之后，再以一种更客观的视角来回顾今天写的逻辑，往往会想通很多问题。

而且，『精力』本身也会影响编码质量。在上午精力充沛的时候会敢于面对挑战，愿意处理更难的问题、考虑更多细节；到晚上身心俱疲的时候，即使脑子还想工作，但其实身体已经很诚实地开始松懈了，遇到难题会倾向于想"这应该没什么大不了的先忽略掉吧"（然后留下一个坑）。

在休息时间思考，然后在工作时间全速编码，这才是一个职业程序员最高效的工作方式。

说起来，我最近找到了一个合适的比喻来形容对技术地掌握程度：游泳🏊‍。所谓的熟练，比喻过来就是"需要换气"的次数更少，能够游地更快更流畅；所谓精通，就是可以忘记教练教的姿势，自己想怎么游就怎么游，而且换了姿势也不必别人慢。

### 13.冷知识：端口号范围

参考：[wiki](https://zh.wikipedia.org/wiki/TCP/UDP%E7%AB%AF%E5%8F%A3%E5%88%97%E8%A1%A8)

首先，0到1023号端口，一般是Unix保留端口号，别占用。这个一般大家都知道。

然后，49152到65535号端口，属于“动态端口”范围，没有端口可以被正式地注册占用。动态端口就是出口端口，任何程序都可以随机使用。因此如果我们自己写的后端服务试图绑定这些端口，那可能会与系统中已经启动的程序冲突。这个问题在生产服务器上一般见不到，因为web服务往往启动的第一件事就是绑定端口；但是在本地开发的时候会埋下一个偶尔出现的隐患，冷不丁遇到了就挺烦人的。

特别是后端程序员，有必要了解一下这个冷知识。

## 展望

目前这个工具的实现大概还只能算是一个原型，如果想要投入到公司其他项目中推广，还需要做更多的脏活（更加抽象以适配更多场景）。

能想到的一些改进：

1. 不局限于『启动服务』，再内置『初始化代码仓库』的脚本，降低准备环境的门槛；
2. 不局限于『windows』，至少要再拓展到『Linux』，以及理论上『Mac』也自然可用了；
3. 不局限于『作一个外部工具』，再考虑参与『业务服务内部』的环境变量配置，以提供更多的灵活性。
4. 不局限于『本地运行』，考虑放入『docker』里获得更好的隔离性和更标准的配置方式，甚至接入『k8s』集群；

目前先个人内测一下吧。

（不过估计这个工具大概率应该就到此为止了，毕竟我位卑言轻我不配做架构/doge ）

## 总结

websocket还挺好玩的，在前端（客户端）能做出有意义的工具 也真的很有成就感。