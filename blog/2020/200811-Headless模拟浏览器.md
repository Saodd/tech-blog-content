```yaml lw-blog-meta
title: Headless 服务端模拟浏览器
date: "2020-08-11"
brev: 做爬虫的人肯定熟悉模拟浏览器，我们今天介绍来自于chrome家族的Headless工具。
tags: [Golang, 中间件]
```

## 基本原理

`Headless Chrome` 指在 `headless` 模式下运行谷歌浏览器。

通俗地说，在「无头」状态下，浏览器不会渲染UI界面，但是其他前端逻辑运行时与真实模拟器完全一致；同时，通过本地端口暴露调试工具，方便其他进程控制当前访问的页面（也就是让我们的业务代码来操作这个浏览器上面的页面）。

这个工具的开发目标是「Web自动化工具」，测试是最重要的功能，而爬虫只是它顺带实现的功能之一。另外，在我们公司当前的场景下，还有一些其他的需求可以用它来实现。

我们以「网页截屏」这个功能作为例子来展开今天的文章。

## 环境准备

其实Chrome浏览器的调试工具是有一条规范的协议的，叫做`Chrome DevTools Protocol (CDP)`。理论上说，任何编程语言，都可以使用实现了这个协议的库来操作chrome浏览器。

因为是服务端应用，我这里选择更轻量、并发性更好的go语言来实现，框架是[`chromedp`](https://github.com/chromedp/chromedp)，它有5.1k star，而且在易用性上我觉得它是做的比较好的，很快就上手了。

```shell
$ go get -u github.com/chromedp/chromedp
```

然后chrome进程我选择用docker环境，镜像地址在[headless-shell](https://hub.docker.com/r/chromedp/headless-shell/)。运行容器后，默认通过9222端口来通信。

```shell
# 容器启动命令
$ docker run -d -p 9222:9222 --rm --name headless-shell chromedp/headless-shell
```

## 示例代码

我们来选择百度首页 www.baidu.com 来进行本次截屏操作。众所周知，百度首页是个动态页面，如果直接发起请求是得不到我们想要的内容的。（不知道的可以用 curl 命令试一下）

首先，我们要建立与chrome进程的通信，然后开启一个新的tab，代码如下：

```go
{
    ctx, cancel := chromedp.NewContext(context.Background(), chromedp.WithLogf(logger.Printf))
    defer cancel()
}
```

这里可能会有点奇怪，为什么没有见到ip和端口号？因为这次使用的是本地默认端口，所以不用显式的配置。如果需要额外配置，请调用`With`开头的那几个函数作为参数传入`NewContext`。这里的`WithLogf`就是指定日志级别以及日志函数的函数。

接下来我们该做些什么？我们想象一下：访问指定的url，等待加载，调整视窗大小，截屏……这几个操作。如果是熟悉go开发的人这时候估计有点头疼，这每个步骤都有可能失败，如何处理异常？幸好，这个框架给我们提供了便利的链式任务执行方法：

```go
{
    var imageBuf []byte
	tasks := chromedp.Tasks{
		chromedp.Navigate(u),
		chromedp.EmulateViewport(1600, 900),
		chromedp.Sleep(5 * time.Second), // 等待页面中的异步加载，这里应该换成更稳固的逻辑。
		chromedp.ActionFunc(func(ctx context.Context) (err error) {
			imageBuf, err = page.CaptureScreenshot().WithQuality(90).Do(ctx)
			return err
		}),
	}
	if err := chromedp.Run(ctx, tasks); err != nil {
		logger.Fatal(err)
	}
}
```

上面的`tasks`就是一系列调试任务的集合，然后通过`chromedp.Run`这个函数来进行顺序执行。

我们还可以看到，除了内置实现的`Navigate`, `EmulateViewport` 等方法外，这个框架还支持我们自定义任务，只要实现特定的函数接口并作为参数传入即可。我这里就是在一个匿名函数中调用截屏接口，来获取截屏返回的字节数组。

最后我们将字节数组写入文件：

```go
{
    if err := ioutil.WriteFile("screenshot2.png", imageBuf, 0644); err != nil {
		logger.Fatal(err)
	}
}
```

搞定！

不过这种模拟浏览器技术的性能肯定是十分糟糕的（相比于其他的服务端业务），所以我只把它当做一种兜底方案来储备。接下来还是要考虑用`node.js`来实现前后端功能的同步。
