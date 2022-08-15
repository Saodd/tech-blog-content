```yaml lw-blog-meta
title: "sentry-javascript 源码速读"
date: "2022-08-11"
brev: "看过源码才能用得放心"
tags: ["前端"]
```

## 背景

`Sentry`这个东西应该很多人都很熟悉吧，在我这里也不是第一次研究它了。

之前写了一个轻量版本的go客户端，名叫[saodd/alog](https://github.com/Saodd/alog) ，我自己使用下来感觉体验非常不错。因此我打算继续沿用这个设计思路，在前端也做一个轻量版的客户端。

为什么需要轻量版呢？因为我正在带的项目实现了一个很重的浏览器插件，而浏览器插件的工作环境是非常复杂的，我希望尽可能地减少与宿主页面之间的相互干扰，让尽可能多的东西在我们自己的掌控之下，减少盲盒的成分。当然，性能也算是一个考虑因素。

## 入门准备：Sentry的基本用法

```js
import * as Sentry from '@sentry/react';
import { BrowserTracing } from "@sentry/tracing";

Sentry.init({
  dsn: '__DSN__',  // 填入一个可以访问到的Sentry服务端地址
  integrations: [new BrowserTracing()],  // 功能1
  // ...
});

Sentry.captureMessage('测试一下吧')  // 功能2
```

## 源码与调试模式

源码仓库地址：[sentry-javascript](https://github.com/getsentry/sentry-javascript) 。这里直接看master分支的最新代码，版本号`7.9.0`左右。

> 下面的讲解过程中，对源码有很多简化，请以实际源码为准。

为了调试，我们要使用`yarn link`功能将其他的仓库对sentry的依赖指向我们刚刚下载的源码仓库的构建产物。由于`sentry-javascript`使用了大量的构建工具（`lerna`,`rollup`等）并且在构建脚本中做了一些奇淫巧计，因此接下来我们也需要用魔法打败魔法，做一些很脏的事情才能让他跑起来。

参考官方文档中的[Contributing指南](https://github.com/getsentry/sentry-javascript/blob/master/CONTRIBUTING.md)，其中提到了三个命令，最后我们需要：

```shell
$ yarn
$ yarn lerna bootstrap
$ yarn build:dev:watch
```

在上面的安装过程中，可能会发现安装特别慢，是因为它依赖了一个很重的库叫做`playwright`，这个库只在测试的时候用到，因此我们现在开发过程完全用不到，因此我们需要手动将`playwright`从所有的`package.json`文件中删去。

然后配置link，有一个脚本可以帮助我们一口气配置所有包，它是：

```shell
$ yarn link:yarn
```

然后我们去我们的前端仓库中（任何一个依赖了sentry进行调试的web项目仓库中），执行link动作：

```shell
$ yarn link "@sentry/browser"
# 或者其他你正在调试的包
```

此时启动web项目，依然会报错，显示`Can't resolve '@sentry/utils/esm/buildPolyfills'`，根据这个去搜一下代码，会发现他们是故意这样写的，然后估计我这里windows平台不支持某些特性导致没能正常运行。

此时我们要去把`buildPolyfills`这个包显式导出

```js
// packages/utils/src/index.ts 添加一行
export * from './buildPolyfills';
```

然后执行全局替换，将全局的`@sentry/utils/esm/buildPolyfills`替换为`@sentry/utils`，这时才能正常运行。

到这里还要注意一个细节，估计构建的时候会将外层的`console.log`全部删掉，因此我们要将它写在函数体里，才能在web页面上观察到输出。

> 到这里我已经累得半死了 _(:з)∠)_

## init：初始化

在上面的示例代码中，引用的是`@sentry/react`这个库的`init`函数，它的作用是注入一些额外的、无关紧要的信息，然后它会继续调用`@sentry/browser`的`init`函数，后者再次调用到`@sentry/core`的`initAndBind`函数。

在`init`函数中用到了一个`BrowserClient`，它本身没什么东西，但是它所继承的父类——来自`packages/core/src/baseclient.ts`的`BaseClient`做了很多事情。

例如，在执行了`init()`之后，抛到全局的异常就会被Sentry捕获并且发送事件到后端，那么这个事件处理器是如何被注册的？

### Integrations

在`init`函数的参数`options`中，有一个字段叫做`integrations`，大概可以理解为是插件的意思，他们将具体的业务逻辑与框架隔离开来，是一种很常见也很优秀的设计（划分到设计模式上应该叫什么来着？）。

`integrations`分为两部分，一部分是Sentry默认指定的`defaultIntegrations`，另一部分是用户自定义的`userIntegrations`(init-Options 中的字段名为`integrations`)

看一眼默认配置，有8个默认的integration：

```ts
// packages/browser/src/sdk.ts
export const defaultIntegrations = [
  new CoreIntegrations.InboundFilters(),
  new CoreIntegrations.FunctionToString(),
  new TryCatch(),
  new Breadcrumbs(),
  new GlobalHandlers(),
  new LinkedErrors(),
  new Dedupe(),
  new HttpContext(),
];
```

### Integration: GlobalHandlers

`GlobalHandlers`是 defaultIntegrations 的其中一种，我们简单看一下它做了什么事情

```ts
// packages/browser/src/integrations/globalhandlers.ts
export class GlobalHandlers implements Integration {
    private _installFunc: Record<GlobalHandlersIntegrationsOptionKeys, (() => void) | undefined> = {
        onerror: _installGlobalOnErrorHandler,
        onunhandledrejection: _installGlobalOnUnhandledRejectionHandler,
    };
    public setupOnce(): void {
        // ...上述两个函数会在这里被调用，然后挂载到Client上去
    }
    // ...
}
```

`_installGlobalOnErrorHandler`和`_installGlobalOnUnhandledRejectionHandler`会负责捕获抛到全局的异常。他们被`GlobalHandlers.setupOnce()`调用，然后通过`addInstrumentationHandler`将handler函数注册到Listener上去，

大概长这样：

```ts
// packages/browser/src/integrations/globalhandlers.ts
function _installGlobalOnErrorHandler(): void {
  addInstrumentationHandler(
    'error',  // 对应 window.onerror
    (data: { msg: any; url: any; line: any; column: any; error: any }) => {
      // ... data 是
    },
  );
}
```

顺带一提，`window.onerror`这个函数本身就有很多参数：

```ts
// packages/utils/src/instrument.ts
function instrumentError(): void {
  _oldOnErrorHandler = global.onerror;

  global.onerror = function (msg: any, url: any, line: any, column: any, error: any): boolean {
    // ...
  }
  // ...
}
```

异常被捕获之后的事情也很容易理解了，Sentry会再添加一些附加的追踪信息（例如日志输出、Breadcrumb、url等信息），然后通过HTTP的方式发送到后端进行收集。然后我们就能在Sentry前端上进行查看和统计分析了。

### Integration: Breadcrumbs

Breadcrumbs 这个单词直译是『面包屑』的意思，根据典故引申出的意思是『痕迹』。

顾名思义，这个中间件(`integration`)会收集一些“沿途的痕迹”，典型的有：console日志输出、xhr/fetch请求、ui事件、exception异常等等，并且他们按照时间进行排序，我们事后可以根据这些蛛丝马迹来尝试复原用户当时的行为路径。

从它的默认配置项中可以得知它监控了哪些东西：

```ts
// packages/browser/src/integrations/breadcrumbs.ts
export class Breadcrumbs implements Integration {
    public constructor(options?: Partial<BreadcrumbsOptions>) {
        this.options = {
            console: true,
            dom: true,
            fetch: true,
            history: true,
            sentry: true,
            xhr: true,
            ...options,
        };
    }
    // ...
}
```

### Integration: HttpContext

它主要负责记录：url, headers, location, referrer, userAgent 这几样东西，类似于是『用户指纹』的功效。

### Integration: TryCatch

尝试帮你补充一些容易遗漏的异常，有这些：

```ts
// packages/browser/src/integrations/trycatch.ts
export class TryCatch implements Integration {
    public constructor(options?: Partial<TryCatchOptions>) {
        this._options = {
            XMLHttpRequest: true,
            eventTarget: true,
            requestAnimationFrame: true,
            setInterval: true,
            setTimeout: true,
            ...options,
        };
    }
    // ...
}
```

### Integration: LinkedErrors & Dedupe

`LinkedErrors`大概是帮你找出相关联的异常，核心功能是`_walkErrorTree`

`Dedupe`是尝试去除重复的异常。

### Integration: 小结

这几个默认中间件的功能都很基础，很常见，也很重要。

除了Sentry本身监听了太多事件这可能会对性能略有影响之外，正常情况下我们就使用这一套默认的中间件就足够满足需求了。比较常见就是在此基础上做做减法，过滤一些我们觉得不想看到的消息，例如非常典型的`ResizeObserver loop limit exceeded`。

## Hub：客户端对象

和大多数库的设计思路相同，Sentry也有一个唯一的对象来挂载各种配置和自定义逻辑，这个东西在Sentry的世界里被称为：`Hub`

从顶层往下看，这个Hub它是挂载在全局对象上的，全局对象则由当前的运行环境决定，代码很经典：

```ts
// packages/utils/src/global.ts
export function getGlobalObject() {
  return (
    isNodeEnv() // Node环境
      ? global
      : typeof window !== 'undefined' // 浏览器环境
      ? window
      : typeof self !== 'undefined' // WebWorker环境
      ? self
      : fallbackGlobalObject
  );
}
```

拿到这个global之后，它会被称为`Carrier`，然后Hub会挂载在它的`__SENTRY__`属性上：

```ts
// packages/hub/src/hub.ts
export function getMainCarrier(): Carrier {
  const carrier = getGlobalObject();
  carrier.__SENTRY__ = carrier.__SENTRY__ || {
    extensions: {},
    hub: undefined,
  };
  return carrier;
}
```

作为实验，我们可以打开一个启用了Sentry的web页面，在控制台上看看`window.__SENTRY__`是否存在。

然后我们再看hub的声明部分，它是一个class：

```ts
// packages/hub/src/hub.ts
export class Hub implements HubInterface {
    // ...
}
```

在它的定义中我们可以看到很多很重要而且也很熟悉的东西，例如`captureException`, `captureMessage`, `captureEvent`三个API函数，例如`Session`, `Transaction`, `Scope`, `Extra`等概念的配置和使用，等等。

### 小实验：劫持captureMessage

```ts
export class Hub implements HubInterface {
    public captureMessage(
        message: string,
        level?: Severity | SeverityLevel,
        hint?: EventHint,
    ): string {
        console.log(message, level, hint)  // 这可以观察到传入的参数
        return ''  // 返回的是一个eventId可以随意
        // ...原来的代码在后面
    }
}
```

## 小结

总的看下来，累赘功能也没有特别多，尚且都在可以接受的范围。

而且配置项也并不算丰富，想配置也配置不了，不如干脆不配置了，只对`init`做一些基本配置就直接上线使用吧。

目前看来使用体验不错。后续补充。

### 示范：修改StackTrace深度

Stack Trace 这个功能呢，对于Webpack打包出来的东西，也就是混淆之后的代码来说，意义不大。因此我们可能有一种想法是，减少它的追踪深度，以此降低性能开销。

它并没有一个直接的设置项来设置这个东西，因此我们需要绕一点远路。

一个简单粗暴的方式是，在最后发送之前拦截一下：

```ts
init({
    beforeSend: (e) => {
        e.exception.values.forEach((ex) => (ex.stacktrace.frames = ex.stacktrace.frames.slice(0, 5)));
        return e;
    },
});
```

> 上面这种操作我在之前go语言中已经玩过了，所以这次用起来轻车熟路的。

另一个治本的方式是，改写源码中的`defaultStackParser`（出自`packages/browser/src/stack-parsers.ts`），从源码层面修改最大值为你想要的值。

## 最后：记得清理源码

刚才在调试源码的时候执行了很多`link`命令，所以最后要记得执行很多`unlink`命令——在所有`build`目录，以及使用link的web项目中都要分别执行`unlink`命令。

一个简单的办法是直接使用`clean`命令。
