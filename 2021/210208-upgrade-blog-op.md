```yaml lw-blog-meta
title: '个人网站升级日记'
date: "2021-02-08"
brev: "k8s, caddy+HTTP3, 把牛刀升级成屠龙刀。"
tags: ["DevOps"]
```

## 前言

不要问我为什么，问就是闲的。

由于是个人项目，不展示代码，只说说思路。

## 在本地使用HTTPS

为了测试方便，我希望在本地直接启动HTTPS服务。由于我这是个人项目，网站证书个人保管，所以复制到本地也就是动动手指的事。如果是公司项目要用，解决方案应该是自签证书。

去 `C:\Windows\System32\drivers\etc` 修改hosts文件，把域名和子域名都指向本地（或者开发机）。

```text
127.0.0.1 lewinblog.com
127.0.0.1 www.lewinblog.com
127.0.0.1 api.lewinblog.com
```

上述行为其实充当了一种网络中间人攻击，劫持了DNS 。而我与中间人的最大区别是，我有正确的证书，所以我的「劫持」是有效的，而普通的中间人即使劫持了DNS也绕不开TLS的验证。但是，如果中间人从某种渠道获得了由有效根证书签发的证书的话，那就真正地实现了HTTPS劫持。

然后把线上服务的Image拉到本地，把原来 docker swarm 的配置文件重写成 k8s 配置文件。运行，然后在浏览器中验证一切正常工作。

然后接下来全部在本地开发，验证完成后再到线上滚动更新。

## 步骤一：Nginx迁移Caddy

为什么要用Caddy呢？因为貌似目前只有Caddy才支持 HTTP/3 ，至于Nginx，它的社区在2020年6月提出要做，以一个新项目的名义做，结果做了半年也没见成品问世……在这个开源技术井喷的时代，Nginx已经没有垄断地位了，这慢吞吞的动作搞不好要凉凉哦。

之前我在 [学习Caddy的文章](../2021/210127-caddy-gin-jwt.md) 中有说过，说Caddy的配置文件非常好写，我今天实际（在尽可能接近生产环境配置的难度下）写了一下，觉得还是很难写，还是要像Nginx一样，这里抄抄那里抄抄。

所以抄了这么多的我也回馈一下社会吧，简要贴一下我的配置：

```text
{
    experimental_http3
}

www.lewinblog.com {
  tls /crt/xxx.crt /crt/xxx.key
  encode zstd gzip

  @static_files {
    path_regexp \.(jpg|jpeg|png|gif|swf|svg|mp4|eot|ttf|otf|woff|woff2|css|js)$
  }
  @static_icon {
    path_regexp \.(ico)$
  }
  header Strict-Transport-Security max-age=31536000;includeSubDomains;
  header @static_files  Cache-Control max-age=31536000
  header @static_icon   Cache-Control max-age=604800
  header               ?Cache-Control no-cache

  root * /xxxx
  try_files {path} /index.html
  file_server
}

lewinblog.com {
  redir https://www.lewinblog.com{uri}
}

api.lewinblog.com {
  tls /crt/xxxxx.crt /crt/xxxxx.key
  reverse_proxy xxxxx:80
}
```

中间那一大坨是关于缓存头的，简单说，就是对于css之类的文件用最长的缓存时间；对ico特殊处理一下只缓存一周；剩下的都用 no-cache 策略，也就是检查 e-tag 的模式。

然后其他的就很简单，前端文件用 try_files 后端项目用 reverse_proxy ，没啥好说的。

不过这里要吐槽一下，Caddy只支持 TLSv1.2 了，想向下兼容吗，别想了。然后加密套件这里也没有设置，我估摸着以它的个(niao)性应该会用安全程度很高的加密套件吧，不想去操心了，毕竟 TLSv1.2 就已经很安全了。

然后还发现了一个新大陆，关于压缩算法，一般主流都是`gzip`吧，这次研究过程中发现一些很新潮的服务商用的是`br`算法，看起来很强大，回头琢磨一下，咕咕咕。

哦对了，差点忘记最关键的 HTTP/3 了。其实吧这玩意，原理很简单，就是用 UDP+应用层保证可靠性 代替了 TCP 。

> 可能内心有点小疑问，为啥不在TCP这一层搞个新协议，而是要基于UDP来做呢？对此我的理解是，TCP/UDP及以下的协议，一般应当算作是「基础设施」，这种东西升级换代的阻力非常大；而HTTP这个级别的应用层协议则是五花八门，搞点新花样大家也见怪不怪，很容易接受。所以基于UDP来搞新协议是明智的选择。

HTTP/3 的配置，在Caddy里面非常简单，就上面那一句话就可以了。然后记得要打通 443/UDP 的传输路径，例如我这里先要在 k8s 的 Service 上指定暴露 UDP，然后还要去云服务商的网络安全策略里设置对UDP放行，很快。

设置完之后稍等一会，就能在浏览器中进行验证了。

## 步骤二：写config

玩了几天的 k8s ，我觉得它的配置文件（或者应该叫做资源文件？）还真是挺好写的。一方面体现在，文档比较齐全，特别是中文文档很齐全，而且估计用的人也很多，搜索问题都能搜到，目前还没遇到搜索不到的问题。

搭建我这个个人网站，需要如下几个应用单元：

- 一个反向代理 (Caddy)
- 一个Web服务 (Golang - gin)
- 一个数据库 (Mongo)

没有折腾Ingress，直接用NodePort的方式简单粗暴地运行。

