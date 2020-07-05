```json lw-blog-meta
{"title":"实战项目：使用Gitlab-CI部署C++项目","date":"2019-09-25","brev":"终于来了个令人兴奋的项目——用Gitlab-CI武装我们现有的C++项目。当真正部署上线之后，你会发现CI比你想象的还要方便。","tags":["DevOps","C"]}
```



## 项目需求

我们公司主要的项目是一个C++项目，大概是每周发一个release。自从新CTO来了以后，不仅大刀阔斧地改革了Git政策，引入了Jira作为项目管理平台；还提出要使用Docker来解决依赖项的痛点。

而使用Docker部署项目的话，最佳搭档必须是CI/CD了。由于我们公司正用着Gitlab，所以就直接使用Gitlab集成的CI功能。

简单说就是将传统的人工cycle转化为自动化的cycle。

本来领导说，咱们一步一步来，先实现在容器内编译并将编译后的文件放在某个地方；然后在容器内实现测试相关的功能；然后我们试着手动制作docker镜像然后上传；最后再自动化blabla……

太罗嗦了，我要一步到位。

## 什么是CI

`CI`的概念我就不说了，网络上太多文章了。全程都是跟着Gitlab的英文文档走的，写得很详细，虽然没有全文检索功能，经常忘记自己之前看的东西放在哪里了……

![CI](/static/blog/2019-09-25-CI.png)

它实际上就是一系列事先定义的脚本，当有特定条件时（比如push到代码库）就触发相应的脚本。

至于你的脚本里写的是编译、是测试、还是部署，完全看你的需求。你写了怎样的脚本，就相应的实现什么功能。

我们典型的发布阶段可能有3个步骤：编译、测试、部署，每个步骤可能有多项脚本要执行。Gitlab-CI完全仿照这种逻辑，设计了`stages`与`jobs`两个概念。比如：

```yaml
stages:
  - build
  - test
  - deploy

build-job:
  stage: build
  script:
    - go build -o xxx xxx.go

test-job:
  stage: test
  script:
    - go test xxx.go

deploy-job:
  stage: deploy
  script:
    - docker run --name xxxx -v xx:xx golang:1.13 mv xxx/ xxxx/
    - docker commit xxxx xxxxx:xxx
    - docker push xxxxx:xxx
```

上面展示的就是一个基本的框架，这里用golang与docker作为例子。首先定义了三个阶段（stage），然后定义了三个任务（job），每个任务都属于一个阶段。同一个阶段的任务可以并发运行，但是两个前后阶段一定是前面的执行完了才会执行后面的，以此来控制流程。

## Gitlab-runner配置

我说过CI其实就是有一个触发器来触发各种脚本，触发器当然是Gitlab本身了，那么谁来执行脚本？

我们需要Gitlab-runner。可以直接在shell上安装运行，也可以安装docker版本的。但是如果你需要执行`docker commit`或者`docker push`这种部署操作，那么你最好是安装shell版本的，否则你会需要docker-in-docker的操作，可能会比较麻烦。

首先选择一台合适的机器来运行runner！因为众所周知，c++项目编译非常痛苦，所以我们需要性能强劲的机器来运行。shell版本的runner非常容易安装，官方宣称是用go语言写的，二进制文件没有任何依赖项。`apt install`分分钟就装好了。

接下来进行配置。首先我们需要从Gitlab的项目页面中获取一个token（这个操作需要项目管理员权限，普通开发者权限看不见）。

然后，我们对runner进行配置，也非常简单，自带导引，我们只需要输入：

```text
$ gitlab-runner register
```

跟着提示输入相应的内容就可以了。配置完成后，我们在gitlab的项目页面可以看到这个runner显示在上面了。

### docker配置

如果是运行的shell版本的runner，我们很可能需要调用docker，那首先要想到调用权限：

```shell
$ sudo usermod -aG docker gitlab-runner
```

然后还有Docker仓库的pull/push权限，注意，要切换到gitlab-runner用户执行登录操作：

```shell
$ sudo su gitlab-runner
$ docker login xxxx.gitlab.local/xxx/xxx/xxx
```

在上面的登录过程中，如果项目使用的docker-registry是自己搭建的，并且基于http服务的，docker会报错（因为docker默认使用https进行交互）。

