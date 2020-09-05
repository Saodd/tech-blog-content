```yaml lw-blog-meta
title: 使用Docker部署Nginx uWSGI Django
date: "2019-06-14"
brev: 学习Django很久了，但是一直都是在其自带的测试服务器 manage.py runserver 上运行，今天来看看生产环境的部署。生产环境当然是首推Nginx了。
tags: [Docker]
```


## 总体框架

```text
Client  <-->  Proxy Server(Nginx)  <-->  uWSGI  <-->  Django
```

用户发起request，指向的是代理服务器，然后由代理服务器（进程）根据规则转发到项目服务器（进程）（这个过程可以做负载均衡），
每个项目服务器里实际又是由uWSGI组织的多个Django工作节点。  

（所以顺便Restful也就顺理成章了，无状态的请求才能适应这套机制。）



## 前提准备：一个Django项目

我们这里已经准备好了，项目名称叫做`apdj`，项目文件映射在容器的`/scripts/APMOS/apmos4_view/apdj`路径下。

```text
apdj
   |-- apdj
   |    |-- settings.py
   |    |-- WSGI.py
   |-- manage.py
```


## 配置项目服务器uWSGI+Django

首先构建一个python容器：

```bash
$ docker run --name mypython --net=some_net -p 20001:20001 -dit python:latest
$ docker exec -it mypython bash

```

>这里要注意，最好构建一个net网络，这样有利于容器之间的互相访问。容器内部的python库的安装我也不说了。
然后要挂载volume的话自己看着办。端口也自己随便选择(这里主要是为了直接测试uWSGI所以暴露了端口)。


### 配置uWSGI

*安装*这里要特别注意，一定要从`pip`安装，有些教程让你从`apt`安装，会很麻烦。

```bash
[root@容器]# python -m pip install uwsgi
```

> 使用`python -m pip`可以给你指定的python版本安装。

安装完成后进入Django的项目目录，尝试运行一下：

```bash
[root@容器]# cd /path/to/your/django
[root@容器]# uwsgi --http 0.0.0.0:20001 --module apdj.wsgi
```

> 端口号一定要是你容器开放的端口号，参数名http意思是在这个端口监听http请求；
module后面跟的是你的`WSGI.py`文件的路径，以python风格表示。

此时会输出一堆内容，像这样的：
```text
detected binary path: /usr/local/bin/uwsgi
uWSGI running as root, you can use --uid/--gid/--chroot options
*** WARNING: you are running uWSGI as root !!! (use the --uid flag) ***
*** WARNING: you are running uWSGI without its master process manager ***
your memory page size is 4096 bytes
detected max file descriptor number: 1048576
lock engine: pthread robust mutexes
thunder lock: disabled (you can enable it with --thunder-lock)
uWSGI http bound on 0.0.0.0:20001 fd 4
spawned uWSGI http 1 (pid: 642)
uwsgi socket 0 bound to TCP address 127.0.0.1:39787 (port auto-assigned) fd 3
uWSGI running as root, you can use --uid/--gid/--chroot options
*** WARNING: you are running uWSGI as root !!! (use the --uid flag) ***
Python version: 3.7.3 (default, May  8 2019, 05:28:42)  [GCC 6.3.0 20170516]
*** Python threads support is disabled. You can enable it with --enable-threads ***
Python main interpreter initialized at 0x562621f6d190
uWSGI running as root, you can use --uid/--gid/--chroot options
*** WARNING: you are running uWSGI as root !!! (use the --uid flag) ***
your server socket listen backlog is limited to 100 connections
your mercy for graceful operations on workers is 60 seconds
mapped 72920 bytes (71 KB) for 1 cores
*** Operational MODE: single process ***
WSGI app 0 (mountpoint='') ready in 0 seconds on interpreter 0x562621f6d190 pid: 641 (default app)
uWSGI running as root, you can use --uid/--gid/--chroot options
*** WARNING: you are running uWSGI as root !!! (use the --uid flag) ***
*** uWSGI is running in multiple interpreter mode ***
spawned uWSGI worker 1 (and the only) (pid: 641, cores: 1)
[pid: 641|app: 0|req: 1/1] 172.21.0.1 () {38 vars in 1060 bytes} [Fri Jun 14 15:30:46 2019] GET / => generated 3878 bytes in 381 msecs (HTTP/1.1 200) 4 headers in 124 bytes (1 switches on core 0)
```
有一些warning不要紧，只要没有error啊，traceback之类恐怖的字眼，并且uwsgi还在运行，就可以了。  
我们试试用浏览器访问20001端口，已经可以正常运行了（输出是上面的最后一行）。  

