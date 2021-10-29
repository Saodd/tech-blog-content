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

先准备一个简单的后端，这里用我熟悉的Go技术栈：

```go
func main() {
	eng := gin.Default()
	eng.GET("/hello", func(c *gin.Context) {
		c.String(200, "I'm Lewin!")
	})
	eng.Run("0.0.0.0:8000")
}
```

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

关于`open()`方法，后面还有可选参数，第三项是`async`，如果显式指定为`false`，那会**同步地**执行这个请求，对，同步地，是不是很蠢。（可以通过`readyState`来观察什么叫做同步地。）所以请不传或者传true 。

(TODO)
