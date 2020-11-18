```json lw-blog-meta
{"title":"使用Docker-swarm部署整个网站服务","date":"2019-07-30","brev":"Docker真是个令人又爱又恨的东西啊……接触了几个月了，之前一直当作虚拟机来用。最近对Docker的理解深刻了许多，所以换上swarm来整体部署。","tags":["Docker"],"path":"blog/2019/190730-Docker-swarm部署整个网站服务.md"}
```



## “虚拟机”模式

之前一直把docker当作虚拟机来使用，在`docker run`的后面附加大量的参数，像这样：

```shell-session
docker run --name nginx -v /xx/:/xx/   \
-v /xx/:/xx/xx/   \
-v /xx:/xx/xx/  \
--net=xxnet -p 80:80   \
-dit nginx
```

```shell-session
docker run --name blogdj -v /xx/:/xx/ -v /xx/:/xx/xx/ \
        --net=xxnet -dit mypython:1.01 \
        uwsgi /xx/uwsgi.ini
```

实质上只是把容器当虚拟机用，容器中只安装必须的库，所有的数据和网络配置等，都储存在宿主机目录下，通过参数传入容器中使用。

虽然看起来有点麻烦，不过我觉得在实际应用中**非常实用**，只需要两行代码就可以完成应用单元的升级：

```shell-session
git pull 
docker restart xxx
```

但为什么最后还是要用`Docker-swarm`来进行编排呢？我有三点思考：

1. 本地调试不方便。当应用规模不断增大，在本地/服务器端来回切换很容易出错（很容易忘记修改配置），所以每次更新都要来回修正两三遍才能搞定。
2. 别说**切换环境**，就连在本地**配置环境**都越来越麻烦。
3. 不能扩容的Docker没有灵魂。

## 转换为swarm时需要考虑的

1. 静态数据一律封装在容器内部，每个容器包含的数据，都必须是能够独立支撑这个单位运行的所有数据。
2. 动态数据一律使用数据库，数据库用volume持久化。
3. 容器之间一律使用tcp端口通信，就算`Unix-socket`能飞也不能再用了（也许通过volume可以实现，但是不管麻烦与否，首先windows不支持，就会很蛋疼）

那么构思一下我的网站的应用架构：

```text
            **proxy**       **WebServer**       **** DB *****
            *       *       *           *       *           *
client -->  * nginx *  ---> * py-Django *  ---> * Mongo     *
            *       *       * go-Gin    *       * Redis      *
            *       *       *           *       * Postgres  *
            *********       *************       *************
```

ok，接下来进入正题。

## 单个镜像的打包

用到的就是`docker build`命令了。

按照之前说的，我们把所有的静态数据封装进入容器中，我们为每个容器编写一个`DockerFile`，像这样：

```docker
FROM nginx:latest

COPY ./xxx       /xxx/
COPY ./xxx/      /xxx/
COPY ./xxx      /xxx/

EXPOSE 80
EXPOSE 443

CMD ["nginx", "-g", "daemon off;"]
```

每次有更新的话，都build一次（tag自己看着办）：

```shell-session
docker build -t mynginx:latest C:/path/to/DockerfileDirectory
```

然后提交到云端，这里因为我的服务器是腾讯云，所以容器服务也选在腾讯云：

```shell-session
docker tag mynginx:latest ccr.ccs.tencentyun.com/yourname/mynginx:latest

docker push ccr.ccs.tencentyun.com/yourname/mynginx:latest
```

> 注意，要从腾讯云pull/push，要先用docker登录哦。

## 容器编排为服务

当架构中的6项服务全部打包以后（根据自己的情况选择是打包还是直接用原版Image），
我们要把它们组织在一起。

写一个`docker-compose.yml`：

```yaml
version: "3.1"   # 版本信息请根据自己安装的版本和需要的功能酌情选择

services:

  dj:
    image: ccr.ccs.tencentyun.com/x/xx:xxx
    deploy:
      replicas: 1
      ...
    networks:
      - xxxnet

  gin:
    ...

  nginx:
    image: ccr.ccs.tencentyun.com/x/xx:xxx
    deploy:
      ...
    ports:
      - "80:80"
      - "443:443"
    networks:
      - xxxnet

  mongo:
    ...

  postgres:
    ...

  redis:
    ...
```

然后要记得定义`networks`和`volumes`

```yaml
networks:
  xxxnet:

volumes:
  xxxpg:
  xxxmg:
  xxxrd:
```

以上操作是什么意思呢？我们通过`replicas: 1`来理解。

实际上，`Docker-swarm`自带负载均衡，每一个`service`，其实都是一组容器的**集合**。

我们通过`service`的名称作为hostname（前面还要加上`stack`的名称），就可以访问到这一组容器中的任意一个（规则由DockerDaemon决定）。

比如我们在`stack myblog`中的`service dj`中，部署了5个Django容器，那么Docker会把它们分别命名为`myblog_dj.1`, `myblog_dj.2`, `myblog_dj.3`, `myblog_dj.4`, `myblog_dj.5`, 

