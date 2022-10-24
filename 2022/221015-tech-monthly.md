```yaml lw-blog-meta
title: "技术月刊：2022年10月"
date: "2022-10-15"
brev: "这个月主要是解决综合问题"
tags: ["技术月刊"]
description: "babel可选链, 受控组件, figma导出SVG, PC客户端icon, publicPath, 插件判断环境, gzip压缩效果"
keywords: "技术月刊"
```

## OPPO Sans

官方网站：[OPPO Sans，用文字探索科技美感](https://www.coloros.com/index/newsDetail?id=72)

偶然看见抖音有产品使用了这个OPPO的字体，仔细一看，似乎这个字体做得还挺认真的。

全免费商用，如果看腻了思源和阿里普惠可以考虑替代一下。

## babel错误编译可选链操作符

『[可选链操作符](https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Operators/Optional_chaining)』，即`?.`操作符。

我当前的`webpack`配置：

```js
{
  loader: 'babel-loader',
  options: {
    presets: ['@babel/preset-typescript'],
    plugins: [],
  },
}
```

我有一段这样的代码，看起来平平无奇：

```ts
const v = xxx.view;
[].push({
  url: v?.getURL(),
});
```

```js
// 编译产物：
const v = yyy_WEBPACK_IMPORTED_MODULE_8__.xxx.view;
[].push({
  url: v?.getURL(),
});
```

上面看起来一切正常，但是像下面这样写就不行：

```ts
[].push({
  url: xxx.view?.getURL(),
});
```

```js
// 编译产物：
[].push({
    url: yyy_WEBPACK_IMPORTED_MODULE_8__.xxx.view.getURL(),
});
```

仔细看，它少了那个关键的`?`，导致程序没有按照预期的逻辑运行，崩溃了。

这个现象大致总结为：当可选链太长的时候（或者遇到 WEBPACK MODULE 的时候？），可选链会被错误地丢弃掉。

一种解决方案是增加`@babel/preset-env`，但是会遇到"regeneratorruntime is not defined"的错误，为此我们还要为项目引入`babel-polyfill`（参考：[Babel 6 regeneratorRuntime is not defined](https://stackoverflow.com/questions/33527653/babel-6-regeneratorruntime-is-not-defined)）。可行，但是我觉得代价比较大，因为我的运行环境是固定的，我并不需要这层额外的polyfill 。

另一种方案，只增加一个babel插件，专门用来解决这个可选链的问题，最终babel配置如下：

```js
{
  loader: 'babel-loader',
  options: {
    presets: ['@babel/preset-typescript'],
    plugins: ['@babel/plugin-proposal-optional-chaining'],
  },
}
```

文档：

- [@babel/plugin-proposal-optional-chaining](https://babeljs.io/docs/en/babel-plugin-proposal-optional-chaining)
- [@babel/plugin-proposal-nullish-coalescing-operator](https://babeljs.io/docs/en/babel-plugin-proposal-nullish-coalescing-operator)

## React受控组件

关于『受控组件』的描述，参考：[Controlled Components](https://reactjs.org/docs/forms.html#controlled-components)

这个概念我一直是很清楚的，但是有一天在写代码的时候，莫名其妙就给我警告："A component is changing a controlled input to be uncontrolled. "

我看了半天，确认我一直用的是受控用法，为啥还会给我警告？？

后来经过仔细排查，发现传入的value的值在某些情况下会是`undefined`，这个值会让React猜测它变成了『非受控组件』，因此提出警告。（当然，会传入`undefined`也确实是我代码有不严谨的地方。）

## Firefox不能使用Figma的导出svg功能

这是偶然发现的一个小细节。我目前的项目，遇到设计那边提供的icon（向量图形）时，我们是直接导出SVG使用的。

我在Mac的电脑上使用Figma，可以直接在图层上右键单击，就能直接把svg代码赋值到剪贴板，直接拷贝到代码项目里去，非常方便。

但是我突然使用Windows后，就发现很神奇，两边的右键菜单居然都不一样。

一开始我以为是Figma有意限制Windows平台的能力（虽然没有任何理由能解释），或者在做什么灰度测试命中了我。可是随后我发现，在Windows上，我可以通过另一个菜单手动导出SVG，能力是一样的，只不过形式上换成了“下载SVG文件”。

于是我隐约感觉是Figma检测了剪贴板相关的能力，然后在他家论坛上看到了[这篇文章](https://forum.figma.com/t/couldnt-copy-svg-or-png/6463)，才理解到原来是Firefox的问题，而不是Windows的问题。

而至于为什么我这里用的是Firefox而不是Chrome，则要扯出另一个槽点：我现在这台电脑上的Chrome打开Figma会导致页面崩溃，大概原因是WebGL没有启动，或者是驱动因素导致WebGL能力没有正确被Figma识别。这个问题我暂时还没有解决，替代方案就是用Firefox。

## pc客户端的icon

MacOS 对icon的处理比较优秀，直接扔给他一个大尺寸（`512x`以上）的`.png`，然后就不用管了，安装后无论大图标、小图标都显示效果良好。

Windows 就蛋疼了，扔一个大尺寸图片过去，安装后在小图标的场景下就会显示出锯齿来。

因此需要一些 [工具](https://www.aconvert.com/cn/icon/png-to-ico/) ，来将`.png`转化为`.ico`。

然后在开发阶段又遇到另一个坑，是windows资源管理器 它对文件的图标是有缓存的，缓存不更新，我们重新构建之后的图标也不会显示出最新的来。解决方案参考：[Icon Cache in Windows 11/10](https://www.thewindowsclub.com/rebuild-icon-clear-thumbnail-cache-windows-10#:~:text=The%20Icon%20Cache%20or%20IconCache,Windows%20draw%20the%20icons%20faster.)

## webpack的publicPath

它的原理，就只是简简单单地地加在路径前面而已。

示例：

```js
// webpack.js
module.exports = {
    output: {
        publicPath: 'build/'
    }
}
```

```js
// 编译后：
// ...
__webpack_require__.p = "build/";
// ...
module.exports = __webpack_require__.p + "0123456789abcdef0000.html";
```

## chrome插件判断运行环境

在优化架构，做代码封装的时候，我产生这样的需求：我需要判断当前的运行时是`content`环境还是`background`环境。

参考答案：[Can js code in chrome extension detect that it's executed as content script?](https://stackoverflow.com/questions/16267668/can-js-code-in-chrome-extension-detect-that-its-executed-as-content-script)

```js
// 不可以用：if (chrome.extension)
if (location.protocol == 'chrome-extension:') {
    if (chrome.extension.getBackgroundPage() === window) {
        // 在 background 环境
    } else {
        // 在 popup/options 环境
    }
} else {
    // 在 contnent 环境
}
```

## gzip压缩效果探究

在配置Nginx的时候，为了性能考虑，往往都会设置gzip进行压缩。

可是，有天我偶然发现，gzip对图片的压缩效果几乎为零。

那么，gzip的压缩效果到底如何？

我做了一个简单的实验：随意选取我的电脑中的不同种类的文件每种一个，使用Golang内置的`"compress/gzip"`包进行压缩，压缩等级`9`即最大压缩，得到结果：

| filename | Raw-Size | Gzipped-Size | Compression Rate |
|----------|---------:|-------------:|-----------------:|
| .woff2   |    27964 |        27997 |           -0.12% |
| .webp    |   182568 |       182651 |           -0.05% |
| .jpg     |   217231 |       216765 |            0.21% |
| .gif     |   679144 |       676537 |            0.38% |
| .png     |   407788 |       404865 |            0.72% |
| .woff    |   522388 |       516978 |            1.04% |
| .mov     |  3047334 |      2952818 |            3.10% |
| .mp4     |  3273341 |      3122262 |            4.62% |
| .otf     |  8390148 |      7409385 |           11.69% |
| .ttf     | 36144992 |     20539387 |           43.18% |
| .html    |     1221 |          646 |           47.09% |
| .ico     |   226234 |        89531 |           60.43% |
| .svg     |     7486 |         2469 |           67.02% |
| .css     |     2768 |          907 |           67.23% |
| .js      |    19133 |         5658 |           70.43% |
| .json    |    17878 |         2174 |           87.84% |

从上表可以看出，gzip能够有效压缩的，大体上分为两类。

- 一类是 json, js, css, html, svg 这类文本文件；
- 一类是 ico, ttf, otf 这类虽然是二进制，但是含有许多重复内容的文件。

至于其他的，例如 mp4, jpg, png 等常见格式，它们协议本身就已经含有压缩算法了，不需要再使用gzip再次压缩了。甚至对于 webp, woff2 这种现代化的协议，gzip纯粹就是添乱的。

gzip会带来额外的cpu开销，因此对于压缩率很低的文件来说，还是别用gzip了。

## 技术选型

我这个小小的个人网站，麻雀虽小五脏俱全，至少 MySQL, Mongo, Golang, JS/TS 等多种技术都有在使用。

### 数据库：MySQL vs Mongo

Mongo是我最初的数据库，也是我最熟悉的数据库。但说是熟悉，不如说是熟练，其实我对它底层的一些机制是还不太清楚的。相比之下，MySQL的八股文太多了，整个业界都把MySQL研究得底朝天了，我很轻松就能学习到MySQL各种底层机制。而对Mongo底层机制不熟悉，导致我在运维操作上总是没有信心。

除了运维以及学习知识上的考虑，Mongo对事务(ACID)支持的不完善，也是我下定决心要使用MySQL的重要理由。

于是我上了MySQL，两个数据库同时在线上运行，负责不同的业务模块。

然后使用MySQL一段时间后，我逐渐发现它的槽点更多：

1. SQL纯文本协议，执行效率低；
2. 在Go语言中缺乏有效的ORM框架，手写SQL很容易出问题；
3. 受范式约束，字段不可再分，没有数组、对象类型导致有时在业务上处理起来比较麻烦；
4. 缺少一些现代化的便利能力，例如正则表达式、文件存储（小文件、大文件）等；
5. 固定库表结构(DDL)是一把双刃剑，有好有坏；

所以我现在的态度是：如果没有强烈的ACID需求，我一定会优先选择Mongo；即使有关系型需求，我可能也会考虑调研其他的现代化的RDBMS，例如[TiDB](https://www.pingcap.com/case-study/embracing-newsql-why-we-chose-tidb-over-mongodb-and-mysql/)

### 开发语言：Golang vs TS

JS由于其强大的动态语言特性，（以及V8引擎强大的执行效率、文档标注系统、各种编译检查工具），再加上TS这种最顶级的类型系统的辅助，用TS来写业务，是真的顺滑得一批。

虽然JS生态有些槽点，开发过程依赖webpack等编译工具需要一定的熟练度，但是熟练之后的能力还是很强大的。

但是动态语言也有其致命弱点：类型并不是100%可靠的。运行时可能会出现一些意外。

因此再结合一门稳固的静态类型语言，一动一静，将会是绝配。这个特性非常重要，因为后端服务处于前端和数据库之间，在前端（JS）和数据库（Mongo）都是动态的情况下，在中间插入一个静态类型语言（例如Golang），直接就能保障整个业务架构的数据流通的健壮性，起到一种事半功倍的效果。

Golang是我的个人偏好，但是我也再难以找出其他能够胜任“静态”这个角色的语言了。虽然Go语言本身也有不少槽点啦，但是世界上没有银弹，Golang作为一门Web时代下诞生的现代化语言，它在语言特性上的保守的抉择，很好地满足了我的架构需求。

我也曾经想过，前后端使用相同的技术（Node.js），这样可以复用一部分代码，进一步优化开发效率。但我仔细思考之后，我依然认为选择一门静态类型语言是有必要的，因此我也就放弃了这样的选择。不是说Node.js不能用，而是从整体架构上来审视，这种动态类型的东西只适合放在一些边缘程序上，例如前端、客户端和后端某些边缘服务上，而核心部分依然需要一门严谨的静态语言来保证系统的健壮性。

> 参考阅读：Golang之父 rob pike 的关于Go语言设计的博客文章： [Less is exponentially more](https://commandcenter.blogspot.com/2012/06/less-is-exponentially-more.html)

## 闲谈：为什么我要翻译技术文章

我的博客文章中有一小部分是其他渠道英文文章的翻译，包括 MDN, The-Go-Blog 等。其中MDN有很多文章其实是有官方的简中版本的，为什么我还要自己翻译一遍？

这个问题，我也曾多次灵魂拷问自己。我的答案是：

1. 我已经真的很习惯于英文技术术语了，如果把术语换成中文，我反而“看不懂”了（例如典型的`git rebase`翻译为`变基`）。就算我不翻译，我自己看我也会看英文版的。
2. 绝大多数团队的技术文章，最官方、最原始、最权威的版本一定是英文版的，至于他们额外提供的中文等其他语言的版本，多半是由热心网友翻译的。翻译难免会有损耗，因此就质量来说，肯定是第一手的英文版的最好。
3. 我用“翻译”这个过程，强迫自己逐字逐句地去阅读原文，并且为了“信达雅”我还会额外参考很多相关资料；相比于我快速地浏览一遍，翻译一遍会让我理解非常深刻。
4. 维持自己的英语阅读能力，以及中文博客写作能力。
5. 作为笔记，供未来的我自己查阅、回忆。

因此对于我认为很重要、很干货的文章，我都会选择翻译一遍。不是要给谁看，是翻译给自己看。

## 闲谈：我的一天

记录一下最近我的生活。说难听点叫单调又重复，说好听点叫规律又充实，但我觉得它们都是有意义的，值得拿出来碎碎念一下。

早上6:30左右，醒来（因为感受到阳光），看看手机，想想今天没什么要紧事，（就算有什么要紧事那也）继续睡。

8:00~9:30期间，反复醒来、然后继续睡。

9:30左右，起床。把昨晚预先泡胀的豆子放进破壁机开始豆浆程序，然后去洗脸刷牙。如果9:00就清醒了的话，我会把衣服扔进洗衣机去洗。

9:40~10:00，弹钢琴，经过一晚的睡眠，昨晚练习时生涩的部分自然就变得流畅了，这个过程很像是“发酵”。

10:00左右，清洗豆浆机，晒衣服，换衣服，出门去公交车站。

公交只要2块钱，而且人少，空调巨给力，它让我顺利度过了杭州炎热的夏天。路上刷一下手游的体力。

10:30左右，到达公司。去茶水间拿一袋小面包，和自带的豆浆一起吃，一边看看今天的工作计划。

专注工作，直到11:30或者12:30，如果要给徒弟上课的话就可能拖到13:30，下楼吃饭。

吃饭地点就那么两三家轮换着，一家水煮肉（主要吃米饭），一家包子铺（吃4个包子很舒服），偶尔跟同事一起去其他店，或者偶尔心血来潮自己带一次便当。一边吃饭一边刷B站视频，一般看科普、人文、美食这类能够学习到技能的视频。

饭后继续专注工作，中途可能只起身两三次，直到18:00下班。

我坚持走路回家，因为这是全天唯一的运动量了。路线也已经很熟悉了，身体自己会走，大脑就天马行空、思考人生。

18:30左右到家，开始做晚饭。一般炒两个菜，一荤一素，用时40-50分钟，米饭煮好的时候菜也差不多炒好了。

一边吃饭一边上B站，追番或者看些娱乐向视频。

饭后20:00~24:00，弹琴，弹累了就刷视频，刷到弹钢琴的大佬的视频，自己又有了弹琴的力气，继续弹琴，如此循环。

有时会追番，一晚上的时间差不多可以把整季动画（12集左右）追完，追完顺手写个影评更新到自己的博客上。有时候需要网购，网购其实还挺花时间的，我的几乎一切生活物资都是网购的，包括生鲜食品。

24:00洗澡，又有了弹琴的力气，让我再弹会儿。哦对了，我的琴是电钢琴，不扰民。

1:00打开手游，把体力清空，把日常任务做完。如果遇到活动剧情，那就得多玩一会儿才能搞定。

1:30睡觉。争取能活到35岁吧哈哈哈。
