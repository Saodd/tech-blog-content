```lw-blog-meta
{"title": "WebSocket初体验——网页聊天室", "date": "2019-07-10", "tags": ["Web"], "brev": "受Yanni老板的启发，我的第一个Web应用打算做成一个网页聊天室。之前用Go+TCPsocket实现过控制台版本，不过在控制台上体验很差。今天来看一下Go+JS+Websocket是否能够实现预期的结果。"}
```

## 先看`Go`+`TCPsocket`实现的聊天室

主要逻辑很简单，采用典型的C/S结构，服务器监听端口，将所有的客户端连接都保存在`map`里，
当有任意客户端发送消息来了，服务端就遍历所有的活动连接，把消息广播出去。

### Go: ChatroomServer

```golang
var ConnMap map[string]*net.TCPConn

func Main_ChatroomServer() {
    var tcpAddr *net.TCPAddr
    ConnMap = make(map[string]*net.TCPConn)
    tcpAddr, _ = net.ResolveTCPAddr("tcp", "0.0.0.0:9999")

    tcpListener, _ := net.ListenTCP("tcp", tcpAddr)

    defer tcpListener.Close()

    for {
        tcpConn, err := tcpListener.AcceptTCP()
        if err != nil {
            continue
        }

        fmt.Println("A client connected : " + tcpConn.RemoteAddr().String())
        // 新连接加入map
        ConnMap[tcpConn.RemoteAddr().String()] = tcpConn
        go tcpPipe(tcpConn)
    }

}

func tcpPipe(conn *net.TCPConn) {
    ipStr := conn.RemoteAddr().String()
    defer func() {
        fmt.Println("disconnected :" + ipStr)
        conn.Close()
    }()
    reader := bufio.NewReader(conn)

    for {
        message, err := reader.ReadString('\n')
        if err != nil {
            return
        }
        fmt.Println(conn.RemoteAddr().String() + ":" + string(message))
        // 这里返回消息改为了广播
        boradcastMessage(conn.RemoteAddr().String() + ":" + string(message))
    }
}

func boradcastMessage(message string) {
    b := []byte(message)
    // 遍历所有客户端并发送消息
    for _, conn := range ConnMap {
        conn.Write(b)
    }
}
```

### Go: ChatroomClient

```golang
var quitSemaphore chan bool

func Main_ChatroomClient() {
    var tcpAddr *net.TCPAddr
    tcpAddr, _ = net.ResolveTCPAddr("tcp", "192.168.1.213:9999")

    conn, _ := net.DialTCP("tcp", nil, tcpAddr)
    defer conn.Close()
    fmt.Println("connected!")

    go onMessageRecived(conn)

    // 控制台聊天功能加入
    for {
        var msg string
        myScanln(&msg)
        if msg == "quit" {
            break
        }
        b := []byte(msg + "\n")
        conn.Write(b)
    }
    <-quitSemaphore
}

func onMessageRecived(conn *net.TCPConn) {
    reader := bufio.NewReader(conn)
    for {
        msg, err := reader.ReadString('\n')
        fmt.Println(msg)
        if err != nil {
            quitSemaphore <- true
            break
        }
    }
}

func myScanln(a *string) {
    reader := bufio.NewReader(os.Stdin)
    data, _, _ := reader.ReadLine()
    *a = string(data)
}
```
### 实现效果（客户端）

```shell
root@973b12641a87:/scripts/Learning/Golang/src# go run run2.go
connected!
yeah!
192.168.1.213:60894:yeah!

yahoo
192.168.1.213:60894:yahoo

^Csignal: interrupt
```

以上的Golang代码仅仅使用了`net`包，
我们可以很直观地感受到，连接非常简单轻量，并没有包含乱七八糟的头部数据（我指的是`http`那种头部）。
所以如果要实现一些高级特性，比如加密、浏览器支持之类的，还要自己另外想办法。

我们接下来看一下`WebSocket`：

## WebSocket简介

