```yaml lw-blog-meta
title: "[TheGoBlog] Command PATH security in Go"
date: "2021-01-25"
brev: "发布时间2021-01-19，一个安全小知识get"
tags: [Golang]
```

# Command PATH security in Go

[原文链接](https://blog.golang.org/path-security)

Russ Cox  
19 January 2021

今天发布的安全补丁（`1.15.7`和`1.14.14`）修复了一个`go get`命令在不受信任的目录下进行查找时，可能导致远程执行的漏洞。我们希望人们知道这具体意味着什么，并且清楚这是否会影响到他们的代码。这篇文章详细描述了BUG，我们的补救措施，如何评估你的程序是否被波及，以及你可以做什么去补救。

## Go命令 与 远程执行

在一开始就是这样设计的：大部分`go`命令，包括`go build`, `go doc`, `go get`, `go install` 和 `go list`，它们**不会**执行从网络上下载的任意代码。然而，除此之外，`go run`, `go test` 和 `go generate` 则**会**运行任意代码——这是它们的使命。所以现在，当`go get`被发现可以执行任意代码时，我们认为这是一个严重的BUG。

如果我们要求`go get`不能执行任意代码，那么不幸的是，任何它所调用的程序，例如编译器和版本控制系统，都必须在安全的考虑范围之内。我们曾经就发现过编译器和版本控制系统的bug会成为go的bug。

然而，这次的BUG，是完完全全我们自己的问题，不能甩锅给`gcc`或者`git`了。

## Commands and PATHs and Go

所有的操作系统都有一个「可执行路径`executable path`」的概念（例如Unix的`$PATH`，Windows的`%PATH%`，以下简称为`PATH`），它会是一个目录的列表。当你在命令提示符中输入一个命令，shell会在这些目录中按顺序地寻找你输入的那个命令对应的可执行文件；如果找到了就执行第一个找到的，找不到就返回提示。

在Unix中，这个概念第一次被提出是在1979年，文档中这样解释：

> The shell parameter $PATH defines the search path for the directory containing the command. Each alternative directory name is separated by a colon (:). The default path is :/bin:/usr/bin. If the command name contains a / then the search path is not used. Otherwise, each directory in the path is searched for an executable file.

> shell参数`$PATH`定义了命令程序的搜寻路径。每个目录由`:`符号分隔，默认值是`:/bin:/usr/bin`。如果命令名称中包含`/`则不会使用这些搜寻路径；否则会在所有这些路径中搜索。

注意默认值：当前路径（是写在最前面的**空字符串**，让我们称其为`dot`）是在`/bin`和`/usr/bin`的前面的。而在MS-DOS和Windows中则是硬编码强制先搜索当前路径。

有一篇[经典论文](https://people.engr.ncsu.edu/gjin2/Classes/246/Spring2019/Security.pdf) 证明过，把dot放在PATH前面是很危险的。例如，你`cd`到一个路径下然后准备执行`ls`，这是一个很普通的操作，但是假如当前路径下也有一个名叫`ls`的可执行文件，那么你就会执行到这个可执行文件。如果它是恶意的，那么你就惨了。正因为如此，后续的Unix系统都把这个最前面的dot去除掉了，只有Windows还保留着。

在Go里，PATH的搜索工作是由`exec.LookPath`执行的，而后者是被`exec.Command`调用的。为了适应不同平台，`exec.LookPath`在Unix中按Unix的规则，在Windows中按Windows的规则。例如，这条命令：

```go
out, err := exec.Command("go", "version").CombinedOutput()
```

的行为会跟你直接在操作系统shell中执行的完全一样。也就是说，在Windows中，它会先尝试执行`.\go.exe`。

> 在Powershell中改掉了这个毛病，不再优先搜索当前路径。而在 cmd.exe 和 Windows C library 中依然保留。所以Go选择与cmd保持一致。）

## The Bug

当`go get`下载并且构建了一个包，包含`import C`的包，它会运行`cgo`去把相应的C代码编译为Go等价代码。注意，`go`命令会在那个包的源码目录下调用`cgo`。并且当`cgo`生成出了Go输出文件之后，`go`命令会调用Go编译器去处理刚才生成的那些Go文件，以及调用宿主机的C编译器（gcc或者clang）去构建C代码。

那么问题一：`go`命令去哪里寻找宿主机的C编译器？——答案当然是通过PATH去寻找。幸运的是，虽然是在包的源码目录下调用C编译器，但是PATH搜索的起点是在`go`命令被调用的地方：

```go
cmd := exec.Command("gcc", "file.c")
cmd.Dir = "badpkg"
cmd.Run()
```

所以，即使有`badpkg\gcc.exe`文件存在，它也不会被执行。

问题二：`go`命令去哪里找`cgo`？——它用的是从GOROOT计算得来的路径，这比前面更安全：

```go
cmd := exec.Command(GOROOT+"/pkg/tool/"+GOOS_GOARCH+"/cgo", "file.go")
cmd.Dir = "badpkg"
cmd.Run()
```

但是，问题三：`cgo`去哪里调用C编译器？——它要处理一些它创建的临时文件，意味着：

```go
// running in cgo in badpkg dir
cmd := exec.Command("gcc", "tmpfile.c")
cmd.Run()
```

现在，因为`cgo`是在`badpkg`源码目录下，因此Windows用户会执行到`badpkg\gcc.exe`而不是系统中那个正确的gcc，这就是漏洞所在。

Unix用户是安全的，因为，首先dot不在PATH中，其次源码目录中的可执行文件在默认情况下是没有执行权限的。但是，对于把dot加在PATH前面，并且使用GOPATH模式的用户，会收到与Windows用户相同的漏洞攻击。（如果你就是这样的，那今天就是个把dot从PATH中去掉的好日子！）

## The Fixes

所以问题的根源到底在哪里？

第一种答案，是cgo不应该在不受信任的目录下搜索可执行文件。如果是这样，那么解决方案就是让go命令给cgo指定绝对路径就好了。

第二种答案，是把锅甩给dot路径。用户可能会希望在命令提示符中直接使用当前路径下的可执行文件，而不希望子进程也在当前路径下搜索。那么解决方案就是在PATH搜索时忽略dot。

我们认为以上两种观点都是对的，于是我们对两点都做了修复。现在go命令会给cgo传递C编译器的完整路径。除此之外，cgo、go和go发行版中的其他命令现在都使用`os/exec`包的变体，如果它以前使用的是dot的可执行文件，那么就会报告错误。`go/build`和`go/import`也用相同的策略去调用go命令和其他工具。

出于谨慎考虑，我们同样对`goimports`和`gopls`，以及`golang.org/x/tools/go/analysis`和`golang.org/x/tools/go/packages`等调用go命令作为子进程的工具和库做了相同的修复。所以，接下来就去对Go和所有你用到的工具做一次升级吧。

## 你的程序会受影响吗？

如果你在你自己的代码中使用`exec.LookPath`或者`exec.Command`，那么只有当你（或者你的用户）可能会运行不受信任的代码时，你才需要关心这个安全漏洞。

如果你确实在乎这件事，请知悉：我们发布了一个新的严格的包`golang.org/x/sys/execabs`，把你原来的`os/exec`替换成它吧。

## 默认强化 os/exec

我们开了一个issue来讨论是否要改变在Windows下优先寻找dot这一行为。支持改变一方的理由是，改变之后可以杜绝今后类似的安全漏洞，而且Powershell抛弃了这个行为说明这个行为已经被承认是错误的了。反对改变的一方的理由是，它会带来 breaking change，会影响到那些特意想要在当前路径下搜索可执行文件的程序的逻辑，我们不知道有多少程序这样做了，但是如果我们改了那它们就可能会遭遇难以发现的异常了。

所以我们在前面说的`golang.org/x/sys/execabs`就是一个折衷的方案，当尝试执行到当前路径下的文件时会报错（假如你想调用一个叫prog的程序）：

```text
prog resolves to executable in current directory (.\prog.exe)
```

这有利于那些依赖dot特性的程序去发现到底发生了什么。另外，对于这些特意想要使用当前路径的程序，可以用`exec.Command("./prog")`去替换`exec.Command("prog")`，这样就可以确保执行到当前路径的文件，并且这在所有系统（包括Windows）都受到支持。

## 我的小结

虽然这篇文章是在讲安全的东西，不过我倒是意外学到调用子进程的方法。

然后还学到PATH这个冷门知识……
