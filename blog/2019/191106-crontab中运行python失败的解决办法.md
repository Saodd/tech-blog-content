```lw-blog-meta
{"title": "crontab中运行python失败的解决办法", "date": "2019-11-06", "tags": ["OS"], "brev": "最近在将公司项目打包进Docker中，除了python依赖环境，crontab也是重要的部分。但是在转移crontab的时候遇到很奇怪的问题，python不能正常执行。"}
```

## 首先cron是正常的

在镜像中是安装了最新的cron，实例化容器后手动启动cron进程：

```shell-session
$ docker run -dit -e PYTHONPATH=xxxx python:3.7.4 bash
$ docker exec -it xxxx crontab /srv/apmos/ApmosReconcile/crontab
$ docker exec -it xxxx service cron start
```

先测试了一条最简单并且也可以观测的命令，执行正常：

```text
* * * * * echo "haha" > ${PROJECT_PATH}/../log.txt
```

## python版本问题

在bash中和在cron中运行的是完全不同的python版本：

```shell-session
$ python --version
Python 3.7.4
$ which python
/usr/local/bin/python
```

```text
* * * * * python --version >/srv/log.txt
# 输出 Python 2.7.16
* * * * * which python >/srv/log.txt
# 输出 /usr/bin/python
```

### 解决办法

指定python程序的绝对路径

```text
* * * * * /usr/local/bin/python some_python.py
```

或者在crontab文件头部设置`PATH`变量，将python3的目录放在最前面

```text
PATH=/usr/local/bin:xxxxx:xxxxx:xxxxx
* * * * * python some_python.py
```

## 环境变量问题

在Docker容器实例化时传入的`-e PYTHONPATH=xxx`参数，在cron中是无效的，进而导致报错『No module named 'xxx'』。

### 解决办法1

在crontab每行前面临时设置环境变量

```text
* * * * * export PYTHONPATH=${PROJECT_PATH}; python some_python.py
```

### 解决办法2

最好的办法，是在crontab最上方定义`PYTHONPATH`。语法类似`shell`，但是不支持变量给变量赋值，每个变量都要通过值来赋值。

```text
PROJECT_PATH='/srv/apmos/ApmosReconcile'
PYTHONPATH='/srv/apmos/ApmosReconcile'

* * * * * python some_python.py
```

下面的赋值方法是无效的：

```text
PYTHONPATH=${PROJECT_PATH}
```
