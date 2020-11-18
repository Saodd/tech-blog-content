```json lw-blog-meta
{"title":"认证机制：Session与JWT","date":"2020-06-04","brev":"Web应用中如何对客户端身份进行验证，这是一个非常基础的问题。","tags":["前端"],"path":"blog/2020/200604-session与jwt.md"}
```



## 引言

我们先想象一个情景：假如我们有一栋房子（或者一家公司办公楼），我们想做身份认证机制，有哪些思路？

- 思路一：装上防盗门，然后发钥匙（或者门卡之类的）。只要有钥匙，不管是谁都可以进。
- 思路二：请几个保安（或者前台小姐姐）在门口守着，然后准备一个名单，只有在名单上的人才可以进。

但是这两种方案各有缺点：

- 思路一的缺点：如果不想让某个人进来了，那必须把整个门换掉；此时所有的有效用户都要重新领取新钥匙。
- 思路二的缺点：有额外的开销（请保安的钱），以及性能瓶颈（名单只有一份）。

这两种方案分别对应web世界中的两种主流身份认证机制：

- 思路一：JWT
- 思路二：Session

## 第一部分：Session

我们先讲 Session。因为它的思路是把身份认证信息存放在服务端，因此更安全，更可控，所以往往会作为兜底的解决方案。

### 1.1 Session的签发流程

- 客户端登录后（或者仅仅在游客访问时），服务端生成一个唯一身份标识码，在服务端保存（内存or数据库）。
- 同时以set-cookies的形式存回客户端。
- 每次客户端请求时，附带上这个标识码；服务端到数据库中检验无误后，则认为该客户端有效。

```text
    Client                     Server
      |
      |-----1.POST /login------->|
                                 |
      |<-----2.SetCookie---------|
      |
      |------3.GET /secret------>|
                                 |
       <-----4.HTTP200-----------|
```

它的关键点在于：一个唯一身份标识码(session-id)。这个值是随机生成，而且数值巨大，一般不可能被暴力破解。并且是以服务端中保存的为准。

更具体的用户信息，例如用户名，等级，分组等信息，可以随该 id 一起放在数据库中。常规做法是将对象序列化后作为二进制字符串存在 Redis 中，也有一些简单的实现是放在数据库中（例如Django的默认配置）。

### 1.2 Session特性

它的优点是认证身份可以回收，可以强制下线。实现方式很简单，只要把对应数据库中的记录删除（或者标记为不可用）就可以了。

它的缺点，有一个是容易遭受CSRF攻击。不过这点是Cookie的弱点，Session表示并不背这个锅。另一个是它的性能缺陷，结局方案无非就是水平拓展加机器嘛，数据库就分库分表主从读写分离咯，总有办法的。鲁讯曾经说过，钱能搞定的问题都不是问题！

### 1.3 源码剖析

这里留个坑。

## 第二部分：JWT

JSON Web Token（JWT）是一种开放标准（RFC 7519）。它的主要思路就是，将用户的（敏感信息除外的）身份认证信息存放在客户端；每次客户端请求时都带上这个信息，服务端直接可以在本机验证而无需依赖数据库。

数据存放在客户端，那么如何保证数据不被客户篡改？答案就是加密签名。使用一个服务端才知道的秘钥对身份信息签名，然后跟身份信息一期交给客户端。每次客户端请求时，服务端对信息重新签名比对，这样就确保了不被篡改。

