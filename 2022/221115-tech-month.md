```yaml lw-blog-meta
title: "技术月刊：2022年11月"
date: "2022-11-24"
brev: "offscreen canvas, Secure contexts, Generator, MediaSource, npm public"
tags: ["技术月刊"]
description: "offscreen canvas, Secure contexts, Generator, MediaSource, npm public"
```

## 检测用户是64位还是32位

[Detect 64-bit or 32-bit Windows from User Agent or Javascript?](https://stackoverflow.com/questions/1741933/detect-64-bit-or-32-bit-windows-from-user-agent-or-javascript)

核心还是检查UA字段，但是可能出现的值有很多，因此这种检测可能并不完全可靠。

我的一个未经验证的灵感：写几行简单的wasm脚本运行一下，看看int的长度。

## 检测当前运行环境是WebWorker

- [Any standard mechanism for detecting if a JavaScript is executing as a WebWorker?](https://stackoverflow.com/questions/7507638/any-standard-mechanism-for-detecting-if-a-javascript-is-executing-as-a-webworker)
- [Reliably detect if the script is executing in a web worker](https://stackoverflow.com/questions/7931182/reliably-detect-if-the-script-is-executing-in-a-web-worker)

```js
if (typeof WorkerGlobalScope !== 'undefined' && self instanceof WorkerGlobalScope) {
    // huzzah! a worker!
}
```

## offscreen canvas

参考阅读：

- [HTMLCanvasElement.transferControlToOffscreen()](https://developer.mozilla.org/en-US/docs/Web/API/HTMLCanvasElement/transferControlToOffscreen)
- [OffscreenCanvas — Speed up Your Canvas Operations with a Web Worker](https://developer.chrome.com/blog/offscreen-canvas/)

简而言之，这个东西就是可以把 canvas 传到 webworker 中去进行操作，有效提升页面性能。（原本`HTMLCanvasElement`是一个DOM元素，是无法在WebWorker中访问的）

> 图像可以丢到Worker中去绘制，但是音频数据目前依然只能在主线程中播放。需要使用[Web Audio API](https://developer.mozilla.org/en-US/docs/Web/API/Web_Audio_API)

## Secure contexts

参考：[Secure contexts](https://developer.mozilla.org/en-US/docs/Web/Security/Secure_Contexts)

有许多先进的 Web API 可能都会要求 『Secure contexts』（直译为：安全上下文）环境下才可以使用。

什么是安全上下文：

- 加了 TLS 的（`https://`, `wss://`）
- 本地资源，例如：
  + `http://127.0.0.1`
  + `http://localhost`
  + `http://*.localhost`
  + `file://`

> 值得一提的是，localhost 结尾的域名会被强制解析到`127.0.0.1`地址上去，这个过程不受浏览器代理插件（如SwitchyOmega）的影响，因此它的实际意义并不大。

如何判断当前是否正处于安全上下文环境，在JS中可以这样写：

```js
if (window.isSecureContext) {
    // ...
}
```

当然，也可以直接检测你所需的API是否存在，例如`SubtleCrypto`, `Clipboard`, `VideoDecoder`等，（这种方式还顺便把兼容性也一起考虑了）。

这个东西的存在，说是说“保护用户设备安全”，但我目前个人认为它并没有什么实际价值，反而只是给我们开发人员捣乱。

在web开发、测试阶段，如果没有https的本地支持而又想要满足安全上下文的要求，最可行的解决办法是[强制浏览器对特定域名开启安全上下文](https://stackoverflow.com/questions/34878749/in-androids-google-chrome-how-to-set-unsafely-treat-insecure-origin-as-secure)。即，在`about://flags`设置页面中指定开启`unsafely-treat-insecure-origin-as-secure`。

## Generator

`Generator`，译名生成器，这个概念其实几乎所有的语言都能够支持（据我所知至少Python和JS都有原生的API，Go不需要但是也可以自己模拟实现）。

我很早就知道这个概念，但是似乎一直没有找到特别合适的场景。

直到我开始做音视频处理，有流式数据处理需求的时候，发现这个东西可能有点用。（但是实际写出了一版之后再看，其实并不好用，真的不如自己管理状态）

参考阅读：[Generator - MDN](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Generator) 。注意它支持`async`用法，即每个`next()`都返回一个`Promise`（但是 Generator对象 本身依然是同步的而不是 Promise）。

关于Typescript类型，则是通过三个泛型变量来声明：

```typescript
interface Generator<T = unknown, TReturn = any, TNext = unknown> extends Iterator<T, TReturn, TNext> {
    next(...args: [] | [TNext]): IteratorResult<T, TReturn>;
    // ...
}
```

上面这个类型系统有些讨厌，约束得不是很死，我们在写代码的时候要多写几行才能完美适配类型。

### AsyncGenerator

在异步用法中有几个语法细节提示一下。

```ts
// 普通函数写法
async function* gen() {
  yield 0
}

// 类方法写法
class A {
  // 注意 * 的位置
  async *gen(): AsyncGenerator<number, null, string> {
    // 接收值和返回值的类型都与上面的泛型类型相关联
    const receivedValue: string = yield 0;
    return null;
  }
}
```

### 预激生成器

如果需要在`.next(value)`中传入值，那么要注意了，生成器第一次"生成"的时候，是还没有这个传入值的。

因此需要抛弃第一次循环，也就是所谓的"预激"。具体到代码上，就是创建了生成器之后要立即额外`.next(null)`一次。这里的逻辑怎么理解呢，就是在创建生成器的时候，生成器函数内部并没有被执行，而是从第一次`.next()`的时候才会开始执行；第一次执行到`yield`的时候才会交出控制权并等待输入，等第二次`.next()`开始才是预想中的循环。建议你自己写个demo测试一下便于理解。

参考：[Generator.prototype.next()](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Generator/next)

> "预激"这个术语我是从Python世界中了解到的（估计是硬造出来的术语，参考：[预激协程](https://jarvisma.gitbook.io/pythonlearn/4.3-sheng-cheng-qi/chapter4.3.4)），我目前暂时没有在JS的世界里找到相关的术语，如果你知道的话，欢迎写信告诉我

## MediaSource与MediaStream的区别

参考阅读：

- [MediaSource vs MediaStream in Javascript](https://stackoverflow.com/questions/51843518/mediasource-vs-mediastream-in-javascript)
- [MediaSource, MediaStream and video/audio tags](https://csplite.com/csp152/)

简而言之，两个是完全不同的东西，而且也互相之间不能兼容、转换。

`MediaSource`是更传统的媒体操作方法，是经典的`URL.createObjectURL()`+`append buffer`组合拳的一部分。更灵活、兼容性更好，相关资料也更丰富。

`MediaStream`则是最近有点热度的`webRTC`体系下的东西，只能用`webRTC`相关API去操作。它相对更底层，效率更高，但是如上面的参考文章所说它可能夹带了很多Google的私货。很多最新的音视频API都跟这个家族相关。

## npm public

参考官方文档：[Creating and publishing scoped public packages](https://docs.npmjs.com/creating-and-publishing-scoped-public-packages)

在Golang的世界里，只需要一个git仓库，就能任意实现“发包”的动作。然而，在js的世界里，发包需要到宇宙中心`npm`上去发布（发布到自建npm也是相同原理）。

要在`npm`上发包，首先需要注册一个账号（废话）。但是需要提醒的是，注册之后，账号必须要开通“两步验证(2FA)”才允许发包（也可以去设置就是不用它）

> 说到`2FA`这个话题，其实最痛的点无非就是云端备份/迁移了。我用过许多工具，包括`Authy`（特点：可以上传到云端）， `Microsoft Authenticator`（我以为它可以上传），最后我还是回到了最简单的`Google身份验证器`（虽然不能传到云端，但似乎可以在手机之间迁移了）。但话说回来，云备份是有风险的，如果账号数量不是特别多的话，每次换手机（2FA设备）的时候重新生成一遍，也不是不能接受的事情。

npm上发包有三种模式，第一种，发布到全局的命名空间里，例如`react`；第二种，发布到个人/组织名下，包名以个人/组织为前缀，例如`@types/react`；第三种，发布为私有包，这个需要付费开通，实际上是把资源文件托管在`npmjs.com`上了。

发包所需的配置，基本上都是写在`package.json`里进行配置的。

通过`.gitignore`或者`.npmignore`来过滤掉那些敏感数据或者你不想公开的内容。参考[Developer Guide](https://docs.npmjs.com/cli/v8/using-npm/developers)

最后发布之前，最好确认一下会有哪些文件被发布到`npm`上去。

```shell
npm pack
```

上述命令会在当前目录下生成一个`.tgz`文件，其中的内容就是将会上传到npm的所有内容了。

最后执行发包命令：

```shell
npm publish --access public
```

## spread语法的时间复杂度

`...`这个很常见的操作符有它的名字，叫做[spread语法](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/Spread_syntax)，与它相反的操作叫做[rest语法](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Functions/rest_parameters)，示例代码：

```ts
const arr = [1,2,3]
const b = [...arr]  // spread 
```

上面示例的数组的 spread 操作，是实现数组浅拷贝的很经典的操作。既然是浅拷贝，那么根据一般认知，它应该至少是O(n)的复杂度。

即使某些实现（例如V8）给它做了一些优化，但是只能把它当作“意外之财”，至少我们在做算法复杂度分析的时候是不能考虑那些特殊优化的。

甚至，为了实现`iterator`这个通用协议，这种浅拷贝的代价可能还比想象中更大。参考V8引擎团队关于这个问题的优化分析：[Speeding up spread elements](https://v8.dev/blog/spread-elements)

## sentry-plugin注入了多余的内容

我之前在[《技术月刊：2022年5月》 - 7.sentry上传sourcemap](/2022/220525-some-fe-skills.md#7-sentry上传sourcemap) 中提到过sentry上传sourcemap的操作方式。

今天遇到了新的坑：使用了`@sentry/webpack-plugin`这个插件打包的代码，其中有一份代码无法正常运行。

追查源码，在`@sentry/webpack-plugin`这个库的`SentryCliPlugin.injectEntry`函数中，描述了它将会在指定的`entries`中注入一个额外的文件，注入之后的执行效果大致如下：

```js
// 即挂载一个全局变量 window.SENTRY_RELEASE = {id: "1.0.5"}
("undefined" != typeof window ? window : void 0 !== n.g ? n.g : "undefined" != typeof self ? self : {}).SENTRY_RELEASE = {id: "1.0.5"}
```

这个注入其实是多余的（因为我们正常情况下都会通过别的手段注入这类信息）、而且是危险的（你一个插件不要随便做一些乱七八糟的事情啊！我以为你仅仅只是上传文件这一个功能！！）。

关闭这个功能的方式，在插件配置项中指定`entries: []`。
