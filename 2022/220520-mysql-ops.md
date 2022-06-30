```yaml lw-blog-meta
title: "MySQL基本用法——运维篇"
date: "2022-05-20"
brev: "主从、集群、分库分表"
tags: ["中间件"]
```

## 主从复制

主从复制是一切运维操作的基础。这个『从』，可以是slave，也可以是对等的另一个master，也可以是备份磁盘；『复制』因此也就是一方导出+另一方导入结合的过程。

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
- 复制行数据（所有更新过的行都会进行同步）
- 混合模式

仅复制语句肯定是执行效率最高的，但是复制行数据可以更早发现主从之间的不一致（还记得mysql从库也可以写入吗……）

### 复制流程

1. master有binlog线程，把操作写入binlog文件
2. slave有io线程，将binlog放入自己的 relay log
3. slave的执行线程，执行 relay log

## 插曲：什么是高可用

谷歌云的 [这篇文章](https://cloud.google.com/architecture/architectures-high-availability-mysql-clusters-compute-engine#consider_your_requirements_for_ha) 讲得挺好的，简单翻译一下：

当你在考虑高可用的时候，费用是一个不可忽视的方面，因为你需要至少两倍的硬件资源来保证高可用。当你选择MySQL高可用方案时，你需要考虑这些问题：

- 哪些服务和用户依赖着数据层（Data Tier）？
- 运维预算有多少？
- 数据层不可用的期间，会给你的业务带来多少损失？
- 你需要多高的自动化程度？
- 要达到多高的可用性，99.5%, 99.9% 还是 99.99% ？
- 预期故障恢复时间（RTO）是多少？

当评估RTO的时候，这些因素需要被考虑：

- 故障侦测时间
- 备用虚拟机就绪时间
- 存储设备恢复时间
- 数据库应用恢复时间
- 业务应用恢复时间

高可用架构需要实现以下功能：

- 一种侦测主节点上发生的故障的机制
- 一种将备用节点升格为主节点的过程
- 一种将查询路由(query routing)切换到新的主从关系上的过程
- （可选）一种将配置恢复到原始的主从拓扑结构的方法

## 群组复制

参考官方文档： [Chapter 18 Group Replication](https://dev.mysql.com/doc/refman/8.0/en/group-replication.html)

> 注意，这里用的是8.0版本文档，另外也有5.7的文档。

`Group Replication`提供弹性、高可用性、容错性。

它可以以单主模式运行，能够自动选举主节点，并且保证只有一个节点会接受写入操作；也可以以多主模式运行。

我们已经提供了内置的 群组关系服务`group membership service`，它可以持续关注节点的情况，当任意节点加入或者退出的时候做出相应的处理。

但是这种机制保证的是整个组的可用性。开发者必须知晓，客户端连接到的某一个具体的数据库实例发生异常时，此时整个群组依然可用，客户端必须重新连接其他可用的实例。这个过程需要借助其他的东西，例如连接器、负载均衡器、路由器或者其他形式的中间件。`Group Replication`本身并没有内置这种功能的中间件。

`Group Replication`是以MySQL插件的形式提供的，（每个实例分别）经过配置即可使用。另一种替代方案是`InnoDB Cluster`。

[todo](https://dev.mysql.com/doc/refman/8.0/en/group-replication-configuring-instances.html)

## SSL配置

mysql通讯协议是建立在TCP之上的纯文本协议，因此在通讯过程中可能被被抓包分析得出其中的内容。

从MySQL5.7开始引入了SSL协议，使得C-S两端的通信内容全部加密。（注意，这里SSL只保证通信路径上的安全，并不保证账户密码本身的安全）

但毕竟加密、解密是需要额外计算资源的，根据一些文章的分析，SSL带来的损耗可能高达20%左右。而数据库又往往是一个应用的关键瓶颈，所以对于“MySQL是否一定要用SSL”这个问题，我们可能不得不在性能和安全之间做一个取舍。我个人认为，如果只是在内网之内通讯，这个网络环境是相对安全的（但也并不绝对安全，参考`零信任安全模型`），可以考虑不用SSL ；而如果需要开放到更大的网络环境中，特别是互联网环境下（这个其实很不安全，除了通讯安全之外还要考虑一些其他的安全因素），则必须强制使用SSL。

言归正传，接下来说说怎么配置。

使用SSL首先需要自签证书。可以自己用`OpenSSL`去操作，也可以借助MySQL自带的工具，例如MySQL从零开始初始化的时候会自动生成一套证书（Docker容器去`/var/lib/mysql/`目录里找），或者手动执行`mysql_ssl_rsa_setup`工具，一共得到8个证书：

```text
ca-key.pem  ca.pem
client-cert.pem  client-key.pem
private_key.pem  public_key.pem
server-cert.pem  server-key.pem
```

如果你使用从外部导入的现成的证书，那么你需要给`mysqld`提供其中3个证书，配置如下（[官方文档](https://dev.mysql.com/doc/refman/5.7/en/using-encrypted-connections.html)）：

```ini
[mysqld]
ssl_ca=ca.pem  # 指向你导入的文件路径
ssl_cert=server-cert.pem  # 指向你导入的文件路径
ssl_key=server-key.pem  # 指向你导入的文件路径
require_secure_transport=ON  # 可选，强制要求所有客户端使用ssl
```

至于如何将配置文件导入docker容器，请参考[镜像文档](https://hub.docker.com/_/mysql)

从客户端登录时，也需要指定其中3个证书（[官方文档](https://dev.mysql.com/doc/refman/5.7/en/using-encrypted-connections.html)）：

```shell
mysql --ssl-ca=ca.pem
      --ssl-cert=client-cert.pem
      --ssl-key=client-key.pem
```

至于如何判断当前所使用的连接是否正在使用ssl，似乎没有一个定论（[参考](https://dba.stackexchange.com/questions/36776/how-can-i-verify-im-using-ssl-to-connect-to-mysql)），目前看来不能通过SQL语句（例如`SHOW STATUS`）来判断，只能依靠客户端本身的功能或者配置来判断。

接下来还要主动要求客户端配置ssl 。如果是按我上面的安全策略模型，只对部分场景要求ssl的话，那么需要的操作是：设置一个专用账户，单独对这个账户要求必须ssl登录。（[官方文档](https://dev.mysql.com/doc/refman/5.7/en/alter-user.html#alter-user-tls)）

```sql
# 创建用户时
GRANT ALL PRIVILEGES ON XXX.* TO 'lewin'@'%' REQUIRE SSL;
# 修改用户时
ALTER USER 'lewin'@'%' REQUIRE SSL;
```

## 监控指标

我以为会有比较成熟的组件，但是大概搜了一下，似乎只发现 [Prometheus + Granafa](https://segmentfault.com/a/1190000022336871) 的方案，看起来配置起来还是需要一点功夫的。

## 慢查询日志

在实际业务中，慢查询是一个比较重要的监控指标，我们经常可以从慢查询里发现一些代码中的问题。当然，出现慢查询了那都已经是事后的被动检查机制了。在开发过程中主动做一些性能预测和优化也同样重要，二者结合效果更佳。

它是MySQL内置的功能，但是默认关闭，需要我们主动打开。检查方式：

```sql
show variables like 'slow_query_log'; 
```

## 同步表结构

我们如果要修改表结构，那么肯定先在测试环境调试完毕之后，再同步到线上数据库中去。

那么这里就存在一个同步的操作。

一般借助工具来完成。我手头有Jetbrains家的`DataGrip`，只要同时选中两个table，然后右键单击选择"Compare"，即可快速对比两个table的区别并且给出相应的`alter`语句，非常强大。除此以外也有很多开源工具可以选用，我就不提了。

但如果项目的数据库是管理比较严格的，可能会不允许这类工具直连线上数据库。这时候我们可以考虑通过`SHOW CREATE TABLE <table-name>`命令得到线上数据库表的DDL，在测试环境克隆一个表，然后再用上述工具去处理就行了，毕竟我们最后需要的是它生成出来的`alter`语句而已。

甚至还可以在测试环境创建一定规模的数据来预测操作所需时间（约等于服务器停机时间）

对于大型应用的数据库，特别是做了主从、分库分表的大型数据库，可能需要更多更复杂的处理流程，例如复制表、滚动更新等策略。这个话题对于普通的研发工程师来说有些超纲了，暂且先不研究它。可以读一下[这篇文章](https://www.cnblogs.com/wangtao_20/p/3504395.html)
