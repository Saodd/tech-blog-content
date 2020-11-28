```yaml lw-blog-meta
title: 实战项目：使用Gitlab-CI部署Go项目
date: "2019-09-27"
brev: 之前部署了C++项目，但是由于我对C++工具链不太熟，而且没有项目管理权限，因此做的很保守。现在自己创立一个Go项目，再来看一下Gitlab-CI的用法。
tags: [DevOps, Golang]
```


## 构思

这个阶段的主要目标是：

- 通过Gitlab-CI执行pipeline流程，目前只需要test阶段；
- test阶段需要获取coverage数据，最好获得html的报告；
- 使用最新的`Go Modules`进行依赖管理。

在完成上面的主要目标以后，接下来我们还可以考虑：

- 自动编译，并提交镜像；
- 自动部署运行（并保持运行的容器是最新的版本）；
- 以美观方便的形式展示coverage报告。

后面三项额外目标，我已经有思路，等实现以后在下一篇博客中写吧。现在来讲一下前面三个主要目标。

## 配置Gitlab

首先在Gitlab上新建一个项目，这里命名为`apmogo`（虽然看起来很像mongo，不过这个单词是ap-mo-go的缩写）。

然后找一台可用的服务器，先安装shell版本的`Gitlab-runner`，然后注册到apmogo项目中，并做好一些简单的设置工作。这里不仔细说了，前面的博客有介绍。

然后建立一个简单的`.gitlab-ci.yml`文件，这里我们只需要一个stage一个job就可以了：

```yaml
stages:
  - test

test:
  stage: test
  script:
    - "docker run --rm \
      -v golang-pkg:/go/pkg \
      -v $(pwd):/go/src/apmogo \
      --workdir /go/src/apmogo \
      golang:1.13.1 bash -c \
      'go test -coverprofile=/coverage.out ./... ' "

```

注意这里有个细节。我们如何管理依赖库的代码？不可能每个pipeline都重新将所有依赖下载一遍吧？

我的答案是——使用Docker-volumes来进行保管。我们只需要将它存放起来就行了，go自己会根据`go.mod`文件去选择使用哪些依赖。

## 写Go程序代码

这个项目第一个任务是改造之前说过的『价格监视程序』，在那个程序中用到了`github.com/go-redis/redis`，同时这次我还想引入`github.com/robfig/cron`。

我们就为它们分别写一段代码，然后分别写测试代码：

```go
// cron程序代码
package main

import (
    "fmt"
    "github.com/robfig/cron/v3"
    "time"
)

func main() {
    fmt.Printf("start at: %s\n", time.Now())
    c, err := setCron()
    if err != nil {
        fmt.Println(err)
    }

    c.Start()
    defer c.Stop()
    select {}
}

func setCron() (c *cron.Cron, err error) {
    cst, _ := time.LoadLocation("Asia/Shanghai")
    c = cron.New(cron.WithLocation(cst))

    // 写任务吧
    _, err = c.AddFunc("55 19 * * *", func() { fmt.Println("haha!") })
    if err != nil {
        return
    }

    return
}
```

```go
// cron测试代码
func Test_setCron(t *testing.T) {
    _, err := setCron()
    if err != nil {
        t.Fatal(err)
    }
}
```

```go
// redis程序代码
package apdb

import "github.com/go-redis/redis"

var apRedisOptions = &redis.Options{Addr: "192.168.1.242:6379"}

func GetRedis() *redis.Client {
    rd := redis.NewClient(apRedisOptions)
    return rd
}
```

```go
// redis测试代码
func Test_ApRedisOptions(t *testing.T) {
    rd := GetRedis()
    _, err := rd.Ping().Result()
    if err != nil {
        t.Fatal(err)
    }
}
```

好，代码非常简单，主要意思就是看看能否正常建立cron任务，然后看看能否连接Redis数据库。

接下来我们要用最新的`Go Modules`模块来做依赖管理。所有的依赖信息最终会记录在`go.mod`和`go.sum`这两个文件中。

