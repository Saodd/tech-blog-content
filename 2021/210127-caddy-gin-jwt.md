```yaml lw-blog-meta
title: 'JWT跨服务认证方案设计'
date: "2021-01-27"
brev: "玩一下Caddy，和gin的中间件设计。以JWT来实现跨服务身份认证的场景来实践。"
tags: ["中间件"]
```

## 前言

今天玩点啥呢……

不想搞太大，就随手实践一下最近的一个想法吧：在对身份安全性要求不高的场景下，最简便地实现跨服务的身份认证。

关于JWT，我之前有 [文章](../2020/200604-session与jwt.md) 做过简单的介绍。其核心原理就是将一些数据以加密的形式放在前端。对于一些安全性要求不高的场合，用它来代替Session可以省很多事。

关于gin，我之前也有 [文章](../2020/200623-Gin框架源代码速读.md) 简单地理解过它的源代码。其中间件的核心逻辑是通过`c.Next()`方法来实现顺序调用。

gin当然有现成的[JWT框架](https://github.com/appleboy/gin-jwt) 。但是今天我来简单地实践一下，用同样的思路、自己的方式来写一遍。

> 本文用到的代码可以在 [我的Github](https://github.com/Saodd/learn-caddy) 中找到。

## Caddy是啥

简单说，它是用Go写的，性能还凑合（比Nginx差一个档次但是不至于差数量级）。但是最大的优势是它的配置文件极其简单，并且针对TLS做了大量的优化，以及最大的好处——支持自动续期HTTPS证书。还有各种可有可无的方便小特性，不一一列举。

> 用过Vxxx代理的同学可能会有所感触，配置过程中根本都没见到证书长什么样子，服务就自动地带s了，相当地方便。这也是吸引我来学习它的理由之一。

## Caddy入门

### 0. 基本运维

再强调一遍，它是Go写的。所以理论上它应该只需要一个二进制文件就可以在你的系统中运行了。

不过作为一个有轻微洁癖的人，我还是选择Docker：

```shell
docker pull caddy:2.3.0-alpine
```

有点难受的是，阿里云的镜像加速仓库中没有包含caddy的镜像，所以这个速度相当的慢。不过慢也慢得有限，毕竟alpine，忍忍吧。（不能忍的同学自己折腾一下代理，反正我在windows上折腾了好一会都不行，回头再去Linux上试试……）

### 1. 基本操作

[官方教程](https://caddyserver.com/docs/getting-started)

启动容器：

- 注意除了常规的80和443端口之外，还有一个2019是管理端口，不开也行。
- 我这里直接挂载配置文件进去，快速修改快速验证。

```shell
docker run --rm --name caddy -p 80:80 -p 443:443 -p 2019:2019 -v ${pwd}/Caddyfile:/etc/caddy/Caddyfile -it caddy:2.3.0-alpine
```

此时会提示你，Caddyfile是空的，配置无效，退出。

然后试着写第一个Caddyfile（这里 10.0.1.232 是我本机ip地址）：

```text
10.0.1.232:80
respond "Hello, Lewin! No.1."
```

就这两行？对，就这两行。然后在浏览器中立即可以验证。是不是有点离谱？

> 虽然有点遗憾的是，JetBrains家还没有任何针对Caddy的插件，所以写配置文件就没有任何代码提示了。（Nginx是有的）

启动Caddy之后，我们可以在日志中观察到它将配置文件转化为了JSON格式的文件。我们试着cat看一下：

```json
{
  "apps": {
    "http": {
      "servers": {
        "srv0": {
          "listen": [
            ":80"
          ],
          "routes": [
            {
              "handle": [
                {
                  "handler": "subroute",
                  "routes": [
                    {
                      "handle": [
                        {
                          "body": "Hello, Lewin! No.1.",
                          "handler": "static_response"
                        }
                      ]
                    }
                  ]
                }
              ],
              "match": [
                {
                  "host": [
                    "10.0.1.232"
                  ]
                }
              ],
              "terminal": true
            }
          ]
        }
      }
    }
  }
}
```

两者对比很直观，JSON是配置文件的最终形态，Caddyfile只是一种快捷方式。我觉得大多数情况我们用简单的Caddyfile就可以了。

除了写配置文件之外，我们还可以通过它的管理端口来动态地修改配置文件。强大的是，这个过程是真·动态，可以在不停机、不阻塞服务的情况下完成配置的更新。关于这一块不详细讲。

### 2. 多Host

写监听两个host，返回不同的内容：

```text
10.0.1.232:80 {
respond "Hello, 232!"
}

:80 {
respond "Hello, localhost!"
}
```

访问浏览器可以进行验证。此外，我们依然可以检查容器内生成的JSON配置文件，会发现此时生成了两个route规则，很直观。

### 3. 静态文件

反向代理服务器的另一重要功能是直接提供静态文件。规范语法在 [文档](https://caddyserver.com/docs/caddyfile/directives/file_server) 中可以找到，我这样写：

```text
10.0.1.232:80 {
    file_server * browse {
        root /etc/caddy/
    }
}
```

因为Caddy默认会隐藏掉目录下的Caddyfile，所以这次启动时，我需要多挂载一些文件进去以便验证：

```shell
docker run --rm --name caddy -p 80:80 -p 443:443 -p 2019:2019 -v ${pwd}:/etc/caddy -it caddy:2.3.0-alpine
```

当指定开启`browse`模式时，直接打开根目录甚至会展示一个简单而且还算好看的静态文件浏览界面，挺惊喜的。

### 4. 反向代理

请参阅 [详细文档](https://caddyserver.com/docs/caddyfile/directives/reverse_proxy)

先写一个极其简单的gin服务：

```go
func main() {
	app := gin.Default()
	app.GET("/business/1", func(context *gin.Context) {
		context.String(200, "Hello, Im business code.")
	})
	app.Run("0.0.0.0:30001")
}
```

然后配置Caddy去访问它：

```text
:80 {
    reverse_proxy /business/* 10.0.1.232:30001
}
```

这里值得一提的是，当上游服务器挂掉的时候，Caddy也是可以正常启动的。而Nginx默认在启动时要求上游服务器都是健康的。

### 5. 反向代理进阶：负载均衡

我们直接在Caddyfile里配置两个上游（而此时只启动了一个）：

```text
:80 {
    reverse_proxy /business/* 10.0.1.232:30001 10.0.1.232:30002
}
```

在浏览器中访问，偶尔能成功，偶尔会失败，说明负载均衡生效了。

接着我们稍微改动之前的gin代码，在30002端口再启动一个服务。然后继续可以在浏览器中验证，caddy随机地把我们地请求分配给两个服务上去。

负载均衡有很多策略，它默认是`random`即随机选择。如果我们再加上健康检查策略，那么这个服务就"高可用"了，caddy不会把我们的请求代理到健康检查失败的上游服务上去：

```text
:80 {
    reverse_proxy /business/* {
        to 10.0.1.232:30001 10.0.1.232:30002
        health_path /business/1
        health_interval 1s
    }
}
```

不过需要强调的是，在某个上游服务挂掉之后、下一次健康检查之前，这个服务依然会被代理到，这个期间外部就会遭遇失败。

### 6. HTTPS

只要给它指定监听443端口，或者指定一个域名作为host，它就会自动将HTTP请求308定向到HTTPS。

在本地也可以使用证书！不过仅限于**本机**，它的原理是把内置的一个证书安装到当前的系统中。所以在Docker中运行的Caddy，在容器之外是无法访问的（因为外面没有安装它的根证书）。

一个比较典型的用法是 Caddy + `Let's Encrypt` （一家提供免费证书签发的机构），应该可以很轻松地让你的网站带上S~

这里我就暂时先不体验了，等我的云服务到期了之后再试试看吧。

## 用JWT实现跨服务身份认证

我们先不用gin的中间件写法，先直接在视图函数中处理JWT的签发和认证环节。

首先选择加密算法。当前2021年来说，最强大的加密算法应该是ECC算法了吧。但是使用公钥-私钥对稍微麻烦一点点点点，所以我这里暂时用次一档的方案，用对称加密的AES256来实现。

然后继续偷个懒，直接用我之前写过的一个[加密小工具](https://github.com/Saodd/giary) 中的封装：

```shell
go get github.com/Saodd/giary
```

然后写一个认证服务。它的功能就是给Cookie里写入JWT。不过，说是JWT，我这里并没有完全按照规范来做：因为我不需要前端也能识别Token的内容，因此我只需要那个加密的部分就可以了。

需要注意的是Golang在计算加密数据时都是以`[]byte`的形式在运算。如果要写进Cookie里，根据HTTP对Headers的要求，我们把加密后的数据转换为Base64是最好的。

```go
package main

import (
	"encoding/base64"
	"encoding/json"
	"github.com/Saodd/giary/giary"
	"github.com/gin-gonic/gin"
	"learn-caddy/common"
	"time"
)

func main() {
	// 实例化一个用于加密解密的封装对象
	var cc = giary.NewClient([]byte(common.Secret))
	app := gin.Default()
	app.GET("/auth", func(context *gin.Context) {
		// 生成Token
		token, _ := json.Marshal(&common.UserToken{Name: "Lewin Lan", Expired: time.Now().Unix() + 60})
		tokenCipher := cc.Seal(token) // 加密时可以前后加盐，这里暂时不折腾了
		tokenCipherB64 := base64.StdEncoding.EncodeToString(tokenCipher)
		// 设置Cookie
		context.SetCookie(common.CookieKey, tokenCipherB64, 3600, "/", "localhost", false, true)
		context.String(200, "Auth Passed.")
	})
	app.Run("0.0.0.0:30000")
}
```

上面用到了一些常数，这些常数需要在两个服务之间共享，因此我写在一个单独的package里：

```go
package common

const Secret = "8237yrhoq8u3rfgh-/4t2q-+"
const CookieKey = "LewinToken"

type UserToken struct {
	Name    string
	Expired int64
}
```

此时我们在浏览器中访问，就可以在Response Headers里看到这条Cookie：

```text
Set-Cookie: LewinToken=yLhOTw......D11G5xpj6dGSi; Path=/; Domain=localhost; Max-Age=3600; HttpOnly
```

还要注意在上面的auth服务中还有一些细节。我在UserToken中人为地规定了过期时间是当前时间+60秒，是个很短的时间。而我给Cookie设置的过期时间是3600秒，这样待会我们就能在business服务中观察到客户端发来了一个过期的Token。

接下来我们写另一个服务，它要解密之前放进Cookie里的Token，然后JSON反序列化，然后验证用户身份以及是否过期。

```go
func main() {
	// 实例化一个用于加密解密的封装对象
	var cc = giary.NewClient([]byte(common.Secret))
	app := gin.Default()
	app.GET("/business/1", func(ctx *gin.Context) {
		// 获取Cookie
		tokenCipherB64, err := ctx.Cookie(common.CookieKey)
		if err != nil {
			ctx.String(http.StatusUnauthorized, "Token Not Found.") // 在正式产品中请不要给出这么详细的错误提示
			return
		}

		// 解密Token
		tokenCipher, err := base64.StdEncoding.DecodeString(tokenCipherB64)
		if err != nil {
			ctx.String(http.StatusUnauthorized, "Base64 Decode Failed.") // 在正式产品中请不要给出这么详细的错误提示
			return
		}
		token, err := cc.Open(tokenCipher)
		if err != nil {
			ctx.String(http.StatusUnauthorized, "AES Decode Failed.") // 在正式产品中请不要给出这么详细的错误提示
			return
		}
		var user common.UserToken
		if err = json.Unmarshal(token, &user); err != nil {
			ctx.String(http.StatusUnauthorized, "JSON Decode Failed.") // 在正式产品中请不要给出这么详细的错误提示
			return
		}

		// 验证用户Token是否有效
		if user.Expired < time.Now().Unix() {
			ctx.String(http.StatusUnauthorized, "Token Expired.") // 在正式产品中请不要给出这么详细的错误提示
			return
		}

		// 执行业务
		ctx.String(200, "Hello, Im business code. 30001")
	})
	app.GET("/_/health", func(ctx *gin.Context) {
		ctx.Status(200)
	})
	app.Run("0.0.0.0:30001")
}
```

再把Caddyfile稍作修改：

```text
:80 {
    reverse_proxy /business* {
        to 10.0.1.232:30001
        health_path /_/health
        health_interval 1s
    }
    reverse_proxy /auth* {
        to 10.0.1.232:30000
    }
}
```

此时，之前的`/business/1`这个路径的资源，就被我自定义的Token规则所保护起来了。要访问它，首先要去`/auth`路径获取Token才行。

调通之后稍作等待，等60秒之后再次访问`/business/1`，就会发现此时Token已经过期了，要重新认证一次才行。

## 改造为gin的中间件

得益于Golang的接口式设计，我只要简单地把 签发/验证 环节的代码提取出来放在一个函数里就可以了。而且最美妙的是，中间件函数的签名与视图函数的签名完全一致，真正实现无痛迁移，连变量名都不需要改。

不过还是有需要改的地方的。在中间件中如果要立即放弃当前请求不再进入业务函数（视图函数），那么要使用`Abort`系列的方法，直接return是不行的。

> 读过gin源码的同学会知道，gin的整个请求处理过程是在一个函数列表上依次执行的。

认证服务代码：

```go
var cc = giary.NewClient([]byte(common.Secret))

func AuthMiddleware(ctx *gin.Context) {
	// 这里是之前的签发Token的代码
	token, _ := json.Marshal(&common.UserToken{Name: "Lewin Lan", Expired: time.Now().Unix() + 60})
	tokenCipher := cc.Seal(token)
	tokenCipherB64 := base64.StdEncoding.EncodeToString(tokenCipher)
	ctx.SetCookie(common.CookieKey, tokenCipherB64, 3600, "/", "localhost", false, true)
	// 执行下一个
	ctx.Next()
}

func main() {
	app := gin.Default()
	app.Use(AuthMiddleware) // 这里把上面自己写的中间件设置到引擎中
	app.GET("/auth", func(context *gin.Context) {
		context.String(200, "Auth Passed.")
	})
	app.Run("0.0.0.0:30000")
}
```

business服务代码不再赘述了~

## 总结

emm……跟想象得差不多，一切都进行的很顺利。果然是个茶余饭后打发时间的轻松任务~

Caddy这个东西呢，（可能是因为我已经比较熟悉Nginx了）整个体验过程几乎没有遇到任何的坑，就主观感受来说我觉得完全可以给100分满分。

不过我可能还是倾向于认为它更像是一个玩具（挺厉害的够玩很久很久的玩具），而不是最顶尖的选择。不过，对于个人或者初创团队来说，选择Caddy来替代Nginx，应该会是个很不错的选择。
