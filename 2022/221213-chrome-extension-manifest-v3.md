```yaml lw-blog-meta
title: "浏览器插件 Manifest V3 应对方案"
date: "2022-12-13"
brev: "chrome 即将强制推行 Manifest V3，作为开发人员应该如何应对"
tags: ["前端"]
keywords: "Manifest,V3,Chrome,extension"
```

## chrome extension Manifest V3

在『浏览器插件』这个领域，chrome浏览器、 chromium系浏览器、甚至包括firefox都是遵循 『chrome extension Manifest 规范』的。

之前的主流协议版本是『V2』，它赋予了浏览器插件以几乎无所不能的权限，许多热门的浏览器插件、以及我们项目组正在商业化运营的产品都是基于V2的能力所构建的。

但是随着『[V3](https://developer.chrome.com/docs/extensions/mv3/intro/)』版本的推出，V2的生命周期也开始进入倒计时。参考[Manifest V2 support timeline](https://developer.chrome.com/docs/extensions/mv3/mv2-sunset/)，几个关键时间点：

> 2022年12月9日更新：根据[Pausing Manifest V2 phase-out changes](https://groups.google.com/u/0/a/chromium.org/g/chromium-extensions/c/zQ77HkGmK9E)的通知，v2的废止计划将被延期，目前具体时间未定。（看起来至少延期3个月以上，也许2023年能这样苟过去了……）

- 2022年6月：V2版本的插件将无法上架『Chrome Store』
- 2023年1月：从chrome112开始，在开发分支上（可能）关闭对V2的支持
- 2023年6月：从chrome115开始，在所有分支，包括正式版上（可能）关闭对V2的支持
- 2024年1月：企业版插件也将不再支持。商店中所有V2的插件都将被移除

因此对于一个商业化的插件来说，可以认为2023年6月是真正的deadline 。

### V3的典型受害者：Tampermonkey

Tampermonkey 是一个非常强大的、允许用户随意修改并向网页中注入一些脚本逻辑的浏览器插件。从技术上来讲，我认为这个插件几乎是把浏览器插件的能力用到了极致，是所有其他插件值得学习的老前辈。

但是它的能力还是受到底层浏览器的限制。V3的一些改变对它来说将是毁灭性的打击。更多信息可以从[issue#644](https://github.com/Tampermonkey/tampermonkey/issues/644)获知。

除了 Tampermonkey 之外，另一类典型的受影响的插件是广告拦截类插件，虽然原理略有不同，[参考阅读](https://nordvpn.com/blog/manifest-v3-ad-blockers/)。

## 应对方案

关于V2插件如何改造为V3，可以参考官方文档[Migrating to Manifest V3](https://developer.chrome.com/docs/extensions/mv3/mv3-migration/#man-sw)这篇文章，但它讲的主要是关于`manifest.json`本身的。

具体到业务逻辑代码上还有很多需要注意的地方，例如：

- CSP策略中的`script-src`将被限制为仅能使用`self`，即去除了`unsafe-eval`这项关键能力。导致`eval`相关的动态生成代码能力全部禁用。
- `background`模块 从 `Background Page` 替换为 `ServiceWorker` 而引发的一系列问题（参考本文下一章节）。
- 一批旧的API将被正式废弃，例如：[chrome.runtime.getURL vs. chrome.extension.getURL](https://stackoverflow.com/questions/32344868/chrome-runtime-geturl-vs-chrome-extension-geturl)

如果V2的某些能力是插件运行必需的，那么有以下几种方式可以尝试：

- 方案一：阻止用户更新chrome，保持在115版本以下即可（“幸好”，国内环境下chrome一般不会自动更新）
- 方案二：放弃chrome，引导用户使用其他浏览器。
  + firefox 也将升级到V3，但是一些细节会做得与chrome不同，可以仔细研究一下
  + 360浏览器，实质上是chromium87等低版本的封装，目前（以及可见的未来）仅支持V2
- 方案三：自己打包客户端App（electron.js），甚至浏览器（基于chromium），这样你可以为所欲为，你的业务逻辑将不再受到“插件”的规则的制约。

### ServiceWorker的代码兼容处理

在 Service Worker 中，`window`这个全局变量是不存在的，解决方案是用`typeof`去判断，这个语法非常安全。参考["window is not defined" service worker](https://stackoverflow.com/questions/49664665/window-is-not-defined-service-worker)

`XMLHttpRequest`这个东西也不存在，如果为了代码一致性坚持要在 ServiceWorker 中使用`axios`，则使用[@vespaiach/axios-fetch-adapter](https://www.npmjs.com/package/@vespaiach/axios-fetch-adapter)这个库来做垫片处理。

```js
import fetchAdapter from '@vespaiach/axios-fetch-adapter';

export const client = axios.create({
  adapter: typeof XMLHttpRequest === 'undefined' ? fetchAdapter : undefined,
});
```

## chrome store 插件上架流程

参考[谷歌拓展商店](https://developer.chrome.com/docs/webstore/)

首先[注册开发者账号](https://developer.chrome.com/docs/webstore/register/)，需要一个邮箱，然后交5美元注册费。

在[控制台-账号界面](https://chrome.google.com/webstore/devconsole)中完善账号基本信息，包括验证电子邮箱地址。

在[控制台](https://chrome.google.com/webstore/devconsole)中点击『+ New Item』按钮，将预先准备好的zip文件拖入上传框内。

上传完毕后，也就自动创建了一个“应用”（术语叫`Item`）。在发布之前必须补充基本信息（store listing）、隐私政策（privacy practices）

> 可以了解一下隐私政策生成网站[freeprivacypolicy](https://www.freeprivacypolicy.com/blog/chrome-apps-extensions-privacy-policy) 生成出来的是英文版，但是其中主要内容都是常见的格式条款。实在不行的话，网上随便搜一下也能找到中文版的隐私政策文本供参考。

当所有条件都满足之后即可进入下一步。（后续步骤略）

## 关于360浏览器

虽然我很讨厌360，一方面是它所谓的“安全卫士”做出了许多流氓行为却依然敢声称自己“安全”这点让我非常鄙视；另一方面，从更单纯的技术上来说，它依然是chromium内核的封装，而且从使用体验上来说封装质量也不算高，因此它也不是什么值得我尊敬的技术产品。

但不得不承认，在国内chrome缺席的情况下，360浏览器可以说成功地扮演好了替代者的角色，在国内的市场份额非常高。（全靠同行衬托是吧）

在『浏览器插件』这个领域，360也有着一套与chrome几乎一致的商店体系（[360浏览器应用开放平台](https://open.chrome.360.cn/)）。它[支持的是V2版本的插件规范](https://open.chrome.360.cn/extension_dev/manifest.html)，虽然在实现细节上与chrome有一些细微的差异（我们项目就踩过坑），但大差不差吧。

关于『V3强制升级事件』这场风波，即使在离deadline已经不远的现在，我似乎依然没有看到360浏览器有打算跟进的计划。也许是它目前内核版本依然仅仅只有87的原因吧，离115还早着呢。