WebSocket 协议在2008年诞生，2011年成为国际标准。所有浏览器都已经支持了。

它的最大特点就是，服务器可以主动向客户端推送信息，客户端也可以主动向服务器发送信息，是真正的双向平等对话，属于服务器推送技术的一种。

### 为什么需要它

我们知道，正常的Web应用中，是客户端发起请求-服务器应答的模式。
那么对于一些复杂的应用，比如最简单的聊天室，要如何实现？能想到的就是用js脚本定时询问服务器。
但这种方法显然太低效了。

而原生的Socket，又太简单太底层了。

所以这种两种需求相加，自然就会催生出一种既有类似于Socket的效率，又像http那样兼容丰富的技术，那就是Websocket了。

![WebSocket结构图](/static/blog/2019-07-10-WebSocket-Structure.jpg)

### WebSocket 数据帧结构

```text
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
 +-+-+-+-+-------+-+-------------+-------------------------------+
 |F|R|R|R| opcode|M| Payload len |    Extended payload length    |
 |I|S|S|S|  (4)  |A|     (7)     |             (16/64)           |
 |N|V|V|V|       |S|             |   (if payload len==126/127)   |
 | |1|2|3|       |K|             |                               |
 +-+-+-+-+-------+-+-------------+ - - - - - - - - - - - - - - - +
 |     Extended payload length continued, if payload len == 127  |
 + - - - - - - - - - - - - - - - +-------------------------------+
 |                               |Masking-key, if MASK set to 1  |
 +-------------------------------+-------------------------------+
 | Masking-key (continued)       |          Payload Data         |
 +-------------------------------- - - - - - - - - - - - - - - - +
 :                     Payload Data continued ...                :
 + - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - +
 |                     Payload Data continued ...                |
 +---------------------------------------------------------------+
 ```

 我们自己写服务端和客户端测试一下：

```golang
func Main_WebSocketServer() {
    var tcpAddr *net.TCPAddr
    ConnMap = make(map[string]*net.TCPConn)
    tcpAddr, _ = net.ResolveTCPAddr("tcp", "0.0.0.0:9999")

    tcpListener, _ := net.ListenTCP("tcp", tcpAddr)

    defer tcpListener.Close()

    for {
        tcpConn, err := tcpListener.AcceptTCP()
        if err != nil {
            continue
        }
        fmt.Println("A client connected : " + tcpConn.RemoteAddr().String())
        // 把客户端请求全文打印出来
        reader := bufio.NewReader(tcpConn)
        for {
            message, err := reader.ReadString('\n')
            if err != nil {
                return
            }
            fmt.Println(string(message))
        }
    }
}
```

然后在浏览器中执行：

```js
var ws = new WebSocket("ws://192.168.1.213:9999");
```

可以看到服务端输出：

```text
A client connected : 192.168.1.213:63930

GET / HTTP/1.1

Host: 192.168.1.213:9999

Connection: Upgrade

Pragma: no-cache

Cache-Control: no-cache

Upgrade: websocket

Origin: chrome-search://local-ntp

Sec-WebSocket-Version: 13

User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.100 Safari/537.36

Accept-Encoding: gzip, deflate

Accept-Language: en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7

Sec-WebSocket-Key: /v/rffaLbTlI2yvDFqxIaQ==

Sec-WebSocket-Extensions: permessage-deflate; client_max_window_bits
```

另外，如果我们使用`wss`连接，我们会看到服务器输出一堆乱码，因为传输的内容被加密了：

```js
var ws = new WebSocket("wss://192.168.1.213:9999");
```

## 初步实现：`net/websocket`包的使用

那么我们开始进入主题。前面是用`http`包建立的简陋服务器，
接下来我们首先来看一下`net/websocket`是否好用。
我们写一个最基本服务器端，功能是建立`websocket`后，将客户端所有的消息打印出来：

