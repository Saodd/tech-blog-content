```yaml lw-blog-meta
title: "技术月刊：2022年12月~2023年01月"
date: "2023-01-13"
brev: "一些案例分析和前端小细节"
tags: ["技术月刊"]
```

## 案例学习：ref.current的卸载时机

日常巡查时，我在我们故障收集平台上看到这样一条错误信息：

```text
Cannot read property 'querySelectorAll' of null
```

然后我追踪了一下源码，简化后的源码大概长这样：

```tsx
const ContentList = () => {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const callback = () => {
      Array.from(ref.current.querySelectorAll(`...`))
    };
    const ob = new ResizeObserver(callback);
    ob.observe(ref.current);
    return () => ob.disconnect();
  }, []);

  return (
    <div ref={ref}>
    </div>
  );
};
```

简单解释一下，上面的代码主要逻辑是，每当`ref`指向的元素尺寸变化时候，会重新执行回调函数。

你能看出上面的代码有什么问题吗？

为什么`ref.current`会变成`null`？为什么它变成`null`了却依然被`ResizeObserver`监听着？为什么`ob.disconnect()`没有按期望地运行？

根据[react官方文档 - Using the Effect Hook](https://reactjs.org/docs/hooks-effect.html#example-using-hooks)，它强调了：React保证了在effect运行之前DOM已经被更新了——这个规则对于`cleanup`函数依然有效。

具体对于上面的示例代码来说，在`ContentList`被卸载时、在运行`ob.disconnect()`之前，DOM会先被更新过，因此`callback`也先被再次触发，可触发之时`ref.current`已经变成`null`了，因此报错。幸运的是，这个小报错并不会影响程序的正常运行。

解决方式就是简单地判断一下`rec.current`是否存在即可。

## react: Callback Refs

参考官方文档: [Callback Refs](https://reactjs.org/docs/refs-and-the-dom.html)

我们在组件上使用`ref`属性，常见用法一般都是借助`React.useRef()`生成的东西来间接使用。

但实际上，`useRef()`并不是什么神奇的黑魔法。如果有需要的话，我们可以用回调函数来实现对元素引用的更精细的控制。基本用法如下：

```tsx
const ContentList = () => {
  const myRef = useCallback((elem: HTMLDivElement) => {
    console.log(elem);
  }, []);
  
  return (
    <div ref={myRef}></div>
  );
};
```

- 当组件挂载时，（`componentDidMount`之前，）`myRef`函数会接收当前组件对应的DOM元素对象。
- 当组件更新时：
  + 如果复用DOM元素，`myRef`不会被调用
  + 如果替换为了新的DOM元素，则`myRef`会被调用两次。第一次是旧的元素被卸载时触发的，参数为`null`；第二次时新元素挂载时触发的，参数为新的DOM元素对象。
- 当组件卸载时，`myRef`函数会接收一个`null`作为参数。

这里还有一个小细节，如果`myRef`函数本身发生变化时（例如不用`useCallback`而是用普通的内联函数），旧的函数先被参数为`null`调用，然后新的函数被参数为DOM元素调用，一共调用了两次。（虽然都是调用两次，但是其原理与DOM元素更新是有所不同的。）

总的来说，`ref`的执行逻辑与`useEffect`非常相似，只不过前者是针对 DOM 的，而后者是针对 Component 的。

> 思考题：Callback Refs 能不能用来解决前一章节描述的 ResizeObserver 的引用问题？

##  案例学习：react复用元素

说起『react的元素复用』这个话题，我相信大多数人都能立即想到`key`的相关用法。

最近我们有个需求，是要做一个交互式的`transition`动画效果，为了实现这个效果，也是需要考虑DOM复用问题的。然后昨天我在 Code-Review 的时候，发现我们同学写出的代码有些奇怪，代码简化后如下所示：

```tsx
const ContentList = () => {
  const [value, setValue] = useState(0);

  // 我们打算在这个子组件上实现transition动画，因此希望保证DOM元素被正确复用
  const Child = () => {
    // 这里用ref回调函数是为了来检测DOM元素是否正确被复用，也可以用其他方式实现
    return <div ref={console.log}>这是子组件</div>;
  };

  return (
    <div>
      
      {/* 写法一：把它当作组件 */}
      <Child />
      
      {/* 写法二：把它当作函数 */}
      {Child()}

      <span>{value}</span>
      <button onClick={() => setValue(value + 1)}>触发re-render</button>
    </div>
  );
};
```

这位同学使用的是上面的“写法一”，然后在调试过程中发现，动画效果怎么调都调不出来。

让我猜猜，可能大多数同学都无法解释上面两种写法的区别？

回想一下，我们在准备面试背八股文的时候，是不是背过『react diff 算法』？它怎么说的来着？

参考官方文档 [协调(reconciliation)](https://zh-hans.reactjs.org/docs/reconciliation.html)：

> **对比不同类型的元素**  
> 当根节点为不同类型的元素时，React 会拆卸原有的树并且建立起新的树。……举个例子，当一个元素……从`<Article>` 变成 `<Comment>`……会触发一个完整的重建流程。  
> 当卸载一棵树时，对应的 DOM 节点也会被销毁。……

在上面的示例代码中，`Child`这个函数在每次`render`时都会被替换为一个新的函数（虽然新、旧函数的名字和内容都完全相同，但它们是不同的对象）。

因此，如果把它作为`<Child>`组件来使用，那么对于React来说，每次都是完全不同的**组件**，也就意味着每次都会“触发一个完整的重建流程”，这个流程也包括了DOM的销毁和创建。因此，这位同学想要实现的`transition`动画能力自然也就无法达成了。

改进方法，很容易想到，我们只要保证函数组件不变即可，用`useCallback`即可解决。这种方式可以，但没必要，我们用“写法二”即可更简单清晰地实现我们的需求。

## ts re-export

最近我在尝试使用`npm workspace`来管理项目代码。当我尝试导出一个包内的东西的时候，写了这样一段代码：

```ts
export { xxxClass, xxxType } from './components';
```

但是构建时报错：`export 'xxxType' (reexported as 'xxxType') was not found in './components'`

参考：[Cannot re-export a type when using the --isolatedModules with TS 3.2.2](https://stackoverflow.com/questions/53728230/cannot-re-export-a-type-when-using-the-isolatedmodules-with-ts-3-2-2)

原来在导出纯类型的时候不能直接写`export`，要专门导出`export type`，如下所示：

```ts
export { xxxClass } from './components';
export type { xxxType } from './components';
```

## 抖音网页端签名机制

这个话题挺有意思的。

一方面，严格来说这件事情算是“灰色”产业，如果不是字节跳动而是腾讯的话，可能搞不好哪天就去蹲牢了。

另一方面，如果在Github上搜索一些关键字，会搜出不少广告仓库，说什么三千块钱教你破解抖音签名……之类的。（用这个来赚钱，胆子是真肥）

所以虽然我已经完整地总结出了破解方法，但我这次先不详细介绍了，先藏一会吧。

简而言之，抖音的签名有三个部分组成：`msToken`，`X-Bogus`，`_signature`，三个东西分别依赖不同的数据源，而且代码也做了多种方式的保护，从技术方案设计的角度来说，我认为他们已经在理论上做到了极限了。

> 之前我在[总结CSRF](../2021/210922-Dig-CORS.md)的时候提到过，只用一种防御手段很容易被针对性地破解，要使用多种防御手段进行组合才是安全的。

要不是他们在具体执行落地的时候有一些些偷懒的地方，我想他们的签名是绝对不可能被破解的。也正是这个原因，我还是希望讲解破解方法的文章少一些，让他们尽可能晚一些做对抗性修复。如果哪天他们修复了，现有的方法不管用了，我再来详细讲讲。

## 闲谈：程序员的级别

根据我之前工作中的感受，再参考网络中的一些资料（[1](https://www.yaozeyuan.online/2022/10/01/2022/10/%E5%A6%82%E4%BD%95%E9%9D%A2%E8%AF%95%E5%80%99%E9%80%89%E4%BA%BA-%E5%89%8D%E7%AB%AF%E6%97%A9%E6%97%A9%E8%81%8A%E7%AC%94%E8%AE%B0/) [2](https://www.zhihu.com/question/265982393) 等等），目前总结出我对程序员的评价标准是：

- 初级（P4）：
  + 技术能力：能做，但是日常工作仍然需要指导，产出质量可能也不尽如人意。
  + 面试表现：入门级八股能答出，基本代码题能顺利写出，但是无法深入。
- 高级（P5）：
  + 技术能力：能做完，偶尔遇到困难经过指点之后能自行解决，产出质量合格。
  + 业务能力：能讲清楚自己做的是什么
  + 面试表现：常规八股扎实掌握，对框架和API熟悉；写出代码能考虑更多情况；能解决easy算法题，挑战medium题。
- 资深（P6）：
  + 技术能力：能做好，在分工领域（例如前端）内能解决复杂的问题，对其他领域有所了解并有效配合产出，能输出影响（做出有质量的技术分享）。
  + 业务能力：充分理解业务，准确挖掘关键需求
  + 面试表现：从八股知识点出发、结合项目经验讲方案，讲底层原理；掌握常见算法思路。
- 专家（P7）：
  + 架构能力：具备深度和广度，心中有全套的解决方案（前端+后端+运维+更多），只需要简单的需求讨论就能落地（创业能力），甚至反向驱动业务发展；
  + 实现能力：写代码又快又好，有成熟的工作方法，保证无论遇到什么情况都能妥善处理；
  + 业务能力：对自家产品的内外情况有全面的认识，对行业状况有充足的了解；
  + 小团队领导能力：抗住压力，给团队找方向，撑起空间；能带好5人左右小组，或者小公司的经理、架构师。
- 高级专家（P8）：
  + 领导能力：有优异的成绩能服人，能说黑话来唬人；能扛旗子，争取资源，打造团队；能解决各种奇奇怪怪的疑难杂症，作为小公司或者大部门的守门员的存在；
  + 技术竞争力：保证核心技术竞争力，深入理解日常工作中不会接触到的底层细节或者行业标准（例如协议、交互、渲染、基建），能根据场景自己定制框架，或者能做出独创性技术（公开渠道查不到的解决方案）；
- （P9）再往上则需要参与管理、或者行业影响力，不再是凭个人能力能达到的了，还需要一定的机遇。

一个“优秀”的程序员，他应该在最晚1-2年左右达到高级水平（顺利的话校招就是P5），3-5年达到资深水平（P6），5-8年达到专家水平（P7）。超过这个时间范围，我认为最多就只能被认为是“平凡”了。

我对我自己的评价是已经达到了P7的水平，但是还需要一些时间来适应角色，目前正在争取向P8的要求看齐。

在4年的程序员职业生涯中，我从一个自学转行的低端python脚本小子，干到（有实无名的）资深后端，再干到（名副其实的）资深前端、项目前端负责人、前端面试官，这份成绩值得骄傲，但同时，当我开始评价别人的时候，我也必须时刻提醒自己不能总以自己的成绩去要求别人。因此像这样总结一份职级能力对照表，对我来说还是很有意义的。
