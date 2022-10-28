```yaml lw-blog-meta
title: "Web Worker 试玩"
date: "2021-10-26"
brev: "解锁前端多线程能力"
tags: ["前端"]
```

## 背景

在前面学习 `React Fiber` 和 `PIXI` 那些东西的时候我们可以发现，页面主线程的性能 在页面很复杂的时候会变成一个令人无法忽视的东西。

页面多复杂的时候？其实这种场景离我们并不远，至少离我不远，我之前在开发针对抖店页面的浏览器插件的时候就已经体会到了性能瓶颈。本身抖店的页面就很复杂很粗糙，有很重的框架、大量的轮询和PreFlight，本身性能就很差了，我还给上面加了大量的UI和业务逻辑，emmm……一言难尽……

我们都知道JS是单线程运行的，今天的主角`Web Worker`就是允许JS使用多线程能力。放到业务上来说，就是允许将一部分重CPU逻辑放到另一个线程中去，给主线程（UI线程）减负，提升整体性能和用户体验。

> 参考阅读： [Using Web Workers - MDN](https://developer.mozilla.org/en-US/docs/Web/API/Web_Workers_API/Using_web_workers)

## 简介

Web Worker 是一种简单的方式来实现在后台线程中执行脚本。worker线程的运行不会打扰UI线程。不仅可以做CPU密集任务，也可以通过`XMLHttpRequest`（有一些限制）或者`fetch`来实现IO操作。

创建worker线程之后，要通过「消息 messages」和「事件句柄 event handler」来与主线程进行通信。具体而言是使用`.postMessage()`方法，注意在这个过程中，数据是拷贝的而不是直接传引用的，也就是说，一份数据不能同时属于多个线程。（也可以用`transfer`来实现转让，避免拷贝）

主要用法很简单，先`new Worker()`就可以创建一个对象。它的参数是一个入口js文件的地址（url）。

在worker线程中的全局上下文不是`window`，要用`self`代替。

在worker线程中，几乎可以做任何事情，除了直接操作DOM、除了使用`window`上的一些方法和属性。你可以使用`WebSocket`、`IndexedDB`等特性。

worker可以制造新的worker 。

可以用`if (window.Worker)`来判断当前浏览器是否支持 Web Worker 。

## 插曲：工具配置

这里使用当前最新的构建环境（`webpack@5.49.0` + `webpack-dev-server@4.2.0` + `jsx` + `ts`）。

这个版本的 HMR 对于 Web Worker 不兼容，表现为：注入的contentHash与实际产生的文件名不同。解决方案是`devServer`配置`hot: false`项目（我自己摸索的，如果不对请指正）。

为了避免IDE报错，还要：

- 配置ESLint： 在 `.eslintrc.json` 配置 `env.worker: true`
- 配置Typescript: 在Worker代码文件第一行插入三斜线注释`/// <reference lib="webworker" />`

## 基本用法

接下来展示一下基本用法。我们在主线程中创建一个worker线程：

```tsx
const worker = new Worker(new URL('./deep-thought', import.meta.url));
```

> Webpack的相关说明参考: [Web Workers](https://webpack.js.org/guides/web-workers/) ，特别注意 `new Worker(new URL(..., import.meta.url))`这几个东西必须在代码上写在一起，URL对象如果经过函数传递，就无法被Webpack识别了。

原生语法是`new Worker()`，里面传入一个url参数，这里一定要配合后面的`import.meta.url`，这个东西会让webpack把实际的文件名（配置在`ouput.filename`里的那个，可以带contentHash的那个）注入进去替换掉。

我们的主线程和Worker线程的代码都可以是`.tsx`，但是在new的入参里不要带后缀，就跟正常的import一样的写法。

接下来我们添加一些逻辑。我们从主线程里去访问Worker线程，然后将返回的结果展示在UI上，代码如下：

```tsx
// 主线程: index.tsx
const worker = new Worker(new URL('./deep-thought', import.meta.url));

// 第1步：主线程发送
worker.postMessage({
    question: 'How are you?',
});
// 第4步：主线程收到响应消息
worker.onmessage = ({ data: { answer } }) => {
    setAnswer(answer);
};
```

```tsx
// Worker线程: deep-thought.ts

// 第2步：Worker接收消息
self.onmessage = ({ data: { question } }) => {
    // 第3步：Worker发回另一条消息
    self.postMessage({
        answer: 'You asked: ' + question,
    });
};
```

稍微讲解一下。从主线程通过`worker.postMessage`发送了一个数据，然后在Worker中通过`self.onmessage`收到了这条消息，消息的参数是一个event，其中的一个字段`data`是刚才发送过来的数据结构体。接下来，反过来，Worker通过`self.postMessage`发送数据，主线程通过`worker.onmessage`接收。

接下来分享一些工程实践中的经验：

### 捕获异常

捕获异常可以这样写，（但是注意这样捕获的异常是Worker线程中冒到顶层的异常，一般可用于加载阶段的异常捕获）：

```tsx
worker.onerror = (e) => {
  // e.preventDefault()
  setAnswer(`${e.filename} 的第 ${e.lineno} 行出现了错误: ${e.message}`);
};
```

关闭Worker，可以在主线程用`worker.terminate()`方法，或者worker线程内的`self.close()`方法。

### 函数封装

WebWorker这个东西呢，我们使用它的时候，往往肯定是用来解决一些特殊问题的。

此外还要考虑到某些不支持WebWorker的浏览器环境，即使不兼容，我们也会需要想办法直接在主线程上勉强运行原本设计在WebWorker中运行的代码。

因此从架构的视角来看，WebWorker中实现的能力，应该都以函数的形式进行封装。这层封装应当对上层透明，即业务方在调用能力的时候可以不用关心下面到底是WebWorker还是主线程（甚至还有其他环境）。

这个封装的过程，我就不展开讲了。大概是需要比较扎实的TS类型体操技能 + 一定的JS面向对象思维，然后通信过程可能需要自己定义`RequestID`这种东西来把响应内容送回到正确的调用方去。

如果不想自己封装的话，可以考虑使用开源库，例如：[comlink](https://github.com/GoogleChromeLabs/comlink)

### 数据的拷贝

WebWorker 的一个常见用途可能是用来处理视频之类的重CPU型操作，这些操作往往也伴随着大量二进制数据的转移，如果每次都要内存拷贝的话，性能上未免有些难受。

其实，它的`postMessage()`方法上已经原生提供了对象转移的方法，（[文档](https://developer.mozilla.org/en-US/docs/Web/API/DedicatedWorkerGlobalScope/postMessage)）

核心用法是把需要转移的对象放在一个列表上：

```ts
const data = new ArrayBuffer(8)
self.postMessage(
    { someData: data },   // message, 类型是any
    [data],  // transferList, 类型是 Transferable[]
)

// type Transferable = ArrayBuffer | MessagePort | ImageBitmap

// 还要注意，转移之后，这边的 data 对象依然存在并且可以访问，但是它的 byteLength 会变成 0 （这个特性可以用来判断是拷贝还是转移）
```

> 值得一提的是，[SharedArrayBuffer](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/SharedArrayBuffer) 也可以实现避免拷贝的需求，不过它相对危险且麻烦，一般只用在一些其他的特殊场合，在WebWorker与主线程的通信中一般不考虑使用它。

### 引入其他脚本文件

在Worker中还可以引入其他js文件，但是要用特殊的`importScripts()`方法，这个方法是阻塞的：

```tsx
importScripts('foo.js', 'bar.js'); // 引入两个文件
importScripts('//example.com/hello.js'); // 跨域引入文件
```

不过webpack一般会帮你bundle好，所有代码都放在一个js文件中了，正常来说应该不需要额外载入脚本。但大概也会有需要的场景吧，（例如你非要想用CDN资源的话，）示例用法：

```tsx
declare const axios: any;

console.log(axios);  // 这里会报错，因为还没有加载
importScripts('https://cdn.jsdelivr.net/npm/axios@0.21.1/dist/axios.min.js'); // 加载过程是同步阻塞的，不是异步的
console.log(axios);  // 这里可以访问✅

self.onmessage = function ({ data: { question } }) {
  console.log(axios); // 这里可以访问✅
  // ...
};
```

## Shared workers

前一章中介绍的都是「专用 Dedicated」workers 的用法。它只能被创建它的代码（线程）所访问到。

还有另一种叫做「共享 Shared」workers 。大概意思就是可以一个worker线程实例同时与多个页面主线程进行交互，这种情况下应该可以做到数据共享和数据推送。

我尝试了一下，但是发现 IDE(Goland) 、ts、webpack 的支持都大有问题，困难重重，而且可能应用场景也十分有限，所以不再继续深入研究。

大概感觉就跟浏览器插件的 background-script 的工作方式几乎一样。总之就是要像TCP一样先建立连接，然后通过连接去进行通信。

## Service Worker

它是 WebWorker 的一种特殊类型。也略过。