更具体的信息可以参考：[JSON Web Token 入门教程 - 阮一峰](https://www.ruanyifeng.com/blog/2018/07/json_web_token-tutorial.html)。

### 2.1 JWT的构造

JWT是由 `.` 符号分隔的三部分组成，分别是头部`Header`，有效载荷`Payload`和签名`Sign`。

一个标准的JWT的例子：

```text
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiemhhbmdzYW4ifQ.ec7IVPU-ePtbdkb85IRnK4t4nUVvF2bBf8fGhJmEwSs
```

看起来杂乱无章，但这实际上只是base64处理过的字符串而已。我们可以借助Linux下的`echo "" |base64 -d`命令，来对各个部分进行解密：

```shell-session
$ echo "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"|base64 -d
{"alg":"HS256","typ":"JWT"}%

$ echo "eyJ1c2VyX2lkIjoiemhhbmdzYW4ifQ"|base64 -d
{"user_id":"zhangsan"%  
```

可以看到，第一部分定义了一些协议相关的东西（其实是没什么用的，可以省去，服务端自己知道协议就可以了）。然后第二部分是有效载荷部分，也就是客户端身份认证的信息。第三部分是无法解析的，因为它是第二部分的签名。

那么如何控制这个认证信息的时效性呢？不可能签发一次，永久有效吧。——实现方法就是在载荷部分加入这次签发的有效期。

### 2.2 JWT签发流程

- 客户端请求登录
- 服务端验证后，将必要信息（user_id，有效期等）写入载荷部分，然后用服务端秘钥进行签名得到jwt字符串
- 服务端将jwt字符串发回客户端
- 客户端可以将jwt存放在localStorage中，下次会话依然有效。（当然也可以放在Cookie）
- 服务端可提供专门的接口用于jwt以旧换新，减少重新登录次数，提升用户体验。

### 2.3 JWT特性

它的优点就是拓展性，可以轻松实现多端跨域登录。

在现代的大型Web应用，往往都是由多项服务、多个域名同时提供服务的，其中可能就会有一个专门负责签发认证的服务。使用jwt可以减少服务之间的依赖，从而允许更灵活的部署方式。

它的缺点主要有两个。第一是无法回收已经签发的认证信息。关于这点的解决方案呢，就是不要在敏感操作上使用jwt，而选择使用session的思路，在服务端控制（或者换个说法，将jwt储存在服务端数据库中）。第二点是工程管理方面的，前面提到了jwt使用一个服务端秘钥来进行签名，那么如果这个秘钥（及签名算法）泄露了，攻击者就可以任意伪造身份信息。

### 2.4 源码剖析

首先是前端的jwt储存与使用。我们这里参考[ant-design-vue-pro](https://github.com/vueComponent/ant-design-vue-pro)的实现。它的做法是，将jwt储存在`localStorage`中，然后在`Axios`上加一个请求拦截器，在每个请求的http头部放入jwt字段。

```js
request.interceptors.request.use(config => {
  const token = storage.get(ACCESS_TOKEN)
  if (token) {
    config.headers['Access-Token'] = token
  }
  return config
}, errorHandler)
```

然后是后端，我们参考 [gin](https://github.com/gin-gonic/gin) 框架以及第三方jwt中间件 [jwt-gin](https://github.com/appleboy/gin-jwt)。

先看一下这个中间件的基本使用，只需要简单的注册到gin路由中就可以了：

```go
func main() {
	g := gin.New()
	j := getMiddleWare()
	{
		g.GET("/login", j.LoginHandler)
		userModule := g.Group("/user")
		userModule.Use(j.MiddlewareFunc())
		userModule.GET("secrete", secreteHandlerFunc)
	}
	g.Run("0.0.0.0:8000")
}
```

然后我们在配置中看一下它做了一些什么事情：

```go
func getMiddleWare() *jwt.GinJWTMiddleware {
	return &jwt.GinJWTMiddleware{
		Realm: "Lewin-JWT",
		Key:   []byte("q23786tr1b1c634t5biq8c234y5o13"),  // 服务端秘钥
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			if v, ok := data.(*User); ok {
				return jwt.MapClaims{  // 这里决定了在载荷部分放什么数据
					"user-haha": v.UserName,
					"id-haha":   v.ID,
				}
			}
			return jwt.MapClaims{}
		},
		Authenticator: func(c *gin.Context) (interface{}, error) {
			username := c.Query("user")
			password := c.Query("pwd")
			if username == "admin" && password == "admin" {  // 这里验证客户登录时的用户密码
				return &User{
					UserName: "lewin",
					ID:       0,
				}, nil
			}
			return nil, jwt.ErrFailedAuthentication
		},
	}
}
```

然后我们试着登录，会得到中间件返回的一些基本信息和我们想要的jwt：

```text
GET http://localhost:8000/login?user=admin&pwd=admin

HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
Date: Fri, 12 Jun 2020 16:05:18 GMT
Content-Length: 237

{
  "code": 200,
  "expire": "2020-06-13T01:05:18+08:00",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1OTE5ODE1MTgsImlkLWhhaGEiOjAsIm9yaWdfaWF0IjoxNTkxOTc3OTE4LCJ1c2VyLWhhaGEiOiJsZXdpbiJ9.OvmCZEp14WU5apdwDn0SFLbvCKMLk0DgvlrXsAb-EC4"
}
```

尝试对载荷部分进行解码，我们可以看到我们在上面的代码中自定义的认证信息：

```shell-session
$ echo "eyJleHAiOjE1OTE5ODE1MTgsImlkLWhhaGEiOjAsIm9yaWdfaWF0IjoxNTkxOTc3OTE4LCJ1c2VyLWhhaGEiOiJsZXdpbiJ9"|base64 -d
{"exp":1591981518,"id-haha":0,"orig_iat":1591977918,"user-haha":"lewin"}% 
```

### 总结

Session OR JWT？这是一个问题。

1. 项目大小？项目阶段？其实这个并不能作为选择的依据。`Django`默认的实现是session，`Flask`默认的实现是（变种）JWT，各种项目可能都会有各种需求，还是根据实际需求来决定。
2. JWT可以用于一些特殊用途（例如第三方认证，简单省事，不会对认证源产生负担）。
3. 对安全性较高的应用应该选择Session。
4. 取长补短：二者混用，或者多级权限控制。
5. 难以取舍的时候，可以先上Session。
