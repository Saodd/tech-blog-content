```yaml lw-blog-meta
title: "详解CORS"
date: "2021-09-22"
brev: "跨域安全细节"
tags: ["安全"]
```

## 背景

最近手头的工作呢，某种方面来说其实就是在对抖店web端做逆向破解。

在做的过程中不断挖掘他们所使用的安全手段，然后结合自己的实践去思考，然后发现自己的安全体系还有一些没有完全清楚的地方。

其他的知识可以偷懒，但是安全知识必须完备。

## 参考阅读

- [Are JSON web services vulnerable to CSRF attacks?](https://stackoverflow.com/questions/11008469/are-json-web-services-vulnerable-to-csrf-attacks)
- [Cross-site request forgery](https://en.wikipedia.org/wiki/Cross-site_request_forgery)
- [Mitigating CSRF attacks in Single Page Applications](https://medium.com/tresorit-engineering/modern-csrf-mitigation-in-single-page-applications-695bcb538eec)

## TL;DR

1. 后端CORS中间件：
   + `Access-Control-Allow-Credentials: true`
   + `Access-Control-Allow-Origin` 必须包含你所有来源的域名（一般是主域名+www域名），且不能是`*`
   + 如果你的请求中有额外的Header，则必须在`Access-Control-Allow-Headers`中指定。（这个会导致`PreFlight`，请考虑清楚后使用）
2. 后端设置Cookie时：
   + `Domain`必须是你的主域名，例如`Domain=lewinblog.com`
   + `HttpOnly`和`Secure`请务必设为true
   + `Max-Age`也请务必设置一个合理值，不要设为永久
3. 前端请求时：
   + `XMLHttpRequest.withCredentials`要设为`true`，在axios中则是`{withCredentials: true}`

## CSRF

`CORS`意思是「跨域资源请求」，是一个中性词。在现代SPA应用中，将API放在子域名中，或者需要访问第三方资源（Google-Analytics等）时，都需要这个方案的支持。

`CSRF`意思是「跨域请求伪造」，既可以指一种攻击行为，也可以指对应这种攻击行为的防御手段。

在前面的TLDR章节中，能应对所有的`xhr`请求。它依赖的最关键的特性，是浏览器在执行跨域请求时会携带`Origin`原始域名信息，这样服务端在收到请求之后可以进行检查并判断是否允许来自这个域名的请求。

> TIPS: 注意区分 Origin 和 Referer .

而网页端的请求还有另一种，也是更古老且大家更熟悉的，叫做`form`。

在配置了上面的安全策略之后，我依然可以在另一个域名伪造请求来访问。

假如我是黑客，`http://localhost:8000`是我做的一个钓鱼网站，我在其中做了一个CSRF表单：

```html
<form action="https://api.lewinblog.com/status" method="get">
    <button>Attack!!</button>
</form>
```

`https://api.lewinblog.com/status` 这个接口是一个需要用户登录状态的GET接口，此时我在`http://localhost:8000`点击这个表单，Duang，跳转到了`https://api.lewinblog.com/status` 这个页面并且返回了用户数据（即通过了session认证）。查看请求内容，可以发现cookie被提交了。

但是通过`form`会导致页面跳转，所以攻击者依然无法获得接口请求的数据。

所以这里的防御方式，是在`GET`接口中只返回信息，不做操作。

那么，攻击者可以去寻找那些有「副作用」的接口来进行攻击，典型的是`POST`请求。

```html
<form action="https://api.lewinblog.com/test" method="post">
  <input name="data" value="给我打钱！"/>
  <button>Attack!!</button>
</form>
```

点击提交！表单内容被正常提交了，咦，但是居然响应了`403`了呢。原来是被gin的`CORS`插件给拦回去了。

那么再试试，把插件关掉，会发生什么？

——会发现请求没有带cookie，认证失败。这是被之前设置的cookie的`Domain`属性给保护了。

> 准确地说是 [`Same-site`属性](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie/SameSite) ，默认值是`Lax`，即仅在导航跳转的时候才会携带cookie，其他情况下不携带。

## gin CORS 插件源码赏析

它做的事情非常简单直观，并且足够强大：

```go
func (cors *cors) applyCors(c *gin.Context) {
	origin := c.Request.Header.Get("Origin")
	if len(origin) == 0 {
		// request is not a CORS request
		return
	}
	host := c.Request.Host

	if origin == "http://"+host || origin == "https://"+host {
		// request is not a CORS request but have origin header.
		// for example, use fetch api
		return
	}

	if !cors.validateOrigin(origin) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	if c.Request.Method == "OPTIONS" {
		cors.handlePreflight(c)
		defer c.AbortWithStatus(http.StatusNoContent) // Using 204 is better than 200 when the request status is OPTIONS
	} else {
		cors.handleNormal(c)
	}

	if !cors.allowAllOrigins {
		c.Header("Access-Control-Allow-Origin", origin)
	}
}
```

1. 获取`Origin`字段，如果没有，则说明不是跨域请求，直接放过。
2. 再额外检查一下来源是不是本机域名，如果还是来源于本域名，直接放过。（在容器化状态下应该不会走进这个分支）
3. 检查`Origin`是否合法，检查的依据是启动时的配置参数，如果不合法则返回403。（前面的form表单被拒绝就是在这里）
4. 如果是`OPTIONS`方法，那说明是个`PreFlight`，那么不需要执行这个请求，返回`204`和正确的Headers即可。
5. 正常执行请求，并且要在Header里表达允许这个`Origin`的访问。

## csrf token

到此为止，无论是`xhr`还是`form`的CSRF，统统被封死。

如果还是不放心，也可以有一些额外的手段。

在上个世代还在流行`form`的时候，csrf的标准防御手段是在html中直接植入`csrf token`。到了现在SPA应用中，形式要发生一些改变。

以抖店（巨量百应）为例，页面在加载之后，会由js发起一个请求，获取一个`csrf token`，维护在运行时中，后续的请求都会将其添加到Header中一并提交。

这种思路的逻辑其实很简单，正常的csrf攻击都是一次性的请求，那么我们给请求增加复杂度，至少要求两个请求一起工作才能生效，这也能极大地抑制csrf的可能性。

## 结语

当然，这些防御手段都是基于浏览器特性的支持。so，如果你一定要支持那些使用危险的过期的浏览器的用户，那可能需要更加深入的研究，以及，明确你到底要支持到哪种程度。

然后，除了CSRF之外，在前端还常常提及另一种攻击`XSS`，这个防御起来相对简单，不想再总结一遍了。