接着稍微优化一下，我们在项目根目录下（manage.py旁边）创建一个`uwsgi.ini`文件，写入：

```ini
[uwsgi]
chdir = /scripts/APMOS/apmos4_view/apdj
module = apdj.wsgi:application
socket = 0.0.0.0:20001
master = true
workers = 2
daemonize = （日志文件路径，先去创建好）
disable-logging = true

```

> 注意，是给`--socket`参数配置的端口号（之前是http），意思是在这个端口监听socket请求（nginx转发过来的）


这样我们就可以简化启动命令了：

```shell
# Ctrl+C结束上一个uwsgi进程
# 还在刚才的路径下运行,否则要改成绝对路径
[root@容器]# uwsgi uwsgi.ini
```

这样只会给出一条提示：
```text
[uWSGI] getting INI configuration from uwsgi.ini
```
日志就要去你刚才指定的文件中去找了：
```text
*** Starting uWSGI 2.0.18 (64bit) on [Fri Jun 14 07:41:29 2019] ***
compiled with version: 6.3.0 20170516 on 14 June 2019 05:04:27
os: Linux-4.9.125-linuxkit #1 SMP Fri Sep 7 08:20:28 UTC 2018
nodename: 7da0359ad079
machine: x86_64
clock source: unix
pcre jit disabled
detected number of CPU cores: 2
current working directory: /scripts/APMOS/apmos4_view/apdj
detected binary path: /usr/local/bin/uwsgi
uWSGI running as root, you can use --uid/--gid/--chroot options
*** WARNING: you are running uWSGI as root !!! (use the --uid flag) *** 
chdir() to /scripts/APMOS/apmos4_view/apdj
your memory page size is 4096 bytes
detected max file descriptor number: 1048576
lock engine: pthread robust mutexes
thunder lock: disabled (you can enable it with --thunder-lock)
uwsgi socket 0 bound to TCP address 0.0.0.0:20001 fd 3
uWSGI running as root, you can use --uid/--gid/--chroot options
*** WARNING: you are running uWSGI as root !!! (use the --uid flag) *** 
Python version: 3.7.3 (default, May  8 2019, 05:28:42)  [GCC 6.3.0 20170516]
*** Python threads support is disabled. You can enable it with --enable-threads ***
Python main interpreter initialized at 0x55c7af4d9470
uWSGI running as root, you can use --uid/--gid/--chroot options
*** WARNING: you are running uWSGI as root !!! (use the --uid flag) *** 
your server socket listen backlog is limited to 100 connections
your mercy for graceful operations on workers is 60 seconds
mapped 218760 bytes (213 KB) for 2 cores
*** Operational MODE: preforking ***
WSGI app 0 (mountpoint='') ready in 1 seconds on interpreter 0x55c7af4d9470 pid: 648 (default app)
uWSGI running as root, you can use --uid/--gid/--chroot options
*** WARNING: you are running uWSGI as root !!! (use the --uid flag) *** 
*** uWSGI is running in multiple interpreter mode ***
spawned uWSGI master process (pid: 648)
spawned uWSGI worker 1 (pid: 649, cores: 1)
spawned uWSGI worker 2 (pid: 650, cores: 1)
```

> 这里吐槽一下uWSGI的日志，我看见了一句：  
Fri Jun 14 15:05:07 2019 - uWSGI worker 2 screams: UAAAAAAH my master disconnected: i will kill myself !!!  
"2号苦工尖叫了起来：呜啊啊啊，我的主人抛弃我了，我要结束我的生命！！！"  
笑死我了好吗哈哈哈哈！！！

然后再用浏览器测试一下，搞定！








## 配置Nginx

Nginx运行在另一个容器中：
```shell
$ docker run --net=somenet --name mynginx -p 8080:80 -d nginx
```

> 你也可以不加`-d`，仔细观察它的输出。
net必须跟你前面Django项目的容器的是同一个，这样才能互相可见。至于volume自己安排，把静态文件挂载进去。

