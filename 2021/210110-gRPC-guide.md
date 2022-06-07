```yaml lw-blog-meta
title: "gRPC 入门教程"
date: "2021-01-10"
brev: "序列化无非是为了通信。尝试用Go,Python,Node三种语言分别快速实现RPC代码。"
tags: ["中间件"]
```

## 前言

本文以Golang为主。但是我觉得 [官方教程](https://grpc.io/docs/languages/go/basics/) 的例子太复杂了，要理解这个例子中的业务含义就足够费劲了，让人无法把精力集中在对 gPRC 本身的学习上。

因此类似 [我的Rabbitmq入门教程](../2020/200906-Rabbitmq入门.md) 一样，我自己来写教程用例，希望能写得更简单、更好理解一点。

## 主要思路

1. 定义 .proto 文件，写一对简单的Golang代码实现远程调用(RPC)；
2. 把上述CS架构替换为其他的语言(Python/Node)
3. 更新 .proto 文件，增加stream。

代码地址： [github](https://github.com/Saodd/learn-grpc)

## proto 定义

没有什么比"回声"更简单的了吧，这也是web框架的常见demo形式。

功能是，客户端给服务端发一个数据结构`Sentence`，服务端更新其中一个字段`speaker`后，将数据结构返回。并将这个远程函数命名为`Echo`：

```protobuf
// hello.proto
syntax = "proto3";
option go_package = "./hello_go"; // 编译为go需要这个，会创建子目录

service Chat {
  rpc Echo (Sentence) returns (Sentence);
}

message Sentence {
  string speaker = 1 ;
  string content = 2;
}
```

## Step1: Go的基础实现

### 准备工作

先要准备一个命令行工具`protoc`，找到[教程](https://developers.google.com/protocol-buffers/docs/downloads) 中的下载页面，选择对应的平台（我是win64）。下载得到zip文件后，自己解压缩，然后把bin目录添加到系统环境变量中，确认在终端中可以使用`protoc`工具。

按教程中get两个模块：

```shell
go get -u google.golang.org/protobuf/cmd/protoc-gen-go  google.golang.org/grpc/cmd/protoc-gen-go-grpc
```

之外，还要记得去主动更新grpc的依赖，否则会遇到`grpc.SupportPackageIsVersion7`不存在的尴尬局面：

```shell
go get -u google.golang.org/grpc
```

此时你的`go.mod`文件中应该包含：

```text
github.com/golang/protobuf v1.4.3
google.golang.org/grpc v1.34.0 // indirect
google.golang.org/protobuf v1.25.0
```

然后我们使用编译工具将 .proto 转化为 Go 代码：

```shell
protoc --go_out=. --go-grpc_out=. hello.proto
```

执行后，在`./hello_go`文件夹中就会出现`hello.pb.go`和`hello_grpc.pb.go`的文件，这两个文件挺复杂的，人类不好阅读，建议直接根据proto文件去想象我们有哪些东西可以用，然后借助IDE的提示来完成后续代码。（不过，如果想象不到的时候，还是得去这俩文件里去找，找大写开头的结构和方法。）

然后我们在`hello_go`文件夹下面分别建立两个子文件夹，`hello_go/server`和`hello_go/client`用于存放服务端和客户端的main代码。

### Go服务端代码

其实proto中的`service`完美对应着Golang中的`interface`，所以我们在服务端要做的事情很简单，就是写一个struct去实现那个接口就行。

```go
// hello_grpc.pb.go
type ChatServer interface {
    Echo(context.Context, *Sentence) (*Sentence, error)
    mustEmbedUnimplementedChatServer()
}
```

不过现实是残酷的，gRPC-go不允许我们自己实现接口，必须继承那个预置的struct然后重写需要用到的方法：

```go
// hello_grpc.pb.go
type UnimplementedChatServer struct {
}

func (UnimplementedChatServer) Echo(context.Context, *Sentence) (*Sentence, error) {
    return nil, status.Errorf(codes.Unimplemented, "method Echo not implemented")
}
func (UnimplementedChatServer) mustEmbedUnimplementedChatServer() {}
```

这里注意一下它的命名规则，我们在proto中定义的`service`叫`Chat`，所以在`hello_grpc.pb.go`中分别生成了`ChatClient`和`ChatServer`，我们继承和重写的时候根据这个名字去寻找就可以了。

于是开始写代码：

```go
type ChatServer struct {
	hello_go.ChatServer
}

func (s *ChatServer) Echo(ctx context.Context, sentence *hello_go.Sentence) (*hello_go.Sentence, error) {
    fmt.Println("收到: ", sentence)
	sentence.Speaker = "Lewin-Server"
	return sentence, nil
}
```

然后呢，监听tcp的部分也要我们自己写，虽然可以抄，但是还是得操心啊。主要流程：

1. 监听一个tcp;
2. 实例化一个grpc服务器对象；
3. 把需要用到的RPC服务对象，注册到grpc服务器对象中去；（类似gin中的注册路由）
4. 开始监听。

```go
func main() {
	lis, err := net.Listen("tcp", "localhost:5005")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	hello_go.RegisterChatServer(grpcServer, &ChatServer{})
	grpcServer.Serve(lis)
}
```

### Go客户端代码

结合上面的服务端代码，我们先猜想一下，客户端需要做哪些事情？

- 建立连接
- 调用服务

emmm似乎就是这么简单！开始做吧！

```go
func main() {
	// 1. 建立连接。这里可以配置很多选项，例如 TLS, JWT 等
	var opts = []grpc.DialOption{
		grpc.WithInsecure(), // 本地不安全连接必须指定这个
	}
	conn, err := grpc.Dial("localhost:5005", opts...)
	if err != nil {
		log.Fatal("连接失败: ", err)
	}
	defer conn.Close()

	// 2. 实例化一个专用客户端对象
	client := hello_go.NewChatClient(conn)
	// 3. 执行调用
	req := hello_go.Sentence{
		Speaker: "Lewin-Client",
		Content: "Hello, world!",
	}
	resp, err := client.Echo(context.Background(), &req)
	if err != nil {
		log.Fatalln("调用失败: ", err)
	}
	fmt.Println("收到回复: ", resp)
}
```

试着运行一下，一切顺利~

## step2-1: Python实现

准备工作：

- `pip install grpcio-tools`
- 前面准备好的`hello.proto`

### 编译

python有些不太一样，用的不是`protoc`，而是python内部的模块，并且参数名也有不同：

```shell
python -m grpc_tools.protoc -I. --python_out=hello_py --grpc_python_out=hello_py hello.proto
```

这样，就得到了`hello_py/hello_pb2.py`和`hello_py/hello_pb2_grpc.py`两个文件。

在grpc文件中，由于Python是面向对象的语言，因此函数代码会更整齐一些；但是在pb文件中，由于Python是动态类型语言，因此类型定义很难读。

> 由于Python没有类型，所以proto自己实现了一套类型的约束，所以也就别指望有IDE的提示了，自己瞎JB猜吧。  
> 写Python就是这样，能跑起来全看运气。（生气脸

### 客户端

我们先从容易的开始吧，客户端只需要调函数就可以了，在Python里也是很简单的：

```python
import grpc
import hello_pb2_grpc
import hello_pb2

if __name__ == '__main__':
    # 1. 建立连接，准备客户端
    channel = grpc.insecure_channel("localhost:5005")
    stub = hello_pb2_grpc.ChatStub(channel)

    # 2. 调用
    req = hello_pb2.Sentence(speaker="Lewin-Client-Python", content="Hello, world!")
    resp = stub.Echo(req)
    print(resp)
```

此时 localhost:5005 端口还运行着刚才写的Go的服务端。就这样，就实现了跨语言的RPC调用。

### 坑：编码问题

在上一篇 protobuf 教程中有讲到，它的 string 是utf-8编码的，而Python的默认编码并不是。所以会出现如下诡异现象：

```python
req = hello_pb2.Sentence(speaker="Lewin-Client-Python", content="Hello, world!!中文")
resp = stub.Echo(req)

print(resp)          # 打印: content: "Hello, world!!\344\270\255\346\226\207"
print(resp.content)  # 打印: Hello, world!!中文
```

我没有看源码，不过估计是在get装饰器上有个编码转化。但也正因为它的行为如此诡异，我们在传输中文时一定要特别小心。

### 服务端

服务端代码其实也挺好写的……（就是没有类型提示，万恶的动态类型

先继承并重写服务函数：

```python
class ChatServer(hello_pb2_grpc.ChatServicer):
    def Echo(self, sentence, context):
        print("收到: ", sentence)
        sentence.speaker = "Lewin-server-python"
        return sentence
```

然后监听端口，提供服务：

```python
import grpc
import hello_pb2_grpc
from concurrent import futures

if __name__ == '__main__':
    # 1. 服务器实例
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    # 2. 注册路由
    hello_pb2_grpc.add_ChatServicer_to_server(ChatServer(), server)
    # 3. 启动服务
    server.add_insecure_port('localhost:5006')
    server.start()
    server.wait_for_termination()
```

试着运行一下，现在可以尝试 py-py py-go go-py go-go 四种方式的互相调用了，一切顺利~

不过对于Python还需要提醒的是，Python并没有原生的异步调用，grpc在这里看起来是自己封装了一套`futures`的用法，但是看起来也有点诡异，所以还需要更仔细的研究才敢正式使用。

## step2-2: Node实现

Node 的部分更加奇葩，它允许你在运行时直接加载`.proto`文件，而不需要预先生成……

好吧你NB，但是我选择静态生成。

### 准备&编译

先安装全局的编译工具，然后编译：

```shell
npm install -g grpc-tools

grpc_tools_node_protoc --js_out=import_style=commonjs,binary:hello_node --grpc_out=grpc_js:hello_node hello.proto
```

编译后我们得到了`hello_node/hello_pb.js`和`hello_node/hello_grpc_pb.js`两个文件，分别对应着数据结构和服务。

为了后续的运行，我们还需要安装一些依赖，直接写到`package.json`里然后去`install`吧：

```json
{
  "dependencies": {
    "@grpc/grpc-js": "^1.2.3",
    "@grpc/proto-loader": "^0.5.5",
    "google-protobuf": "^3.14.0",
    "grpc": "^1.24.4",
    "grpc-tools": "^1.10.0"
  }
}
```

### Node客户端

此时我们先用着之前写的Go服务端，然后来写Node客户端。

由于教程上都是动态加载的方式，因此下面使用静态代码的代码是我自己写的：

```js
const messages = require('./hello_pb');
const services = require('./hello_grpc_pb');
const grpc = require('@grpc/grpc-js');

function main() {
    // 1. 建立连接
    const client = new services.ChatClient("localhost:5005", grpc.credentials.createInsecure())

    // 2. 准备请求数据。
    // 这里很恐怖的是，初始化数据居然是通过Array传入的……所以强烈建议使用setXXX方法来准备参数……
    const req = new messages.Sentence(["Lewin-Client-Node", "Hello, world!"])
    req.setContent("Hello, 世界!")
    console.log(`发送: ${req.getSpeaker()} | ${req.getContent()}`)

    // 3. 执行调用。
    client.echo(req, function (err, resp) {
        console.log(`结果: ${err} | ${resp.getSpeaker()} | ${resp.getContent()}`)
    })
}
```

这里有一些东西让我不太舒服：

第一，依然没有类型的代码提示。而且需要再次强调的是，在 new 一个 message 的时候，传入的初始参数居然是 Array …… 也就是说，如果不小心搞错了顺序，救都救不回来。因此务必使用`setXXX`的方法来设置参数。

第二，不支持 async/await 的调用方式（即 Promise 的写法）。也许是我编译参数给的不对？或者用`promisify`可以包装一下吗？暂时还不确定。

总之姑且是跑起来了~

### Node服务端

服务端依然是那几个步骤：实现服务函数、初始化服务器、注册服务路由、监听端口：

```js
const echo = (call, callback) => {
    console.log(`收到: ${call.request.getSpeaker()} | ${call.request.getContent()}`)
    call.request.setSpeaker("Lewin-Server-Node")
    callback(null, call.request)
}

function main() {
    const server = new grpc.Server();
    server.addService(services.ChatService, {echo})
    server.bindAsync('0.0.0.0:5007', grpc.ServerCredentials.createInsecure(), () => {
        server.start();
    });
}
```

然后这里又遇到了奇怪的行为，即，服务函数的参数，是通过`call.request`去访问的，对，不论参数是什么类型，都是用`request`去访问。这个行为让我觉得很怪异。

然后是在注册服务路由的环节，IDE一直提示我上面的`echo`函数不符合它的类型定义……可我明明是参考官方示例来抄的，没有做什么奇怪的事情啊……很怪异。

罢了，总之姑且是跑起来了~ 现在我可以实现 py, node, go 之前的任意互相调用了。

## step3: stream类型

其实也没有什么特别的。就不详细展开了。

简单说，在Golang中，就是直接把它当作一个连接来处理，一端循环地`Send()`，另一端循环地`Recv()`直到`EOF`。

## 总结

首先必须吐槽一下， [grpc.io的文档](https://grpc.io/) 页面的加载速度实在是慢。简单排查了一下是jquery和bootstrap的CDN资源挂了（也许是我没出国），但是它们挂了也没影响页面的正常运行，估计是框架的锅？但是对于一个这么成熟的框架的文档来说，不应该出现这种事情。

然后，无论是`protobuf` 还是 `gPRC`，我感觉它们的文档都很不友好。特别是对 Python 和 Node 来说，让我感到一种“二等公民”的感觉。毕竟，给没有类型的语言增加类型约束这件事，本身就是很奇怪的行为（虽然不得不做）。

不过呢，总体来说还是挺满意的。 gRPC 给我打开了新世界的大门。

虽然以前我们用 HTTP 也同样是一种 RPC，理念是相似的。不过，在与 protobuf 结合之后，我发现原来在不同服务之间快速生成 RPC 逻辑是这么简单的事情。

俗话说，「量变产生质变」，当某种技术的复杂度降低到一定程度后，它能发挥的作用也会有一个质的飞跃。

我想今后我会把 gPRC 作为我个人默认的服务间通讯的最佳选择。