```golang
package learnWeb

import (
    "fmt"
    "golang.org/x/net/websocket"
    "net/http"
)

func Main_WebSocketServer() {
    http.Handle("/echo", websocket.Handler(wssHandler))
    http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
        http.ServeFile(writer,request,"C:/Users/lewin/github/Learning/Golang/src/learnWeb/20190710-WebSocketClient.html")
    })
    http.ListenAndServe("0.0.0.0:9999",nil)
}

func wssHandler(conn *websocket.Conn)  {
    fmt.Println("connect +1 ")
    defer func() {
        fmt.Println("connect closed: ")
    }()
    
    rec := make([]byte, 4096)
    var n int
    var err error
    for {
        n, err = conn.Read(rec)
        if err != nil {
            break
        }
        if n > 0 {
            fmt.Println(string(rec[:n]))
            wssBroadcast(rec[:n])
        }
    }
}
```

在上面的代码中，我们依然使用`http`包去监听`http请求`，
但是使用了`websocket.Handler`来建立`WebSocket`连接。
它们会帮我们处理好线程的问题，我们这里就不多费心了。

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Test WebSocket</title>
</head>
<body>

<input id="input" type="text"/>
<button onclick="send()">Send</button>
<pre id="output"></pre>


<script>
    var input = document.getElementById("input");
    var output = document.getElementById("output");
    var wss = new WebSocket("ws://192.168.1.213:9999/echo");

    wss.onopen = function (evt) {
        output.innerHTML += "Status: Connected\n";
    };

    wss.onmessage = function (evt) {
        output.innerHTML += "Server: " + evt.data + "\n";
    };

    wss.onclose = function (evt) {
        output.innerHTML += "Connection closed.\n";
    };

    function send() {
        wss.send(input.value);
        input.value = "";
    }
</script>
</body>
</html>
```

配合上面这个html页面，我们就可以确认前端与后端的Websocket是否连接正常。

## 升级`gin-gonic/gin`+`gorilla/websocket`的技术组合

我们一般要写api的话，首先想到的应该是`gin`之类的框架吧，功能比较多，有备无患嘛。
但`gin`本身是没有`websocket`的处理机制的，所以我们引入另一个专业的框架`gorilla/websocket`。
这次我们写一个复读机服务器：

路由部分（其实主页面我用`Django`实现了，不过还是把这部分贴出来便于理解）：

```golang
{
    g := root.Group("/chatroom")
    {
        g.GET("", chatroom.Hall)
        g.GET("/ws", chatroom.WebSocket)
    }
}
```

控制部分：

```golang
package chatroom

import (
    "fmt"
    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
    "net/http"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func (r *http.Request) bool { // 一定要这个函数，否则报错
        return true
    },
}

func Hall(ctx *gin.Context) {  // 聊天室大厅页面
    ctx.HTML(http.StatusOK, "chatroom.html", nil)
}

func WebSocket(ctx *gin.Context) {  // websocket响应
    conn, _ := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
    wsHandler(conn)
}

func wsHandler(conn *websocket.Conn) {
    fmt.Println("connect +1 ")
    defer func() {
        fmt.Println("connect closed: ")
    }()

    for {
        msgType, msg, err := conn.ReadMessage()
        if err != nil {
            break
        }
        // Print the message to the console
        fmt.Printf("%s sent: %s\n", conn.RemoteAddr(), string(msg))

        // Write message back to browser
        if err = conn.WriteMessage(msgType, msg); err != nil {
            return
        }
    }
}
```

实现效果：

![复读机版本效果图](/static/blog/2019-07-10-WebSocket-SimpleEcho.png)



## 完善聊天业务逻辑

### 同时服务多位在线用户

既然要面对多个用户，那我们在建立连接的时候就分别使用`Go程`来单独监听：

```golang
func WebSocket(ctx *gin.Context) {
    conn, _ := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
    go wsListener(conn) // 启动
}

