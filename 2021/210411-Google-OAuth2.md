```yaml lw-blog-meta
title: "Google OAuth2.0的接入与用户系统建立"
date: "2021-04-11"
brev: "技术并不难，但是实现起来比想象中麻烦很多。"
tags: ["架构","安全"]
```

## 前言

虽然很久很久之前，我就已经考虑要做一套用户系统，以及在此基础上再做一套博客文章评论系统。但毕竟个人网站备案是不能开评论的，做这个得游走在灰色地带，因此懒小人占了上风，一直没做。

但是最近，我在玩《戴森球计划》，很肝，两周时间玩了约60个小时，可以说除了上班就是在想和玩这款游戏。当我开始准备批量建立生产线的时候，我发现我需要一个统计工具，详细记录每个生产单位（~~微服务~~）的消耗速率和产出速率，以避免“水多加面面多加水”的窘境。

好吧，作为一个真·全干程序员，这个工具当然得自己做了。于是我连数据库表结构都设计好了。

再于是，我想，这玩意不能我一个人用啊，我得开放出来大家用。开源暂时不考虑，因为这东西不仅在客户端有大量交互，还需要服务端来保存数据，是一套完整的Web应用架构，要做成开源的话，我得为了封装而写很多没什么意思的代码，所以懒得开源了。所以直接以Web服务的形式提供（~~SAAS~~）。

好吧，既然要给大家提供免费的服务，那就势必需要一套用户系统。所以回到原点，我还是得拿出那份沉睡已久的技术设计稿来做这套用户系统。

## Google OAuth2.0 篇

