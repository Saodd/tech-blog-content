```yaml lw-blog-meta
title: "技术月刊：2022年9月"
date: "2022-09-15"
brev: "做得多了，写得少了"
tags: ["技术月刊"]
```

## js复制到剪贴板

旧的方式是使用`.execCommand()`，但这种方式已经[不推荐使用了](https://developer.mozilla.org/en-US/docs/Web/API/Document/execCommand)

新的方式是使用 [Clipboard](https://developer.mozilla.org/en-US/docs/Web/API/Clipboard)，兼容性[还不错](https://caniuse.com/?search=navigator.clipboard)

但是要特别注意，这个API仅限在HTTPS（或者localhost）环境下使用，因此可以考虑[写一个wrapper](https://stackoverflow.com/a/65996386/12159549)来兼容开发与线上环境。

## React Suspense

参考阅读： [深度理解 React Suspense](https://segmentfault.com/a/1190000018386975)

简而言之，`Suspense` 是通过捕获 `throw Promise` 来得知当前异步加载状态的。

简单观察了一下，react在 v16 v17 v18 三个版本下的行为都有比较大的不同，因此（在没有深入研究的情况下）我做不到尝试自己实现一个Suspense。

如果真的想深入研究的话，我认为只能直接去看react的源码进行调试，或者自己从零开始撸react框架。不能依赖react现有的框架能力，因为它底层做了很多事情，暴露的API的能力可能也有限，不能满足我们深入研究的需求。

话说回来，虽然我在多个项目中都有用过，但其实我自己还真的从未对 `Suspense` 这个东西感兴趣过，因为它的适用场景太有限了（仅限于配合 `lazy()` 使用），而如果在业务代码中有这种需求，完全可以用 `useEffect` 或者 `useLayoutEffect` 来实现，更简单更清晰。

## webpack同时构建多个target

参考：[Multiple targets #1122](https://github.com/webpack/webpack/issues/1122)

最佳实践是在`webpack.config.js`中导出一个数组：

```js
module.exports = [
    {
        name: "client",
        entry: "/path/to/client/entry.js",
        target: "browser",
    },
    {
        name: "server",
        entry: "/path/to/server/entry.js",
        target: "node",
    },
];
```

至于两者之间共同的部分，则借助`webpack-merge`来帮助处理。

## webpack直接引入外部script

之前我的做法是手动在`index.html`中添加script标签，这种方式其实维护起来特别累。

后来我才突然发现，在`webpack 5`中，已经加入了对这种方式直接的支持。示例用法：

```js
module.exports = {
    // ...
    externalsType: 'script',
    externals: {
        react: ['https://cdn.jsdelivr.net/npm/react@17.0.2/umd/react.production.min.js', 'React'],
        'react-dom': ['https://cdn.jsdelivr.net/npm/react-dom@17.0.2/umd/react-dom.production.min.js', 'ReactDOM'],
    },
};
```

重点关注配置中的`externalsType`字段；它有很多个值可以选择，另外还有一个有趣的选项是`promise`，有兴趣可以自行了解。

**但是**，通过这种方式引入的script，是不能保证顺序的。（其实想一想也合理，例如已知`antd`依赖`react`，但是webpack既然已经不去分析`antd`里面的内容了，它也就自然不知道这两个库的依赖顺序了。）

话说回来，我折腾了几年的cdn配置，一直没有找到一个十全十美的解决方案。『薅CDN的羊毛』这个事看起来美好，但是落到实践中时，坑太多了，弊大于利。因此我现在很多时候都是不想用这个功能的。

## WeakSet解决重复问题

最近做了这么一个事情，我侵入了`XMLHttpRequest`内部，截取响应的数据内容，代码如下所示。

```ts
window.XMLHttpRequest.prototype.open = function (this: XMLHttpRequest): void {
  this.addEventListener('readystatechange', (ev) => {
    if (this.readyState === 4 && this.status === 200) {
      console.log(this.responseText);
    }
  });
}
```

但是在实际运行过程中，我发现在某些情况下（例如`301`重定向），那么这个`open()`方法会被执行两次，于是我插入的代码也会被执行两次，造成一些麻烦。

于是我想到借助一个`WeakSet`，来排除重复的请求对象。关键代码如下：

```ts
const reqSet = new WeakSet<XMLHttpRequest>();
window.XMLHttpRequest.prototype.open = function(){
  if (reqSet.has(this)) return
  reqSet.add(this)
  // ...
}
```

这样就解决了问题。

之前背八股的时候，一直在想`WeakMap`和`WeakSet`到底有什么样的应用场景；如今总算是用上了，而且很及时有效地解决了问题。这可以算是“厚积薄发”了吧：）

## XMLHttpRequest的error事件

在解决上面的问题的时候，引申出另一个问题。

我一开始其实是监听的`addEventListener('load')`，那么疑问就来了，当发生异常的时候（即`error`事件触发的时候），`load`到底是会执行呢，还是不执行呢？

参考这个 [问题](https://stackoverflow.com/questions/6783053/xmlhttprequest-is-always-calling-load-event-listener-even-when-response-has-e)

先抛结论：当抛出`error`事件的时候，`load`不会触发。

但是这个`error`的时机可能会跟我们预期的稍有区别。

如果请求被响应了例如`400`、`500`的状态码时，在浏览器看来这个请求本身其实是执行正常的，因此不会抛出异常，会正常触发`load`事件。只有请求本身出了问题，例如被跨域政策拦截下来了，此时才会有`error`事件抛出。

`Fetch`的行为逻辑也是相同的。

之前我在[《XMLHttpRequest 与 Fetch》](../2021/211029-XMLHttpRequest-Fetch.md)中说遇到`400`“会”抛出异常的，是`axios`附加的逻辑，而`XMLHttpRequest`本身并不会抛。

## Proxy劫持construct

```js
// 被劫持的类
class A {}

const Fake_A = new Proxy(A, {
  construct(target, argumentsList, newTarget) {
    const a = Reflect.construct(target, argumentsList, newTarget);
    // 这里可以拿A类的对象a来做事情
    console.log('我劫持啦', target === A, a);
    return a;
  },
});

const a = new Fake_A()  // 运行看看
```

## JSON遇到undefined

在js的世界里，表达“空”的概念，有两个东西：`null`和`undefined`。但是在JSON规范里，只有`null`这一个。

然而，`JSON.stringify()`对这个问题的处理是有坑的：

```js
JSON.stringify(undefined)  // 'undefined'
JSON.stringify({ a:undefined })  // '{}'
```

`undefined`居然会被JSON反序列化成`'undefined'`！这个行为导致了JSON居然不能安全地将序列化地内容进行反序列化！（这个用术语来说叫什么来着……）

```js
const a = undefined
JSON.parse(JSON.stringify(a))  // boooooooom!!
```

对上面问题的解决方案是，多用一个对象包装一下：

```ts
const a: any;
JSON.parse(JSON.stringify({ data: a})).data
```

## 闲谈：“松弛感”

先看V2EX上的一篇帖子：[现在年轻人这么刚的么？](https://www.v2ex.com/t/877840)

光是看楼主贴出的聊天记录，我就已经充满了不适感了；再一看下面的回复列表，用楼主的话来说，我也再次体会到了“人与人认知差异还是蛮大的”。

我想我似乎产生了一种与鲁迅先生当年相同的感觉：“学医已经救不了中国了”。当然这句话肯定也是一时气话，严谨地说，放到V2EX这儿应该是：“学编程也救不了一部分中国人了”。

其实我一直坚信，一个优秀的程序员，他也自然会是一个优秀的社会人。因为他需要终生学习（否则会被淘汰）、勇于探索（业务迭代太快），他要接受自己的不完美（再牛的大佬也会写BUG），他既要全局思考（架构设计）还要考虑每个细节（边界条件），要准备预案（应对oncall），还经常要做各种妥协（好or快）；在这样的环境下，最后脱颖而出的人自然会具有许多优秀品质。

但实际上呢，一个典型的程序员，他可能如上面所描述的，在日常工作中表现比较优秀；但是，他的精神追求似乎并不会从这份职业中受益。相反，甚至可能有害——就如上面帖子中那些，在我看来是“缺乏信仰”的一部分人。

前几天有天中午，我跟同事出去吃饭。两位同事一直在说公司哪里哪里不好、最近热点新闻又如何糟糕、股市又跌了、炒币又亏了；而我却画风完全不同，我在说我们项目最近做了什么新的技术方案、上个周末我下厨做了好吃的、附近哪家按摩店的小姐姐技术好。

当时我自己都没有意识到，我显得像一股清流。回去的路上，他们说他们“都还在精神内耗”，而我“已经无敌了”，我才反应过来，最近的我似乎已经将这种“松弛感”融入了自己的灵魂中，随时都能散发出乐观平和的正能量来。

接下来，我本来打算讲讲我是如何培养我的正能量的。但是等我打了一堆文字后发现，我想说的，跟随处可见的鸡汤文学似乎也没有什么区别。在这个互联网发达年代，想要鸡汤那是再容易不过了，可是能把鸡汤真正的消化吸收、形成自己的能量的人，却非常少见。说也是白说，所以我把一大段文字都删了。

我最近挺喜欢“跟自己和解”这个说法。不仅要跟自己和解，也要跟这个世界和解——必须承认，无论是主观的自己还是客观的世界，都是存在许多缺陷的；在这样的认知基础上，再抱着“去改善”的心态重新出发，哪怕是一点点的改善也值得感恩，到那时，眼前的景色一定会变得大不一样吧。这，我想就是所谓的『松弛感』了。

## 闲谈：心里缺了一块的人

跟我前后相恋8年的前女友，要结婚了。

消息是从好兄弟那里传来的。毕竟我跟她已经两年多没有任何来往了，哪怕是朋友圈的点赞也没有。

一时间我的思绪翻涌，有太多话想说。可我不能说，我也不会去说。这些语句翻腾了一阵子之后，被我埋进了心底，就像浮空大陆边缘剥落的悬崖，坠入了无底深渊，我的心好像也缺了一块。

我以为我是已经完全放下了的。毕竟分手的时候两人都很平静，仿佛那是当时早已在心中达成好的默契。

可真到了这一刻的时候，我是有点难过的。

哦不，可能不仅仅是『有点』，可能是『很』。

因为不知不觉间，我已经泪如雨下。

要不是周日的下午在公司没有第二个人的存在，我这副狼狈模样可能就要给我丢人了。

但其实我也不知道我在难过些什么。我一边用纸巾堵住眼角，一边思考着。然后我发现，我脑海里挥之不去的是她在婚纱照上的笑容。

她怎么笑得那么灿烂。

跟我在一起的时候，我有让她那样笑过吗？——嗯，那肯定有的，而且应该有很多，只是很多记忆被我封存起来、一时产生了错觉罢了。

但她的笑容依然像一根刺一样扎在我的软肋上。我仿佛看见她的笑容逐渐变得扭曲，然后嘲笑的声音从四面八方传来：“你怎么就没有给她更好的”。

在大部分同龄人都结婚生子、过得像模像样的时候，我却还在苦练着一人敌之术、不修边幅地与世界战斗着。用现在耳机中正在播放的歌词来形容，就叫『长枪刺破云霞 / 放下一生牵挂 / 望着寒月如牙 / 孤身纵马 / 生死无话』。

从这个角度来说，我的性格是有巨大缺陷的，我很难委屈自己去满足那些世俗的期待。

我会感到难过，也只是恨自己不争气。

一曲《年少有为》，不是专门为她而弹，此时放在这里却也再合适不过。

算了，何必掩饰呢。

壬寅年八月廿三，宜买醉。
