```json lw-blog-meta
{"title":"[转载]centos7升级dockerCE最新版","date":"2019-07-30","brev":"因为腾讯云默认提供的centos系统上的docker非常老(1.13.1)，部署的时候发现 docker service logs 是只有新版本才有的功能。所以必须重装一个新版的docker。","tags":["Docker"],"path":"blog/2019/190730-转载-centos7升级dockerCE.md"}
```



## 原文地址

[链接](https://www.cnblogs.com/wdliu/p/10194332.html)，略有删减，侵删。
因为我自己试验了一遍，升级过程中没有遇到任何困难，所以一定要把这篇文章转载一下。

## 删除老版本

停止docker服务：

```shell
systemctl stop docker
```

卸载旧版docker软件包：

```shell
yum erase docker \
    docker-client \
    docker-client-latest \
    docker-common \
    docker-latest \
    docker-latest-logrotate \
    docker-logrotate \
    docker-selinux \
    docker-engine-selinux \
    docker-engine \
    docker-ce
```

删除相关配置文件：

```shell
find /etc/systemd -name '*docker*' -exec rm -f {} \;
find /etc/systemd -name '*docker*' -exec rm -f {} \;
find /lib/systemd -name '*docker*' -exec rm -f {} \;
rm -rf /var/lib/docker   #删除以前已有的镜像和容器,非必要
rm -rf /var/run/docker  
```

## 安装新版本

软件包安装 (注：这个应该是基础的开发包，我没有装):

```shell
yum install -y yum-utils  device-mapper-persistent-data lvm2
```

添加yum源:

```shell
yum-config-manager \
--add-repo \
    https://download.docker.com/linux/centos/docker-ce.repo
```

查看可安装的版本:

```shell
yum list docker-ce --showduplicates | sort -r
```

安装最新版本:

```shell
yum install docker-ce -y
```

启动并开机自启:

```shell
systemctl start docker
systemctl enable docker
```

查看docker版本:

```shell
docker version 


Client: Docker Engine - Community
 Version:           19.03.1
 API version:       1.40
 Go version:        go1.12.5
 Git commit:        74b1e89
 Built:             Thu Jul 25 21:21:07 2019
 OS/Arch:           linux/amd64
 Experimental:      false

Server: Docker Engine - Community
 Engine:
  Version:          19.03.1
  API version:      1.40 (minimum version 1.12)
  Go version:       go1.12.5
  Git commit:       74b1e89
  Built:            Thu Jul 25 21:19:36 2019
  OS/Arch:          linux/amd64
  Experimental:     false
 containerd:
  Version:          1.2.6
  GitCommit:        894b81a4b802e4eb2a91d1ce216b8817763c29fb
 runc:
  Version:          1.0.0-rc8
  GitCommit:        425e105d5a03fabd737a126ad93d62a9eeede87f
 docker-init:
  Version:          0.18.0
  GitCommit:        fec3683
```