registry可能不太好改，那么我们只能改本地的docker配置了。我们[参考这篇文档](https://docs.docker.com/registry/insecure/)，将`/etc/docker/daemon.json`文件创建并添加以下内容：

```text
{
  "insecure-registries" : ["myregistrydomain.com:5000"]   # 地址和端口请改为你的registry
}
```

然后重启docker进程：

```shell
$ service docker restart
```

## 写配置文件

监工有了，工人有了，就差一张图纸了。

我们在项目根目录下（也可以自定义路径，这里不展开讲），创建一个`.gitlab-ci.yml`文件。

这是一个YAML文件，语法规则很正常，不难学；而且也比较常见。稍微看一看就会了。这个文件定义了runner的行为，也就是你在发布流程中需要执行的脚本。

这里要思考一下如何实现。首先我们要`cmake`，然后是`make`，然后是`make test`，然后又要`gcovr`，最后还要`make install`。中间有大量的文件操作，因此要特别注意，Gitlab-runner会给每个job重置当前的环境（只保留git clone的内容）；而jobs之间的文件传递需要一些特别的手段。

这里我们使用docker volume。好处是便于管理，而且出现bug时我们也可以通过人工操作来回到当时的环境进行debug。

首先是编译阶段：

```yaml
build:
  stage: build
  script:
    - "docker volume create APTS-${CI_COMMIT_TAG}-${CI_COMMIT_SHORT_SHA}"
    - "docker run --rm \
      -v APTS-${CI_COMMIT_TAG}-${CI_COMMIT_SHORT_SHA}:/home \
      -v $(pwd):/home/APTradingSystem:ro \
      xxx.xxx.xxx:8888/xxx/aptradingsystem/base:00.000.001 \
      bash -c 'cd /home/ \
      && cmake -S ./APTradingSystem \
      && make -j 12 --quiet ' "
```

在上面的任务中：

- 建立一个以当前commit的tag和sha值为名称的volume；
- 将这个volume和当前路径（项目根目录）挂载进入一个容器中；
- 在容器中执行cmake和make，12个线程并发；生成的文件都放在volume中。

接下来执行测试就很简单了，注意使用相同的名字，这样才能找到编译时用的volume：

```yaml
ctest:
  stage: test
  script:
    - "docker run --rm \
      -v APTS-${CI_COMMIT_TAG}-${CI_COMMIT_SHORT_SHA}:/home \
      -v $(pwd):/home/APTradingSystem:ro \
      xxx.xxx.xxx:8888/xxx/aptradingsystem/base:00.000.001 \
      bash -c 'cd /home \
      && make test ' "
  only:
    - tags
```

然后使用`gcovr`就省略吧，不过这里要注意的是，如何导出覆盖率报告？我们要使用Gitlab-CI的一个字段叫做`artifacts`，这个字段指定的文件或者路径，将会被上传到Gitlab服务器中，我们可以直接从项目网页上查看这些文件：

```yaml
pages:
  stage: deploy
  script:
    - mkdir -p public
    - "docker run --rm \
      -v APTS-${CI_COMMIT_TAG}-${CI_COMMIT_SHORT_SHA}:/home \
      -v $(pwd)/public:/home/public \
      alpine \
      mv home/_public/index.html home/public/_index.html "
    - cp ./public/_index.html ./public/index.html
    - "docker run --rm -v $(pwd)/public:/home/public alpine rm /home/public/_index.html"
  artifacts:
    paths:
      - public
    expire_in: 30 days
```

在上面的过程中要**特别注意**，gitlab-runer执行脚本时，是以『gitlab-runner』用户身份执行的，这个用户我们一般不会设置超级权限；而docker是以『root』用户身份执行的，所以容器创建的文件属于『root』，『gitlab-runner』无法删除，也就无法完成环境的清理工作，所以会报错退出。所以我们要**用容器来删除容器所创建的文件**。

最后是发布了。我们现在相对保守，真正的部署工作还是由人工审核后进行。所以我们这里只是制作镜像并提交。

不过这已经完成绝大部分任务了，你说人工pull一下update一下需要几秒钟？

```yaml
commit:
  stage: deploy
  script:
    - "docker run --rm \
      -v APTS-${CI_COMMIT_TAG}-${CI_COMMIT_SHORT_SHA}:/home \
      -v $(pwd):/home/APTradingSystem:ro \
      xxx.xxx.xxx:8888/xxx/aptradingsystem/base:00.000.001 \
      bash -c 'cd /home/ && make install' "
    - "docker run --name APTS-${CI_COMMIT_TAG}-${CI_COMMIT_SHORT_SHA} \
      -v APTS-${CI_COMMIT_TAG}-${CI_COMMIT_SHORT_SHA}:/home \
      xxx.xxx.xxx:8888/xxx/aptradingsystem/base:00.000.001 \
      bash -c 'mv /home/depot/$(ls /home/depot|grep apts_ |head -1) /root/prod/' "
    - "docker commit APTS-${CI_COMMIT_TAG}-${CI_COMMIT_SHORT_SHA} \
      xxx.xxx.xxx:8888/xxx/aptradingsystem/apts:${CI_COMMIT_TAG}"
    - "docker rm -f APTS-${CI_COMMIT_TAG}-${CI_COMMIT_SHORT_SHA}"
    - "docker push xxx.xxx.xxx:8888/xxx/aptradingsystem/apts:${CI_COMMIT_TAG}"
```

在上面的任务中：

- 先启动一个容器来执行make install；
- 然后创建一个有名字的容器，将编译后的二进制文件放置到指定的位置；
- 将当前容器保存为镜像，这样二进制文件就保存起来了；
- 然后提交，然后适当清理一下。

## 小结

看起来寥寥数语，其实为了写这个小小的配置文件，我还是付出了很多的心血的。

所幸，最后执行情况很完美，没有出现意料之外的情况，我大量的docker使用经验肯定功不可没。

至此，也算是（自认为）彻底打通了DevOps的任督二脉，核心内容应该就是CI了，剩下的无非就是细枝末节或者锦上添花了吧。

感谢领导给我这样一个展现的机会，当然也要感谢自己这数个月以来坚定的学习精神。
