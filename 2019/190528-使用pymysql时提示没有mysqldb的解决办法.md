```yaml lw-blog-meta
title: 使用pymysql时提示没有mysqldb的解决办法
date: "2019-05-28"
brev: 在我的windows上运行sqlalchemy一切正常，但是进入到 python容器(linux环境)中运行就报错。
tags: ["中间件"]
```


## 报错情况

运行`sqlalchemy`时报错：

```shell-session
root@2492dd2ac42b:/scripts/APMOS# python somepython.py
Traceback (most recent call last):
  ...
  File "apmos2_etls/ApmosData/Cash.py", line 49, in __init__
    self.sqla = sqlalchemy.create_engine(MysqlDB_Sever["login"])
  File "/usr/local/lib/python3.7/site-packages/sqlalchemy/engine/__init__.py", line 435, in create_engine
    return strategy.create(*args, **kwargs)
  File "/usr/local/lib/python3.7/site-packages/sqlalchemy/engine/strategies.py", line 87, in create
    dbapi = dialect_cls.dbapi(**dbapi_args)
  File "/usr/local/lib/python3.7/site-packages/sqlalchemy/dialects/mysql/mysqldb.py", line 118, in dbapi
    return __import__("MySQLdb")
ModuleNotFoundError: No module named 'MySQLdb'
```

## 尝试解决

按理说我这个官方的python镜像应当非常健康才对，检查python环境：

```shell-session
root@2492dd2ac42b:/scripts/APMOS/apmos2_etls/DBtools# pip list
Package         Version
--------------- ---------
Django          2.2.1
get             2019.4.13
IMAPClient      2.1.0
numpy           1.16.3
pandas          0.24.2
pip             19.1.1
post            2019.4.13
psycopg2        2.8.2
public          2019.4.13
pymongo         3.8.0
PyMySQL         0.9.3
python-dateutil 2.8.0
pytz            2019.1
query-string    2019.4.13
redis           3.2.1
request         2019.4.13
setuptools      41.0.1
six             1.12.0
SQLAlchemy      1.3.3
sqlparse        0.3.0
wheel           0.33.1
xlrd            1.2.0
XlsxWriter      1.1.8
```

尝试安装`mysqldb`：

```shell-session
root@2492dd2ac42b:/scripts/APMOS# pip install mysqldb
Collecting mysqldb
  ERROR: Could not find a version that satisfies the requirement mysqldb (from versions: none)
ERROR: No matching distribution found for mysqldb
```

竟然提示没有`mysqldb`这个包！

[搜索一番](https://www.pythonanywhere.com/forums/topic/1212/)得出答案：  

> the correct pip package for Python 3.x has changed since the last update on this thread. 
If you're using a virtualenv with Python 3, you can install it like this:

对于python3正确解决办法是：

```shell-session
pip install mysqlclient
```

安装！
```text
root@2492dd2ac42b:/scripts/APMOS# pip install mysqlclient
Collecting mysqlclient
  Downloading https://files.pythonhosted.org/packages/f4/f1/3bb6f64ca7a429729413e6556b7ba5976df06019a5245a43d36032f1061e/mysqlclient-1.4.2.post1.tar.gz (85kB)
     |████████████████████████████████| 92kB 39kB/s
Building wheels for collected packages: mysqlclient
  Building wheel for mysqlclient (setup.py) ... done
  Stored in directory: /root/.cache/pip/wheels/30/91/e0/2ee952bce05b1247807405c6710c6130e49468a5240ae27134
Successfully built mysqlclient
Installing collected packages: mysqlclient
Successfully installed mysqlclient-1.4.2.post1
```
然后问题解决！
