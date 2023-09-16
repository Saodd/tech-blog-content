```yaml lw-blog-meta
title: "浏览器插件注入js脚本"
date: "2023-09-15"
brev: "V3版本manifest为我们提供了新的注入方式"
keywords: "chrome,extension,inject,js,manifest,V3"
tags: ["前端"]
```

# 正文

我们知道，浏览器扩展所执行的`content.js`代码，在V2时代，是只能运行在独立运行时环境的（即所谓的`ISOLATED`环境），是与页面主线程不同的运行时环境。

因此，在V2时代，如果想要访问主线程上的东西，例如`window`上的对象、或者`DOM`中的复杂属性，我们需要另外“注入”一段代码到主线程中去执行、然后再通过某种手段来实现两个运行时之间的通信才能实现。

核心代码如下：

```ts
function injectScript(file: string): void {
  const s = document.createElement('script');
  s.setAttribute('type', 'text/javascript');
  s.setAttribute('src', file);
  document.head.appendChild(s);
}

injectScript(chrome.runtime.getURL('/inject.js'));
```

上面的思路，在V3时代也依然可以运行。

但是在一种特定需求情况下，例如我们要求我们的代码必须在页面加载之前执行，那上面的方法就不能再用了，因为在`document_start`的阶段，`document.head`还是个`null`，无法挂载新的`script`标签。

这次V3的`manifest`给我们提供了一种新的、更加直接有效的方式：我们可以直接指定在`MAIN`运行时中运行代码。[参考链接](https://stackoverflow.com/a/75202975)

核心配置如下：

```json
{
  "name": "script injection",
  "version": "0",
  "manifest_version": 3,
  "minimum_chrome_version": "103.0",
  "content_scripts": [
    {
      "matches": ["*://*/*"],
      "js": ["inject.js"],
      "run_at": "document_start",  // 这一行
      "world": "MAIN"  // 这一行
    }
  ]
}
```