func wsListener(conn *websocket.Conn) {
    fmt.Println("connect: ", conn.RemoteAddr())
    broadcaster.add(conn)
    defer func() {
        fmt.Println("stop: ", conn.RemoteAddr())
        broadcaster.del(conn)
    }()

    for {
        _, msg, err := conn.ReadMessage()
        if err != nil { // 客户端断开连接后会跳出
            break
        }
        broadcaster.msgChan <- msg // 消息存入带缓存的chan中
    }
}
```

我们把收到的消息存入（带缓存的）的`chan`中，以保证一定的性能，
而`broadcaster`是一个由单独`Go程`运行的广播员，负责从`chan`中读取消息，然后广播出去。

```golang
var broadcaster = &broadCaster{msgChan: make(chan []byte, 10), wsConns: make(map[net.Addr]*websocket.Conn)}

type broadCaster struct {
    msgChan chan []byte
    wsConns map[net.Addr]*websocket.Conn
    lock    sync.Mutex
}

func (self *broadCaster) add(conn *websocket.Conn) {
    self.lock.Lock()
    self.wsConns[conn.RemoteAddr()] = conn
    self.lock.Unlock()
}

func (self *broadCaster) del(conn *websocket.Conn) {
    self.lock.Lock()
    delete(self.wsConns, conn.RemoteAddr())
    self.lock.Unlock()
}

func (self *broadCaster) say() {
    for {
        msg := <-self.msgChan
        self.lock.Lock()
        for _,conn := range self.wsConns{
            e := conn.WriteMessage(1, msg)
            if e !=nil{
                fmt.Println("Failed sending to: ", conn.RemoteAddr(), " | Err: ", e)
            }
        }
        self.lock.Unlock()
    }
}

func init() { // 使用init实现单例模式
    go broadcaster.say()
}

```

### 用户身份验证系统

我们知道`gin`有一个`BasicAuth`中间件，可以实现简单的用户认证功能。
但是我这个网站的主体还是`Django`，其中有一个自己写的用户认证中间件，虽然代码不多，但是功能强大。
然后二者之间要协作，肯定需要一个中间人了，我选择`Redis`。

我的思路是：

1. 聊天室主要页面建立在`Django`上。
2. 通过`Django`的逻辑生成`token`，一份附在js代码中送给客户，一份存入服务器的`Redis`数据库供查验。
3. 聊天室主要页面通过调用js代码，与`gin`建立`websocket`连接。

#### 生成`token`

这部分是在`python`环境下完成的：

```python
@wrap_login_required  # 在装饰器内，利用session核验身份
def chatRoom(request: WSGIRequest):
    # 生成token并存入数据库
    userName = request.session.get("user_name", "")
    chatToken = 生成密码()
    rd = redis.Redis(connection_pool=RedisPool)
    rd.set("ChatroomToken" + userName, chatToken, ex=300)
    # 把token代入HTML模板中
    return render(request, "appDevelop/chatroom.html", {"chatToken": chatToken, "userName": userName})
