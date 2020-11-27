```json lw-blog-meta
{"title":"GitLab配置CI环境","date":"2019-08-21","brev":"CI/CD应该是DevOps的关键内容之一吧。本文以Python项目为例，简述GitLab-CI的安装和运行。","tags":["DevOps"],"path":"blog/2019/190821-GitLab部署CI持续集成.md"}
```



## 什么是CI

最近公司升级了GitLab，导航栏内大大的`CI/CD`字眼甚是惹眼。所以我趁此机会试用一下，并且也是熟悉一下所谓的Github/GitLab flow。

`GitLab CI/CD`官方给出的定义是：

> GitLab CI/CD is a tool built into GitLab for software development through the continuous methodologies:
> 
> Continuous Integration (CI)
> Continuous Delivery (CD)
> Continuous Deployment (CD)

包含三大块：持续集成，持续交付，持续部署。

> Continuous Integration works by pushing small code chunks to your application’s code base hosted in a Git repository, and, to every push, run a pipeline of scripts to build, test, and validate the code changes before merging them into the main branch.

持续集成包括push代码，然后构建、测试、验证，然后合并到主分支中。

> Continuous Delivery and Deployment consist of a step further CI, deploying your application to production at every push to the default branch of the repository.

持续交付和持续部署比CI更多一个部署的步骤。即每次更新都部署到产品线上。

听起来很嗨！这也是我一直想要的功能！仰慕已久，快来看看吧。

## 环境安装

### GitLab环境

首先你要有一个GitLab（好像Github上也行，但是公司内部使用一般都流行GitLab吧），版本需要是8.0以上。

然后你要有项目管理权限`project Maintainers and Admin users`，否则你无法设置runner。像我对于公司的项目是没有这种权限的，所以我自己开了一个个人名下的hello-world项目用来试用！

### 设置Runner

其实最核心（我最重视）的部分就是测试了。那我们知道，跑测试代码肯定要机器了，还要依赖环境了，特别是对于Python这种脚本型语言来说特别重要。这种需求下当然是Docker没得说了。

不过先不要太担心依赖的问题，我们设置好runner就知道它很简单了。

![settings](https://cdn.jsdelivr.net/gh/Saodd/tech-blog-pic@gh-pages/2019/2019-08-21-CI-settings.png)

在设置页面，我们可以展开Runners面板，下面会显示出所有当前可用的runner。刚开始如果别人没有设置的话，是肯定没有runner的，我们要从头开始。

先pull镜像：

```shell-session
docker pull gitlab/gitlab-runner:latest 
```

启动，这里要注意，一定要把Docker守护进程的sock套接字给挂载进去，这样runner这个容器才能跟Docker守护进程通信。

```shell-session
docker run -d --name gitlab-runner --restart always \
-v /var/run/docker.sock:/var/run/docker.sock \
-it gitlab/gitlab-runner:latest 
```

将当前容器注册到GitLab的项目中去。我们在刚才的网页中会看到一个`url`和`token`，把他们输入进去。
（注意，这里官方的Doc有点乱，说什么用一个容器来保存配置然后`volumefrom`云云，扯淡吧，我折腾了好久，还是通过`exec`在已经运行的容器内部注册比较靠谱）

```shell-session
docker exec -it gitlab-runner gitlab-runner register
# 有交互，按照提示输入相关信息。成功后会告诉你success
```

![settings](https://cdn.jsdelivr.net/gh/Saodd/tech-blog-pic@gh-pages/2019/2019-08-21-CI-success.png)

当runner前面有一个绿灯的时候，就说明这个runner运行正常了。

## 推送代码

至此就设置完成了，我们推送一个commit试一下。

```python
# main.py

print("hello, world!")
```

```yaml
#.gitlab-ci.yml

image: python:3.7.4

simple-test:
  script:
    - python main.py
```

git add, git status, git commit, git push一气呵成：

![settings](https://cdn.jsdelivr.net/gh/Saodd/tech-blog-pic@gh-pages/2019/2019-08-21-CI-pass.png)

此前如果没有配置runner，每次commit都会显示测试失败的图标。现在显示一个绿色的小勾，强迫症一本满足了。

我们点击小勾，可以看到具体的执行情况：

![settings](https://cdn.jsdelivr.net/gh/Saodd/tech-blog-pic@gh-pages/2019/2019-08-21-CI-log.png)

能看到日志是非常棒的，可以放心地安排更多更复杂的测试了。
