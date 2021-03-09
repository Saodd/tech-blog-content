```yaml lw-blog-meta
title: "Github Actions 入门"
date: "2020-09-13"
brev: 以搭建博客为例，简单讲一下GH-Actions的主要用途。
tags: ["DevOps","技术分享会"]
```

## 目录 

- 1.简介
- 2.搭建一个静态博客——Github Pages
- 3.搭建一个前后分离的博客——Github Actions
- 4.搭建一个自建博客——Github Packages
- 5.其他有趣的操作

> 本文所用到的代码托管在 https://github.com/Saodd/learn-gh-actions

## 1. 简介

### 1.1 历史背景

在我们以前的印象中， `Github` 就是一个存放开源项目代码的地方，或者再放大一点说，对于一些程序员来说，这里也是一个很重要的社交平台。但是说白了，这里只负责看代码，不负责运行代码。

但是我们会有很多针对代码项目的脚本程序，为了减少重复劳动，有了`持续集成(CI)`的概念，就是在一些情况下自动执行这些脚本程序，比如在 commit 时做单元测试，在 Pull Request 时做完整测试，或者在某个时间定时清理，等等。

有一些平台做得更早，比如在开源项目中常见的`Travis`，以及更多用于私有场景的 `Gitlab` 等等。

Github 做的比较晚（2020年初左右正式上线？），背靠微软，有后发技术优势，因此玩法更多也更靠谱。随着这个产品的上线，Github 才可以算是一个综合性的代码托管平台。

### 1.2 它是什么

一句话说：「在一定条件下触发，给你分配一个临时服务器运行你指定的脚本」

### 1.3 它能做什么

它能做「一台云服务器能做的一切事情」

下载、上传、计算、构建，至少主流应用场景是没问题的。

所以问题来了，当你拥有一台云服务器时，你会想做什么？

——接下来以搭建博客作为例子，讲一些主要的使用方法。

## 2. 搭建一个静态博客——Github Pages

在 Github Actions 上线之前，其实还有另一个产品——`Github Pages`，这个相信大家都应该比较熟悉。

所谓 Pages ，就是将静态的页面（或者其他资源），托管在 Github 上，并且享受基础的Web服务。

以及，它会给你分配一个`***.github.io`的三级域名。

### 2.1 一个最简单的博客

Github Pages 是基于分支的，默认会托管名为`gh-pages`的分支上的文件。（可以尝试一下，新建一个仓库，然后向这个分支上推送一个html文件，然后就可以访问它了。）

也可以通过配置，托管其它的分支。（Settings - Github Pages）