```

#### 使用`token`建立连接

这部分是`golang`环境的工作，先修改`路由`：

```golang
{
    g = root.Group("/chatroom")
    {
        g.GET("", chatroom.Hall)
        g.GET("/ws/:name/:token", chatroom.WebSocket)  // 改这里，识别参数
    }
}
```

然后修改`Handler`：

```golang
func WebSocket(ctx *gin.Context) {
    // 从url中读取参数
    token := ctx.Param("token")
    name := ctx.Param("name")
    // 从Redis中读取，并对比
    token2, err := rd.Get("ChatroomToken"+name).Result()
    if err == redis.Nil || token!=token2 {
        ctx.String(403,"token wrong.")
        return
    } else if err != nil {
        ctx.Status(500)
        fmt.Println(err)
        return
    }
    // 这些部分不用改
    conn, _ := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
    go wsListener(conn)
}
```

实现效果：

![UserModule1](/static/blog/2019-07-10-WebSocket-UserModule1.png)
![UserModule2](/static/blog/2019-07-10-WebSocket-UserModule2.png)

### 前端美化，增加用户名，部署到生产环境（Nginx代理）

`Nginx`代理配置：

```conf
location /gin {
        proxy_pass http://bloggin:9999;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
```

`html`部分代码：

```html
<div class="container" style="min-height: 100%">

        <textarea class="col-12" id="output" rows="25" readonly="readonly"></textarea>

        <textarea class="form-control col-11" id="input" rows="6" onkeyup="keySend();"
                  style="position: relative;"></textarea>
        <button class="float-right" id="buttonSend" onclick="send()">Send</button>
    </div>
<script>
        var input = document.getElementById("input");
        var output = document.getElementById("output");
        var wss = new WebSocket("ws://www.lewinblog.com/gin/chatroom/ws/{{ userName }}/{{ chatToken }}");

        wss.onopen = function (evt) {
            output.value += "[提示] 成功连接服务器\n";
            output.scrollTop = output.scrollHeight
        };

        wss.onmessage = function (evt) {
            output.value += evt.data + "\n";
            output.scrollTop = output.scrollHeight
        };

        wss.onclose = function (evt) {
            output.value += "[提示] 与服务器断开连接，请刷新……\n";
            output.scrollTop = output.scrollHeight
        };

        function send() {
            wss.send(input.value);
            input.value = "";
        }

        function keySend() {
            if (event.keyCode == 13)  //回车键的键值为13
                buttonSend.click();
        }
    </script>
```

在Windows上交叉编译Linux：

```shell
# PowerShell环境
SET CGO_ENABLED=0
SET GOOS=linux
SET GOARCH=amd64
go build BlogGin.go
```

最终效果：

![Fianl Version](/static/blog/2019-07-10-WebSocket-FinalVersion.png)

## 收获&不足

1. 整体来说这是一个很综合性的项目。前端涉及`html`/`css`/`js`/`Bootstrap`/`jQuery`等诸多技术，
后端是`Nginx`+`uWSGI+Django`+`Gin`三大模块，把它们糅合在一起有点费劲，但是也的确学到很多东西。
2. `Nginx`还有很多配置选项还没有完全弄懂，感觉很多是涉及到http协议的东西，不是纯粹的编程逻辑了。
3. 网站的主体我目前觉得还是`Django`更强大一些，毕竟发展那么久，用户那么多，
再加上动态脚本语言的特性，所以依然是`Web`应用的最佳选择。
4. `Golang`的部署真的超方便，真的是丢一个`.exe`到服务器就ok了。所以坚决地抛弃各种模板文件，
只把它用作纯api服务，才能最大限度地发挥它这项特长。
5. `Golang`令人称道的工程管理优势，今天觉得感觉一般。当业务逻辑复杂了之后，
依然会出现各种问题，它还是不能从根本上杜绝这些问题。

就这个项目来说，其实还有很多未完善之处：

1. 最明显的是掉线的问题了。还不清楚是`Websocket`本身的问题还是网络波动的问题，连接并不算很稳定。
掉线期间可能会有消息丢失，目前没有做检测、重连、重发的机制。
    > 解答：`Nginx`可能有一些计时机制，阻止了长时间的连接。
2. 用户信息太简单。写代码的时候偷懒，直接用`map[net.Addr]*websocket.Conn`储存用户信息，
导致后期想拓展都不太容易了。所以以后涉及到这类数据，都提早建立好`struct`才比较容易拓展吧。
3. `Redis`在`Golang`中的包，没有连接池`ConnectionPool`的概念。到底是不在乎？还是没必要？
还是真的没实现出来？这个问题暂时不得而知。
    > 解答：在代码注释中发现`Client`对象就是池子，我们直接使用`NewClient()`获取的客户端对象就是线程安全的。

## 参考文献

> 参考1：[用Golang实现 文本广播式聊天服务器/客户端](https://victoriest.gitbooks.io/golang-tcp-server/content/chapter3.html)  
> 参考2：[WebSocket 教程 - 阮一峰的网络日志](http://www.ruanyifeng.com/blog/2017/05/websocket.html)