而我们在应用中只需要访问`myblog_dj`就行了，比如Nginx中：

```conf
location / {
    include  uwsgi_params;
    uwsgi_pass  myblog_dj:80;
    }
```

## 服务栈的部署

`Docker stack`我觉得很难翻译，姑且称之为**服务栈**吧。

 - 一个`stack`，由若干个`service`组成；
 - 一个`service`，由任意个`container`组成。

> 我们通过`Docker-swarm`进行集群部署，一般的管理都是以`service`为单位，很少直接涉及到`container`层级。

直接拿出刚才写的`docker-compose.yml`文件就可以部署啦：

```shell-session
docker stack deploy -c /xxxx/docker-compose.yml myblog
```

这里一定要注意的是，`Docker-swarm`反应比较慢！！一定不要频繁的deploy/rm操作！！一定要确认全部加载完毕之后再进行后一步操作！

使用以下命令确认各项服务的状态：

```shell-session
docker service ls
```

这样我们的服务栈就部署完毕啦。

 - 如果是第一次部署，可能会需要一些初始化的操作，我们使用以下命令：

    ```shell-session
docker ps        # 查询服务栈中容器的具体名称
    docker exec -it xxx bash     # 进入容器中执行命令
    ```

  - 如果是已经部署的服务栈需要更新，我们使用以下命令：

    ```shell-session
docker pull xxxxxxxx       # 把最新的Image拉下来
    docker service update --force xxx_xx      # 强制重启某个service
    docker stack deploy -c xx xx     # 如果yaml文件有更新，直接用这个
    ```

## 附录：本地调试技巧

Web应用的调试的话，**证书**不匹配是一个很大的问题。

一般我们在windows下修改hosts文件就可以解决很大一部分问题：

```text
C:\Windows\System32\drivers\etc
```

然后还要刷新DNS缓存。我这里看好像只需要刷新浏览器的缓存就可以了，以chrome为例：

```text
1. 在chrome://net-internals/#dns 点击 [Clear host cache]
2. 在chrome://net-internals/#sockets 点击 [Flush socket pools]
```

有时还会遇到wss证书不匹配的问题，对于这个只能屏蔽chrome的验证机制了。在chrome启动时附加参数：

```text
"C:\xxx\xxx\xxx\chrome.exe" --ignore-certificate-errors
```

## 小结

在单机环境下运行docker-swarm模式，对于性能肯定会有不小的影响了，我们从它的封装/虚拟化机制就可以感受出来。

另外，对于更新的数据量也会增加很多，只是腾讯云在国内的环境不太在乎这个罢了。

但是也有一些好处，比如每个容器单元独立完整，比如整体部署有利于本地调试等。

最重要的是，这是学习Docker路上的重要环节。

不能伸缩的Docker是没有灵魂的！


## 牢骚几句

个人认为，`Docker`的社区生态环境目前不太好。

1. 首先，它整个生态有很多工具。

    - 最著名也是最基础的部分是`Docker`本体，实质上就是虚拟化技术，是针对单个**应用单元**的打包和部署。
    - `Image`多了，那就要进行**容器编排**了，所以出现了`Docker-compose`，我理解它是一个单机版本的快速部署工具。但我认为不好用。
    - 单机性能不够用了，需要**集群+弹性扩容**，那么就需要`Docker-swarm`来支持了。
    - 最大的问题是，`Docker-swarm`虽然是官方的工具，但并不是最流行的。最流行的应该是`k8s`，我们从招聘信息中可见一斑。

2. 其次，Docker发展非常快，release非常密集。

    - 最新版本是19.03.1，发布于2019年7月25日。上一个常见的版本是18.09.0，发布于2018年11月8日。不到一年的时间中，有10个版本发布。这谁顶得住啊？
    - 新版本对旧版本的兼容性不足。我们从常用的`.yaml`文件的参数就可以发现，从`v 1`到现在的`v 3.7`，参数的变化非常大，旧的参数经常被删除重写。

3. 再次，国内的普及度不算高。

    本身这个技术还并没有得到大规模的应用，敢于吃螃蟹的团队还是极少数（或者大厂团队在Docker早期发展时就迫不及待地进行了二次开发），所以国内可用的资料很少。

    就算有资料，也受限于前面两点原因，多会因为版本兼容的问题处处碰壁。

4. 最后，必须要吐槽Docker命令行提示太少了！

    为了写一句：

    ```shell-session
docker-compose -f xxx.yml -p xxx up
    ```

    就这么简单一个语句，我没注意，总是把up放在前面，写成这样：

    ```shell-session
docker-compose up -f xxx.yml -p xxx
    ```

    然后一直报错，而且一直没有出现任何有价值的提示。官网上、搜索里都只说`docker-compose up`，却极少有实际应用的经验博客供参考，导致这个小问题就折腾了我一个小时……太痛苦了