```yaml lw-blog-meta
title: "MySQL基本用法——运维篇"
date: "2022-05-20"
brev: "八股！"
tags: ["中间件"]
```

## 主从复制

### 基本运维操作

> 注意，这里使用的MySQL版本是5.7 。部分命令在8.0版本不支持。

这里依然全部使用docker来运行MySQL进程，由于MySQL默认会占用容器内的一些资源，为了方便，直接一个容器一个MySQL实例。

先启动一个实例作为MASTER：

```shell
docker run --name m1 -p 7001:3306 -e MYSQL_ROOT_PASSWORD=root -dit mysql:5.7 mysqld --log-bin=mysql-bin --server-id=1 --bind-address=0.0.0.0
```

在这个实例里通过cli执行一些命令：

```sql
GRANT REPLICATION SLAVE ON *.* to 'backup'@'%' identified by 'password';
    
flush privileges;
    
show master status;
```

上面代码的主要意思是：创建一个用户专门用来做同步操作，用户名为`backup`，密码`password`，`%`意思是这个用户可以从任意host登录，`*.*`意思是对所有库、表生效。

然后启动第二个实例（或者更多个）作为SLAVE，启动命令可以几乎完全一致：

```shell
docker run --name m2 -p 7002:3306 -e MYSQL_ROOT_PASSWORD=root -dit mysql:5.7 mysqld --log-bin=mysql-bin --server-id=2 --bind-address=0.0.0.0
```

```sql
change master to master_host='10.0.6.239',
    master_port=7001,
    master_user='backup',
    master_password='password';

start slave;
```

上面两条语句，指定了master的配置信息，然后开启salve模式。随后mysql进程会在后台执行数据同步操作。

我们可以试着在master写入，然后从slave中查询，以验证主从复制模式工作正常。

### 与Redis的比较

它们都是相同的主从同步架构，即主节点上产生操作日志，从节点拉取日志进行同步（默认异步模式）。它们都支持一主多从，多级从节点。

但是我在刚才的实践中发现MySQL存在一些问题：

- master下线后，默认配置下，slave节点每1分钟才尝试重连一次（这也太懒惰了吧！）
- slave节点可写！（而Redis会拒绝写入）
- 各种配置的方式一直给我一种感觉：古老。（Oracle习惯把好东西都藏着掖着是吧）

### 同步还是异步？

1. 异步模式：默认模式。类似Redis，事务执行完毕后直接返回客户端，不等待同步动作。
2. 半同步模式：至少1个slave同步成功即可
3. 同步模式：每个事务确认所有slave同步成功后，才返回客户端。

![同步模式](https://p3-juejin.byteimg.com/tos-cn-i-k3u1fbpfcp/9340d676fdb24b38963a0d08ab4fbc64~tplv-k3u1fbpfcp-zoom-in-crop-mark:1304:0:0:0.awebp)

（[图片来源](https://juejin.cn/post/6967224081410162696) ）

异步模式就是类似Redis的AP模式，同步模式就是类似etcd的CP模式，根据业务可靠性要求来选择。（一般可能选择半同步模式即可吧？）

> 同步的过程其实就是对binlog文件做offset找位置然后增量同步，这种模式在故障恢复的时候可能产生麻烦。因此有了一种新的模式：GTID [参考](https://www.cnblogs.com/rickiyang/p/13856388.html) ，核心逻辑是给操作生成一个全局唯一的ID

### 复制的内容？

- 复制语句
- 复制行数据：所有更新过的行都会进行同步
- 混合模式

仅复制语句肯定是执行效率最高的，但是复制行数据可以更早发现主从之间的不一致（还记得mysql从库也可以写入吗……）

### 复制流程

1. master有binlog线程，把操作写入binlog文件
2. slave有io线程，将binlog放入自己的 relay log
3. slave的执行线程，执行 relay log

## innodb引擎

### 与 myisam 的区别？ 
