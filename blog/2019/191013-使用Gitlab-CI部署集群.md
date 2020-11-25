```json lw-blog-meta
{"title":"实战项目：使用Gitlab-CI部署集群","date":"2019-10-13","brev":"前面说了如何对单个项目持续集成，再讨论一下如何将多个项目组合在一起，进行集群部署。","tags":["DevOps"],"path":"blog/2019/191013-使用Gitlab-CI部署集群.md"}
```



## 构思

以我的个人网站为例，实现了前后端分离之后的架构示意图如下：

> 说起来觉得还是可以吹一下的。我在10.1决定`前后端分离`，10.2决定使用`Angular`，10.7上线了一个基础版的`Angular`网站，10.8用`go-gin`重写了后端两个api，10.9配置持续集成与部署脚本，短短九天时间就将整个网站架构完全颠覆了一遍，而且前端还是用边学边练的Angular框架。

```text
             |---------|
             |         |  ->  Nginx(static files)
Browser -->  |  Nginx  |
             |(Angular)|      |-------------|
             |         |  ->  | Go-Gin(api) |  ->  MongoDB
             |---------|      |-------------|  ->  Redis
```

如图所示，我的网站项目会同时在三个仓库上进行开发，分别是部署了`Angular`的门户容器、部署了`Go-Gin`的纯api容器、和一个负责提供博客静态内容的容器。

得益于前后端分离的设计，我可以根据需要对其中任意一个进行修改，而不影响其他组件。

那么如何将这些组件容器组合在一起呢？

## 为各组件配置持续集成

### 博客内容项目

我的所有博客内容都[开源在Github上面](https://github.com/Saodd/Saodd.github.io)的。
为了应用Gitlab-CI，我先必须将这个项目在Gitlab上做个镜像。

所幸，Gitlab为所有的**开源项目**提供免费的持续集成以及`Shared-Runner`服务。
只需点几下鼠标，这个项目在Gitlab上就镜像了一份。

![saodd-mirror](https://saodd.github.io/tech-blog-pic/2019/2019-10-13-saodd-mirror.png)

然后就是写配置文件啦。首先写了个Golang程序，负责在容器启动时向Mongo数据库中注册一些信息，以保证api提供的数据是最新的。

然后为了将项目封装为容器（`Image`），我们要写`Dockerfile`，不赘述了，这里注意的是只把有用的文件打包起来，减少空间占用。

最后就是写Gitlan-CI配置文件，大概就是每次commit都要重新打包一次。

这样，我就得到了一个每次自动更新的docker-image，放在`gitlab-registry`里：

![gitlab-registry](https://saodd.github.io/tech-blog-pic/2019/2019-10-13-gitlab-registry.png)

### API项目

对Golang项目进行测试，然后编译，是非常基础的任务了，不再赘述。

不过这里说个细节，golang的官方镜像体积还是比较大的，我们考虑使用`golang-alpine`镜像，并且将编译后的二进制文件放进`alpine`镜像中，体积小得不可思议。（同理，Nginx也使用`nginx-alpine`镜像）

但是要注意编译问题，Golang官方说：虽然我已经自举啦balabla但是对于网络应用还是优先选择cgo原因是balabala如果需要纯go编译那就要加个参数：

```shell-session
$ go build -tags netgo
```

这样编译之后的文件才能完全没有动态链接库的依赖，可以放心的在`alpine`镜像中运行。

### 前端项目

前端项目也是同理，而且`Angular`与`Golang`是一家人，所以它们的部署风格也极其近似（这也是我选择`Angular`的原因之一）。

我们把`ng build --prod`的文件放在一个`nginx-alpine`镜像中。

## 服务器端部署

好了，镜像都集齐了，那么如何部署到服务器呢？

- 思路1：让Gitlab-CI自动触发，每次更新了镜像之后都命令服务器更新集群。
- 思路2：写个简单的shell脚本，让服务器每天定时拉取最新镜像，并更新集群。

我选择了方案二。因为组件之间毕竟还是有依赖的，我希望人工确认各个组件都正常编译之后，再一起更新。

那么在服务器上的脚本如何构建呢？首先我们写个脚本，主要包括`docker pull`、`docker stack deploy`和`docker service update --force`这三个命令。（也可以一步到位使用`docker stack deploy --with-registry-auth`参数，不过我还是希望分步执行更加稳妥一点）

然后使用人见人爱的`crontab`，选个良辰吉时就好了。我们可以考虑在凌晨运行，此时网络负载低；或者在上午运行，这样便于立即处理突发状况。

## 运行监控

理想的集群部署应该是基于`k8s`的，而且Gitlab专门有面板来对接`k8s`，虽然我还没有用上（只有一台小服务器，实在搞不起这个），但是略略感觉Gitlab对于它的支持应该会是非常强大的，而且应该包括运行状态的监控。

我目前只能用Dashboard看看各个项目的pipeline执行情况了，聊胜于无吧：

![DashBoard](https://saodd.github.io/tech-blog-pic/2019/2019-10-13-DashBoard.png)

## 问题：为什么要用集群

**方便部署**是显而易见的了，不过我想再强调一个好处：**便于测试**。

我可以直接在本地windows机器上启动一个完整的集群（`stack`），只要配置得当，这个集群会跟服务器部署的集群一模一样；接着通过`docker exec`进入容器内部进行调试，最后调试结果所见即所得，可以完美复现在服务器端，不会再冒出额外的Bug来（毕竟手动改配置的时候难免会忘记一两个）。

也许讨论**便利性**会有争论的余地，但是说到**稳定性**，我相信没有任何一家公司/团队会放弃这个优势。

这，就是Docker真正强大的地方。
