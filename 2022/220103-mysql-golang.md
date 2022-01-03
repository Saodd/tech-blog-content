```yaml lw-blog-meta
title: "MySQL基本用法"
date: "2022-01-03"
brev: "Golang环境下的MySQL基本用法 + 事务专题"
tags: ["中间件"]
```

## 背景

目前公司的业务用的都是Mongo，不用维护表列，不用拼SQL语句，写一般的业务是挺爽的。但它的缺点也很明显，在需要关系/需要事务的场景下，用Mongo强行实现需求太蛋疼了，我不想再经历第二次了。

而且严格说起来，一个后端程序员，没精通MySQL，确实有点说不过去。所以我还是得抽出时间，以当前的技术水平，重新把MySQL用法完整地过一遍。

## 入门篇：增删改查

### 安装

最新版本是`8.0`，但是估计目前多数生产环境都是以`5.5`或者`5.6`为主吧。这里暂时不考虑兼容问题，直接使用最新版本学习特性。

```shell
$ docker pull mysql
$ docker run --name mysql -p 3306:3306 -e MYSQL_ROOT_PASSWORD=root -dit mysql
```

在初始化启动的时候必须指定root用户的认证方式，应该是三选一吧，这里用最简单的密码的方式。

> 这里有个小插曲，一开始我以为它默认用户密码是`admin`，结果试了几次才反应过来是`root`，也是有点搞笑。

### 客户端库

