```yaml lw-blog-meta
title: "HTTP缓存控制: Cache-Control"
date: "2022-09-23"
brev: "一文弄明白HTTP的缓存机制"
tags: ["前端"]
description: "本文彻底讲解了HTTP的缓存机制，包括 Cache-Control, Clear-Site-Data, Conditional Request, ETag 等概念和常规使用方式。"
keywords: "缓存,HTTP,Cache Control,Etag"
```

## 背景

昨天对我的个人网站做了一些功能迭代。其中一个点是实现了一个简单的图片存储服务，把番剧图片全部保存在我的服务器上了。（看了一眼，我这服务器竟然续费了5年，可以尽情折腾了……）

既然涉及到图片资源，那就要考虑HTTP缓存问题了。

之前使用缓存的场景，大部分都是通过 Nginx 的配置直接实现的，`add_header Cache-Control`这样写就行了；也有一些在后端服务内实现的，Go语言环境下，`c.Header("Cache-Control", "max-age=600")`这样写也就完事了。

可是，`Cache-Control`还有别的用法吗？`Etag`到底是如何生成和使用的？今天全面整理一下。

## Cache-Control

原文：[Cache-Control](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control) ，本节内容是对这篇文章的提炼总结，精简了部分内容。

`Cache-Control`是一个`HTTP`头部字段。它既可以出现在Request里，也可以出现在Response里。

大多数情况下，我们讨论的是后者，也就是服务器向客户端发回的响应头中携带这个字段。

先解释一些术语：

- 『Shared cache（共享缓存）』：可以被Proxy和CDN储存起来的、供任意用户重复使用的缓存。
- 『Private cache（私有缓存）』：包含了用户个人数据的内容，只能给某个具体的用户重复使用，不能共享。例如：包含了用户信息、认证token的响应。
- 『Reuse（复用）』：把缓存过的响应体，直接在本地返回给后续的请求。
- 『Revalidate（再验证）』：向服务器询问某个资源是否依然新鲜(`fresh`)。这个过程通常是通过`conditional request`来完成的（后面介绍）
- 『Fresh（新鲜的）』：响应体依然是有效的，可以复用
- 『Stale（过期的）』：响应体已经过期，不能直接复用；如果通过再验证（`Revalidate`），则会重新变为新鲜的（`fresh`），并且可以复用了。
- 『Age（年龄）』：某个响应体自生成以来经过的时间，它是评判缓存是否新鲜（`fresh`）的标准。

然后分别介绍 `Directives`，翻译为指令，意思是可以写在`Cache-Control`内部的可选字段。

### max-age

`max-age=N` 意思是缓存在`N`秒内可以认为是新鲜的。

注意，这个计算时间是自响应生成以来，而不是自客户端接收以来。在中间有Proxy、CDN这类中间人的场景会出现。这种情况下中间人还要提供一个`Age`头，来告诉客户端这个缓存在中间人这里已经经过了多长时间：

```text
Cache-Control: max-age=604800
Age: 100
```

```nginx
# Nginx配置示例：
location ~* \.(jpg|jpeg|png)$ {
    add_header Cache-Control "max-age=604800";  # 1week
}
```

### no-cache no-store

`no-cache`意思是“不要缓存”，这种情况下，客户端必须每次都再验证（`revalidate`）之后才可以使用。

但注意，`no-cache`依然可能会“储存”；如果连储存也要禁止，则使用更严格的`no-store`。

`no-cache`一般用在可能经常更新的内容上。典型例子是SPA应用的`index.html`文件。

```nginx
# Nginx配置示例：
location / {
    try_files  $uri /index.html;
    add_header Cache-Control "no-cache";
}
```

### must-revalidate

当缓存过期后，也就是超过`max-age`之后，按理来说客户端必须验证之后才能重新使用；但是有个特殊情况，当客户端无法与服务器重新连接的时候（因网络故障等），HTTP允许客户端继续使用过期的缓存。

`must-revalidate`则禁止了上述情况，即禁止客户端使用过期的缓存。

```text
Cache-Control: max-age=604800, must-revalidate
```

### immutable

`immutable`告诉客户端，缓存新鲜的时间内，不会发生改变。也就是说，“新鲜期内不要来问我”！

对现代web网页的一种最佳实践是，如果某些静态资源从来不会改变的话，给静态资源的URL中加入版本号或者Hash值，然后设定`immutable`等指令，这样可以减少不必要的请求发到服务器上去。（这种实践模式称为`cache-busting`，译为“缓存爆裂”）

```text
Cache-Control: public, max-age=604800, immutable
```