这时候我们从外部访问宿主机的8080端口（即Nginx的默认端口80，也是Http的默认端口），
可以看到Nginx自带的提示。  

接下来就把Nginx和uWSGI联系起来。

Nginx的配置文件是`/etc/nginx/conf.d/default.conf`，你可以进入Nginx的`容器`内部用cat去操作，
也可以在宿主机上写好然后用volume挂载进去，也可以在`docker build`的时候给他安排好。总之八仙过海自己想办法。

主要要写入的内容（也就是转发规则，如果原来的文件有冲突要适当调整）：
```text
server {
    listen       80;
    server_name  随便起个名字？;

    # 这里转发所有的"/..."
    location / {   
        include  uwsgi_params;
                 uwsgi_pass  mypython:20001;
    }

    # 这里转发所有的"/static"
    location /static {
        alias /scripts/APMOS/apmos4_view/apdj/static;
    }
}
```
覆盖了原来的文件之后，我们重启一下Nginx:
```shell
# 在宿主机
$ docker restart mynginx

# 或者在容器内部操作，不过最后你还是要出来重启的
[root@容器]# service nginx restart

```

在浏览器中访问，此时二者已经连接起来了。搞定！！

这样就形成了预期的框架结构：
```text
Client  -->  访问80端口，Nginx监听
Nginx  -->  根据规则转发到20001端口，uWSGI监听
uWSGI  -->  把请求分配给空闲的worker
Django ---  执行
```

## 进阶：使用Unix-Socket

现在`Nginx`与`uWSGI`之间还是使用TCP-socket进行通讯，其实还可以用Unix-socket性能更好。

> 2019-07-13 注：有一些文章表明，Docker在`bridge`网络模式下，可能使用的就是`Unix-socket`；
> 只不过是伪装了，在容器内表现还是`tcp-socket`。（但是我个人还没有在官方文档中找到明确证据）
> 我觉得是有可能的，因为这种性能问题是应当被考虑到的。如果真是这样的话，
> 我们就没必要去自己设定`Unix-socket`了。

修改`uwsgi.ini`文件:

```ini
[uwsgi]
chdir = /scripts/APMOS/apmos4_view/apdj
module = apdj.wsgi:application
socket = /docker/uwsgi.sock    # 改这里！注意，这里是python容器中的路径，要从宿主机挂载进来
master = true
workers = 2
daemonize = （日志文件路径，先去创建好）
disable-logging = true

```

对应的，修改`Nginx`的配置文件：

```text
    # 这里转发所有的"/..."
    location / {   
        include  uwsgi_params;
                 uwsgi_pass  unix:///docker/uwsgi.sock;  # 改这里！注意，这里是nginx容器中的路径，要从宿主机挂载进来
    }

```

实质就是通过文件系统来连接套接字。所以我们也要相应的**重启两个容器**，并挂载同一个目录进去：

```shell
$ docker run ... -v /docker/:/docker/ ...
# 前面一个/docker/是宿主机的目录，后面/docker/是容器内的目录
```

重启之后，尝试访问网页，成功！
并且我们可以在`uWSGI`容器内找到之前定义的日志文件，可以看到这样一行：

```text
uwsgi socket 0 bound to UNIX address /docker/uwsgi.sock fd 3
```

但是要注意，使用Unix-Socket虽然性能会好一些，不过也大大地降低了可维护性和可拓展性，
以我个人经验来看，如果不是特别需要的话，还是不要用了。

## 进阶：使用`.pid`文件管理uWSGI进程

修改`uwsgi.ini`文件，增加一行:

```ini
pidfile=/scripts/uwsgi.pid
stats=/scripts/uwsgi.status
```

这样就会把`uWSGI`的进程信息写入这个文件中，我们可以通过这个文件便捷地管理`uWSGI`进程。

```shell
# 查看状态
[root@container]# uwsgi --connect-and-read /scripts/uwsgi.status

# 重启/关闭
[root@container]# uwsgi --reload /scripts/uwsgi.pid
[root@container]# uwsgi --stop /scripts/uwsgi.pid

```

# 小结

`Nginx`和`uWSGI`两个东西还是有点小坑的，不过经历了一遍就好了，其实也并不难。

