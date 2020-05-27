```lw-blog-meta
{"title": "在Linux中访问windows共享盘", "date": "2019-08-27", "tags": ["OS"], "brev": "为了在容器中方便地访问外部数据（以取得测试用例文件等），刚好看到自己的电脑上开启了共享，那就看一下如何在Linux中使用吧。"}
```

## 使用mount

```shell
sudo mount  \
    --verbose  # 打印日志
    -o --options  # 设置
    -t --types  # 文件系统类型
    [source] [destination]
```

在Linux机器上挂载windows主机硬盘，我们选择的类型是`-t cifs`即`普通网络文件系统Common Internet File System`；

然后要附带用户名和密码`-o username=lewin,password=123`；

网络位置是`//192.168.1.1/c`，前面是你的IP或者Hostname，后面是你的共享文件夹，这些在你的windows主机上查看。

目标位置是你在Linux上的文件夹位置，即挂载点。

### 报错：wrong fs type, bad option, bad superblock

如果`mount`程序提示以上信息，其实下面还附带了详细信息：

```text
for several filesystems (e.g. nfs, cifs) you might need a /sbin/mount.<type>  helper program
```

意思是你需要一个`/sbin/mount.<type>`文件，那么这个怎么来呢，我们相应地安装就好了：

```shell
sudo apt install cifs-utils
# nfs装这个 sudo apt install nfs-common
```

使用命令检查一下是否正确安装：

```shell
lewin@aphkapmosprod02:~$ ls /sbin/mount.cifs
/sbin/mount.cifs
```

[参考来源](https://askubuntu.com/questions/525243/why-do-i-get-wrong-fs-type-bad-option-bad-superblock-error)

### 报错：mount error(112): Host is down

出现这个报错是什么原因呢？

> This could also be because of a protocol mismatch. In 2017 Microsoft patched Windows Servers and advised to disable the SMB1 protocol.

因为协议不匹配。在2017年微软打了一个补丁，禁用了`smb1`协议。

那么我们显式地指定另一个协议`smb2`（或者`smb3`）。
完整命令如下：

```shell
sudo mount --verbose -t cifs -o username=lewin,password=123 //192.168.1.213/c ./windows/ -o vers=2.0
```

挂载成功！检查一下挂载的状态：

```shell
lewin@aphkapmosprod02:~$ df -h
//192.168.1.213/c                     931G  107G  824G  12% /home/users/lewin/windows

lewin@aphkapmosprod02:~$ ls ./windows/
360Downloads            Go            #......
```

[参考来源](https://serverfault.com/questions/414074/mount-cifs-host-is-down)

## 使用SFTP

`mount`适用于比较长久的挂载，如果是突发地、临时性地拷贝文件，那首选还是`SFTP`了。

那么难点在于服务端，如何在windows上架设sftp服务？

首先想到的是通过软件，比如moba家族的`MobaSSH`，提供一个图形化的界面，基于`openssh`并兼容所有主流的ssh客户端。非常方便。

![mobaSSH](/static/blog/2019-08-27-mobassh.png)

开启运行后，相当于一个普通的windows服务，并不会挂在任务栏中。通过windows用户账户登录即可。

### 使用docker容器

在安装MobaSSH的过程中，突然想到：SFTP无非就是一种服务，提取宿主机的文件，通过22端口提供给网络上的用户。

为什么不用docker呢？

好吧，这里我懒得折腾了，因为MobaSSH已经装好了。不过Docker-hub上是有这个项目的，而且有10M+下载数量，说明非常好用。

[仓库地址](https://hub.docker.com/r/atmoz/sftp)

示例用法：

```shell
docker run \
    -v /host/upload:/home/foo/upload \
    -p 2222:22 -d atmoz/sftp \
    foo:pass:1001
```
