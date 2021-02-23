```yaml lw-blog-meta
title: "proto3 入门教程"
date: "2021-01-09"
brev: "Protobuf 是一种高效的数据序列化技术。"
tags: ["中间件"]
```

## 前言

Protocol Buffer 是一个语言无关、平台无关的数据通信格式。

原文地址: [Language Guide (proto3)](https://developers.google.com/protocol-buffers/docs/proto3)

## 1. 定义一个消息类型

我们需要创建一个`.proto`文件。

假如你想定义一个搜索请求的消息格式，其中包含查询字符串、分页页码、分页大小三个内容，可以用protobuf定义如下：

```
syntax = "proto3";

message SearchRequest {
  string query = 1;
  int32 page_number = 2;
  int32 result_per_page = 3;
}
```

- 第一行表明了你正在使用`proto3`的语法，如果不指定的话则会被默认为是`proto2`，它必须是除了空行、除了注释行之外的第一行。
- `SearchRequest`这个「消息`message`」定义了三个字段（键值对），每个字段都有名字和类型。

### 字段类型 Field Types

上面的 int32 和 string 都是「纯量类型`scalar types`」。除此之外还有复合类型`composite types`。

### 字段编号 Field Numbers

注意到没，每个字段的定义后面还跟了一个唯一编号。这个编号是用来决定字段在二进制格式中的排列顺序的。因此不能随便更改。

需要注意的是，序号 1-15 需要占用1个字节（包括序号和类型），16-2047 则需要占用两个字节。因此尽量保留 1-15 序号给你最常用的字段。最好还要记得在序号中留出一些空档以方便未来添加字段。

序号可用值的范围是 1-536870911 （2的29次方减一），并且排除 19000-19999 这一段。

### 字段规则 Field Rules

- `singular`: 允许该字段出现0次或1次；（简单说就是可以没有）（这是protobuf3的默认字段）
- `repeated`: 允许该字段出现0次或任意次，有顺序。（简单说就是一个列表）

### 定义多个消息类型

可以在一个 `.proto` 文件中定义多个消息类型。这允许你把一些相关的消息类型放在一起。

### 添加注释 Comments

使用C语言的注释风格，即 `//` 和 `/* ... */`

### 保留字段 Reserved Fields

有时你可能还是会删除，或者临时注释掉某些字段。可是如果你的同事在不知情的情况下复用了你之前所使用的字段编号，那么可能会引发问题。

因此可以用保留字段。可以写序号，或者写字段名。

```
message Foo {
  reserved 2, 15, 9 to 11;
  reserved "foo", "bar";
}
```

### 我们把.proto文件生成了什么？

使用编译工具来处理`.proto`文件，会生成与你选择的语言对应的代码。

- 对于Go来说，会生成一个`.pb.go`文件，其中给你定义的每个protobuf消息类型都有对应的结构体。
- Python有些不太一样，它生成的是一些原类`metaclass`（用于继承）
- 其他语言省略。

## 类型对照表

protobuf中的类型 与 具体语言中的类型有一个固定的转换关系。

例如 double 在Golang 中会转换为 float64.

请 [前往官网查看](https://developers.google.com/protocol-buffers/docs/proto3#scalar)

## 默认值

如果不指定值，则会使用零值：

- string: 空string
- bytes: 空bytes
- bool: false
- numeric(数字类型): 0
- enums(枚举类型): 第一个枚举值，也就是0
- message: not set. （由各语言的规则决定）

## 枚举类型 Enumerations

用`enum`关键字，像这样：

```protobuf
syntax = "proto3";
message SearchRequest {
  enum Corpus {
    UNIVERSAL = 0;  // 这里是枚举值，且第一个必须是0
    WEB = 1;
    IMAGES = 2;
    LOCAL = 3;
    NEWS = 4;
  }
  Corpus corpus = 4;  // 注意这个4是字段序号，而不是枚举值！！
}
```

如果需要把多个名称赋给同一个枚举值（起到“别名”的效果），那么就指定`allow_alias`选项：

```protobuf
message MyMessage1 {
  enum EnumAllowingAlias {
    option allow_alias = true;
    UNKNOWN = 0;
    STARTED = 1;
    RUNNING = 1;
  }
}
```

枚举值的取值范围是int32，并且不能使用负数。

你可以把枚举定义在message里面。也可以放在外面，这样就可以被当前文件中的所有message所用。你也可以通过`_MessageType_._EnumType_`来访问写在另一个message内部的枚举类型。

在反序列化时，如果遇到了不能识别的枚举值（没有定义在proto文件中的值），则会把这个值留在结构体中，然后具体的表现由语言来决定。开放型枚举类型的语言，例如C++和Go，你可以直接访问到枚举值；而对于封闭型枚举类型的语言，例如Java，你可能需要一些特别的访问器。

在序列化时，不能识别的枚举值会被正常序列化进去。

可以给枚举值保留：

```protobuf
enum Foo {
  reserved 2, 15, 9 to 11, 40 to max;
  reserved "FOO", "BAR";
}
```

## message嵌套

你可以把一个message类型作为另一个message的字段：

```protobuf
message SearchResponse {
  repeated Result results = 1;
}

message Result {
  string url = 1;
  string title = 2;
  repeated string snippets = 3;
}
```

### 导入定义

（在除了Java之外的语言中）你可以使用另一个 .proto 文件中定义的 message.

使用`import`关键字。如果使用`import public`关键字，则会让下一级导入也能继续使用。

```protobuf
// one.proto
message Hello {
  string name = 1;
}
```

```protobuf
// two.proto
import pulic "one.proto";
import "zero.proto";
// 下面可以使用one.proto 中定义的 Hello 了
```

```protobuf
// three.proto
import "two.proto";
// 下面可以使用one.proto中的定义，但是不能使用`zero.proto`中的定义。
```

> 这里踩一个坑，在JetBrains家的IDE中，对于proto文件一般使用`Protocol Buffer Editor`这个插件。但是，这个插件不能直接识别import的proto文件。需要手动将要import的目录标记为`Sources Root`即可正确解析文件路径并且跳转。

### 直接嵌套 Nested

也可以直接把一个message写在另一个message里面。外面可以通过`a.b`的形式去访问。

```protobuf
message SearchResponse {
  message Result {
    string url = 1;
    string title = 2;
    repeated string snippets = 3;
  }
  repeated Result results = 1;
}
```

## 更新message定义

- 不要改变字段序号。
- 增加字段，直接增加就可以了。
- 删除字段，可以，但是你得保证你的代码没有用到被删除的字段。（并且建议你给它改名，或者用保留）
- （可以改名。）
- int32, unit32, int64, uint64, bool 之间可以相互转换。
- sint32, sint64 之间可以相互转换。
- string, bytes 之间可以相互转换，但是要求bytes是UTF-8.
- fixed32 兼容 sfixed32, fixed64 兼容 sfixed64.
- emun 可以兼容int32, uint32, int64, unit64, 但是有一些风险要注意。
- oneof

## 未知字段

有时可能接收方用的是旧的proto，收到新的proto序列化过的数据时，可能有部分字段对接收方来说是未知的。

proto3不支持，但是proto3.5支持。

## Any 类型

## Oneof 类型

## Maps 类型

## Packages

用于给一个 .proto 文件指定引用名称，避免多个 import 时的命名冲突。

## 服务 Services

```protobuf
service SearchService {
  rpc Search(SearchRequest) returns (SearchResponse);
}
```

详情请参考我的下一篇博客 [gRPC入门教程](../2021/210110-gRPC-guide.md)

## 转化为JSON

## 选项 Options

有文件级别的Option，也有message级别的，还有字段级别的，等等。

列举一些常用的：（看了下大多都是C++和Java的，都略过）

- `deprecated`: 在大多数语言中，都没有实际的效果，一般用作提示。（每种语言应该都有一套标注deprecated的规范，反正Golang是有的，可以被Goland正确识别并且显示。）

```protobuf
int32 old_field = 6 [deprecated = true];
```

## 编译

把 .proto 文件编译为你所需要的语言的代码。至少需要一个out选项，选择指定的语言，例如：

```shell
protoc --go_out=hello_go hello.proto
```