对于初次引入、并且希望指定版本的依赖库，我们使用`go get`命令来导入它。对于间接依赖的库，我们使用任意go命令时都会自动添加。如果需要删除依赖，我们使用`go mod tidy`命令。

注意在上述所需的两个依赖库中，`cron`是使用了Modules功能，而`redis`还没有，所以二者的命令会有区别。

```shell-session
$ go mod init gitlab.apcapital.local/lewin/apmogo
go: creating new go.mod: module gitlab.apcapital.local/lewin/apmogo

$ go get github.com/go-redis/redis
$ go get github.com/robfig/cron/v3

$ cat C:/Users/lewin/mycode/apmogo/go.mod

module gitlab.apcapital.local/lewin/apmogo

go 1.13

require (
    github.com/go-redis/redis v6.15.5+incompatible
    github.com/robfig/cron/v3 v3.0.0
)

```

这样依赖管理就设置好了。然后我们运行测试看一下：

```shell-session
$ go test -cover ./...
ok      gitlab.apcapital.local/lewin/apmogo/build/cron  0.498s  coverage: 35.7% of statements
ok      gitlab.apcapital.local/lewin/apmogo/utils/apdb  0.781s  coverage: 100.0% of statements
```

## 完善CI配置

好，上面的覆盖度是分别针对每个package的，那这些packages作为一个整体，整体的覆盖度如何得到呢？

官方文档并没有给出直接的答案，搜索了一下也只有一些古老的办法使用bash命令，逐个package进行测试然后汇总……很麻烦……

不过最后误打误撞，发现我所需要的功能其实已经在`go test`中实现了。我们看一下如何操作：

```shell-session
$ go test -coverprofile=coverage.out ./...
ok      gitlab.apcapital.local/lewin/apmogo/build/cron  0.455s  coverage: 35.7% of statements
ok      gitlab.apcapital.local/lewin/apmogo/utils/apdb  0.847s  coverage: 100.0% of statements

$ go tool cover -func coverage.out
gitlab.apcapital.local/lewin/apmogo/build/cron/cron.go:9:       main            0.0%
gitlab.apcapital.local/lewin/apmogo/build/cron/cron.go:21:      setCron         71.4%
gitlab.apcapital.local/lewin/apmogo/utils/apdb/redis.go:7:      GetRedis        100.0%
total:                                                          (statements)    43.8%

```

好的，我们将这个命令整合到Gitlab-CI中，让它自动执行。同时，另外生成一份html报告，它非常直观，强烈建议你试一试！

```yaml
test:
  stage: test
  script:
    - "docker run --rm \
      -v golang-pkg:/go/pkg \
      -v $(pwd):/go/src/apmogo \
      --workdir /go/src/apmogo \
      golang:1.13.1 bash -c \
      'go test -coverprofile=/coverage.out ./... \
      && go tool cover -func /coverage.out \
      && go tool cover -html /coverage.out -o _coverage.html' "
    - cp _coverage.html coverage.html
    - "docker run --rm -v $(pwd):/go/ -w /go/ alpine rm /go/_coverage.html"
  artifacts:
    paths:
      - ./coverage.html
    expire_in: 30 days
```

## 整合代码覆盖度

要将`go test`命令的输出与Gitlab结合起来，这一点Gitlab的开发者已经帮我们设计好了。它会从pipeline的输出中，用正则表达式去寻找所需的信息。

注意观察上面的输出，然后我们在项目设置中（需要项目管理员权限），写入正则表达式：`\(statements\)\s*?\d+.\d+%`。

同时我们还可以给项目加上`Badges`，也就是那些在Github上经常见到的小标签。最终效果如下：

![apmogo](../../tech-blog-pic/2019/2019-09-27-apmogo.png)

ok，这样我们实现了三个主要目标，此时其实已经可以算是将CI的框架都准备好了，接下来进行完善就好了。
