```yaml lw-blog-meta
title: "前端架构：微前端"
date: "2025-02-10"
brev: "技术KPI考核必备"
tags: ["前端"]
```

## 背景

说起“微前端”这个概念，我印象中大概是三、四年前火过一阵子，但由于当时后端的“微服务”概念先火了几年，再加上当时我还不是专职的前端开发，因此我当时对这个概念是有些嗤之以鼻的，简单了解之后觉得并不实用，就没有深究了。

几年后的今天，最近在招聘过程中发现，候选人的简历中提到“微前端”经验的频率似乎有比较明显的提升。因此也提醒了我，是该补补课了。

## 简而言之

一句话概括，微前端主要用来解决——“多技术栈并存问题”。

对应于现实中，大概有两种情况，一是有使用旧框架的历史项目实在维护不动了，想要切换新框架；二是对于超大型项目，为了并行开发，支持各团队以更加独立的方式各自开发，然后借助微前端技术糅合成一个巨型应用。

## 与后端对比

对比来看，其实“微前端”与“微服务”的思考逻辑几乎是相同的。

微服务也是为了解决巨型后端应用的难题，拆分成多个服务各自独立运行、维护，互相之间通过基座——反向代理（Nginx等）和 远程调用（HTTP、gRPC等）进行协作，共同组成一个巨型应用。

他们产生的代价也是相似的。拆分之后会提升开发和维护的成本，特别是如何有效复用代码又成为新的问题。

## 微前端的几种实现方案

