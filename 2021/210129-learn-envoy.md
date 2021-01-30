```yaml lw-blog-meta
title: 'Envoy入坑日记'
date: "2021-01-29"
brev: "Envoy是集群间通讯代理工具。学过它之后，我好像知道为什么Caddy没有列入CNCF了。"
tags: ["中间件"]
```

## 学习资料

首先，老生常谈，放上[官方文档](https://www.envoyproxy.io/docs/envoy/v1.17.0/) 但我觉得这个官方文档，写了等于没写，我把 Getting Started 部分读了一遍，发现我变得更加蒙蔽了。

于是，我再推荐 [中文版文档](https://www.servicemesher.com/envoy/intro/what_is_envoy.html) 这份文档好歹是列在官方文档的列表里的，而且我看了下，翻译质量非常高（我也翻不到更好了），所以还是很值得去看的。虽然看了之后依然一脸蒙蔽，没好到哪去。

最后，强烈推荐你直接去看[官方仓库](https://github.com/envoyproxy/envoy) 中的 Examples ，并且推荐从 front-proxy 部分开始看。

> 我悟了，程序员的本质果然是复印机。

## 基本运维

它是用C++写的，反正我是不想看它的源代码，依照惯例，直接Docker走起：

```shell
docker pull envoyproxy/envoy
```

## 基本操作1：HTTP代理

HTTP代理的部分，我体验下来，觉得跟前两天刚刚研究的 [Caddy](./210127-caddy-gin-jwt.md) 在功能上是高度重合的。也就是说Envoy可以胜任一个最前端（或者称为`边缘`）的代理职责的。

所以偷个懒，继续沿用我在研究caddy时所写的 [两个服务](https://github.com/Saodd/learn-caddy) 。

有一说一，Envoy的配置文件真的太太太太太复杂了，而且是用非常反人类的`yaml`格式写的，严重依赖缩进，一不小心就会把配置写坏。

> 什么？你说YAML的设计目标是为了人类可读？……那你我之间可能有一个不属于人类……

有一些概念先讲解一下，有利于理解配置文件的写法：

1. Envoy的本职工作是 L3/L4代理（也就是直接操作TCP），但是也能做 L7代理（也就是处理HTTP），所以HTTP代理仅仅只是它的一个模块而已。
2. `listener` 是它监听的入口，`cluster`是它代理的资源。

然后接下来的例子，我将启动一个auth服务，它是一个gin框架搭建的HTTP服务，它负责`/auth`路径的资源，它将运行在`10.0.1.232:30000`地址上。我尝试用Envoy去反向代理到它上面去：

```yaml
static_resources:
  listeners:
    - name: listener_0
      # 这里是Envoy监听的地址
      address:
        socket_address:
          protocol: TCP
          address: 0.0.0.0
          port_value: 10000
      filter_chains:
        - filters:
            - name: envoy.filters.network.http_connection_manager
              typed_config:
                "@type": type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager
                stat_prefix: ingress_http
                route_config:
                  name: local_route
                  virtual_hosts:
                    - name: local_service
                      domains: [ "*" ]
                      routes:
                        # 这里配置路由转发规则
                        - match:
                            prefix: "/auth"
                          route:
                            cluster: service_auth  # 这个名字对应 下面定义的cluster的名字
                http_filters:
                  - name: envoy.filters.http.router
  clusters:
    - name: service_auth
      connect_timeout: 0.25s
      type: LOGICAL_DNS
      dns_lookup_family: V4_ONLY
      lb_policy: ROUND_ROBIN
      load_assignment:
        cluster_name: service_auth
        endpoints:
          - lb_endpoints:
              # 这里定义 被代理的上游资源
              - endpoint:
                  address:
                    socket_address:
                      address: 10.0.1.232
                      port_value: 30000
```

然后把这个配置文件挂载进去。在镜像中，配置文件的位置在`/etc/envoy/envoy.yaml`：

```shell
docker run --rm -p 10000:10000 -v ${pwd}/envoy.yaml:/etc/envoy/envoy.yaml -it envoyproxy/envoy
```

然后在浏览器中直接访问`localhost:10000`，就像Caddy一样，然后就能观察到一切都顺利运行，我们的请求被成功地代理到那个auth服务上去了。

## 基本操作2：HTTP负载均衡与健康检查

在之前学习caddy时还写了另一个服务，它负责`/business/1`这个路径的资源。为了测试负载均衡，我将会在30001和30002端口上分别启动这个服务实例。

更新 envoy.yaml ，这次不全量复制了，我觉得应该能看懂吧：

```yaml
- match:
    prefix: "/business"
  route:
    cluster: service_business
```

```yaml
- name: service_business
  connect_timeout: 0.25s
  type: strict_DNS  # 注意DNS的模式要换一个
  dns_lookup_family: V4_ONLY
  lb_policy: ROUND_ROBIN  # 负载均衡模式是轮流
  
  # host配置，可以简写成这样
  #      hosts: [
  #        {socket_address: { address: 10.0.1.232, port_value: 30001 }},
  #        {socket_address: { address: 10.0.1.232, port_value: 30002 }},
  #      ]
  load_assignment:
    cluster_name: service_business
    endpoints:
      - lb_endpoints:
        # 这里定义了两个服务的地址
        - endpoint:
            address:
              socket_address:
                address: 10.0.1.232
                port_value: 30001
        - endpoint:
            address:
              socket_address:
                address: 10.0.1.232
                port_value: 30002
```

如果我只启动一个business服务，那么在浏览器中访问时会发现，有时请求成功有时失败，这是因为在没有健康检查的情况下，它会无脑地将请求进行分配，于是被发往还没启动的端口的请求自然就会失败了。

然后继续配置健康检查。注意，健康检查的配置是`clutser`的下一级：

```yaml
# static_resources:
#   clusters:
#     - name: ...
        health_checks:
            - timeout: 1s
              interval: 10s
              interval_jitter: 1s
              unhealthy_threshold: 1
              healthy_threshold: 1
              http_health_check:
                path: "/_/health"  # 这里是你自己定义的路径
```

> 说到这里我真的必须要吐槽，这个文档真的毫无参考价值，还是得靠抄。参考 [Envoy 的健康检查 - 我是阳明](https://cloud.tencent.com/developer/article/1649321)  
> 姑且贴一下官方文档对于 Health Check 的定义，谁爱看谁去看吧： [网址](https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/core/health_check.proto)

## 基本操作3： HTTP失败重试

只有健康检查是不够的，如果在两次健康检查的间隔之间有上游服务挂掉的话，那么这次失败的请求就会传导到下游，而这是我们不希望看到的。我们需要的是一种自动的失败重试机制。

但这里必须要先提醒，正常来说我们只能对**幂等**的操作来简单的执行自动重试。对于有副作用的操作，最大的问题在于，这个重试的动作对下游调用方来说是无法感知的，可能会导致严重的后果；请三思而后行，并且谨慎地配置重试规则。

既然自动重试这么危险，那有没有好的解决方案？——有的，Envoy提供两种方式来配置自动重试，一种方式是直接针对路径的（Envoy的配置），另一种方式是针对请求的（下游客户端自己在请求中加一个Header来指定可以重试）。

我这里暂且只介绍前者，在Envoy上对路径设置重试规则：

```yaml
# routes:
  - match:
      prefix: "/business"
    route:
      cluster: service_business
      retry_policy:
        retry_on: connect-failure, 5xx
        num_retries: 5
```

文档在： [retry_policy](https://cloudnative.to/envoy/api-v3/config/route/v3/route_components.proto.html#envoy-v3-api-field-config-route-v3-routeaction-retry-policy) 
-> [config.route.v3.RetryPolicy](https://cloudnative.to/envoy/api-v3/config/route/v3/route_components.proto.html#envoy-v3-api-msg-config-route-v3-retrypolicy) （PS：好像总算找到了文档的正确食用方式……）

上面的配置的意思呢，是针对`/business`路径的这个路由的规则，设置`retry_policy`属性。其中定义了重试条件是`connect-failure`（在TCP层面无法与上游建立连接）或者`5xx`（返回了500系列的 HTTP STATUS），以及最大重试次数`num_retries`是5次。

然后做一些测试。我给上游服务加一个规则，让它交替返回200和500：

```go
app.GET("/business/1", CheckAuthMiddleware, func(ctx *gin.Context) {
    if coin = !coin; coin{
        ctx.Status(500)
        return
    }
    ctx.String(200, "Hello, Im business code. 30001")
})
```

然后启动Envoy，然后启动两个上游服务30001和30002，此时一切正常。然后干掉其中一个上游服务，继续在浏览器中验证，依然100%可用，失败的请求会自动重试到另一个服务上去。

在启用失败重试的同时，我们依然可以开启健康检查，它们可以互相作为补充，达到更优的状态。

## 小插曲：尝试模拟gRPC掉线情况

HTTP太简单了，没啥好说的，我们接下来说一下gRPC。虽然它的实质也是HTTP/2，但是我的理解是它对稳定性要求更高一些？毕竟在客户端进行了显式的建立连接的操作。

这次依然偷懒，使用我之前写过的一个gPRC的Demo，主要功能就是一个Echo方法，会将传入的文本稍作修改之后返回。 [代码地址](https://github.com/Saodd/learn-grpc)

为了模拟"部分服务挂掉"的场景，我这次直接抄家伙，上`docker swarm`，建立一个2副本的服务。我们知道swarm的service直接内置了服务发现和负载均衡，这2个副本构成的服务映射在本机的5005端口上，由swarm提供的负载均衡来分配请求：

```yaml
services:
  echo:
    image: learn-grpc:go
    deploy:
      mode: replicated
      replicas: 2
      restart_policy:
        condition: on-failure
    ports:
      - "5005:5005"
    networks:
      - net
```

然后我在宿主机上启动一个 客户端 去调用这个service 。一开始我尝试每 100ms 调用一次，结果发现：

- 建立连接之后后续的请求都是从同一个容器中返回的，grpc的请求并没有被swarm正确地负载均衡；
- 杀掉一个容器，客户端立即重新连接了另一个容器，看起来无缝切换。

我疑惑，gRPC的稳定性有这么强吗？

然后我试着把客户端的调用频率缩短到 1ms，再杀掉服务容器，这次终于发现客户端抛出了异常：

```text
rpc error: code = Unavailable desc = transport is closing
```

看起来，在某一次请求发出之后、还未得到回复之前，如果连接断开，这次请求是会返回异常的。

这次实验说明，swarm这类框架附带的负载均衡，更像是L3这种低层次的负载均衡，对于复用同一个连接的连续请求，是无法做均衡的。可算让我找到突破点了，接下来我们来看看Envoy能不能做到：

1. 对每个gRPC单个请求做负载均衡（而不是客户端连上一个服务端之后就穷追猛打）
2. 断线瞬间造成gRPC请求失败时的自动重试

> swarm提供的LB的确是L3/L4的，参考阅读： [stackoverflow](https://stackoverflow.com/questions/38717965/whats-the-mechanism-of-inner-load-balancing-along-with-docker-swarm-v1-12)

## 基本操作4：gRPC代理与失败重试

gRPC底层依然是HTTP/2，所以它的`listener`和`cluster`依然是HTTP的，跟刚才差不多，只要稍作修改：

在`listener`这边要指定grpc过滤规则：

```yaml
routes:
    - match:
        prefix: "/"
        grpc: {}  # 注意增加了这个
```

以及，在`cluster`这边要指定http2的处理规则：

```yaml
clusters:
    - name: service_echo
      connect_timeout: 0.25s
      type: STRICT_DNS
      lb_policy: ROUND_ROBIN
      http2_protocol_options: { }  # 注意增加了这个
```

至于失败重试，也可以继续沿用刚才HTTP的配置：

```yaml
route:
    cluster: service_echo
    retry_policy:
      retry_on: connect-failure, 5xx
      num_retries: 2
```

不过上面的 5xx 这个规则就不适用了，相应地，Envoy提供了针对gRPC响应状态码的处理规则。（网页一下子找不到了，自己去翻吧）

在当前的配置下，我们用刚才启动的 swarm 集群的那个 echo 服务来进行测试，会发现一切正常运行，当强行杀掉服务中的一个容器，Envoy会妥善地处理重试，客户端也不会感知到错误。

## 基本操作5：gRPC负载均衡

TODO: 搞不定了，文档太难搞了，暂停一下，去其他项目找点灵感先……
