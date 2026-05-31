---
title: 'CI服务器磁盘清理命令'
description: '总结 CI 服务器常见磁盘占用来源，并给出 Docker 镜像、构建缓存、容器日志和系统日志的排查与清理命令。'
pubDate: '2026-05-25'
updateDate: ''
tags: ['运维']
keywords: 'CI服务器,磁盘清理,Docker,build cache,containerd,日志清理'
---

## 综述

在一台只负责运行 CI 的服务器上，常见堆积这些资源：

- 构建得到的 Image
- 构建过程中的 Build Cache
- 容器日志
- 系统日志

尤其是前两项。在我们项目组的前端CI服务器上，前两项一般每周增长几十GB。

## 找出大体积目录

```shell
sudo du -xh / | sort -rh | head -30
```

其中第一个参数`/`表示在哪个目录开始查找。每次运行，找到一个大体积目录后，可以递归执行这条命令。最后得到的结果大概是：

```shell
$ sudo du -xhd1 /var/lib/ | sort -rh
# 94G     /var/lib/containerd
# 94G     /var/lib
# 244M    /var/lib/apt
# 132M    /var/lib/docker
```

这就能说明是`docker`占用了空间。

Docker本身也提供了查询占用的命令：

```shell
docker system df -v
```

## 清理Image

```shell
docker system prune -f
```

准确地说不仅仅是废弃的镜像，也包括所有的docker认为可以安全删除的资源：

- stopped containers
- unused networks
- dangling images（没有tag、无人引用的镜像层）
- build cache（部分）

这条命令相对安全，可以放心地每周执行一次。由于清理了缓存，会一定程度上降低下次构建速度，不过很快就会恢复。

在此基础上，这条命令支持更多参数：

```shell
docker system prune -af --volumes
```

其中，`-a`会清理包括有Tag的镜像，`--volumes`会清理有命名的数据卷。这条命令比较危险，而且也不是日常维护所需，不要使用。

## 清理构建缓存

上面的命令只能清理大约一半的硬盘占用增长量，剩下的量基本属于构建缓存占用，也就是`/var/lib/containerd`目录下的东西。

清理命令：

```shell
docker builder prune --all
```

这条命令会彻底清理所有构建缓存，也同样会导致更大程度地降低下次构建速度。构建缓存包括`go mod download`, `pnpm i`这类在构建中间步骤安装的依赖。

因此不建议经常使用，只作为发现硬盘空间不足时、手动执行的兜底工具。

不过也有一些辅助参数，可以根据时间或者体积来判断，供参考：

```shell
docker builder prune -af --filter "until=168h"
docker builder prune -af --keep-storage 20GB
```

## 清理容器日志

一条命令查询所有容器的日志体积：

```shell
docker ps -aq | while read id; do
  name=$(docker inspect --format='{{.Name}}' $id | cut -c2-)
  log=$(docker inspect --format='{{.LogPath}}' $id)
  size=$(sudo du -sh $log 2>/dev/null | cut -f1)
  echo "$size  $name"
done | sort -hr
```

日志文件的生命周期跟随容器本身，因此清理是容易且不需要操心的。对于某些长期运行的容器，可以考虑配置Docker守护进程来限制日志体积：

```json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "100m",
    "max-file": "3"
  }
}
```

## 清理系统日志

查询：

```shell
journalctl --disk-usage
```

清理：

```shell
sudo journalctl --vacuum-time=7d
```

## 定时执行

最常见的定时任务的部署方式无非是`crontab`，这很简单，但是又很麻烦——我指的是，它“很不云原生”，它需要污染宿主机器。

跟[neilpang](https://github.com/acmesh-official/acme.sh/wiki/Run-acme.sh-in-docker)这位老哥的想法一致，我极度讨厌在一台docker宿主机上安装任何东西，哪怕只是一个简单的shell脚本。

因此我更建议启动一个常驻容器（`-d`），在容器内运行一个内置了crontab能力的二进制程序（尤其是golang编写的），通过子进程`docker cli`（甚至是直接进程内RPC）来执行清理或者其他任务。不要反驳我，毕竟，在AI时代，写一个简易版本的crontab的难度跟喝水差不了太多。
