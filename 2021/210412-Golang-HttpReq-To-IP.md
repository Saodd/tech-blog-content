```yaml lw-blog-meta
title: "Golang将HTTP请求发往指定的IP"
date: "2021-04-12"
brev: "记录一下实现"
tags: ["Golang"]
```

## 需求

在Web后端开发过程中，往往有多个环境（本地、测试、线上），而在面对Nginx这类根据Host来反向代理的中间件时，在代码中直接填写相应环境的IP地址是不会生效的。

> 例如：我在本地启动了一个 api.lewinblog.com 的服务，为了调试它，如果我向localhost发送请求，会被Nginx丢掉。

当然一种方式是改写系统的`Hosts`文件来实现域名与IP的特别映射。

但是有的时候也不想频繁去改这个`Hosts`文件，有时希望直接在代码中（测试用代码）就能直接实现保持Host的情况下将请求发往指定的IP，而非DNS解析的IP。（这个事情并不矛盾）

## 改写IP

关键是改写`http.Client.Transport.DialContext`，它是一个函数类型，它接收从`http.Request`中传来的url，然后建立并返回底层的TCP连接。

这个过程可以实现连接池。也可以实现我们现在需要的IP改写。

参考文章: [stackoverflow - golang force http request to specific ip (similar to curl --resolve)](https://stackoverflow.com/questions/40624248/golang-force-http-request-to-specific-ip-similar-to-curl-resolve)

```go
func newHttpClient() *http.Client {
	// 1. 先准备一个Dialer对象
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}
	transport := &http.Transport{
		// 2. 插入特别的改写条件，然后继续利用原先的DialContext逻辑
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if addr == "api.lewinblog.com:80" {
				addr = "127.0.0.1:80"
			}
			return dialer.DialContext(ctx, network, addr)
		},
	}
	// 3. 构建http.Client
	return &http.Client{
		Transport: transport,
	}
}
```

## 代理

代理(`Proxy`)跟直接改写IP是不同的。主要逻辑就是改写`http.Client.Transport.Proxy`，这在我 [之前的文章](../2020/200116-Go配置代理.md) 中介绍过了，不再赘述。
