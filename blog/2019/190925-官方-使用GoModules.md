```json lw-blog-meta
{"title":"[官方] Using Go Modules","date":"2019-09-25","brev":"发布时间2019-03-19，因为近期发布了一篇《Migrating to Go Modules》与之关联，所以找出来看一下。","tags":["Golang"],"path":"blog/2019/190925-官方-使用GoModules.md"}
```



# Using Go Modules

[原始链接](https://blog.golang.org/using-go-modules)

Tyler Bui-Palsulich and Eno Compton  
19 March 2019

## Introduction

`Go 1.11`和`Go 1.12`包含了对`Go modules`模块的提前支持，它是Go的新的依赖管理系统。它让依赖库的版本更加易读并且更容易管理。

一个`module`就是一些package的集合，它存放在一个包含`go.mod`文件的目录下。`go.mod`文件定义了所有module的路径（也是引用路径），也包含了依赖的要求。每条依赖要求都包含路径和版本。

从`Go 1.11`开始，go命令就已经支持在包含go.mod文件的项目中运行modules工具。从`Go 1.13`开始，modules将会作为默认的开发选项。

接下来按照顺序，展示一下使用modules工具进行开发的常规流程：

## Creating a new module

在`$GOPATH`之外的地方创建一个空的文件夹，作为整个项目的根目录（因为传统Go项目需要GOPATH的支持，这里特意避开GOPATH）；并创建一个新的文件`hello.go`：

```go
package hello

func Hello() string {
    return "你好，世界。"
}
```

然后为其写个测试文件：

```go
package hello

import "testing"

func TestHello(t *testing.T) {
    want := "你好，世界。"
    if got := Hello(); got != want {
        t.Errorf("Hello() = %q, want %q", got, want)
    }
}
```

此时，这个目录下有一个package，但是不是一个module，因为这里没有`go.mod`文件。我们运行测试会看见：

```text
PS C:\Users\lewin\mycode\learn-go-modules> go test
PASS
ok      _/C_/Users/lewin/mycode/learn-go-modules        0.447s
```

因为我们在GOPATH之外运行，所以会假设一个虚拟的路径。

接下来我们将其转化为一个module，我们使用`go mod init`，然后重新运行测试：

```text
PS C:\Users\lewin\mycode\learn-go-modules> go mod init example.com/learn-go-modules
go: creating new go.mod: module example.com/learn-go-modules
PS C:\Users\lewin\mycode\learn-go-modules> go test
PASS
ok      example.com/learn-go-modules    0.454s
```

恭喜！这样你就创建了自己的module并进行了测试。`go.mod`文件现在长这样：

```text
module example.com/learn-go-modules

go 1.13
```

## Adding a dependency

我们引入一个外部的模块：

```go
package hello

import "rsc.io/quote"

func Hello() string {
    return quote.Hello()
}
```

```shell-session
$ go test
go: finding rsc.io/quote v1.5.2
go: downloading rsc.io/quote v1.5.2
go: extracting rsc.io/quote v1.5.2
go: finding rsc.io/sampler v1.3.0
go: finding golang.org/x/text v0.0.0-20170915032832-14c0d48ead0c
go: downloading rsc.io/sampler v1.3.0
go: extracting rsc.io/sampler v1.3.0
go: downloading golang.org/x/text v0.0.0-20170915032832-14c0d48ead0c
go: extracting golang.org/x/text v0.0.0-20170915032832-14c0d48ead0c
PASS
ok      example.com/hello    0.023s
$
```

go命令会解析所有的导入项（根据go.mod中定义的），如果有不存在于`go.mod`文件中的导入项，就会自动寻找并将其导入`go.mod`文件，并使用最新版本。

在我们的例子中，我们使用`go test`命令，自然也会触发上述过程。

## Upgrading dependencies

在Go的世界观中，版本号由三个部分组成：主要版本、次要版本、补丁版本，比如『v0.1.2』这个样子。

我们列出现在所有的modules：

```shell-session
$ go list -m all
example.com/learn-go-modules
golang.org/x/text v0.0.0-20170915032832-14c0d48ead0c
rsc.io/quote v1.5.2
rsc.io/sampler v1.3.0
```

可以看到golang.org/x/text这个模块的版本号有些混乱，这是一个`go modules`生成的虚拟版本号。我们可以手动更新一下它：

```shell-session
$ go get golang.org/x/text
go: finding golang.org/x/text v0.3.2
go: downloading golang.org/x/text v0.3.2
go: extracting golang.org/x/text v0.3.2

$ go list -m all
go: finding golang.org/x/tools v0.0.0-20180917221912-90fa682c2a6e
example.com/learn-go-modules
golang.org/x/text v0.3.2
golang.org/x/tools v0.0.0-20180917221912-90fa682c2a6e
rsc.io/quote v1.5.2
rsc.io/sampler v1.3.0
```

那如果更新之后出了BUG（新版本不兼容），就像这样：

```shell-session
$ go test
PASS
ok      example.com/learn-go-modules    0.613s

$ go get rsc.io/sampler
go: finding rsc.io/sampler v1.99.99
go: downloading rsc.io/sampler v1.99.99
go: extracting rsc.io/sampler v1.99.99

$ go test
--- FAIL: TestHello (0.00s)
    hello_test.go:8: Hello() = "99 bottles of beer on the wall, 99 bottles of beer, ...", want "你好，世界。"
FAIL
exit status 1
FAIL    example.com/learn-go-modules    0.420s
```

怎么办？我们可以手动指定版本号来下载（并且尝试）：

```shell-session
$ go list -m -versions rsc.io/sampler
rsc.io/sampler v1.0.0 v1.2.0 v1.2.1 v1.3.0 v1.3.1 v1.99.99

$ go get rsc.io/sampler@v1.3.1
go: finding rsc.io/sampler v1.3.1
go: downloading rsc.io/sampler v1.3.1
go: extracting rsc.io/sampler v1.3.1

$ go test
PASS
ok      example.com/learn-go-modules    0.514s
```

注意上面的`@v1.3.1`是用来指定版本号的，如果不指定，默认值是`@latest`。

## Adding a dependency on a new major version

在Go的世界观中，不同的**大版本号**的路径是不同的。上面的`rsc.io/quote`指的是v1版本的，因此`go get`只会拉取这个大版本下的最新的小版本。如果我们要更新大版本，必须显式地更改其导入路径为`rsc.io/quote/v3`这样。

> 译者注：即通过这种方式，要求库开发者必须保证同一个大版本内前后兼容；如果有不兼容的内容，那么必须放在另一个大版本中，而且通过不同的路径进行导入。  
> 个人认为这种理念挺好的，不过毕竟不是每个开发者都是Go语言出身，所以想必在相当长的时间内都会因为这种政策而导致撕逼事件吧哈哈，直到这种理念真正得到大多数人的认可。

将之前的`hello.go`新增一个不同大版本的依赖项：

```go
package hello

import (
    "rsc.io/quote"
    quoteV3 "rsc.io/quote/v3"
)

func Hello() string {
    return quote.Hello()
}

func Proverb() string {
    return quoteV3.Concurrency()
}
```

并且增加测试代码：

```go
func TestProverb(t *testing.T) {
    want := "Concurrency is not parallelism."
    if got := Proverb(); got != want {
        t.Errorf("Proverb() = %q, want %q", got, want)
    }
}
```

然后执行，然后看一下现在的依赖项版本：

```shell-session
$ go test
go: finding rsc.io/quote/v3 v3.1.0
go: downloading rsc.io/quote/v3 v3.1.0
go: extracting rsc.io/quote/v3 v3.1.0
PASS
ok      example.com/learn-go-modules    0.501s

$ go list -m rsc.io/q...
rsc.io/quote v1.5.2
rsc.io/quote/v3 v3.1.0
```

## Upgrading a dependency to a new major version

接下来，我们可能需要逐步将旧版本的代码更新，使其使用新版本的依赖。这一步操作很简单，我们无需任何额外操作。

将`hello.go`文件中旧版本代码更新到新版本（v3版本使用的函数是helloV3）：

```go
package hello

import "rsc.io/quote/v3"

func Hello() string {
    return quote.HelloV3()  // 这里升级为最新的用法
}

func Proverb() string {
    return quote.Concurrency()
}
```

## Removing unused dependencies

删除某项依赖必须要手动执行。我们使用`go mod tidy`命令：

```shell-session
$ go mod tidy
$ go list -m all
example.com/learn-go-modules
golang.org/x/text v0.3.2
golang.org/x/tools v0.0.0-20180917221912-90fa682c2a6e
rsc.io/quote/v3 v3.1.0
rsc.io/sampler v1.3.1
```

## Conclusion

复习一下，我们学了这几个命令的用法：

- `go mod init`
- `go build`, `go test`
- `go list -m all`
- `go get`
- `go mod tidy`

## 我的小结

这样看来，Go语言还是敢于大刀阔斧地改革，这次又在包管理机制上面提出了新的理念。

虽然还没有具体在项目中使用，不过我觉得这个理念非常合理，下次一定要尝尝鲜！
