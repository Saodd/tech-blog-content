```yaml lw-blog-meta
title: "grpc-web 尝鲜"
date: "2022-06-07"
brev: "看来目前在前端还是别想grpc了，老老实实REST吧"
tags: ["中间件"]
```

## 准备服务端程序

后端使用我最熟悉的go语言，具体执行步骤可以参考我之前的博客：[gRPC 入门教程](../2021/210110-gRPC-guide.md) 。

把示例代码的监听地址改为`0.0.0.0`，启动服务端程序后，它在监听`5005`端口。（可以用go的客户端去测试一下连通性。）

## Nginx代理grpc流量

由于`gRPC`使用的通信协议是`HTTP/2`，这个在浏览器端是不被直接支持的，而且js也无法访问到原始的`HTTP/2`的数据帧，因此需要一个转化。

而目前市面上应该只有Envoy才支持这种转化，相关内容在下一章中讲解。

对于我们熟悉的Nginx，它只能代理gRPC到gRPC的流量，参考文章：[Publishing gRPC Services](https://www.nginx.com/blog/deploying-nginx-plus-as-an-api-gateway-part-3-publishing-grpc-services/)

```nginx
events {
    worker_connections  1024;
}

http{
    log_format grpc_json escape=json '{"timestamp":"$time_iso8601",'
    '"client":"$remote_addr","uri":"$uri","http-status":$status,'
    '"grpc-status":$grpc_status,"upstream":"$upstream_addr"'
    '"rx-bytes":$request_length,"tx-bytes":$bytes_sent}';
    
    
    map $upstream_trailer_grpc_status $grpc_status {
        default $upstream_trailer_grpc_status; # grpc-status is usually a trailer
    ''      $sent_http_grpc_status; # Else use the header, whatever its source
    }

    server {
        listen 50051 http2;
        access_log  /var/log/nginx/access.log  grpc_json;
        error_log  /var/log/nginx/access.log error;

        location / {
            grpc_pass grpc://10.0.15.233:5005;
        }
    }
}
```

## 用Envoy将HTPP转化为gRPC

参考：[官方文档](https://grpc.io/docs/platforms/web/basics/) 

### 配置Envoy

Google那帮人当然是推荐你使用它们自家的`Envoy`来做代理。

再次吐槽，Envoy的yaml格式配置文件是真的难写！配置项有这么多就算了，官方文档真的很烂，根本看不懂，而且stackoverflow上面也几乎没有相关的有意义的帖子。配置过程极度痛苦，我有必要贴一下我的配置，否则搞不好下次我自己都不能复现了：

```yaml
static_resources:
  listeners:
    - name: listener_0
      address:
        socket_address: { address: 0.0.0.0, port_value: 10000 }
      filter_chains:
        - filters:
          - name: envoy.filters.network.http_connection_manager
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
              codec_type: auto
              stat_prefix: ingress_http
              route_config:
                name: local_route
                virtual_hosts:
                  - name: local_service
                    domains: ["*"]
                    routes:
                      - match: { prefix: "/" }
                        route:
                          cluster: echo_service
                          timeout: 0s
                          max_stream_duration:
                            grpc_timeout_header_max: 0s
                    cors:
                      allow_origin_string_match:
                        - prefix: "*"
                      allow_methods: GET, PUT, DELETE, POST, OPTIONS
                      allow_headers: keep-alive,user-agent,cache-control,content-type,content-transfer-encoding,custom-header-1,x-accept-content-transfer-encoding,x-accept-response-streaming,x-user-agent,x-grpc-web,grpc-timeout
                      max_age: "1728000"
                      expose_headers: custom-header-1,grpc-status,grpc-message
              http_filters:
                - name: envoy.filters.http.grpc_web
                  typed_config:
                    "@type": type.googleapis.com/envoy.extensions.filters.http.grpc_web.v3.GrpcWeb
                - name: envoy.filters.http.cors
                  typed_config:
                    "@type": type.googleapis.com/envoy.extensions.filters.http.cors.v3.Cors
                - name: envoy.filters.http.router
                  typed_config:
                    "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
              access_log:
                - name: envoy.access_loggers.file
                  typed_config:
                    "@type": type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog
                    path: "/dev/stdout"
  clusters:
    - name: echo_service
      connect_timeout: 0.25s
      type: logical_dns
      http2_protocol_options: {}
      lb_policy: round_robin
      load_assignment:
        cluster_name: cluster_0
        endpoints:
          - lb_endpoints:
              - endpoint:
                  address:
                    socket_address:
                      address: 10.0.15.233
                      port_value: 5005
```

上面的配置里，Envoy将监听`10000`端口，反向代理到后端服务`5005`端口上去。`access_log`部分是我添加的，用于简单观察日志。

### 准备js编译环境

第一，要能编译js，因此需要：

```shell
yarn add grpc-tools
```

第二，编译web部分代码，选用的是官方的`grpc-web`这个库，需要配套的工具：[protoc-gen-grpc-web](https://github.com/grpc/grpc-web/releases) 来负责编译。下载解压后添加到环境变量中。

第三，运行时需要依赖这个：

```shell
yarn add google-protobuf
```

### 编译web代码

执行编译命令：

```shell
grpc_tools_node_protoc hello.proto --js_out=import_style=commonjs:./proto --grpc-web_out=import_style=commonjs,mode=grpcwebtext:./proto
```

输入了两个参数，因此我们也顺利在`./proto`目录中得到`hello_grpc_web_pb.js`和`hello_pb.js`两个文件。

然后写一些代码来进行调用，我用的是React框架，这里只贴出部分关键代码：

```typescript
import proto from './proto/hello_grpc_web_pb';

const client = new proto.ChatClient('./api');  // 因为涉及跨域，因此我用这个api前缀用于反向代理

const req = new proto.Sentence();  // 我们proto中定义的请求体格式，包含 Speaker 和 Content 两个参数
req.setSpeaker('Chrome102');
req.setContent('收到请回答！');

client.echo(req, {}, (err, resp) => {
    console.log(resp);
});
```

webpack devServer 也要相应配置一下代理，将请求指向Envoy服务（前面启动的监听`10000`端口的哪个）

在浏览器里调试一下，可以发现其实没有什么魔法，`grpc-web`这个库只是将rpc的endpoint转化在url里了，然后参数则以二进制base64格式来进行传输（当然这个行为根我们编译时指定的参数有关）。

### typescript支持

需要安装这个：

```shell
yarn add grpc_tools_node_protoc_ts
```

然后编译的时候增加一个`--ts_out=./proto`参数即可。会额外编译出两个`.d.ts`文件。

但是，仔细看一下，其中一个ts是给node用的，并不能直接拿来给 grpc_web 的产物使用！（也许有别的库可以支持，但是目前我没找到，以后看到再补）

## 总结

总体来说，gRPC这套东西依然不能直接应用于前端，至少目前没有看到可靠的转化工具。如果按上面介绍的`grpc-web`+`Envoy`来做的话，后端的改造代价会很大。除非你后端本来就已经用上Envoy了，这不现实，它还远远不算是常规的后端中间件啊。

距离我上次体验 gRPC + Envoy 这套东西，已经过去一年多的时间了。可是我感觉这一年来它们丝毫没有进步：依然依赖多个编译工具，而且没有一个完善的文档可以让人一次性跑起这个Demo来，需要大量的折腾才能成功。

这让我感触蛮深的，做开源项目并不仅仅是技术好就行；要维护一个可靠的文档、让好的技术能够推广出去，这可能才是最可贵的。
