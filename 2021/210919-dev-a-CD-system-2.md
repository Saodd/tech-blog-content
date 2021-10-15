```yaml lw-blog-meta
title: "搭建一个自动部署(CD)系统-优化篇"
date: "2021-09-19"
brev: "关键词：systemd websocket"
tags: ["运维"]
```

## 背景

在 [前一篇文章](../2021/210905-dev-a-CD-system.md) 里，我手撸了一个简易版的自动部署系统。使用起来当然没什么问题，不过生命不息折腾不止，之前留下可以优化的点，今天来深入研究一下。

## 优化一：systemd托管

`nohup`这个东西还是有坑的，第一，目前我是记录`pid`来结束上一个进程，那么在系统重启、进程号打乱的情况下，乱杀进程会很危险；第二，缺乏自动重启进程的手段。

当然，上述问题多写点几行代码也能解决，不过咱还是要有点追求的，今天玩一下`systemd`。

这个是主流Linux发行版都会携带（并且非常依赖）的一个系统服务组件。参考阅读：

- [Systemd 入门教程：命令篇 - 阮一峰的网络日志](https://www.ruanyifeng.com/blog/2016/03/systemd-tutorial-commands.html)
- [将 Web 应用丢给守护进程 - Cloud-Cloudys](https://cloud.tencent.com/developer/article/1656548)

使用`systemd`对于目前的我来说主要有这么些好处：

- 更新后，重启进程方便
- 宿主机重启后，重启进程方便
- 日志统一管理

同时也带来了坏处，也就是要向系统目录中写入一个配置文件，在某种角度来说是污染了宿主机，这个我是不太喜欢，不过污染程度有限，可以接受。

### systemd启动

具体怎么做呢，简而言之，关键就是一个配置文件（这里我命名为`jenky-runner.service`）：

```ini
[Unit]
Description=Jenky Runner Service

[Service]
ExecStart=/home/lewin/code/jenky-runner/runner
Restart=always
User=lewin
Group=lewin
Environment=PATH=/usr/local/sbin:...省略...
WorkingDirectory=/home/lewin/code/jenky-runner

[Install]
WantedBy=multi-user.target
```

上面配置文件的意思就是呢，进程启动命令是`/home/lewin/code/jenky-runner/runner`这个可执行文件，总是重启，然后用`lewin`用户身份运行，然后还指定了环境变量和工作目录。

把它拷贝到系统目录中，然后做一些操作：

```shell
sudo cp jenky-runner.service /etc/systemd/system
sudo systemctl daemon-reload
sudo systemctl start jenky-runner.service
```

这样，这个自定义的服务就启动了，可以查看日志检查一下：

```shell
sudo journalctl -u jenky-runner
```

### systemd重启

这里要分成两种情况。

第一种是systemd的配置需要更新，一般很少啦，但也不排除这种情况。

做法是，先`cp`配置文件，然后`systemctl daemon-reload`，然后`systemctl restart xxx`。

代码与前一章节的相同。

第二种是服务进程文件需要更新，这个会比较常见，至少对于我们自己托管的服务来说。

首先，正在运行的二进制文件是不允许被覆盖写入的，所以我们在启动进程的时候就要把原始执行文件`cp`成一个新名字，然后记得`chmod`。

然后要更新的时候，正常构建覆盖旧的文件，然后`systemctl stop xxx`，然后`cp`，然后`systemctl start xxx`。

```shell
sudo systemctl stop jenky-runner.service
cp runner runner_d
chmod +x runner_d
sudo systemctl start jenky-runner.service
```

## 优化二：embed脚本

借助Golang构建时的`embed`功能，我们把CD流程中所需要使用的shell脚本直接打包在二进制文件中，这样可以大大简化部署时所需要同步的文件数量。示例代码：

```go
//go:embed play.sh
var script string

func play() {
	cmd := exec.Command("/bin/bash", "-c", script)

	cmd.Start()
	cmd.Wait()
}
```

不过这样带来一个小问题是，`cmd.String()`的内容就很长了，不利于日志的处理和查看；但同时又带来好处，因为执行内容都在这里面。

## 优化三：websocket提供日志

部署过程中的日志最好还是要能够查看的，否则实在令人焦躁。（不确定是否出现故障）

之前的实现，是只提供一个 HTTP GET 接口，当然如果愿意手动刷新的话，那也勉强可以观察任务的执行状态。

但更高级的实现当然是Websocket了。

### ws前端

前端代码很简单，建立一个`Websocket`对象，监听`message`事件就可以了，错误处理都可以不要（不过还是要考虑一下创建新的部署之后的处理）。

```typescript jsx
  useEffect(() => {
    const ws = new WebSocket(WEBSOCKET_HOST + '/ws/taowai/watch-log');
    ws.addEventListener('message', (e) => {
        pushLog(e.data);
    });
    const i = setInterval(() => {
        if (ws.readyState === ws.OPEN) ws.send('ping');
    }, 2000);
    return () => {
        ws.close();
        clearInterval(i);
    };
  }, [ts]);
```

### ws后端：基本操作

上次用websocket都是两年前了，有点生疏了，姑且先复习一下Golang中的API基本操作。

使用两个框架`gin`+`gorilla/websocket`，在视图函数中，先`Upgrade`建立连接，然后分别读和写：

```go
func play(c *gin.Context) {
	conn, _ := upgrader.Upgrade(c.Writer, c.Request, nil)

	go writer(c, conn)
	reader(conn)
}
```

由于浏览器中调用`ws.close()`之后，并不会立即强行关闭连接，而是有60秒的等待时间，所以服务端这边姑且也要读取PING消息，避免一些意外的泄露情况：

```go
func reader(conn *websocket.Conn) {
	defer conn.Close()
	for {
		conn.SetReadDeadline(time.Now().Add(time.Second * 10))
		tp, _, err := conn.ReadMessage()
		if err != nil || tp == websocket.CloseMessage {
			return
		}
	}
}
```

写的话，先看一个简单的每秒写一个时间戳的实现：

```go
func writer(c context.Context, conn *websocket.Conn) {
	defer conn.Close()
	tk := time.NewTicker(time.Second)
	defer tk.Stop()
	for {
		select {
		case <-c.Done():
			return
		case <-tk.C:
			conn.SetWriteDeadline(time.Now().Add(time.Second * 10))
			if err := conn.WriteMessage(websocket.TextMessage, []byte(time.Now().Format(time.RFC3339)+"\n")); err != nil {
				if !errors.Is(err, syscall.EPIPE) {  // 连接可能直接断开了
					alog.CE(c, err)  // 处理错误
				}
				return
			}
		}
	}
}
```

### ws后端：读取日志

然后思考一下如何把一个日志文件输出出去。

在Linux环境下我们知道使用`tail -f`来追踪日志，所以其实一种思路是启动一个子进程来运行这个命令然后接到websocket上。

先看一个 [粗糙的实现](https://github.com/gorilla/websocket/blob/master/examples/filewatch/main.go) ，它是每次文件更新后都读取整个文件，可能适合JS开发热加载的场景，但其实不适合我们这里的日志流场景。

然后可以再看一个 [Golang实现的讨论](https://www.reddit.com/r/golang/comments/60ck9o/why_is_it_hard_to_mimic_tail_f_behavior/) 回复中有人提到了 [hpcloud/tail](https://github.com/hpcloud/tail) 这个库，简单看了下API还是很直观的。

最后我还是选择折腾一下，用`tail`来看吧。

首先，因为`websocket`并不是传统意义的流，所以需要写一个`Writer`中转一下，这里简单粗暴地搞一下：

```go
type ExecContainer struct {
	Ctx context.Context
	Out chan []byte
}

func (c *ExecContainer) Write(p []byte) (n int, err error) {
	select {
	case <-c.Ctx.Done():
	case c.Out <- p:
	}
	return len(p), nil
}
```

然后启动子进程。这里要特别注意，因为`tail`命令不会自己结束，所以要defer杀死子进程，防止进程泄漏。

```go
    cmd := exec.Command("tail", "-f", "-n", "+1", lastFile)
	cmd.Stdout = writer
	cmd.Stderr = writer
	if err := cmd.Start(); err != nil {
		alog.CE(c, err)
		return
	}
	defer cmd.Process.Kill()
	go func() {
		cmd.Wait()
		cancel()
	}()
```

然后视图层就不断地从`chan`里读取数据，丢回`websocket`里去就行了。

### ws其他细节

- 前端可以丢进`textarea`里，获得良好的文字阅读体验。但是要注意设置等宽字体。
- 前端还可以在接受到日志数据的时候，自动滚动到底部。
- 后端处理ws时启动了一个新的Go程，这个是不在gin框架的保护下的，所以要记得`recover`。
- websocket是可以跨域的，可以不用nginx代理，减少一点性能负担。

## 小结

systemd 配置起来确实挺复杂的，不过如果只取我们需要的一小部分功能，这个复杂度还是可以接受的。用上之后感觉体验不错。

然后涉及到websocket这种场景的时候，确实会增加许多需要额外考虑的因素。不过从性能上来说，还是比轮询和长连接清爽得多的。
