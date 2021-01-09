```yaml lw-blog-meta
title: "gRPC 入门教程"
date: "2021-01-10"
brev: "序列化无非是为了通信"
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

## 准备工作

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

## Step1: Go的基础实现

没有什么比回声更简单的了吧，这也是web框架的常见demo形式。

功能是，客户端给服务端发一个数据结构`Sentence`，服务端更新其中一个字段`speaker`后，将数据结构返回。并将这个远程函数命名为`Echo`：

```protobuf
// hello.proto
syntax = "proto3";

service Chat {
  rpc Echo (Sentence) returns (Sentence);
}

message Sentence {
  string speaker = 1 ;
  string content = 2;
}
```

然后我们使用编译工具将其转化为 Go 代码：

```shell
protoc --go_out=hello_go --go-grpc_out=hello_go hello.proto
```

在执行上述命令之前，要先手动建立一个文件夹叫`hello_go`。执行后，在那个文件夹中就会出现`hello.pb.go`和`hello_grpc.pb.go`的文件，这两个文件挺复杂的，人类不好阅读，建议直接根据proto文件去想象我们有哪些东西可以用，然后借助IDE的提示来完成后续代码。（不过，如果想象不到的时候，还是得去这俩文件里去找，找大写开头的结构和方法。）

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

未完待续……
