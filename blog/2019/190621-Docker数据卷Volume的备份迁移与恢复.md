```yaml lw-blog-meta
title: Docker数据卷Volume的备份，迁移，恢复
date: "2019-06-21"
brev: 之前仗着 Docker 强大的特性，直接就在自己的桌面电脑上起了很多服务。但是为了公司长远考虑，还是要把这些容器迁移到服务器中。
tags: [Docker]
```


## 主要思路

[Docker官方](https://docs.docker.com/storage/volumes/#backup-restore-or-migrate-data-volumes)
并没有给出特别好的工具。主要的思路就是通过一个可以运行`tar`的容器，加载指定的`volume`，
压缩出来然后复制到宿主机指定的位置，再起一个容器进行解压缩。

原理很简单，很原始，但是还是很可靠的。下面我们来小试牛刀一下，目标是储存了一点点数据的`mongo`容器。

## 备份

首先看一下`mongo`容器的状态（并没有什么意义，只是关心一下而已 @_< ）:

```shell
PS > C:/Users/lewin> docker ps
CONTAINER ID        IMAGE               COMMAND                  CREATED             STATUS              PORTS                      NAMES
c4fe2af92a81        mongo               "docker-entrypoint.s…"   3 weeks ago         Up 6 days           0.0.0.0:20012->27017/tcp   apmos_mongo_1
```

关键还是要知道这个容器的数据卷存放在哪里。我们去找到当时写的`compose.yml`文件：

```yaml
version: '3.1'

services:
  mongo:
    image: mongo
    restart: on-failure
    volumes:
      - mongo:/data/db    ## 这里数据卷挂载的位置就是数据库保存的目录了
    ports:
      - 20012:27017
    command:
      mongod --port 27017 --dbpath /data/db --replSet rs0 --bind_ip 0.0.0.0

volumes:
  mongo:
```

注意了，这个`volume`是我手动建立的。那么我们按照官方的做法：

```shell
PS > docker run --rm --volumes-from apmos_mongo_1 -v 宿主机目录:/backup alpine tar cvf /backup/mongo.tar /data/db
#                  ↑ 加载需要备份的卷  ↑                                                               ↑卷中的目录

```

这样我们就在`宿主机目录/mongo.tar`得到了我们的备份文件。

## 迁移

复制的工作就不用说了，我们把备份文件放在`另一宿主机目录//mongo.tar`。

## 恢复

我们登录另一台宿主机，也就是迁移至的宿主机。

先创建一个`volume`，因为是数据库的卷，所以我们慎重一点，给它命名，这样更不容易被误删。

```shell
lewin@某服务器:~$ docker volume create mongo
mongo

```

然后把备份包解压到这个`volume`里面：

```shell
lewin@某服务器:~$ docker run --rm -v mongo:/data/db -v /home/users/lewin/docker/:/backup alpine sh -c "cd /data/db && tar xvf /backup/mongo.tar --strip 1"

```