参考： 
- [官方：准备工作](https://developers.google.com/identity/gsi/web/guides/get-google-api-clientid)
- [官方：服务端代码](https://developers.google.com/identity/protocols/oauth2/web-server)
- [民间：用oauth2协议登录访问谷歌API](http://www.zchengjoey.com/posts/%E4%BD%BF%E7%94%A8oauth2%E7%99%BB%E5%BD%95%E8%AE%BF%E9%97%AE%E8%B0%B7%E6%AD%8CAPI/)

这已经是一套非常成熟而且流行的方案，不仅官方文档解释得详细（详细得让我找不到想要的东西），民间也有大量的文章来描述这个流程。

简而言之，整个认证过程：

1. 引导用户跳转Google页面，登录/选择一个Google账号并授权你的应用。
2. Google发出一个授权码(`authorization code`)**给用户**，并引导用户拿着这个code跳转到你的应用。
3. 你的应用从用户那里拿到授权码，访问Google服务器交换一个访问码(`access code`)以及用户信息。

### 步骤一：引导用户去授权

这个步骤很简单，就是拼凑URL罢了。在URL中表明你的应用身份(`clientID`)，Golang实现：

```go
package views

func LoginGoogleRedirect(c *gin.Context) {
    c.Redirect(http.StatusPermanentRedirect, google.BuildLoginUri())
}
```

```go
package google

func BuildLoginUri() string {
    u, _ := url.Parse("https://accounts.google.com/o/oauth2/auth")
    q := u.Query()
    q.Set("response_type", "code")
    q.Set("client_id", config.GoogleOAuthClientID)
    q.Set("redirect_uri", config.GoogleOAuthRedirect)
    q.Set("scope", "https://www.googleapis.com/auth/userinfo.profile https://www.googleapis.com/auth/userinfo.email openid")
    q.Set("access_type", "offline")
    q.Set("state", "")
    u.RawQuery = q.Encode()
    return u.String()
}
```

这个应用身份(`clientID`)要你先去 [Google APIs Console](https://console.cloud.google.com/apis/dashboard) 注册一下。

### 步骤二：接收用户的授权码

授权之后是一个跳转，跳转只能是GET，所以授权码是放在query参数里的，Golang实现：

```go
func LoginGoogleCallback(c *gin.Context) {
    //error := c.Query("error")
    code := c.Query("code")
    state := c.Query("state")
}
```

从链接中取得了`code`之后，接下来我们要用它去交换一个访问码。

从你的应用服务器访问Google，这一步需要一点魔法，自己想办法。（~~我才不会告诉你可以用CF呢~~）

请求地址 `https://oauth2.googleapis.com/token` ，方法POST，携带一个JSON：

```go
type AccessTokenReq struct {
    Code         string `json:"code"`
    ClientId     string `json:"client_id"`
    RedirectUri  string `json:"redirect_uri"`
    GrantType    string `json:"grant_type"`
    ClientSecret string `json:"client_secret"`
}
```

正常响应体：

```go
type AccessTokenResp struct {
    AccessToken  string `json:"access_token"`
    ExpiresIn    int64  `json:"expires_in"`
    TokenType    string `json:"token_type"`
    Scope        string `json:"scope"`
    RefreshToken string `json:"refresh_token"`
    IdToken      string `json:"id_token"`
}
```

异常响应体（可以用组合结构体）：

```go
type AccessTokenError struct {
    Err              string `json:"error"`
    ErrorDescription string `json:"error_description"`
}
```

用户信息是放在`id_token`这个字段里的，它是一个标准的JWT结构，其中的数据信息放在第二段（术语叫`Payload`），我们将其取出并用base64解码，即可获得我们所需的信息：

```go
type UserInfo struct {
    Sub           string `json:"sub"`  // 用户在Google的唯一标识码
    Email         string `json:"email"`
    EmailVerified bool   `json:"email_verified"`
    Picture       string `json:"picture"`
    Name          string `json:"name"`
    // ...
}

func parseUserInfo(c context.Context, idToken string) (*UserInfo, error) {
    idTokenWords := strings.Split(idToken, ".")
    idTokenJson, _ := base64.StdEncoding.DecodeString(idTokenWords[1])
    var info UserInfo
    json.Unmarshal(idTokenJson, &info)
}
```

## 身份认证设计

在自己的数据库中将用户信息保存/更新之后，接下来要给前端返回一个身份标志。

此时有两种主流方案可以选择（从逻辑上说也只有两种），一种是Session（数据保存在后端），一种是JWT（数据保存在前端）。

秉承「杀鸡用屠龙刀」的原则，我这个小破站准备一步到位，选择两种方案的组合。即JWT作为一级认证，负责一些不重要的请求的认证，Session作为二级认证，负责重要的操作认证。

实质上，只是因为Session涉及查数据库操作，为了性能考虑，将一些操作“下放”给JWT来代劳。

其实Session也可以有两种，一种是纯缓存(Redis)的Session，这种难以被“踢下线”，实质上安全性与JWT高不了多少，另一种是需要查数据库(Mongo/MySQL)的Session。因此Session又可以根据这个安全性分为两级。

## Session 篇

### 步骤一：下发 SessionID

从Google认证流程登录的用户，当然认为是最高等级的身份认证，因此我的应用在这里应该给用户返回一个高级的认证信息，即Session。

```go
func LoginGoogleCallback(c *gin.Context) {
    c.SetCookie("my-session", session, user.SessionMaxAge, "", "lewinblog.com", true, true)
}
```

注意了这里有很多参数（坑）：

1. `MaxAge`会指定浏览器的行为，超过这个声明周期之后会被浏览器删除，因此可以用来做登录期限限制。（当然我们不能信任用户代理，真正的限制得靠后端）
2. `Domain`用于跨域。不指定的话就仅限于颁发Cookie的域名才可用。由于我这里有多个子域名，颁发Cookie的是`api.lewinblog.com`，因此我要将domain设置为顶级域名，这样子域名才可用。
3. `Secure`会指定浏览器仅在https的情况下使用这个Cookie。
4. `HttpOnly`会指定浏览器仅在HTTP请求时使用。

### 思考：HttpOnly 的安全

> 2021-04-26更新：CORS，CSRF，XSS，这几个概念缩写太多，之前有点搞混了，请以现在的内容为准。

`HttpOnly`禁止的是JS脚本通过`document.cookies`来访问 cookie 中的内容。这个特性会成为 XSS 攻击的目标。

XSS实质是注入攻击。实践中，选择一个成熟的前端框架，不直接将用户输入在前端执行，中招的概率应该非常小。

但是毕竟注入这种事情，对于脚本语言来说是无法完全避免的。为了安全考虑，打开`HttpOnly`，让JS无法接触到，那么就算注入了也无法获取到Cookie，增加一份安全性。

> 其实如果注入成功了，那就算不直接访问cookie，也可以做很多事情。所以还是要做好注入防御才行。

### 思考：Domain 的安全

`Domain`正是防御CSRF攻击的有效手段。我们来重新审视一下CSRF攻击过程。

首先详细定义一下CSRF，它是攻击者在用户访问其他域名网页时，**利用用户浏览器中储存的Cookie发起请求**来实现攻击。因此它实质上是针对用户浏览器（用户代理）的攻击，因此我们只要针对浏览器规范来进行防御即可。

- 参考： [MDN规范: CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
- 参考： [Cross Site Request Forgery](https://guides.codepath.com/websecurity/Cross-Site-Request-Forgery)

一种方式是基于GET请求的。花样比较多，例如伪造一个`<a>`标签，或者在`<img>`标签的`src`中置入链接，都会发起GET请求。应对这种攻击的方式，可以是让服务端只接受POST请求，或者，（因为攻击者只能发请求而拿不到返回数据）只将查询类的请求（不修改用户数据）放在GET上。

一种方式是基于 [POST](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods/POST) 请求。原生的POST则是form，攻击者在攻击者的网页中置入指向被攻击者的`<form>`标签，诱导用户点击form来发送虚构的请求。应对这种攻击的方式是拒绝form请求（只接受json请求）。

> 鉴别form请求的方式，一是检查`content-type`是否是`application/x-www-form-urlencoded`（还有 multipart 等类型）。二是直接读取请求的BODY，form会以`键=值&键=值`的形式排列，这与JSON是有根本上的不同的，因此直接丢给JSON解析，解析失败那就肯定不是JSON请求了对吧 :)

在浏览器中要构造非form请求则必须通过 [XHR](https://developer.mozilla.org/en-US/docs/Web/API/XMLHttpRequest) 的方式，请求过程如下：

- 浏览器（以下都指代那些符合规范的浏览器）对符合某些规则的请求，在发出请求之前会有个`preflight`去检查跨域规则。
- 浏览器在执行XHR请求时，都会附带一个`origin`头。
- 因此服务端可以针对它做一个过滤，将来自白名单以外的域名的请求全部干掉，这样CSRF就消失了。
- 同时，浏览器也会检查每个跨域请求的响应头，检查其`access-control-allow-origin`和`access-control-allow-credentials`是否符合安全策略。

Golang用一个三方库的中间件实现：

```go
import "github.com/gin-contrib/cors"

func main() {
    app := gin.New()
    app.Use(CORS())
}

func CORS() gin.HandlerFunc {
    cfg := cors.Config{
        AllowWildcard:    true,  // 允许通配符
        AllowOrigins:     []string{"https://lewinblog.com"},  // 允许的域名白名单
        AllowCredentials: true,  // 允许请求携带凭证（即Cookies）
        // ...
    }
    return cors.New(cfg)
}
```

上面还提到`通配符`的概念。有一种情况，假如服务端程序员想要偷懒，可以设置一个`*`作为白名单（即允许所有域名）。在这种情况下，请求会被正常发出去，但是在返回时，如果浏览器发现响应头中包含`access-control-allow-origin: *`会自作主张地将请求拦下，不返回给前端js代码。这是一种额外的辅助安全策略。参考 [MDN](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS/Errors/CORSNotSupportingCredentials)

总结：要在服务端正确配置CORS各项参数。

> 什么？你问那些用不规范浏览器的用户怎么办？那你告诉我我能咋办？浏览器是用户的代理，如果浏览器都不能信任，只会让我们失去讨论的基础罢了。恕我直言：过期技术活该去死。

### 步骤二：前端使用 Cookie

XHR请求要指定一个额外参数，才会携带Cookie。在Angular中的实现：

```js
this.http.get<MyDataStruct>(url, {withCredentials: true})
```

### 步骤三：后端验证 Cookie

也是写一个gin中间件来完成这个事情。验证之后，装进Context里即可，后续请求可以从Context中取出用户信息。

```go
func RequireSessionMiddleware() func(*gin.Context) {
    return func(c *gin.Context) { 
        // 1. 获取sessionId
        session, _ := c.Cookie(SessionCookieName)

        // 2. 查表
        user, _ := GetUserBySession(ctx, session)
        if user == nil {
            c.String(http.StatusUnauthorized, "认证失败，请重新登录。")
            c.Abort()
            return
        }

        // 3. 用户信息注入context
        c.Set("user", user)
        c.Next()
    }
}
```

至于怎么查表，这要看数据库设计。

我这里没有用Redis，只有Mongo（实在懒得运维，能少一个组件就少一个）。其实我认为，在访问量不大、数据结构简单的情况下，Mongo也能抗住很大的并发才对（以后有时间做个压测吧）。

因此我的数据库设计，有一个`User`表，有一个`Session`表。每次查询时使用Mongo的聚合查询（类似MySQL的join）。做好相应的索引。

至于是用聚合查询更快，还是做两次`find_one`更快，我感觉是后者更快。但是前者只有一次请求，更省事，~~而且聚合查询真难写还挺练手的~~，所以我选择前者。

## JWT 篇

由于我不需要在前端使用JWT中的明文内容，所以我选择了：全密文+自定义序列化格式。

简而言之，将接口访问权限分为三级：最高级要求Session，第二级要求JWT（我的变种JWT），第三级是公开。

Session和JWT的内容都写在Cookie里，然后写一个简单的gin中间件来进行判断。如果JWT过期，则fallback到Session并重新签发JWT。

用户系统认证设计就是这么简单。