> 参考阅读：[微前端的那些事儿](https://github.com/phodal/microfrontends)，前半部分讲得不错，通俗易懂。

第一种，也是最简单的方案，借助后端反向代理，依据URL来提供不同的微应用。这就是最简单也最常见的“微前端”方案，所以从后端工程师的角度来看，说它是一种噱头确实不算过分。但是这种方案有一些缺陷，比如切换应用需要浏览器页面导航、不同子应用之间状态难以共享等 ~~（还有太简单了很难算成KPI啊）~~ ，虽然我个人觉得问题不大，但是如果产品对用户体验有苛刻的要求、或者前端工程师想搞点KPI的话，这种方案还是显得简单原始了。

第二种，用`iframe`来改进，也是最简单也最古老的方案之一。它可以稍微缓解页面导航问题（子页面导航不算导航？），但是隔离得过于充分又会进一步加剧应用之间的通信难度。

第三种，随着`react`等现代化框架和`webpack`等工程化工具的发展，我们对web应用的控制力越来越精细，因此在同一个页面框架运行时中运行多个应用也变得可能。因此就有了以`single-spa`为基础的一些框架的诞生，典型代表是`qiankun`。

第四种，浏览器、或者说web标准本身也在努力，提出了`WebComponent`方案也能实现微前端的能力，或者说它支持的更加细致，它实际上实现了一种“微组件”能力。但它的问题是与react等主流框架风格不符，选择这条路线往往需要对原代码做根本性的侵入式改造。

在多种因素考虑下，第三种方案，以`qiankun`为代表的微前端框架逐渐成为了现在很多大型项目的选择。

## qiankun的基本原理

它把整个巨型应用划分为1个`基座`和若干个`子应用`。

`基座`放在应用主入口的html中，它往往只包含少量通用UI（Header、Menu等），它运行之后根据条件判断选择当前所需的`子应用`，然后加载子应用的js、css等资源，将子应用渲染在指定的节点上（比如页面主体main、content部分）。

既然这些应用都运行在同一套HTML、JS、CSS运行时中，那么最重要的问题就在于如何做隔离了。

JS隔离主要需要关注全局变量，因此用到`windowProxy`技术。css隔离主要借助webpack运行时来修改类名（添加前缀），或者借助`ShadowDOM`也可以做更彻底的隔离。除此之外还有相对url路径等一些小问题，都有对应的办法可以解决。

## qiankun的基本用法

使用qiankun框架，不需要太多改造就能将现有的普通SPA应用改造为微前端化的SPA应用。

`基座`应用的改造重点在于，它需要知道所有子应用的情况，需要给它一个对象列表。

示例代码：

```ts
import { registerMicroApps, start } from 'qiankun';

registerMicroApps([ // 把信息注册给qiankun框架
  {  // 第1个子应用的信息
    name: 'react app',
    entry: '//localhost:7100',
    container: '#yourContainer',
    activeRule: '/yourActiveRule',
  },
  { // 第2个子应用的信息
    name: 'vue app',
    entry: { scripts: ['//localhost:7100/main.js'] },
    container: '#yourContainer2',
    activeRule: '/yourActiveRule2',
  },
]);

start();  // qiankun框架开始运行，根据条件来加载所需的子应用
```

以上JS，再加上一个HTML文件，几乎就是基座应用的全部代码了，甚至可以不需要webpack来打包。

`子应用`所需改造也很少。现代web渲染框架都提供了`unmount`能力，以`react-18`为例，核心API为：

```ts
const root = ReactDOM.createRoot();
root.render();  // 加载

root.unmount();  // 卸载
```

子应用各自分别以`umd`格式进行打包。加载时，首先判断是否是qiankun框架环境，如果是，就给qiankun框架暴露`mount`和`unmount`等生命周期函数；如果不是，那就执行一次自己的`mount`函数（即子应用可以脱离基座独立运行）。

示例代码：

```tsx
let root: ReactDOM.Root;

export async function mount() {
  root = ReactDOM.createRoot(document.getElementById("root"));
  root.render(
    <BrowserRouter basename={window.__POWERED_BY_QIANKUN__ ? "/app1" : "/"}>
        <App />
    </BrowserRouter>,
  );
}
export async function unmount(props) {
    root.unmount();
}

if (!window.__POWERED_BY_QIANKUN__) {
    mount();
}
```

子应用的工程代码可以与基座和其他子应用完全独立，也可以用monorepo等方案放在一起。

详细用法请参阅[官方文档](https://qiankun.umijs.org/guide/getting-started)。

## qiankun可能带来的问题

“微前端”三个字，顾名思义，讲究的是如何拆分前端应用，因此，它带来的缺点就在于如何组合多个应用，我举一些例子：

第一，最明显的是代码复用问题。为了视觉交互统一而建立内部的组件库是很常见的事情，但是在微前端架构下这很容易变成一件棘手的问题；哪怕不涉及UI，仅仅只是一些JS实现的业务逻辑代码，只要依赖了第三方库，就都需要考虑不同子应用之间的兼容问题。

第二，多应用之间的协调问题。多个子应用有各自的版本和开发进度，那么在开发、测试、甚至线上环境中，如何协调各应用、选择正确的对应版本将会变得令人困惑，很容易产生一些难以覆盖的测试点或者难以复现的BUG。

第三，拆分隔离很难做到彻底。在实际工作中总是计划赶不上变化，随着业务发展，我们总是难免会遇到各种稀奇古怪的需求，一个需求同时涉及修改基座和多个子应用的情况并不少见，这样原本一个人半天的简单工作量放在微前端架构下也许就变成了三个人扯皮一天了。

第四，公共代码负责人问题。也许在一开始设计架构时会有强势的架构师来主导，公共代码库能够落实到人，可是一段时间后，特别是经过人事变动后，这类基建代码可能陷入责任纠纷或者是集体摆烂屎山拉屎的糟糕状况。

因此，我认为微前端架构仅适用于大公司的巨型应用，小公司一般是不应该动用这把牛刀的。

## WebComponent简介

> 参考阅读：[WebComponent是个什么东西？](https://juejin.cn/post/6956206468316004382)，讲得不错，通俗易懂。

WebComponent其实是由一系列相关API共同实现的、我愿称之为“微组件”的技术方案。

其最核心的API是`customElements.define`，用白话说，就是自定义一个HTML元素标签并注册，之后就可以快速复用这个标签。示例代码如下：

```ts
window.customElements.define('myButton', MyButtonClass)
```

上述代码中，`MyButtonClass`用于实现这个组件的内部逻辑，示例代码如下：

```html
<html lang="en">
<body>
    <myButton></myButton>
</body>
```

```ts
class MyButtonClass extends HTMLElement {
  constructor() {
    super();
    const div = document.createElement('div');
    div.innerText = 'hello, world!';
    this.appendChild(div);
  }
}
window.customElements.define('myButton', MyButtonClass)
```

上面是我们编写的一份HTML和一份JS代码，在运行之后，页面会被渲染成如下结果：

```html
<html lang="en">
<body>
    <myButton><div>hello, world!</div></myButton>
</body>
```

也就是说，在原生的div、span、h1等标签之外，我们额外定义了一个名叫"myButton"的标签，并且可以在页面中任意使用这个新的标签。

新的自定义标签（`customElements`），可以通过标签属性（`props`）和事件回调函数来实现数据流动，可以利用原生的`HTMLTemplateElement`组织代码，可以利用`ShadowDOM`来实现代码隔离。从理论上来说是能够替代现代框架所提供的能力的。

但显然它用起来并不方便，其元素内部相当于是封装成了一个黑盒，操作手感类似`video`标签；且与当前主流的三大框架写法不同，不能直接快速改；同时还有Typescript类型兼容性等工程上的难题。

总之，我认为这项技术的实用价值是弊大于利的。不过这种实现思路给`Vue`框架提供了重要的灵感，也可以算是以精神继承的形式存活下来了。