> immutable 的[兼容性](https://caniuse.com/?search=immutable)约等于没有，不用它也没什么区别。只用`max-age`足够达成 cache-busting模式 的需求。

### stale-

`stale-while-revalidate`意思是，当缓存过期后，可以宽限一段时间；在宽限时间内，缓存可以继续复用（`reuse`），但同时客户端也会尝试再验证（`revalidate`）这个缓存。

```text
# 缓存7天(604800)有效，第8天(86400)是宽限期
Cache-Control: max-age=604800, stale-while-revalidate=86400
```

`stale-if-error`意思也是指定一个宽限期，在宽限期内，如果遇到服务器异常（500,502,503,504），则可以暂时继续使用这个过期缓存。

```text
Cache-Control: max-age=604800, stale-if-error=86400
```

### 请求头中的Cache-Control

上面所说的，都是在响应体中的`Cache-Control`。（这个头一般只对Proxy、CDN等中间服务器有效，源服务器可以忽略。）

只讲两个常见的：

`no-cache`：当浏览器`force reloading`（强制刷新、禁用缓存）的时候，会携带这个指令。

`max-age`：当浏览器`reloading`（当作document打开并刷新时），会携带`max-age=0`

## Clear-Site-Data

参考：[Clear-Site-Data](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Clear-Site-Data)

上面提到了，现代的Web应用，如果使用了`cache-busting`模式，那么在缓存过期、再验证(`revalidate`)之前，服务端将没有任何手段控制客户端的缓存行为。

如果遇到一些异常情况，例如客户端误把异常响应缓存了下来，那就很糟糕了，要通过技术辅助手段清除缓存（或者强制刷新）之后才能恢复。这种情况很容易会提高客诉率。

> 这个场景，我还真的遇到过，必须点名批评 cdn.baomitu.com ，它会把上游资源的异常以200的形式返回给客户端，然后客户端一直缓存着错误的响应体内容。这个BUG曾经困扰了我们很长一段时间，因为难以本地复现，导致我们处理客户咨询的时候一直找不到问题的原因。

现代浏览器提供了一点点帮助，`Clear-Site-Data`可以从服务端发出指令，让客户端清除缓存。

## conditional request

参考：[conditional request](https://developer.mozilla.org/en-US/docs/Web/HTTP/Conditional_requests)

HTTP有个概念叫做`conditional request`（译为“条件请求”），通过将受影响的资源与验证器(`validator`)的值进行比较，可以改变请求的执行。这样的请求对于验证缓存的内容、验证文档的完整性(比如在恢复下载时，或者在服务器上上传或修改文档时防止更新丢失)等情况都很有用。

条件请求会根据 method, headers 的不同而进行不同的处理。

> 简而言之，所谓“条件请求”，就是普通的HTTP请求中多加几个请求头就是了。在后端的处理方法与普通请求相同，只需额外判断几个请求头即可。

### Validators

`Validators`（验证器）就是用来检验是否一致的标准。

- `Strong validation`，强验证器：逐字节地对比。（例如使用md5）
- `Weak validation`，弱验证器：通过某种简化的方式来对比。

例如，如果有两个页面，它们仅仅是Footer里的日期不同。在强验证器模式下，有字节不同，那么就应当认为这是两个不同的资源；而弱验证器可以认为它们是同一个资源。

HTTP默认使用强验证器，但是也指定了可以使用弱验证器的情况。

### Conditional headers

验证过程中可能会用到几个Header：

`If-Match`：如果`ETag`与头部中指定的任意一个值相符的话，那就成功（即继续执行请求）。（一般用于POST等不安全方法。涉及到“更新冲突问题”）

`If-None-Match`：如果`ETag`与头部中指定的所有值都不相符的话，那就成功（即继续执行请求）。（一般用于GET等不安全方法。例如，当 ETag 未曾改变时，验证失败，即返回`304 Not Modified`）

`If-Modified-Since`, `If-Unmodified-Since`：使用`Last-Modified`而不是 ETag 来作为判断条件。

`If-Range`：范围判断，可以用 Etag 或者 Last-Modified 。失败则响应 `200 OK`并附带完整资源内容，成功则响应`206 Partial Content`。（一般用于“断点续传”）

> “断点续传”涉及到几个新的Header以及通信流程，例如 Accept-Ranges, Ranges, Content-Range 等，这里不展开讲，有兴趣的同学可以前往MDN原文，上面有比较详细的图文讲解。  
> “更新冲突问题”（the lost update problem）也可以去看图文了解。

## ETag

参考：[ETag](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/ETag)

`ETag`，全名`entity tag`（译为：“完整性标签”）是一个 HTTP Response Header ，用来表明某个资源的具体版本。它是上文所说`Validator`的一种。

> 注意，ETag 是放在响应头里的，它的值会在请求头的其他字段中（例如 If-None-Match ）使用到。

标准语法：

```text
ETag: W/"<etag_value>"
ETag: "<etag_value>"
```

前面可选的`W/`，用来表明这个ETag是个『弱验证器』。（再次解释，如果弱验证器的值相等，则证明两个资源“在语义上是同一个”，而不一定是“所有字节完全相等”）。

`"<etag_value>"`，注意双引号是规范中说明必需的。其中 etag_value 是一个 ASCII字符串 。它的生成方法并没有规定，它可以是资源内容的hash值、时间戳、版本号等任意内容。

后端处理，用 md5 或者 sha1 生成一个hash来作为 ETag 即可。

> [参考](https://serverfault.com/questions/690341/algorithm-behind-nginx-etag-generation)：Nginx 默认生成的 etag ，是由 last_modified_time 和 content_length_n 组成的。并且会加上 `W/`标记。

## 小结

其实看一遍下来，`Cache-Control`的设置依然保持原来的使用方式就可以，就是`no-cache`或者`max-age`就行。然后`ETag`和`If-None-Match`是这次新学到的，使用体验效果挺好的。

还有一大块内容没讲到，是关于Proxy如何处理缓存的。我感觉这是一个比较庞大的问题，暂时不管，等我有需求了再来研究吧。
