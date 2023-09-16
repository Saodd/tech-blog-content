```yaml lw-blog-meta
title: "electron 打开 about:blank#blocked 的新窗口"
date: "2023-09-15"
brev: "拦截新窗口时遇到了一个无效的url"
keywords: "electron,open,window,blocked,url,setWindowOpenHandler"
tags: ["前端","疑难杂症"]
```

# 正文

我们现在的项目产品简单说可以算是`electron`打造的一个浏览器，为了实现浏览器的核心功能，我们拦截了页面中打开新窗口的事件，并由我们客户端内部逻辑去处理。

今天遇到一个oncall，某些网页上，在页面内打开新窗口时，会被electron的事件监听器读取到一个奇怪的值，

我尝试打个日志看一下，主要代码如下：

```ts
view.webContents.setWindowOpenHandler((details) => {
  console.log(details);
  // ... 这里有些业务逻辑省略 ...
  return { action: 'deny' };
});
```

输出：

```text
{
  url: 'about:blank#blocked',
  frameName: '',
  features: '',
  disposition: 'foreground-tab',
  referrer: { url: '', policy: 'strict-origin-when-cross-origin' },
  postBody: undefined
}
```

这个`details`看起来很奇怪，因为它的`url`是个无效的值，而正常情况下这个值应该是需要被打开新窗口的url地址。但是诡异的事情就在这里，尽管我在这个事件中无法读取到url，但是如果我不去拦截、让electron原本默认的逻辑去处理，是可以打开一个新窗口并导航到正确的url上去的；更别说，同一个网站的同一个行为在常规浏览器（chrome等）中表现更是正常。

我依然不能解释其中的原因，大概猜测，问题可能是electron在处理chromium的事件的时候丢失了一些附带信息，没有暴露到js层来，以至于开发者无法针对这类特殊的事件进行处理。

解决方案参考[这里](https://stackoverflow.com/a/62817116/12159549)。

最后我决定先让这个新窗口在后台隐藏打开，

```ts
view.webContents.setWindowOpenHandler((details) => {
  if (details.url === 'about:blank#blocked') {
    // ... 隐藏模式打开新窗口（用BrowserView即可）...
    return { action: 'allow', outlivesOpener: false, overrideBrowserWindowOptions: { show: true } };
  }
  return { action: 'deny' };
});
```

然后从这个窗口中取得真实url之后，再执行原本的逻辑：

```ts
// 对隐藏窗口的事件进行监听，获取真实url
view.webContents.on('did-create-window', (window, details) => {
  if (details.url === 'about:blank#blocked') {
    window.webContents.on('will-navigate', (evt, url) => {
      // ... 这里拿到了真实的url，可以去做事情了（例如在前台标签页打开显示）...
      window.destroy();
    });
  }
});
```

实际体验速度还可以，正常网速情况下不会出现明显的延迟情况。