[go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) 但它只是标准库 [database/sql](https://golang.org/pkg/database/sql/) 的一种实现（或者叫插件），所以我们用到的API都是后者提供的，看文档也是看后者的。

```shell
$ go get -u github.com/go-sql-driver/mysql
```

> 小插曲：在这个库里又看到了欢乐的Gopher吉祥物，关于这只土拨鼠的来历， [参考](https://go.dev/blog/gopher) 

参考 [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) 的文档，尝试建立连接：

```go
import (
    "database/sql"
	"github.com/go-sql-driver/mysql"
	"time"
)

func main() {
	config := mysql.Config{
		User:   "root",
		Passwd: "root",
		Net:    "tcp",
		Addr:   "10.0.6.239",
		DBName: "LearnMysql",
	}
	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		panic(err)
	}
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	if err := db.Ping(); err != nil {
		panic(err)
	}
}
```

上面的代码中我夹带了私货。

首先，我们可以不需要手动拼接`DSN`字符串（例如`root:root@10.0.6.239/LearnMysql`），而是可以借助`mysql.Config`以结构化的形式去生成。这个可以根据项目的运维习惯来选择。

然后最后的`db.Ping()`，是用来测试连接是否成功的。前面的`sql.Open()`只会校验参数是否正确，不一定会真的去连接。加上Ping之后，立马就报错说这个数据库不存在（此时确实还未创建）。

中间三个配置项是比较重要的，`db.SetConnMaxLifetime()`指定连接的最长生存时间，以允许driver主动关闭并重连而不是被动地被server/OS/中间件意外关闭掉，这个值建议小于5分钟。`SetMaxOpenConns()`和`SetMaxIdleConns()`是连接数相关的，根据项目实际运维情况取一个合理的值就好。

我们在开发时可能还会关心`连接池`的设置，这个东西已经在 database/sql 里默认实现了，我们只需要设置连接数就行了。

### 创建一个数据库

这种操作我们一般不会通过Golang这种业务代码去做吧，不过还是可以了解一下的：

```go
func CreateDatabase(db *sql.DB) error {
	_, err := db.Exec("CREATE DATABASE LearnMysql")
	return err
}
```

创建成功时我们看不到什么反馈（`Exec()`方法没有返回信息），但是失败时会返回err 。

### 查询数据表

> 提示：JetBrains系列的数据库插件非常强大。尤其在Golang语言中，由于SQL直接是由标准库实现了接口，因此插件能力进一步提升，谁用谁知道。

先通过其他客户端工具，在数据库中创建两个表，然后尝试去查询它们的名字。

原始命令应当是这样的：

```shell
mysql> show tables;
+----------------------+
| Tables_in_LearnMysql |
+----------------------+
| test1                |
| test2                |
+----------------------+
2 rows in set (0.01 sec)
```

在Golang中稍微有些原始，我们要一行一行地将数据取出来，自己组装成数组：

```go
// step1
func ShowTables(db *sql.DB) (names []string, err error) {
	cur, err := db.Query("SHOW TABLES ")
	if err != nil {
		return nil, err
	}
	var name string
	for cur.Next() {  // 遍历查询结果
		if err := cur.Scan(&name); err != nil {  // 反序列化其中一行
			return names, err
		}
		names = append(names, name)
	}
	return names, nil
}
```

### 列的类型

每个`Query()`命令都会返回一个类似表格的东西（`*Rows`），它的Column告诉我们查询出来的是什么东西，然后下面的列就是查询到的数据结果。

我们尝试把上面的`SHOW TABLES`命令的列的类型打印出来看看：

```go
func ShowCursorColumnTypes(cur *sql.Rows) {
	tps, _ := cur.ColumnTypes()
	for _, tp := range tps {
		fmt.Println(tp)
	}
}
// &{Tables_in_LearnMysql true false false false 0 VARCHAR 0 0 0xe9a0a0}
```

它的结构体定义如下，对照查看：

```go
type ColumnType struct {
	name string

	hasNullable       bool
	hasLength         bool
	hasPrecisionScale bool

	nullable     bool
	length       int64
	databaseType string
	precision    int64
	scale        int64
	scanType     reflect.Type
}
```

可以看出，定义的内容还是比较详细的。有了这些信息，我们可以自己实现（或者依赖第三方库实现）一些反序列化的方法，直接将Query的结果转化为我们预先定义好的结构体。

### 插入数据

这是原始命令：

```shell
mysql> INSERT INTO test1 (name) values ('manual handle');
Query OK, 1 row affected (0.03 sec)
```

可以看到，它的返回值并不是一个表格（即不能转化成`*Rows`对象），因此我们使用`Exec()`方法去执行这条SQL语句就行了：

```go
// step2
func Test1InsertOne(c context.Context, db *sql.DB, name string) error {
	res, err := db.ExecContext(c, "INSERT INTO test1 (name) values (?)", name)  // 注意用?传递参数
	if err!= nil {
		return err
	}
	fmt.Println(res.LastInsertId()) // 6 <nil>
	fmt.Println(res.RowsAffected()) // 1 <nil>
	return nil
}
```

特别注意！在写SQL的时候（以及任意脚本语言代码的时候），都要有意识地去防御脚本注入攻击。在Golang中的做法是，将参数直接交给`Exec()`，千万不要自己用`fmt.Sprintf()`去拼接SQL语句！

### 查询数据

原始命令：

```shell
mysql> SELECT id,name FROM test1;
+----+--------------------+
| id | name               |
+----+--------------------+
|  1 | 手动插入           |
|  2 | 2                  |
|  3 | manual handle      |
|  4 | insert from golang |
|  5 | insert from golang |
|  6 | insert from golang |
+----+--------------------+
6 rows in set (0.00 sec)
```

> TIPS：在mysql中如果插入了中文，很有可能出现乱码（????这样子的）。一种解决方案是在启动mysql客户端的时候加入`--default-character-set=utf8`参数，另一种方法是直接配置mysqld上， [参考](https://stackoverflow.com/questions/6787824/mysql-command-line-formatting-with-utf8)

在Golang里没什么特别的，所以这次看一下单个查询是怎么写的吧：

```go
// step3
type Test1Model struct {
	ID   int
	Name string
}

func Test1FindOne(c context.Context, db *sql.DB, name string) (*Test1Model, error) {
	row := db.QueryRowContext(c, "SELECT id, name from test1 WHERE name=?;", name)
	if err := row.Err(); err != nil {
		return nil, err
	}
	var model = new(Test1Model)
	if err := row.Scan(&model.ID, &model.Name); err != nil {
        // 省略了ErrNoRows的处理
		return nil, err
	}
	return model, nil
}
```

区别是使用的`QueryRow()`方法，用法上基本跟`Query()`是一样的，只不过返回值我们不需要用`Next()`去循环，而是只取一次。

### 改删数据

```mysql
update test1 set name='updated-name' where id=2;
```

```mysql
delete from test1 where id=2;
```

## 进阶篇

### LIKE

[参考](https://www.runoob.com/mysql/mysql-like-clause.html) `LIKE`实质上是一个简化版本的正则表达式

```mysql
# 查询name中含有 an 的行
SELECT * FROM test1 WHERE name LIKE '%an%';
```

### UNION组合

[参考](https://segmentfault.com/a/1190000007926959) `UNION`的中文译名叫『组合查询』，就是把多个`SELECT`的结果简单地拼在一起。

基本上等同于`WHERE x=x or x=x`，但是可以通过`UNION ALL`来保持不去除重复项目。

在跨表查询时，字段名可以不同，只要字段类型和字段数量一致就行，查询后会简单粗暴地将结果拼在一起。

例子：假如我们有两张表，一个班级表，一个学生表，其中学生表是这样：

```shell
mysql> SELECT * FROM student WHERE status=1;
+----+--------+--------+--------+
| id | name   | status | nick   |
+----+--------+--------+--------+
|  1 | 李明   |      1 | 小明   |
|  2 | 张红   |      1 | 小红   |
+----+--------+--------+--------+
2 rows in set (0.01 sec)
```

我们写一个UNION查询，注意我们从学生表里查的是`nick`，但是最后结果被放在了`name`这一列里：

```mysql
SELECT id,name,'class' as table_name FROM class WHERE status=1
UNION
SELECT id,nick,'student' as table_name FROM student WHERE status=1;
```

```text
+----+--------+------------+
| id | name   | table_name |
+----+--------+------------+
|  1 | 一班   | class      |
|  2 | 二班   | class      |
|  1 | 小明   | student    |
|  2 | 小红   | student    |
+----+--------+------------+
```

### JOIN连接

[参考](https://www.runoob.com/mysql/mysql-join.html) 它主要用来做跨表查询。之前说的`UNION`主要是以行数上纵向拼接，而这个`JOIN`主要是在列数上横向拼接。

两个表的数据未必完全对应，因此就要做集合算法，`INNER JOIN`是取交集，`LEFT JOIN`是保障左表完全取出，`RIGHT JOIN`是保障右表完全取出。

例子，依然是两个表，一个班级表，一个学生表：

```text
+----+--------+--------+
| id | name   | status |
+----+--------+--------+
|  1 | 一班   |      1 |
|  2 | 二班   |      1 |
|  3 | 三班   |      0 |
+----+--------+--------+
+----+--------+--------+----------+
| id | name   | status | class_id |
+----+--------+--------+----------+
|  1 | 李明   |      1 |        1 |
|  2 | 张红   |      1 |        2 |
|  3 | 肖白   |      0 |        4 |
+----+--------+--------+----------+
```

然后写一个查询，将学生姓名和匹配的班级名称一起查出来：

```mysql
SELECT s.name AS student_name, c.name AS class_name FROM student s
INNER JOIN class c on s.class_id = c.id;
```

```text
+--------------+------------+
| student_name | class_name |
+--------------+------------+
| 李明         | 一班       |
| 张红         | 二班       |
+--------------+------------+
```

如果分别改成LEFT和RIGHT，效果如下：

```text
| 李明         | 一班       |
| 张红         | 二班       |
| 肖白         | NULL       |
```

```text
| 李明         | 一班       |
| 张红         | 二班       |
| NULL         | 三班       |
```

> TIPS: JOIN应该算是关系型数据库的关键用法之一了吧，作为平时都用Mongo的人我觉得这个特性还是挺香的，虽然Mongo也有聚合查询可以实现跨表查询。

### VIEW视图

[参考](https://www.cnblogs.com/geaozhang/p/6792369.html) 实质上就是把常用的查询保存下来，后续可以很方便地复用。它是一个逻辑表，并没有包含真正的数据。

例如我们把前一节中查询学生名字+班级名字的查询条件 复制过来就可以创建一个VIEW：

```mysql
create view student_class as
    SELECT s.name AS student_name, c.name AS class_name FROM student s
    LEFT JOIN class c on s.class_id = c.id;
```

然后就可以当作这个视图表是真实存在的，直接在它上面继续做查询：

```shell
mysql> select * from student_class where class_name IS NOT NULL;
+--------------+------------+
| student_name | class_name |
+--------------+------------+
| 李明         | 一班       |
| 张红         | 二班       |
+--------------+------------+
```

视图也可以是"可更新视图"，即可以在上面做UPDATE等操作。可更新视图有一些定义上的要求，[参考](https://blog.csdn.net/nimeijian/article/details/51958758) 上面我定义的这个视图就不是可更新的。

视图本质上就是一种保存下来的查询语句，因此如果源表被修改了(`alter table`)，视图本身依然存在，但是尝试对视图进行查询的时候就会报错；此时需要修改视图的定义让它重新恢复正常。

视图可以嵌套。

### PROCEDURE存储过程

它与视图有点相似，都是保存在服务端的、一些预先设置好的逻辑。不同的是，视图是只读的，一般不带副作用；而存储过程是一段代码（函数），它不能嵌入SELECT指令，而且逻辑中往往会带有副作用。

### 预处理语句

[Mysql中的定义](https://dev.mysql.com/doc/refman/8.0/en/sql-prepared-statements.html) 由于SQL本身是一种脚本语言，因此它需要一个编译的过程。在日常使用时经常会出现一些很模板化的语句，我们一般只会改变其中一小部分条件（例如where里的数值）。在这种场景下，可以用预处理语句来节省编译和传输的开销。

它有两种，一种是客户端预处理（也就是我们开发的代码里），另一种是SQL脚本预处理（也就是交给MySQL服务端去处理）。

在Go中的用法： [Using prepared statements](https://go.dev/doc/database/prepared-statements)

### 函数与运算符

由于SQL是一种脚本语言，我们写大段大段的SQL就是为了让MySQL一口气把它处理完，因此在SQL语法中加入一些常用的运算符和工具函数是有必要的。

运算符主要是加减乘除、大小比较、集合运算、空、模糊匹配等。

函数主要是字符串、日期时间、数学运算相关功能的。

### 索引

这其实又是一个很大的话题，下次专门拿出来总结一次吧。

简单说，从表象上来看，有主键、普通索引、唯一索引、联合索引、外键 等很多种类。

深入到底层储存和算法，可以分为聚簇索引/非聚簇索引，可以分为B树/B+树，还有哈希索引、全文索引等等，很多概念。

围绕索引的特性，又能展开性能相关的讨论，坑很深。

### 加锁

`select`语句后面是可以跟`for`来加锁的。要讨论加的锁是什么，那就离不开索引的知识。所以这个也跟下次跟索引一起讨论。

### explain执行分析

`EXPLAIN SELECT ...`可以检查一条SQL语句执行的可能的情况，一般用于帮助定位性能问题（慢查询）。这块也是跟索引紧密相关的，所以也下次跟索引一起研究。

## 运维篇

作为一个后端程序员，不一定要亲手去操作数据库运维，但是懂些原理还是必须的。

最重要的就是MySQL的主从模型了吧，有了它之后可以推导到备份恢复、负载均衡、故障转移等内容。

然后还有`grant`权限视角、密码与安全(TLS)、管理事件（定时任务）、日志监控等等比较冷门的知识。

另开一篇文章研究。

## 专题：TRANSACTION事务

`TRANSACTION` 事务应该是关系型数据库中最重要的特性了（没有之一）。[参考](https://www.runoob.com/mysql/mysql-transaction.html) 我们重点关注一下 

所谓的NoSQL，并不是说"不用SQL语法"，而是说的是"No"掉了传统关系型数据库中的『关系』这个大包袱，更具体一点到使用的层面就是抛弃了『事务』这个特性。丢掉这个大包袱之后，NoSQL们就能在运行性能、拓展性等方面提升到以前无法想象的程度。

最经典的例子就是 余额和流水 的关系了，如下两张表：

```text
+----------+----------+-----------+
| user_id  | balance | user_name |
+----------+----------+-----------+
| 10000233 |      100 | Lewin     |
+----------+----------+-----------+
+----+----------+--------+
| id | user_id  | amount |
+----+----------+--------+
|  1 | 10000233 |    100 |
+----+----------+--------+
```

假如我们要给一个用户扣费，那么大致操作流程是：

1. 检查用户当前余额是否足够
2. 在流水表中新增一条记录
3. 修改余额表中的记录

正常运行没问题，可是在极端情况下，假如有很多个请求并发对同一个用户进行扣费，那么就有可能出现余额与流水不一致的情况，因为在计算新的余额的时候，当前的余额可能已经被其他的请求修改掉了。

> 所以事务本质上就是一种锁，一种带原生回滚机制的锁。当然你可以说，数据库没有事务没关系，可以借助其他中间件（例如Redis\etcd等）额外加一个锁保护并发操作。可以是可以，而且我们目前项目中就是这么干的；但是这仅限于简单的场景，当查询条件变得复杂之后，你确定你自己加的锁能够保护到所有的查询条件吗？以及更重要的回滚和隔离级别问题，不是一个简单的锁可以解决的。

默认配置下，事务是自动提交的，即每一条命令都单独构成一个事务。在需要时我们通过`BEGIN`命令来显式开启一条事务，接着可以输入多条查询语句。

### 事务：隔离级别

要观察事务的作用，首先需要了解一下『隔离级别』的概念以及它们分别对应的四种异常情况。[参考](https://developer.aliyun.com/article/743691)

然后还要再强调一下，隔离级别是对单个连接生效的，而不是对整个数据库。（具体到代码上，由于连接一般都是由连接池管理的，所以可能是每次开启事务时单独指定）

### 事务并发问题：脏读

『脏读』，顾名思义，读到了脏的数据，脏的意思就是另一个事务还正在处理还没有提交的、处于一种不确定状态的数据。

我们先准备一段代码作为"受害者"，它在`READ UNCOMMITTED`隔离级别的情况下做一次普普通通的查询。

```go
func BalanceDirtyRead(db *sql.DB) {
	tx, _ := db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadUncommitted})
	defer tx.Commit()
	var row BalanceRow
	res := tx.QueryRow("SELECT user_id,balance,user_name FROM balance WHERE user_id=?", 10000233)
	res.Scan(&row.UserID, &row.Balance, &row.UserName)
	fmt.Println(row)
}
```

然后我们通过另一个连接，也就是直接从mysql命令行客户端起一个修改数据的事务：

```mysql
begin;
update balance set balance=99 where user_id=10000233;
# 稍后执行 rollback;
```

此时我们通过`READ UNCOMMITTED`（读未提交）级别的事务去`select`，就会读到待会将被rollback的错误的数值`99`，这就是『脏读』。

如果我们将隔离级别提升到`READ COMMITTED`（读已提交），则不会发生脏读。

### 事务并发问题：不可重复读

读已提交 的级别下，依然不能避免这种问题：当开启一个很长的事务，前后多次查询同一个条件的时候，如果中途被别的已提交的事务修改了，那么在这个长的事务中前后读取的结果就不一致了。（用白话翻译，就是我以为我有事务保护着，但是没想到还是别别人改掉了）

我们先准备一段代码作为"受害者"，先开启事务然后读取一次，等待5秒之后再读一次：

```go
func BalanceNonRepeatableRead(db *sql.DB) {
	tx, _ := db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	defer tx.Commit()
	var row BalanceRow
	{
		res := tx.QueryRow("SELECT user_id,balance,user_name from balance WHERE user_id=?", 10000233)
		res.Scan(&row.UserID, &row.Balance, &row.UserName)
		fmt.Println(row)
	}
	time.Sleep(time.Second*5)  // 此时用另一个事务去修改
	{
		res := tx.QueryRow("SELECT user_id,balance,user_name from balance WHERE user_id=?", 10000233)
		res.Scan(&row.UserID, &row.Balance, &row.UserName)
		fmt.Println(row)  // 不可重复读
	}
}
```

在sleep的时候快速地通过另一个终端去修改这个数据：

```mysql
UPDATE LearnMysql.balance t SET t.balance = 98 WHERE t.user_id = 10000233;
```

这时在前面的终端中虽然依然在事务中，但是出现了不可重复读，读到了最新的数字`98`而不是事务之前读到的`100`。

将隔离级别提升到`REPEATABLE READ`（可重复读），就能解决不可重复读的问题。（嗯，听起来像是废话）这也是Mysql的默认隔离级别。

在`REPEATABLE READ`级别下，事务中的数据一定会保持一致，类似于是在`begin`的时候对当前数据库做了一次快照，不管外面怎么改，在事务中查询的都是这个快照的内容。

### 事务并发问题：幻读

`REPEATABLE READ`级别做的快照，它只能保存当前已经存在的数据。如果快照不够用了要去外面读取，就会出现新的类似"不可重复读"的情况，这里叫做幻读。

我们准备一段代码当作"受害者"，同样是读取一次，等待5秒，再读一次：

```go
func BalancePhantomRead(db *sql.DB) {
	tx, _ := db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	defer tx.Commit()
	{
		res, _ := tx.Query("SELECT user_id,balance,user_name from balance WHERE balance.user_id>0")
		PrintAllBalance(res)
	}
	time.Sleep(time.Second * 5)  // 此时用另一个事务去insert/delete
	{
		res, _ := tx.Query("SELECT user_id,balance,user_name from balance WHERE balance.user_name=? for share", "Lewin")  // 注意这里条件不同并且还 for share
		PrintAllBalance(res)  // 幻读
	}
}
```

这里尝试创造幻读的场景还不是那么容易的。

参考 [MySQL InnoDB RR(可重复读)隔离级别能否解决幻读](https://gaoooyh.github.io/2021-09-28-MySQL-InnoDB-RR(%E5%8F%AF%E9%87%8D%E5%A4%8D%E8%AF%BB)%E9%9A%94%E7%A6%BB%E7%BA%A7%E5%88%AB%E8%83%BD%E5%90%A6%E8%A7%A3%E5%86%B3%E5%B9%BB%E8%AF%BB) 大概意思是，Mysql在RR级别下，在事务初期时候创建的快照会对后续一直生效，后续的读都是`快照读`，所以我们需要一些特别的手段去破坏它，让后面的读取变成`当前读`，才能制造出幻读的场景。简而言之：核心区别是`快照读`or`当前读`，在只读时是可以避免幻读，在读写时可能会因为update操作使得不可见的行变得可见，从而出现幻影行。

## 小结

MySQL作为当前最主流的技术之一，本身就是八股文中的重点必考内容。因此在中文社区中，相关的文章数不胜数，质量也是参差不齐，这与我平时研究Golang和前端知识时的"冷清感"是截然相反的，这个现象挺有趣的哈哈哈。

它作为数据库，也理所应当是每个合格的后端程序员必须掌握的知识，而且为了数据安全，我们还必须把每个细节都抠清楚。所以写着写着发现这篇文章的内容越来越多越来越多，也是尴尬。不得不留下两个专题，另外开文章进行研究。下次一定，咕咕咕~