> 不过这里提一句，腾讯云是提供免费的3节点控制平面的，但是代价是必须要用它的付费负载均衡服务。之前我以为这个负载均衡就是普通的负载均衡，现在学了k8s才知道它是直接对接k8s的负载均衡……

前面两个应用都是无状态的，所以只需要 Deployment 和 Service 就可以了。而数据库是有状态的，就额外需要 PersistentVolumeClaim 了，也就是声明卷，跟之前Docker的用法类似，不过k8s对这个概念做了更抽象的封装，支持更多底层硬件服务。

最后呢，由于这一下子写了这么多东西，我就给顺手加了一个 Namespace 便于管理。

## 步骤三：滚动更新策略

k8s最强的地方就在于不停机滚动更新了。我这里没有过于深究，毕竟我这个个人网站对可用性完全没有要求呢（只有我自己对自己的要求）。简单来说主要有这么几个方面：

1. 滚动策略：比如预期10个Pod，那么在滚动更新期间，最多能额外开几个，最少要保证留下几个可用，这类的设定。
2. 健康检查策略：比如有个 ReadinessPod(读探针) 的东西，实质上就是在Pod启动阶段的健康检查。检查通过之后才会被注册到服务中。
3. 镜像拉取策略：一般用tag为`latest`的镜像是常态吧，这种状态下可以让它每次强制拉取最新的镜像，达到简化操作的目的。

关于镜像拉取这里多说一点，因为对于私有仓库来说，涉及到 docker login 的问题。

在k8s中是通过`secret`来解决的。我们可以给kubectl直接传入明文的用户、密码，也可以传入认证后的token，参考 [从私有仓库拉取镜像 - 官方文档](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/) （注意这里中文文档比英文文档少了一部分内容）

我简单归纳一下，我的操作是先在宿主机（管理节点）上执行 docker login ，然后登录后有个token会保存在当前用户的目录下：

```shell
cat ~/.docker/config.json
```

然后用这个文件来创建一个 k8s secret ，这东西也是一种k8s资源，跟 Deployment 那些东西属于相同的抽象层级。设置方法：

```shell
kubectl create secret generic regcred \
    --from-file=.dockerconfigjson=<path/to/.docker/config.json> \
    --type=kubernetes.io/dockerconfigjson
```

这里有个坑，坑了我好久，就是：secret 也是要区分 namespace 的，所以如果要在应用内使用它，则在创建的时候要把它放到指定的命名空间里去`--namespace xxx` （或者可以通过`.namespace`的方式跨命名空间访问？好像不太安全，我不确定）

然后在Pod里面配置：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  template:
    spec:
      imagePullSecrets:
        - name: regcred  # 写在这个层级
```

然后还有 ConfigMap 其实我是需要研究一下的，以后有时间再说吧。

## 步骤四：在云服务器上安装k8s

经过上面的步骤，我的个人网站就完全可以由一个 .yaml 文件所描述了，我只要拿着这个配置文件，到哪里我都可以瞬间重新搭建我的网站。

接下来就是把它应用到云服务器上。我这里用的是腾讯云，2u4g，单机运行绰绰有余。

安装过程在我 [之前的k8s的文章](../2021/210206-learn-k8s.md) 中介绍过，不再赘述。不过这次在完全"正常"的网络环境下安装k8s，即使是这几天已经反复安装过几遍的我也踩了好久的坑……

1. `apt-get`不认环境变量中的代理变量，要去`/etc`里写个文件。
2. `k8s.gcr.io`的镜像拉不下来。虽然国内好像有镜像源，但是至少阿里的镜像源不是最新的？总之拉取失败了。然后Vxxx代理又很神奇地不能识别 docker 的通讯格式，就很奇怪。所以最后我很蠢地，手动从本地上传镜像上去用的……（以后要更新都不知道咋办了）（以后再说吧）
3. `Calico` 的配置时间太久了。以至于我以为哪里搞错了，甚至把 docker 都卸载了重新安装……然后最后发现，只要耐心等等就好了  X)

总之，在代理的帮助下，半自动半手动地、反反复复地、我总算是把k8s给装好了。看见Ready的那一瞬间真的好高兴啊  X)

## 步骤五：部署应用

这一步是最简单的了。把yaml文件拖到服务器上去，apply一下，锵锵锵！！我宕机了一个小时的个人网站就在一瞬间复活了！！(一下子就少了好几个9呢)

打开浏览器验证一下，emmm……发现数据都被清空了呢，毕竟换了一个卷啊。

不过反正我也不依赖服务器上的数据库，直接去 Github 上点几下，让 Actions 重新跑一编，数据就哗哗地导入过来了！！

## 感想

折腾过一遍之后，我觉得k8s还真挺重要的，所谓的「一辈子都用不上」绝对是一种误解，事实正相反，我觉得我接下来天天都可以用它。我觉得它应当像Docker一样普及，每个后端程序员都应该会像吃饭喝水一样地使用它。

所以目前我眼中的程序员，应当划分为3个层次：

1. 交付 git-repo （写基本的业务代码）
2. 交付 Docker Image （解决构建问题，使业务成为独立运行单元，不给别人添麻烦）
3. 交付 Deployment 甚至 Namespace （关注服务间的交互，参与服务治理）

## 后记

其实昨天晚上我还在想着，年前是不是别去折腾呢，毕竟万一没搞定的话，过年这几天都在宕机状态，那我的自信就要崩溃了呢……

不过今天的体验来看，一切都还算顺利，折腾到现在也就才下午6点多而已。这下真的可以安安心心回家过年了。

新年快乐~