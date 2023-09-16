```yaml lw-blog-meta
title: "浏览器js下载文件时的兼容性考虑"
date: "2023-09-15"
brev: "在某些情况下，借助原生a标签实现的下载能力可能会发生意外"
keywords: "electron,open,window,blocked,url,setWindowOpenHandler"
tags: ["前端","疑难杂症"]
```

# 背景

在浏览器页面中使用js下载文件是个非常常见的需求，一般做法是借助原生的`<a>`标签来实现，兼容性最好。

不仅是后端服务器提供的url，就连前端本地生成的二进制数据，也可以借助`Blob`等现代API来实现直接下载。

但是其中有些兼容性的细节还值得深挖。

# 正文

这次又遇到一个oncall。我们产品系列包含一个浏览器插件(chrome extension)，今天突然被反馈说，导出表格并作为文件下载的能力失效了。原本可以正常触发浏览器下载行为、将文件下载到本地硬盘的，而如今会把页面导航到一个`https://xxx.com/3b743d5a-e336-48b8-a65c-4b19a82425a2` 这样的附带一个ID的url上去。

附带的ID看起来很眼熟，结合“下载”这个行为，我几乎可以确定这个ID应该是`URL.createObjectURL`所产生的ID，可是为什么突然不能下载了，而变成导航了呢？

下载变成导航，我的第一个念头是，难道是触发了`a`标签的跨域限制？——可是仔细检查了一番，域名并没有变化。

我在后台的异常收集、日志收集平台里也没能发现任何端倪。

于是只能在代码层面硬找。然后发现一个诡异的事，我在项目工程中自己写的工具函数，同样使用的`URL.createObjectURL`，可以正常运行；而使用三方库[sheetjs](https://www.npmjs.com/package/sheetjs)内置的下载功能却不行。

经过仔细对比，终于发现了，`sheetjs`作为一个支持跨端的三方库，为了兼容性，它的下载文件的部分写的非常完善，相比于我自己写的代码多了`document.body.appendChild()`这一句（一对），多出的两行大概这样：

```typescript
function downloadByDocument(filename: string, data: ArrayBuffer): void {
  const a = document.createElement('a');
  a.download = filename;
  const u = URL.createObjectURL(new Blob([data]));
  a.href = u;
  // document.body.appendChild(a);  // 多了这个
  a.click();
  // document.body.removeChild(a);  // 以及这个
  setTimeout(() => URL.revokeObjectURL(u), 0);
}
```

那么这一句的作用到底是什么？

根据[JS前端创建html或json文件并浏览器导出下载](https://www.zhangxinxu.com/wordpress/2017/07/js-text-string-download-as-html-json-file/)这篇文章所说，它仅仅为了保证 firefox浏览器 的兼容性。

然而，在某些运行环境下，尤其是浏览器插件这种身不由己的运行环境中，如果`document`上被意外加入了事件监听拦截逻辑的话，那多这一行就会导致点击事件被拦截处理，进而导致程序没有按预期执行，也就产生了这次的oncall。

# 拓展阅读：sheetjs源码

最后再仔细观察一下`sheetjs`的源码，它的下载判断逻辑的源码，经过简化后主要结构如下：

```js
function write_dl(fname, payload, enc) {
    if(typeof _fs !== 'undefined' && _fs.writeFileSync) return _fs.writeFileSync();  // node.js 的 fs 库
    if(typeof Deno !== 'undefined') return Deno.writeFileSync();  // Deno
    if(typeof IE_SaveFile !== 'undefined') return IE_SaveFile();  // IE
    if(typeof navigator !== 'undefined' && navigator.msSaveBlob) return navigator.msSaveBlob();  // IE Blob
    if(typeof saveAs !== 'undefined') return saveAs();  // 不知道
    if(typeof chrome === 'object' && typeof (chrome.downloads||{}).download == "function") {
        return chrome.downloads.download();  // 浏览器插件能力
    }
    
    // 如果上述特殊方法都没有，最后进入常规流程
    var a = document.createElement("a");
    if(a.download != null) {
        a.download = fname;
        a.href = url;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        if(URL.revokeObjectURL && typeof setTimeout !== 'undefined') setTimeout(function() { URL.revokeObjectURL(url); }, 60000);
        return url;
    }
    
    // 后面还有一些尝试性的代码，省略
}
```

从上述代码中我们可以看到，一个具有良好兼容性的代码是要考虑非常多的情况的，虽然很多细节对于绝大多数程序员来说都是陌生而且其实也没有必要掌握的。
