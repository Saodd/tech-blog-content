```yaml lw-blog-meta
title: "技术月刊：2022年8月"
date: "2022-08-15"
brev: "又是搞前端的一个月"
tags: ["技术月刊"]
```

## ShadowDOM与FontFace

之前说过ShadowDOM可以起到css样式隔离的作用。

但是`@font-face`不能在ShadowDOM中定义，必须在宿主根页面定义，然后ShadowDOM里面可以直接用，不会隔离。

参考阅读： [definitions in shadowRoot cannot be used within the shadowRoot](https://bugs.chromium.org/p/chromium/issues/detail?id=336876)

## React是如何处理事件的

- [React官方文档: Handling Events](https://reactjs.org/docs/handling-events.html)
- [React官方文档: SyntheticEvent](https://reactjs.org/docs/events.html)

核心思想：React有它自己封装的事件对象，术语叫做`SyntheticEvent`，它的主要功能是提供一种平台无关的实现（a cross-browser wrapper），并且提供了与原生事件对象几乎一致的接口。

具体到代码层面，React是有将这些事件类型都暴露出来的。规范的代码应该正确的引用React的事件类型而不是原生事件类型：

```ts
const handleCancel = useCallback((e: React.MouseEvent) => {
    e.stopPropagation();
    e.preventDefault();
    setVisible(false);
}, []);
```

一个细节值得一提，React做了一个对象池来管理`SyntheticEvent`。

还有，我们知道DOM事件分为『捕获阶段』和『冒泡阶段』，一般情况下我们处理的都是冒泡阶段的事件，而React同样支持捕获阶段的处理，在事件名后面加上Capture即可，例如`onClickCapture`

## 鼠标事件

鼠标事件其实种类非常多，随便就能列举一堆，（[区别](https://www.w3schools.com/jquery/tryit.asp?filename=tryjquery_event_mouseenter_mouseover)）：

- `mouseenter` 与 `mouseleave` ，只对它自己生效
- `mouseover` 与 `mouseout` ，对它自己以及子元素都生效
- `mousemove`
- `mousedown` 与 `mouseup`

但是作为一个资深前端，除了上述mouse相关事件之外，还需要了解更多。

首先很常见的需求，为了兼容移动端，我们必须处理`touch`事件家族，它们与`mouse`家族相似但是并不完全相同，特别是在处理拖动逻辑的时候差别非常明显。

甚至在移动端可能还会遇到`Pen`这种硬件设备。

因此就有了[Pointer](https://developer.mozilla.org/en-US/docs/Web/API/Pointer_events)事件家族，简而言之它们是硬件无关的设备事件。（[它的优势](https://usefulangle.com/post/27/javascript-advantage-of-using-pointer-events-over-mouse-touch-events)）

## offsetTop与offsetParent

我们经常用[offsetTop](https://developer.mozilla.org/en-US/docs/Web/API/HTMLElement/offsetTop)来获取『某个元素距离顶部的距离』，这个“顶部”，实际上指的是[offsetParent](https://developer.mozilla.org/en-US/docs/Web/API/HTMLElement/offsetParent)。

简单记忆 offsetParent 就是设置了`position`属性的元素，与相对位置的规则相似。

## getComputedStyle

使用`getComputedStyle`来查询一个DOM元素的『实际样式』，这在某些特殊场合非常有用。

引申出来的一个知识点：我们通过js的`style`传入的样式，与通过css匹配到的样式，是存在于两个不同的空间中的（也就是所谓的`CSSDOM`）。如果只是简单的`xxx.style.height`这样是取不到从css传入的样式的。

## Element与Node

参考阅读：[Difference between Node object and Element object?](https://stackoverflow.com/questions/9979172/difference-between-node-object-and-element-object)

最核心的区别，我想看这段源码就能够懂：

```ts
interface Element extends Node, Animatable, ChildNode, InnerHTML, NonDocumentTypeChildNode, ParentNode, Slottable {
    // ...
}
```

如上所示，`Element`是继承自`Node`的。

再用大白话解释一下，`Node`的作用是为了组成一个树形结构（即DOM树），而`Element`则是指具体的某种HTML元素（例如div,span,text等），这些Element元素有它们各自的属性和方法，但它们都能以`Node`的身份来组成一个树形结构。

## SASS中的AND符

> 调试方法：在webpack环境中，要给`css-loader`配置一个`options: { modules: false }`，这样生成出的css类名就不是哈希处理的，而是原样的。[参考](https://stackoverflow.com/questions/42601382/imported-scss-class-names-converted-to-hashes)

参考阅读：[The Sass Ampersand](https://css-tricks.com/the-sass-ampersand/)

`Ampersand`这个单词的意思是『and符号』，也就是`&`这个东西。上面这篇文章总结得很好了，我只说一下我感觉比较新鲜的部分：

我们平常是这样用`&`这个符号的：

```sass
.parent{
    &.disabled {}
}
```

这个时候很好理解，`&`就是代表当前类，相当于面向对象编程语言中的`self`或`this`这种关键字。

但你有没有想过下面的代码是什么意思？：

```sass
.parent{
    .child & {}
}
```

```css
/* 编译后：*/
.parent {}
.child .parent {}
```

注意，编译得到的不是`.parent .child .parent{}`，前面没有父类了，这个我感觉比较反直觉，算是一个坑。

然后，假如我需要得到`.child.parent{}`这样的产物（注意中间没有空格），我该怎么写scss代码？

```sass
.parent{
    .child#{&}
}
```

```css
/* 编译后：*/
.parent {}
.parent .child.parent {}
```

注意上面的`#{}`这个语法，它起到[插值](https://sass-lang.com/documentation/interpolation)的作用，里面可以插入普通的文本、`&`、或者sass变量（`$`）。

总之，`&`这个符号更像是一种字符串替换，它的语法跟我想象中差别还是蛮大的。

### CSSModules中的global

参考阅读：[CSS Modules 用法教程](https://www.ruanyifeng.com/blog/2016/06/css_modules.html)

在上一节中，我提到要关闭 CSS Modules 功能，这样css类名才不会被哈希化。

其实还有另一种方式：在CSSModules开启的情况下，使用`:global`语法，后面括号中的内容不会被哈希化，而是保留原样，也就相当于是成为了“全局样式”。示例：

```sass
.parent {
  :global(.haha) {}
}
```

```css
/* 编译后：*/
.ymvpXij_q9boz6_OTlYO {}
.ymvpXij_q9boz6_OTlYO .haha {}
```

与之相对的还有一个`:local`关键字，但我们正常是不需要用到它的。

另外，CSSModules中还有一个有意思的功能，叫`composes`，感觉比sass的`extends`更好用一些。

## 闲谈

### 杭州的早餐

在杭州呆了两年多了，我对这边饮食口味的印象最深的，不是『甜』，而是『咸』。

我记得我还在深圳的时候，那时候吃什么都很好吃，天南地北的美食，在深圳这个移民之都里都能得到完美的复刻。但是在杭州，完全不是这样的，无论是烤鱼、香锅、麻辣、甚至家常小炒，我越吃，却越觉得嘴巴淡，总感觉缺了一些什么味道。

这里似乎只有很纯粹的味道。大家都知道的爱吃甜，是纯的甜，爱吃咸，也是纯的咸。

『咸』给我印象最深刻的，要属这里的早餐。

我觉得那些不起眼的街头早餐店，是最能体现一座城市的饮食风情的。在我的家乡，我们小时候吃的是加了醋果子的拌粉、拌面，豆浆油条，还有几种从未在外地见过的油炸淀粉类食物。其中要属拌粉最是一绝，首先粉干必须是本地产的细粉丝（与米粉、米线不同），下水捞煮之后，晶莹剔透，与各种调味料混合后，调料能很好地吸附在细细的粉丝上，哧溜一把吸进嘴里，脆弹爽口，其中夹杂着几颗店家手作的醋果子（泡菜的一种），瞬间引爆味蕾。如果是拌面，面条也是本地产的特殊品种，我不清楚它的学名，只知道有点像碱水面吧，与粉干同样好吃。光是想想，我的口水就止不住地流。

杭州这里也有拌粉、拌面，不过与我的儿时记忆相比，这里首先粉、面的品种在口感上就远远落了下风。其次，这里的醋果子也是咸甜口味，完全没有能够刺激味蕾的酸味。此外，酱油加得又多，整个一碗端上来显得又黑又黄，其他调味料（香油之类的）似乎都没有，总结来说就是不管卖相还是味道都令我失望。（叹气）

最离谱的要数咸豆浆配油条的吃法。在我的饮食观念里，油条的酥香，与鲜甜的豆浆才最是绝配；尤其是把油条泡进甜豆浆里泡软再吃，既洗去了一部分多余的油脂，又吸收了豆浆那饱满的口感，1+1效果远远大于了2 。可是，杭州人怎么吃豆浆油条呢？首先他们偏好**咸豆腐脑**（咸豆花），首先他们这咸豆腐脑的吃法就不专业，没有香葱，也没有姜末，只有咸味，又是一碗又黑又黄……（哭笑不得）。然后他们一边吃着**咸**的豆腐脑，一边又往嘴里塞**咸**的油条，啊啊啊，我光是看着我都替他们觉得口渴。我也曾经抱着敬畏之心亲口尝试过这样的搭配，但结局依然让我非常失望。总之这种吃法我永远不可能理解吧。

不仅仅是街头小吃，就是自称文化遗产的xx连锁小吃，提供的口味也基本上是这种风格，也就是说，这种感觉不是我的偏见，而是一种事实。

我想，这，就是杭州人（浙江人）对于『咸』的追求。

### 平凡的伟大

作为一个奔三的人，我的同龄人大多数都是已经结婚生子了，动作快的，孩子都很大了，或者二胎三胎都有了。

每个新妈妈都会把孩子诞生的消息发在朋友圈，然后大家纷纷点赞。

我点赞的时候是什么心情呢。这个赞，是我为一个新的生命许下的祝福，也是对承担了生育重任的那位女性的巨大付出而表示的敬意。——即使可能很多妈妈并不是出于什么“为了人类”、“为了社会”这类伟大的理想抱负才生小孩的，更多的可能只是出于一种“人生的惯性”，即“到了这个阶段该生了那就生吧”这样一种似乎怎么看也算不上值得表扬的理由；但无论初心如何，她们的巨大付出依然值得尊敬。

我见过很多人逃避责任。逃避生育的责任、家庭的责任、工作的责任、甚至是对生活、对自己的责任。当然做出选择是每个人的自由，但是这些人也越发反衬出那些承担了责任的人的伟大。

别以为平平无奇的人生就一定与“伟大”无缘。越是经历、越是思考，越能从身边挖掘出这些不起眼的宝藏。