例如: [一个简单的GithubPages仓库](https://github.com/Saodd/tech-blog-content/tree/gh-pages)

原始文件链接： https://github.com/Saodd/tech-blog-content/blob/gh-pages/404.html

Github Pages 链接： https://saodd.github.io/tech-blog-content/404.html

### 2.2 Jekyll模板（了解）

Jekyll 可以算是 Github Actions 的前身，它也提供了从模板语言到静态页面的构建功能。

[Jekyll官方文档](https://jekyllrb.com/)

它的使用非常简单，我们只需要找到一个模板项目，做一些简单的配置，然后开始写内容就可以了：

[一个可爱的模板](https://github.com/xukimseven/HardCandy-Jekyll)

[模板使用效果](https://xukimseven.github.io/)

可能是由于 Github Actions 的上线，Jekyll的运行规则改变了。所以这部分内容不多讲了。

### 2.3 其他静态页面框架

另一个更流行的静态网页生成框架 [Hugo](https://gohugo.io/)

思路一：在本地构建好，上传到 gh-pages 分支。

思路二：利用 Actions 构建，上传到 gh-pages 分支。

有兴趣的小伙伴自行研究。

## 3. 搭建一个前后分离的博客——Github Actions

静态博客有个很明显的缺点——内容与样式耦合在一个项目里，不便于管理。如何不便了？举个例子，当想要更换主题模板的时候，当要迁移项目托管平台的时候，当更换本地电脑的时候，臃肿的项目代码总是会造成各种各样的小麻烦。

另外，最重要的是，模板语言真的很烦啊 ：）

就像我们的Web技术从后端模板渲染发展到MVVM等模式一样，我也希望用这种思路，将博客的各个组成成分分离开——分成前端、后端、内容三个部分。

那么问题来了，Github Actions 只是提供一个临时服务器用于运行简单的脚本，我们要如何在 Github 上运行一个后端？——或者说详细一点，如何运行一个只提供内容服务的后端？

### 3.1 路径的奥妙

我们再看一下前面提到过的一个 Github Pages 托管的文件： 

https://github.com/Saodd/tech-blog-content/blob/gh-pages/404.html

https://saodd.github.io/tech-blog-content/404.html

现在是一个 `404.html`文件对不对？

那么，假如我创建一个文件夹 `./blogs/list`， 然后在这个文件夹里创建一个文件名为 `1` 的文本文件，我们会得到这样一个路径： 

https://github.com/Saodd/tech-blog-content/blob/gh-pages/blogs/list/1

它对应的 Github Pages 的路径是这样的：

https://saodd.github.io/tech-blog-content/blogs/list/1

这是不是一个很典型的 REST 风格的资源路径？

那么，如果我们向这个文件里写入一个 Json 字符串，它是不是就变成了一个"后端接口"？

### 3.2 我的路径解决方案

一个博客，无非就是这么几个接口：

1. 博客分类目录
2. 每个分类对应的列表页
3. 文章详情页

它是一个非常规则的树形结构，因此我们只需要把它的根节点写死在前端代码中，我们就可以访问到树中的任意节点：

https://saodd.github.io/tech-blog-content/index/tags

https://saodd.github.io/tech-blog-content/index/TimeLine/1

https://saodd.github.io/tech-blog-content/blog/2020/200906-Rabbitmq入门.md

### 3.3 先做一套简单的前后端分离的博客页面

接下来开始展示一些 Github Actions 的API用法。我们分别写一点最简单的前后端代码。[代码托管地址](https://github.com/Saodd/learn-gh-actions)

前端的主要功能是，用户点击按钮后，向后端请求"博客列表"资源，然后显示在当前页面上。

```html
<body>
<div id="main"></div>
<button onclick="onClick()">请求后端内容</button>


<script src="https://cdnjs.cloudflare.com/ajax/libs/jquery/3.5.1/jquery.min.js"
        integrity="sha512-bLT0Qm9VnAYZDflyKcBaQ2gg0hSYNQrJ8RilYldYQ1FxQYoCLtUjuuRuZo+fjqhx/qtq/1itJ0C2ejDxltZVFg=="
        crossorigin="anonymous"></script>
<script>
    onClick = function () {
        $.ajax({
            url: "./api",
            success: function (resp) {
                $("#main").html(resp + '<br>' + Date());
            }
        });
    }
</script>
</body>
```

后端是临时的。它主要功能是，代理 index.html, 并实现一个"博客列表"的资源接口。

```go
func main() {
	http.Handle("/", http.FileServer(http.Dir(".")))

	resp, _ := json.Marshal([]*BlogData{
		{"First Blog", "bla bla bla..."},
		{"Second Blog", "Hello, world! "},
	})
	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		w.Write(resp)
	})

	http.ListenAndServe(":8080", nil)
}
```

我们可以在本地访问 http://localhost:8080/frontend.html 确认前后端都工作正常。

那么，接下来如何把这个后端"放"到 Github 上？

### 3.4 Actions 脚本

先按照前面的思路，将后端接口的 json 内容写到一个文件中。我们需要另写一段代码：

```go
func main() {
	resp, _ := json.Marshal([]*BlogData{
		{"First Blog", "bla bla bla..."},
		{"Second Blog", "Hello, world! "},
	})

	f, _ := os.Create("./api")
    defer f.Close()        
	f.Write(resp)
}
```

然后开始写构建脚本，在 `.github/workflows/***.yml` 建立一个yaml文件，然后按照 [Github Actions 的规则](https://docs.github.com/en/actions) 去写（但其实主要内容就是环境变量和shell命令）：

```yaml
name: Go

# 在什么条件下触发
on:
  push:
    branches: [ master ]
  pull_request:
    branches: [ master ]

jobs:

  build:
    name: Build
    runs-on: ubuntu-latest
    steps:

    # 在虚拟机上安装Golang环境
    - name: Set up Go 1.x
      uses: actions/setup-go@v2
      with:
        go-version: ^1.14
      id: go

    # 在虚拟机上拉取master分支代码
    - name: Check out code into the Go module directory
      uses: actions/checkout@v2

    # 后端"构建"
    - name: Build Backend
      run: go run backend_static.go && mkdir -p ./public && mv ./api ./public/api

    # 前端"构建"
    - name: Build Frontend
      run: mv ./frontend.html ./public/index.html && cp ./public/index.html ./public/404.html

    # 将构建的内容发布到 Github Pages
    - name: GitHub Pages action
      uses: peaceiris/actions-gh-pages@v3.6.1
      with:
        # Set a generated GITHUB_TOKEN for pushing to the remote branch.
        github_token: ${{ secrets.GITHUB_TOKEN }}
        # Set Git user.name
        user_name: Lewin Lan
        # Set Git user.email
        user_email: lewin.lan.cn@gmail.com
```

提交代码！然后就可以去 Github 上相应的页面观察 Actions 的输出日志，以及 Pages 的最终效果。（记得要去设置页面打开 Pages 功能）

## 4. 搭建一个自建博客——Github Packages

> 本文只讨论使用 Docker 来构建和管理服务的方法。

毕竟 Pages 只提供静态服务。但是有时我们可能希望我们的博客后端去做更多的事情，而不仅仅是提供 GET 接口而已。这时候我们就要想办法把 Github 和自己的云服务器关联起来，让 Github Actions 的运行结果可以反馈到自己的云服务器上。

主要思路：因为 Actions 提供的是临时服务器，因此只能由 Actions 去登录我们固定的服务器接口。

如何登录？当然只有一个选择—— SSH ——公钥装在自己的服务器上，私钥传入 Actions 运行时的临时服务器上。

登录后做什么？执行容器的更新、替换等工作，完成服务升级。

最终目标：提交代码——（自动测试）——自动构建——自动部署。也就是所谓的「持续集成」。

### 4.1 构建镜像

Github 专门用于存放构建结果的地方叫 Github Packages 。我们可以在任何一个代码仓库的右侧看到它的存在，对于一个新仓库来说，它是空的，随时可用。

接下来我们写一份 Dockerfile 来负责构建一个镜像：

```yaml
FROM golang:1.14.4

COPY . /backend
WORKDIR /backend

RUN go build -o backend backend.go

EXPOSE 80

CMD /backend/backend
```

然后添加 Actions 构建脚本（这里可以新建一个文件，也可以在原有的文件后面添加一个 `job`）：

```yaml
env:
  IMAGE_NAME: simple_blog
  
jobs:
  docker:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Build image
        run: docker build . --file Dockerfile --tag $IMAGE_NAME:latest

      - name: Log into Github-packages
        run: echo "${{ secrets.GITHUB_TOKEN }}" | docker login docker.pkg.github.com -u ${{ github.actor }} --password-stdin

      - name: Push image to Github-packages
        run: |
          IMAGE_ID=docker.pkg.github.com/${{ github.repository }}/$IMAGE_NAME
          IMAGE_ID=$(echo $IMAGE_ID | tr '[A-Z]' '[a-z]')
          docker tag $IMAGE_NAME $IMAGE_ID:latest
          docker push $IMAGE_ID:latest
```

（我们这里只构建后端镜像作为例子，前端镜像的构建是同理。）

然后我们就可以在代码仓库右侧看到它啦。可以试着 pull 下来看一看。

### 4.2 将镜像上传到其他托管平台

以 `hub.docker.com` 为例。

核心思想就是，将自己在 `docker.com` 生成的 token 传入 Actions 脚本中。

但是肯定不能明文的形式写在代码里，此时就需要用到 Secrets 功能。

我们先看一段将镜像上传的 Actions 脚本：

```yaml
- name: Log into Docker-Hub
  run: echo "${{ secrets.DOCKERHUB_REGISTRY_TOKEN }}" | docker login -u YOUR_USER_NAME --password-stdin

- name: Push image to Docker-Hub
  run: # ... 省略
```

我们只需要在 Settings - Actions 中添加对应的键值对就可以了，Actions 在运行的时候会自动将这些 Secrets 代入到脚本命令中。

> 不用担心日志泄露了你的 Secrets -- 它们对应的值在日志中会被打上马赛克。

### 4.3 SSH登录

一种通用解决方案（我在 Gitlab 上使用的方案）是安装 Linux 上的一些SSH管理工具:

```yaml
# 注：这是 Gitlab-CI 脚本
deploy:
  stage: deploy
  image: ubuntu:latest
  before_script:
    - 'which ssh-agent || ( apt-get update -y && apt-get install openssh-client -y )'
    - eval $(ssh-agent -s)
    - echo "$SSH_PRIVATE_KEY" | tr -d '\r' | ssh-add -
    - mkdir -p ~/.ssh
    - chmod 700 ~/.ssh
    - echo "$SSH_KNOWN_HOSTS" >> ~/.ssh/known_hosts
    - chmod 644 ~/.ssh/known_hosts
  script:
    - ssh ubuntu@xxx.xxx.xxx.xxx "~/your/deploy.sh"
```

在 Github 上，我们可以从市场中找到别人写好的脚本来用，例如 [ssh-action](https://github.com/appleboy/ssh-action) ，写法如下：

```yaml
jobs:

  build:
    name: Build
    runs-on: ubuntu-latest
    steps:
    - name: executing remote ssh commands using password
      uses: appleboy/ssh-action@master
      with:
        host: ${{ secrets.HOST }}
        username: ${{ secrets.USERNAME }}
        password: ${{ secrets.PASSWORD }}
        port: ${{ secrets.PORT }}
        script: whoami
```

至于登录服务器后执行的命令，这就看各自的运维工具了。我的命令是：

```shell-session
docker pull xxx && docker stack deploy xxxxx
```

## 小结

- `Github Actions`: 在一定条件下触发，给你分配一个临时服务器运行你指定的脚本。
- `Github Pages`: 托管任意的静态资源（必须是公开的）。
- `Github Pakages`: 托管构建成品（可以是私有的），支持 Docker镜像 等格式。

## 5. 其他有趣的操作

### 5.1 单元测试及报告

大家肯定对这种 Badge 并不陌生： 

![Github-Actions](https://github.com/Saodd/leetcode-algo/workflows/Go/badge.svg?branch=master)

![Gitlab-CI](https://github.com/Saodd/Saodd.github.io.backup-Jun2020/raw/master/static/blog/2019-09-27-apmogo.png)

它往往被设置在开源项目代码的 Readme.md 文件中，并且设置得越多就好像越是炫酷 ：）

Github Actions 对于 Badges 的支持并不多，它只支持一些默认的事件。

比如上面这个就代表着某个分支的某个 workflow 最近一次运行是否正常。如果脚本命令异常退出的话，往往会返回非0值，这个值会被 Actions 捕获到，从而将此次运行标记为失败。

更多事件请自行摸索。[官方文档](https://docs.github.com/en/actions/configuring-and-managing-workflows/configuring-a-workflow#adding-a-workflow-status-badge-to-your-repository)

> Gitlab-CI 对于单元测试有着更好的支持。它可以从日志输出中，用正则表达式去寻找单元覆盖度等指标并记录下来，然后生成带覆盖度的 Badge 。  
> Github Actions 并不支持直接生成覆盖度 Badge, 如果要自己做的话，其实也就是每次生成的html/css，然后挂载在 Pages 上。

> 覆盖度报告一般都可以生成html格式的，因此可以挂载在 Github Pages 上。甚至可以让 Badge 链接到这个覆盖度报告。（这样逼格又更高了呢）（例如： https://github.com/Saodd/leetcode-algo ）

### 5.2 手动触发

#### 思路一：用来开门？

如果 door.i.***.net 可以被外部访问（通过证书等认证方式）

那么可以做一个 Actions 专门用来开门。

权限控制就用 Github 的 Organization 功能，只要在团队内的用户，都可以触发这个 Actions 。

#### 思路二：作为临时服务器，处理高峰期业务？

**技术上**完全可行。只要上传业务代码，然后通过 Secrets 保存数据库密码等信息。只要没有IP限制等问题，理论上可以处理任何业务。

### 5.3 定时触发

周期性地爬取一个数据（例如天气、温度、股价等），然后追加写入到 Pages 中。

甚至与 `Prometheus` 配合起来。

### 5.4 issue 与 comment

`GopherBot` 是 Golang 团队做的一个很有意思的机器人。主要功能是，根据 issue 和 comment 内容的一些特征，做出相应的处理。

https://github.com/golang/go/issues/41289#issuecomment-689658394

它比 Github Actions 出现得更早，应该是读取了用户通知消息来实现的。
