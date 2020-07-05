```json lw-blog-meta
{"title":"[官方] Migrating to Go Modules","date":"2019-09-26","brev":"发布时间2019-08-21，接着上一篇，讲一下如何将现有的项目迁移到GoModules中。","tags":["Golang"],"path":"blog/2019/190926-官方-迁移到GoModules.md"}
```



# Migrating to Go Modules

[原始链接](https://blog.golang.org/migrating-to-go-modules)

Jean de Klerk  
21 August 2019

## Introduction

在Go语言早期发展历史中，有多个依赖管理工具，常见的有`dep`和`glide`等，他们的最大问题就是互相之间差距太大，不能兼容。因此Go官方设计了自己的依赖管理工具。

将`go modules`引入到你现有的项目时，如果你的项目已经打上了v2.0.0以上的版本号（即大版本号不是v1了），那么你需要将你的导入路径修改。

## 从其他依赖管理工具迁移

> 译者注，我暂时用不上，复制过来。

To convert a project that already uses a dependency management tool, run the following commands:

```shell
$ git clone https://github.com/my/project
[...]
$ cd project
$ cat Godeps/Godeps.json
{
    "ImportPath": "github.com/my/project",
    "GoVersion": "go1.12",
    "GodepVersion": "v80",
    "Deps": [
        {
            "ImportPath": "rsc.io/binaryregexp",
            "Comment": "v0.2.0-1-g545cabd",
            "Rev": "545cabda89ca36b48b8e681a30d9d769a30b3074"
        },
        {
            "ImportPath": "rsc.io/binaryregexp/syntax",
            "Comment": "v0.2.0-1-g545cabd",
            "Rev": "545cabda89ca36b48b8e681a30d9d769a30b3074"
        }
    ]
}
$ go mod init github.com/my/project
go: creating new go.mod: module github.com/my/project
go: copying requirements from Godeps/Godeps.json
$ cat go.mod
module github.com/my/project

go 1.12

require rsc.io/binaryregexp v0.2.1-0.20190524193500-545cabda89ca
$
```

`go mod init` creates a new go.mod file and automatically imports dependencies from `Godeps.json`, `Gopkg.lock`, or a number of other supported formats. The argument to `go mod init` is the module path, the location where the module may be found.

This is a good time to pause and run `go build ./...` and `go test ./...` before continuing. Later steps may modify your `go.mod` file, so if you prefer to take an iterative approach, this is the closest your `go.mod` file will be to your pre-modules dependency specification.

```shell
$ go mod tidy
go: downloading rsc.io/binaryregexp v0.2.1-0.20190524193500-545cabda89ca
go: extracting rsc.io/binaryregexp v0.2.1-0.20190524193500-545cabda89ca
$ cat go.sum
rsc.io/binaryregexp v0.2.1-0.20190524193500-545cabda89ca h1:FKXXXJ6G2bFoVe7hX3kEX6Izxw5ZKRH57DFBJmHCbkU=
rsc.io/binaryregexp v0.2.1-0.20190524193500-545cabda89ca/go.mod h1:qTv7/COck+e2FymRvadv62gMdZztPaShugOCi3I+8D8=
$
```

`go mod tidy` finds all the packages transitively imported by packages in your module. It adds new module requirements for packages not provided by any known module, and it removes requirements on modules that don't provide any imported packages. If a module provides packages that are only imported by projects that haven't migrated to modules yet, the module requirement will be marked with an `// indirect` comment. It is always good practice to run go mod tidy before committing a `go.mod` file to version control.

Let's finish by making sure the code builds and tests pass:

```shell
$ go build ./...
$ go test ./...
[...]
$
```

Note that other dependency managers may specify dependencies at the level of individual packages or entire repositories (not modules), and generally do not recognize the requirements specified in the `go.mod` files of dependencies. Consequently, you may not get exactly the same version of every package as before, and there's some risk of upgrading past breaking changes. Therefore, it's important to follow the above commands with an audit of the resulting dependencies. To do so, run

```shell
$ go list -m all
go: finding rsc.io/binaryregexp v0.2.1-0.20190524193500-545cabda89ca
github.com/my/project
rsc.io/binaryregexp v0.2.1-0.20190524193500-545cabda89ca
$
```

and compare the resulting versions with your old dependency management file to ensure that the selected versions are appropriate. If you find a version that wasn't what you wanted, you can find out why using `go mod why -m` and/or `go mod graph`, and upgrade or downgrade to the correct version using `go get`. (If the version you request is older than the version that was previously selected, `go get` will downgrade other dependencies as needed to maintain compatibility.) For example,

```shell
$ go mod why -m rsc.io/binaryregexp
[...]
$ go mod graph | grep rsc.io/binaryregexp
[...]
$ go get rsc.io/binaryregexp@v0.2.0
$
```

## 现有的项目没有依赖管理工具

从零开始很简单，使用`go mod init`：

```shell
$ git clone https://go.googlesource.com/blog
[...]
$ cd blog
$ go mod init golang.org/x/blog
go: creating new go.mod: module golang.org/x/blog
$ cat go.mod
module golang.org/x/blog

go 1.12
$
```

以前没有用过依赖管理的话，初始化的适合只有一条`module`和一个`go`版本的记录（如上所示）。在这个例子中，项目模块的路径设为了`golang.org/x/blog`，因为这是它自定义的导入路径（custom import path）。用户需要使用这个路径来导入项目中的包，所以必须要小心不要轻易改变它。

下一步，使用`go mod tidy`里将现有的依赖都添加到`go.mod`文件中：

```shell
$ go mod tidy
go: finding golang.org/x/website latest
go: finding gopkg.in/tomb.v2 latest
go: finding golang.org/x/net latest
go: finding golang.org/x/tools latest
go: downloading github.com/gorilla/context v1.1.1
go: downloading golang.org/x/tools v0.0.0-20190813214729-9dba7caff850
go: downloading golang.org/x/net v0.0.0-20190813141303-74dc4d7220e7
go: extracting github.com/gorilla/context v1.1.1
go: extracting golang.org/x/net v0.0.0-20190813141303-74dc4d7220e7
go: downloading gopkg.in/tomb.v2 v2.0.0-20161208151619-d5d1b5820637
go: extracting gopkg.in/tomb.v2 v2.0.0-20161208151619-d5d1b5820637
go: extracting golang.org/x/tools v0.0.0-20190813214729-9dba7caff850
go: downloading golang.org/x/website v0.0.0-20190809153340-86a7442ada7c
go: extracting golang.org/x/website v0.0.0-20190809153340-86a7442ada7c
$ cat go.mod
module golang.org/x/blog

go 1.12

require (
    github.com/gorilla/context v1.1.1
    golang.org/x/net v0.0.0-20190813141303-74dc4d7220e7
    golang.org/x/text v0.3.2
    golang.org/x/tools v0.0.0-20190813214729-9dba7caff850
    golang.org/x/website v0.0.0-20190809153340-86a7442ada7c
    gopkg.in/tomb.v2 v2.0.0-20161208151619-d5d1b5820637
)
$ cat go.sum
cloud.google.com/go v0.26.0/go.mod h1:aQUYkXzVsufM+DwF1aE+0xfcU+56JwCaLick0ClmMTw=
cloud.google.com/go v0.34.0/go.mod h1:aQUYkXzVsufM+DwF1aE+0xfcU+56JwCaLick0ClmMTw=
git.apache.org/thrift.git v0.0.0-20180902110319-2566ecd5d999/go.mod h1:fPE2ZNJGynbRyZ4dJvy6G277gSllfV2HJqblrnkyeyg=
git.apache.org/thrift.git v0.0.0-20181218151757-9b75e4fe745a/go.mod h1:fPE2ZNJGynbRyZ4dJvy6G277gSllfV2HJqblrnkyeyg=
github.com/beorn7/perks v0.0.0-20180321164747-3a771d992973/go.mod h1:Dwedo/Wpr24TaqPxmxbtue+5NUziq4I4S80YR8gNf3Q=
[...]
$
```

这个命令会将你所有导入过的包都添加进来。接下来看一下编译和测试：

```shell
$ go build ./...
$ go test ./...
ok      golang.org/x/blog    0.335s
?       golang.org/x/blog/content/appengine    [no test files]
ok      golang.org/x/blog/content/cover    0.040s
?       golang.org/x/blog/content/h2push/server    [no test files]
?       golang.org/x/blog/content/survey2016    [no test files]
?       golang.org/x/blog/content/survey2017    [no test files]
?       golang.org/x/blog/support/racy    [no test files]
$
```

## Tests in module mode

在迁移到`Go modules`后，有些测试代码可能需要调整。

如果一个测试需要在某个包目录中写入文件，那现在会失败，因为现在package目录在module缓存中，是只读的。进一步可能会导致后续的测试用例失败。现在应该将文件复制到一个临时目录中去。

如果一个测试通过相对路径来导入另一个module的文件，那么现在会失败，因为现在的module都是根据大版本号分别存放在子文件夹中，所以相对路径无效了。现在应该将这些文件复制到你的module中，或者将其转化并放在go代码文件中。

如果一个测试需要使用（GOPATH模式下的）go命令，那么现在会失败。现在应该手动添加一个`go.mod`文件，并显式设置`GO111MODULE=off`。

## Publishing a release

由于`Go modules`的版本机制，强烈建议你给你的release打上三级结构的tag，比如：

```shell
$ git tag v1.2.0
$ git push origin v1.2.0
```

再强调一次，同一个大版本之内，必须向后兼容。

## Imports and canonical module paths

每个module都会在go.mod文件中声明自己的路径，而每个导入的包都会以module路径作为前缀。但是，有些仓库可能有多个远程路径（remote import path），比如`golang.org/x/lint`和`github.com/golang/lint`都指向同一个位于`go.googlesource.com/lint`的仓库。由于在这个仓库中`go.mod`记录的是`golang.org/x/lint`，所以只有这个路径才是有效的。

`Go 1.4`提供了一种『规范导入路径』（canonical import paths），即通过`// import comments`这种语法来提示用户如何导入。但不是每个库都会提供这种规范，因此有些用户可能会使用着“不规范”的路径。以前可以不规范，但是使用`Go modules`之后就会报错了。在上面的例子中，你必须将你的`import github.com/golang/lint`改为`import golang.org/x/lint`。

还有一个问题是关于『非v1版本』的module。现在需要在导入路径后面加上大版本号。

## Conclusion

绝大多数用户可以直接过渡到`Go modules`，少数用户需要为不规范的导入和依赖的不兼容去操心。后续还会发布一些博客，讲解如何发布v2以上版本，以及如何处理奇怪的bug。
