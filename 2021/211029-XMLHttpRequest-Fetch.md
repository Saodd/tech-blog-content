```yaml lw-blog-meta
title: "XMLHttpRequest 与 Fetch"
date: "2021-10-29"
brev: "前端http请求原理"
tags: ["前端"]
```

## 背景

说起在前端发请求，那一般默认就是`axios`了对吧，但是要知道，它的底层其实是`XMLHttpRequest`。

所以很长一段时间以来，我都以为前端的请求只有`form`和`XMLHttpRequest`这两种。虽然在chrome控制台里一直可以看见`Fetch/XHR`这个标签，可我总是对前面这个`Fetch`视而不见哈哈哈。

然后最近在写浏览器插件的时候，又见到`fetch()`这个东西，才意识到，我在这里真的有知识盲区。

所以那句话确实有道理：「"不知道"并不可怕，可怕的是"不知道我不知道"」，这句话送给本文每一位读者，再附赠一句：「技术不是万能的，但是没有技术是万万不能的」，希望大家不要局限在自己的知识茧房里，不要小看学习新技术的力量。

## XMLHttpRequest

先解释几个名词。首先是`AJAX`，全称是「Asynchronous JavaScript And XML」，重点是异步请求，这允许只更新页面的部分内容（而不是整页重载）。

然后`XMLHttpRequest`，表面上是请求`XML`资源，但实际上可以请求任何类型的资源。（这名字应该是历史原因）

