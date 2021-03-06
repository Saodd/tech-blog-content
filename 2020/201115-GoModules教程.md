```yaml lw-blog-meta
title: "Go Modules 快速入门"
date: "2020-11-15"
brev: "已经很熟悉作为最下游的调包侠如何使用 Go Modules ，今天来研究一下作为上游的第三方库的编写者应该如何使用这套体系来创建规范的库供他人使用。"
tags: [Golang]
```

## 基本概念

- `Modules`: 指的是一整个第三方库，比如`github.com/gin-gonic/gin`，一个库中可以包含多个Package
- `Package`: 指的是一个 Go 文件夹目录，其中每个`.go`文件都要以`package xxx`开头，是我们项目代码的基本组织结构，不同的Package中调用其他Package的函数使用`xxx.SomeFunction()`这样
- 版本号：三级版本编号，大版本-小版本-补丁版本。 [参考官博3](https://blog.golang.org/publishing-go-modules) 
    + 同一个大版本中，后面的小版本必须兼容前面的小版本，不允许braking changes
    + 一般有新功能增加则使用小版本，只是修复则使用补丁版本。
    + 如果要废弃或者修改小版本中的接口，则考虑先将原接口标记为`deprecated`，然后把修改后的接口用一个新命名。在下一个大版本中再移除旧的接口。 [参考官博5](https://blog.golang.org/module-compatibility)

## 开始尝试

我们先写一个库，上传到github上，然后再起另一个项目，通过`Go modules`来调用前面那个库。

### 步骤一：模拟写一个"第三方库"

代码请看： [learn-go-modules-dep](https://github.com/Saodd/learn-go-modules-dep) ，或者随便写个`.go`文件随便写个函数也可。

代码写好了，我们还需要做一些步骤。

首先，给自己这个库命名，并初始化。注意，这个名称最好是你实际的Git项目网址：

```shell-session
$ go mod init github.com/Saodd/learn-go-modules-dep 
```

此时我们得到了`go.mod`文件，第一行写着你的 Module Name ， 这个路径名称是你在其他项目中调用这个库时使用的引用路径。

至于版本号，则是通过 Git 来天然实现的，与Golang 无关：

```shell-session
$ git add .
$ git commit -m "......"

$ git tag v1.0.0
$ git push origin v1.0.0
```

然后上 Github 查看一下项目代码以及Tag是否成功上传了。

### 步骤二：在另一个项目中调用我们的"库"

我们创建一个 Golang 项目文件夹，第一件事是初始化（并且记得在你的IDE中开启GoModules支持）：

```shell-session
$ go mod init xxxxxxx
```

然后引入我们前面刚刚上传的"库"，注意我们可以手动指定版本号：

```shell-session
$ go get -u github.com/Saodd/learn-go-modules-dep@v1.0.0
```

此时我们可以在`go.mod`文件中找到我们的"库"，以及我们所指定的版本了。同时，在`go.sum`文件中也会记录"库"代码的哈希值。

接下来我们在新项目中建立一个`main.go`随便写点代码，可以发现已经可以使用前面的"库"中的代码了。

### 步骤三：升级调用库的小版本

我们先给前面所写的"库"增加一个函数，然后打上新的标签，上传：

```shell-session
$ git add . && git commit -m "......"
$ git tag v1.1.0
$ git push origin v1.1.0
```

在调用这个库的项目中，我们可以选择直接编辑`go.mod`文件，修改版本号（修改之后IDE应该会自动识别版本并给你的代码做出相应的提示）；我们也可以重新get一下：

```shell-session
$ go get -u github.com/Saodd/learn-go-modules-dep@v1.1.0
```

如果我们不指定版本号，则会自动更新到最新的一个小版本上：

```shell-session
$ go get -u github.com/Saodd/learn-go-modules-dep@v1.0.0  # 恢复到v1.0.0版本
$ go get -u github.com/Saodd/learn-go-modules-dep  # 更新到最新的小版本号上，即v1.1.0版本
```

### 步骤四：升级调用库的大版本

这里特别要注意的是，原来的应用路径（模块名）`github.com/Saodd/learn-go-modules-dep`只能对应`v0`和`v1`版本，如果升级到`v2`及以后，官方推荐策略是在库代码中设置一个文件夹`v2`来存放这个版本的所有代码（ [参考官博4](https://blog.golang.org/v2-go-modules) ）.

新文件夹中也要包括一个新的`go.mod`文件，其第一行应该标注出这个版本的新引用路径`module github.com/Saodd/learn-go-modules-dep/v2`

```shell-session
$ mkdir v2 && cp *.go v2/ && 写代码
$ cp go.mod v2/go.mod && go mod edit -module module github.com/Saodd/learn-go-modules-dep/v2 v2/go.mod
$ git add . && git commit -m "......"

$ git tag v2.0.0
$ git push origin v2.0.0
```

在调用这个库的项目中，我们更新依赖时需要同时指定新的引用路径和新的版本号：

```shell-session
$ go get -u github.com/Saodd/learn-go-modules-dep/v2@v2.0.0
```

> 我的感言：Golang对于版本号的处理，看起来真的有点……奇怪……也就是说，v1版本的代码放在根目录，v2v3v4版本分别建立一个子文件夹然后在子文件夹内自成体系。
> 这个做法，可以解决"在一个项目中引用同一个库的多个版本"这个矛盾问题；但是对于库的开发者来说，这种写法非常非常的不优雅……

### 步骤五：调用一个私有库

回想一下`go.sum`这个东西，它保存着所有用过的库的哈希值。那么这个哈希值会跟谁去对比呢？——会跟Golang官方提供的一个公网数据库（称为`checksum database`）去对比。可是，对于我们的私有库来说，golang官方怎么可能记录私有库的哈希值呢？

因此，对于私有库，需要解决两个问题：

1. git 命令行访问私有库的权限
2. 屏蔽 checksum

对于问题一，我们需要的是git配置的知识。最好的办法是通过ssh来访问，而`go get`命令默认使用https去访问，所以我们需要给git配置一下路径改写，
参考： [官方 - Why does "go get" use HTTPS when cloning a repository?](https://golang.org/doc/faq#git_https)

对于问题二，我们需要给Golang配置一个环境变量，让它忽略掉我们指定域名下的git仓库的哈希值检验，
参考 [stackoverflow - Right way to get dependencies from private repository](https://stackoverflow.com/questions/60585302/right-way-to-get-dependencies-from-private-repository)
或者参考 [官方 - Environment variables](https://golang.org/cmd/go/#hdr-Environment_variables)

还有个小问题需要注意的是，必须要支持`go-import`协议的 Git 服务，才能够配合 go get 命令运行。（我们公司的`Gitea`实测可用，大名鼎鼎的`Gitlab`我猜是必须可用）

首先我在我们公司的私有Git服务上创建一个代码仓库，引用路径为`git.meideng.net/somebody/learn-go-modules-private`，写一些代码，然后打上标签`v1.0.0` 。

然后按上述步骤配置`~/.gitconfig`

```text
[url "ssh://git@git.meideng.net/"]
    insteadOf = https://git.meideng.net/
```

然后在指定环境变量的情况下执行 go get :

```shell-session
$ GOPRIVATE=git.meideng.net go get -u git.meideng.net/somebody/learn-go-modules-private
go: git.meideng.net/somebody/learn-go-modules-private upgrade => v1.0.0
```

这样就搞定了！

小结一下，如果按照我上面步骤来做的话，`go.mod`文件应该长这样：

```text
module xxx

go 1.15

require (
    git.meideng.net/somebody/learn-go-modules-private v1.0.0 // indirect
    github.com/Saodd/learn-go-modules-dep v1.1.0 // indirect
    github.com/Saodd/learn-go-modules-dep/v2 v2.0.0 // indirect
)
```
