```yaml lw-blog-meta
title: "MySQL基本用法——运维篇"
date: "2022-05-20"
brev: "索引、主从、集群、分库分表"
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
- 复制行数据（所有更新过的行都会进行同步）
- 混合模式

仅复制语句肯定是执行效率最高的，但是复制行数据可以更早发现主从之间的不一致（还记得mysql从库也可以写入吗……）

### 复制流程

1. master有binlog线程，把操作写入binlog文件
2. slave有io线程，将binlog放入自己的 relay log
3. slave的执行线程，执行 relay log

## 引擎与索引

### Innodb 与 MyISAM 的区别？

简而言之：MyISAM更适合重读取的应用。

- InnoDB 支持事务（MVVC），支持外键，支持行级锁（必须使用主键）。
- MyISAM 支持全文索引，保存了总行数，只有表级锁。

MyISAM 理论上来说性能优秀一些；但由于**表级锁**，读写互相干扰，因此在读多写多的场景下表现比较差。

InnoDB 的**行级锁**可以减少这种冲突的情况，并且支持很重要的事务特性。

v5.1之前默认引擎MyISAM，之后的默认引擎是InnoDB，一定程度上说明后者更符合常见业务需求。

### 索引结构

**聚簇性：**

- MyISAM 的索引是 非聚簇索引，索引与数据分离，因此可以缓存更多的索引内容，查询性能更好。
- InnoDB 的索引是 聚簇索引，主键与数据结合；其他索引都是**辅助索引**因此查询时多一级跳转。

参考阅读：[MySQL聚簇索引和非聚簇索引的理解](https://segmentfault.com/a/1190000041290817)

**索引的数据结构：**

- B树（多叉平衡树）
- HASH（拉链法）
- FULLTEXT（全文索引）
- RTEEE（很少见）

关于**B树**，其实有[两种结构](https://segmentfault.com/a/1190000020416577) ：

- B-树：每个节点都储存键和值
- B+树：是只在叶子节点保存值，中间节点只有键。

[MySQL](https://dba.stackexchange.com/questions/204561/does-mysql-use-b-tree-btree-or-both) 和 [Mongo](https://stackoverflow.com/a/65733242/12159549) 都是B+树索引，不是B-树。

- B-树的好处是，可能在中途节点就查找到数据，可以立即返回。
- B+树的查询则一定要到达树的底部才能完成。但是将数据集中在叶子节点有巨大好处，因为MySQL是关系型数据库，可能经常需要范围查询，MySQL的B+树是将叶子节点串联起来做成了链表；同时，磁盘、缓存等硬件也对连续读取有更好的性能。因此B+树有优秀的连续查询效率。

> 特别强调！数据库默认索引都是B+树！我们在某些文档中见到的『B-Tree』中间的减号并不是『B-』的意思，而是一个连字符，即『B-Tree === B树 === B+树』

**B树与HASH索引的区别：** [参考](https://blog.csdn.net/oChangWen/article/details/54024063)

- HASH不支持任何范围相关的操作（包括比较、排序、模糊匹配等）
- 联合HASH索引不能使用左方的部分索引（因为多个键组合之后进行哈希运算）
- 在HASH冲突比较多的时候性能降低

然而，常见的InnoDB和MyISAM都不支持HASH索引，因此某种意义上来说可以不管这种类型。（参考：[Table 13.1](https://dev.mysql.com/doc/refman/8.0/en/create-index.html) ）

使用HASH索引的场景一般是简单的单个查询，应该不需要事务的支持，因此直接用Redis或者其他的NoSQL数据库才是正确的选择。

### 建立索引的原则

由于索引要占用存储空间（包括硬盘、内存、缓存），同时在更新数据的时候也要同步更新所有的索引，这都会降低性能表现。因此索引并不是越多越好的，我们需要在保证功能的基础上尽可能地建立最少的索引。