本章参考自 [Using XMLHttpRequest - MDN](https://developer.mozilla.org/en-US/docs/Web/API/XMLHttpRequest/Using_XMLHttpRequest) 和 [AJAX - MDN](https://developer.mozilla.org/en-US/docs/Web/Guide/AJAX/Getting_Started)

### 基本用法

先准备一个简单的后端，这里用我熟悉的Go技术栈，代码省略。

然后为了避免跨域，在前端代理一下后端接口，配置 devServer:

```js
proxy: {
  '/api': {
    target: 'http://localhost:8000',
    pathRewrite: { '^/api': '' },
    changeOrigin: true,
  },
},
```

然后可以开始写前端代码了:

```typescript
async function main() {
  const req = new XMLHttpRequest();
  req.onload = function () {
    console.log(this.responseText);
  };
  req.open('GET', '/api/hello');
  req.send();
}
```

上面的代码中直接替换了`onload`方法，由常识可知，可以用`addEventListener('load',...)`代替。`XMLHttpRequest`一共有7种事件， [参考](https://developer.mozilla.org/en-US/docs/Web/API/XMLHttpRequest#events) 。 [XMLHttpRequest.readyState](https://developer.mozilla.org/en-US/docs/Web/API/XMLHttpRequest/readyState) 与上面说的事件很像，但并不完全相同。

关于`open()`方法，后面还有可选参数，第三项是`async`，如果显式指定为`false`，那会**同步地**执行这个请求，对，同步地，是不是很蠢？（可以通过`readyState`来观察什么叫做同步地）所以请不传或者传true 。

### 处理响应内容

有多种方式取出响应内容：

1. `.responseXML`取出来的是一个DOM对象（TS中是`Document`类型）
2. `.responseText`取出来的是`string`
3. 最原始的形式是`XMLHttpRequest.response`，它被标注为`any`类型

可以通过`XMLHttpRequest.responseType`来指定要将其转化为哪种类型的数据，取值范围`"arraybuffer" | "blob" | "document" | "json" | "text"`，默认值是`""`好像会根据服务端的`contentType`去选择解析方式。如果在`XMLHttpRequest`指定了解析方式，那么就会调用对应的函数去执行，那么也可能会抛出异常（例如指定了`document`却返回了一段`JSON`字符串）。

> 关于在前端处理二进制数据的方法，可以参考 [其他文章](https://www.cnblogs.com/penghuwan/p/12053775.html) 先了解一下，回头我另开文章整理一遍。

### 提交表单

先回顾一下HTML的`<form>`是如何使用的：

- `POST`方法，可以指定`enctype`为`application/x-www-form-urlencoded`, `text/plain` 和 `multipart/form-data`
- `GET`方法

代码的话，在之前的 [详解CORS](../2021/210922-Dig-CORS.md) 文章中已经有一些展示，这里不再重复。

接下来我们用纯JS，也就是`XMLHttpRequest`的方式提交form请求。我们可以借助`FormData`对象来帮助序列化参数。

基本用法：

```typescript
async function main() {
  const form = new FormData();
  form.set('AAAA', 'aaaa');

  const req = new XMLHttpRequest();
  req.open('POST', '/api/hello');
  req.send(form);
}
```

上面的用法会将数据以默认的`multipart/form-data`形式提交。然后我们可以在服务端把 URL 和 BODY 打印出来看看：

```text
/hello
------WebKitFormBoundaryKKA9AAyo38mxhvJ9
Content-Disposition: form-data; name="AAAA"

aaaa
------WebKitFormBoundaryKKA9AAyo38mxhvJ9--
```
`FormData`的最大问题就是不能序列化为字符串。它的好处是比较规范，并且原生就支持上传文件。

然后我们挖掘一下`send()`的入参，可以发现我们可以传很多种类的数据进去：

```typescript
type BodyInit = Blob | BufferSource | FormData | URLSearchParams | ReadableStream<Uint8Array> | string;
```

另一种相似的方式是使用`URLSearchParams`，用法完全相同，只不过在服务端看来，BODY部分的编码格式是不同的，即请求体中的`Content-Type`会是`application/x-www-form-urlencoded`。（提醒，这种方式兼容性更晚）

```typescript
async function main() {
  const form = new URLSearchParams();
  form.set('AAAA', 'aaaa');

  const req = new XMLHttpRequest();
  req.open('POST', '/api/hello');
  req.send(form);
}
```

```text
/hello
AAAA=aaaa
```

如果我们要以JSON格式提交数据，那么我们需要显式地将其转化为字符串传入：

```typescript
async function main() {
  const form = { AAAA: 'aaaa' };

  const req = new XMLHttpRequest();
  req.open('POST', '/api/hello');
  req.setRequestHeader('content-type', 'application/json');  // 既然传入了字符串，那就要显式设置头
  req.send(JSON.stringify(form));
}
```

```text
/hello
{"AAAA":"aaaa"}
```

### axios源码简析

这里简单看一下`axios@0.24.0`的源码，以`axios.post()`为例，首先它的底层是调用的`.request()`方法：

```javascript
utils.forEach(['post', 'put', 'patch'], function forEachMethodWithData(method) {
  /*eslint func-names:0*/
  Axios.prototype[method] = function(url, data, config) {
    return this.request(mergeConfig(config || {}, {
      method: method,
      url: url,
      data: data
    }));
  };
});
```

`request()`这个方法挺长的，其中有很大的篇幅都是处理中间件（也就是`interceptors`）的。执行请求的部分：

```javascript
  try {
    promise = dispatchRequest(newConfig);
  } catch (error) {
    return Promise.reject(error);
  }
```

`dispatchRequest()`里面呢，会拿`config`中指定的`adapter`(适配器)去执行请求。

```javascript
  var adapter = config.adapter || defaults.adapter;
  return adapter(config).then(function onAdapterResolution(response) {...}
```

不指定适配器的话，默认的适配器是`xhr`，xhr适配器逻辑简化如下：

```javascript
function xhrAdapter(config) {
  return new Promise(function dispatchXhrRequest(resolve, reject) {
      // ...从config中取出请求url、data、header等信息，并做处理
      var request = new XMLHttpRequest();
      // ...如果有Basic认证，那么处理
      request.open(config.method.toUpperCase(), buildURL(...), true);
      request.onloadend = function onloadend() {...}
      request.onabort = function handleAbort() {...}
      request.onerror = function handleError() {...}
      request.ontimeout = function handleTimeout() {...}
      // ...一些cookie和header的处理
      request.send(requestData);
      // ...
  }
})
```

所以核心逻辑依然是`open()`然后`send()`，只不过`axios`帮我们做了很多额外的常用功能。

## Fetch

本章节参考自 [Using Fetch - MDN](https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API/Using_Fetch) 

`Fetch`是`XMLHttpRequest`的升级版替代品，它允许控制更多的HTTP参数和行为。在一些特殊技术中，例如`Service Wrokers`，它会很好用。

它与传统的，以`jQuery.ajax()`为例，相比最明显的区别如下：

- 遇到非200响应时，不会抛出异常（而是将`ok`属性设为`false`）
- `fetch()`不会发送跨域cookies，后来改成默认跨域策略是`same-origin`，除非你主动设置
- （原生使用`Promise`）

### 基本用法

```javascript
async function main() {
    const resp = await fetch('/api/hello');
    console.log(await resp.json());
}
```

这会发起一个`GET`请求。然后要注意的是，在`Response`对象中，并没有直接访问Body的属性，而是要通过`json()`这类方法去实现，而且它返回的会是一个Promise .

### 更多参数

```javascript
async function main() {
  const form = new FormData();
  form.set('AAAA', 'aaaa');

  const resp = await fetch('/api/hello', {
    method: 'POST',
    body: form,
  });
  console.log(await resp.json());
}
```

这里的`body`参数，与`XMLHttpRequest`中的data参数完全一致（几乎完全一致，都是`BodyInit`，但是xhr额外支持传入DOM）

### 插曲：跨域认证方式

跨域的坑真的蛮多的，虽然我已经从多个角度总结好几次了，可是到这次还是发现不能完全自信地运用，所以这里再次总结一下：

- 跨域认证方式一：`业务服务`和`认证服务`是**同一个根域名**下的不同子域名，只要`认证服务`将cookie设置在根域名下，那么该根域名下的所有子域名都可以使用这个cookie（不限于根域名，只要是公共祖先域名即可）
- 跨域认证方式二：`业务服务`和`认证服务`的**根域名不同**，只能参考`OAuth`方式，`业务服务`借助浏览器跳转来使用`认证服务`的cookie，认证通过之后拿回一个token跳回`业务服务`换成自己的cookie（或者使用`JWT`）

### 跨域请求

为了测试跨域请求，我将前后端分离并设置在同一个根域名的不同子域名中：

- 在浏览器上做一个代理，将`api.test.com`转发到我的后端服务上，将`www.test.com`转发到前端devServer上
- 通过`www.test.com`打开前端页面
- 后端将Cookie设置在根域名`test.com`上

```javascript
async function main() {
  {
    const resp = await fetch('http://api.test.com/login', {
      method: 'POST',
      credentials: 'include',  // 必须要指定，否则无法接受set-cookie
    });
    console.log(await resp.json());
  }
  {
    const resp = await fetch('http://api.test.com/check', {
      method: 'POST',
      credentials: 'include',
    });
    console.log(await resp.json());
  }
}
```

这里的用法关键是`credentials`参数，设为`include`时将会带上所有可用的credentials，设为`same-origin`时只会发送当前子域名的（即不跨域），设为`omit`时则完全不发送。

其实跨域中还涉及`PreFlight`，不过不影响我们前端正常使用，略过不讲。然后`credentials`这个概念，似乎除了`cookie`还包括Header中的`Basic`认证。

### Request对象复用

```javascript
async function main() {
  const req = new Request('http://api.test.com/hello');
  console.log(await (await fetch(req)).json());

  const req2 = new Request(req, {method: 'POST'})
  console.log(await (await fetch(req2)).json());
}
```

### Response对象

可以自己模拟一个Response，这在`ServiceWorker`里会很有用。写法是：

```javascript
addEventListener('fetch', function(event) {
  event.respondWith(
    new Response(new Blob(), {
      headers: { 'Content-Type': 'text/plain' }
    })
  );
});
```

### Body

与`XMLHttpRequest`类似，Body可以有多种类型，通过不同的属性去获取。而且有趣的是，可以任意从`Request`或者`Response`对象上去获取，它们引用的是同一个Body：

- Request.arrayBuffer() / Response.arrayBuffer()
- Request.blob() / Response.blob()
- Request.formData() / Response.formData()
- Request.json() / Response.json()
- Request.text() / Response.text()

## 小结

`fetch()`比`XMLHttpRequest`的功能更丰富一点，而且原生支持`Promise`，体验上来说会更好。

但是在实际应用中，首先`fetch()`的兼容版本更晚一些，其次`axios`作为xhr的拓展已经可以说是事实上的标准了，所以其实并没有太大的必要去选择fetch。要么作为尝鲜，要么在一些特定场合（例如Workers、插件等）才会选择fetch .
