```yaml lw-blog-meta
title: "React源码速读(v18)"
date: "2022-04-16"
brev: "精通一个框架，至少核心部分的源码得看看吧"
tags: ["前端","源码"]
```

## 背景

我们都知道，React的核心原理是构建了一个虚拟元素树，即Virtual Dom Tree。那么，当某个组件的状态发生改变的时候，整个树里到底发生了什么？

以前，有很多文章都介绍过：一方面，~~只有状态更新过的组件，或者~~准确地说应该叫做元素(`elements`)，只有更新过的元素才会被标记为脏，然后重新运行它的`render()`得到一个新的子元素树，这个标记为脏的过程剪去了父节点及以上的不必要更新；然后新旧子树再做diff，最后反应到DOM上去，这个diff的过程就是剪去了子节点的不必要更新。看起来很完美，效率很高。

然而，在 React fiber 出现之后，似乎有了一些分歧。小伙伴截了个网络博客文章告诉我说，fiber为了保证"优先级"属性能同步到父节点，会从当前脏节点向上回溯直到root，再从root向下遍历。

如果真的是这样的话，那每个叶子节点的更新都要回溯到整棵树，那这个实现也太蠢了，不可能。

不过虽然我99%认为是不可能的，但我依然没有绝对可靠的证据来支持我自己。所以，这React源码还是必须得看了。

安排。

## 归纳现有的文章

react每次会比较整个组件树吗？

这个回答给出了精简的归纳： [Does React always check the whole tree? - stackoverflow](https://stackoverflow.com/questions/34696816/does-react-always-check-the-whole-tree) 其中引用了核心成员的博客： [React’s diff algorithm - @vjeux](https://calendar.perfplanet.com/2013/diff/)

这篇文章归纳如下的主要意思：`setState`会导致组件被标记为脏，随后它调用`render()`构造一颗以它为根的新的组件树，并与旧的树进行对比。

还提到一个细节，只是对两个组件树（的产物vDom树）进行对比的话，由于它们都只是js对象而不是真实的dom，所以这个比较过程(diff)的代价并不算大。

这个过程不会影响到上层组件，但是会影响到下层，也就是说如果对比较高层的组件更新状态，会导致很大一棵树的递归。对这个问题的[解决方案](https://stackoverflow.com/a/40910993/12159549) 是：

- 类组件：使用`shouldComponentUpdate`主动停止递归
- `PureComponent`组件：直接会对props做浅比较，如果没有属性变化则停止递归（从v15.3.0引入）
- 函数组件：
    + 没有生命周期函数，所以一定会re-render；
    + 可以在父组件用`useMemo`或者对函数组件本身`memo()`来达到`PureComponent`的效果
    + 函数组件本身非常轻量（对比类组件），不考虑这个优化也未尝不可。

总之：对VDOM树的任意层级节点更新状态，都可以通过优化来避免这次更新无意义地向上、向下传播。

但是上面的文章有一个问题在于，它们都是在`React Fiber`这个架构之前的文章，并不适用于从v16以后的实际情况。

接下来再看一篇 [解析](https://indepth.dev/posts/1008/inside-fiber-in-depth-overview-of-the-new-reconciliation-algorithm-in-react#render-phase) 大概意思是：reconcile的时候确实从根节点开始，但是会快速跳过那些没有状态改变过的父节点，直到那个改变了状态的子节点。 （话说这篇文章讲得真的非常详细且深入，下次我要把它完整翻译一遍。）

## 获得源码

通过查看`node_modules/react/package.json`中的配置，得知原始仓库在`https://github.com/facebook/react`。

> 这里有两个槽点，第一，直接在github上搜索"reactjs"，会搜出一个`react community`的组织，而不是`facebook`；第二，Facebook 公司早已改名为 Meta，然而仓库地址依然是`facebook/...`，这个东西确实不能随随便便改的啊。我一直想吐槽Facebook改名改得太快太冲动了……

```shell
git clone https://github.com/facebook/react --depth=1
```

下载的默认分支`main`就是当前的最新版本`v18.0.0`。（说来也巧，刚好就是在前两天才发布的这个版本）

虽然fiber是从`v16`推出的，但我也懒得去找旧版本了，就这么看吧，顺便了解一下最新动态。

> 在看源码之前，先提一句，react仓库代码没有使用typescript，应该是使用了 [flow](https://flow.org/en/docs/getting-started/) 作为js类型检查工具。我们开发中使用的类型提示则是来源于另一个仓库 [@types/react](https://www.npmjs.com/package/@types/react) 

**特别声明：为了阅读、讲解方便，我在本文中贴出的代码都是有删减的，实际请以原始仓库的代码为准。代码相关权利以相关项目 [条款](https://github.com/facebook/react/blob/main/LICENSE) 为准。**

## 调试源码

为了更加有效地理解源码，我们需要一定的手段来调试运行源码。准备过程可以 [参考](https://juejin.cn/post/7021095381589032973)

主要流程是，先在`react`代码仓库里构建，然后使用`yarn link`命令，把其他项目里对react的依赖替换为本地刚刚构建出来的版本（类似于`go.mod`里的`replace`用法）。

这个过程我要吐槽一下，构建react居然需要安装java，也是匪夷所思……

## 1. JSX

React万物都始于这样一条语句：

```jsx
ReactDOM.render(<App />, xx)
```

上面是`JSX`给我们提供了极大的便利，它会被`babel`编译为下面这个样子：

```js
ReactDOM.render(React.createElement(App), xx)
```

如果是一个含有多个children的`Element`，则会在`createElement`中追加多个参数，有兴趣可以 [参考](https://juejin.cn/post/6959948160525565960)

## 2. React.createElement

在源码中看到这样的声明：

```js
// react/packages/react/src/React.js
const createElement = __DEV__ ? createElementWithValidation : createElementProd;
```

后者跳转到`./ReactElement.js`文件中，大概意思如下：

```js
export function createElement(type, config, children) {
  let propName;

  // Reserved names are extracted
  const props = {};

  let key = null;
  let ref = null;
  let self = null;
  let source = null;
  
  // ...把 key, ref, self, source 四个保留属性从config里抽出来，剩下的装进props里
    
  // 从arguments第三位开始是子元素，全部装进props.children里
  
  return ReactElement(  // 再把处理过的属性传给下一步
    type,
    key,
    ref,
    self,
    source,
    ReactCurrentOwner.current,
    props,
  );
}
```

> 再稍微解释一下，在`<App size={1} />`这个元素中，type是`'App'`，config是`{size:1}`

所以这一步的作用其实就是处理参数，做一些防御和警告，没有其他实际意义。

接下来`ReactElement`，这个东西虽然是大驼峰，可是它是个函数，大意如下：

> 这部分内容我之前在[React Fiber](../2021/211011-React-Fiber.md#准备知识：术语) 也讲过

```js
const ReactElement = function(type, key, ref, self, source, owner, props) {
  const element = {
    $$typeof: REACT_ELEMENT_TYPE,  // 这是一个Symbol

    // 四个保留属性
    type: type,
    key: key,
    ref: ref,
    props: props,

    // Record the component responsible for creating this element.
    _owner: owner,
  };

  // ...如果是DEV环境，则额外定义一些属性

  return element;
};
```

所以这一步实际上也就仅仅是把前面传入的属性封装成了一个`element`对象返回。

也就是说，整个`createElement()`做的事情也就仅仅是构造了一个`element`（元素），或者准确地说是通过`children`属性从上到下组织起来的一颗元素树，也就是我们所说的 V-DOM 树。

接下来，这颗VDOM树要交给底层的渲染库（在web中就是`ReactDOM`）去将其映射到HTML中去。

## 3. ReactDOM.render

晴天霹雳，`v18`开始，`render()`方法将被逐步废弃，继续使用则保持`v17`的特性；如果要体验`v18`的特性（例如并发），则需要切换到`createRoot()`方法去。

OK我们先继续看下去：

```flow js
export function render(
  element: React$Element<any>,  // React.createElement的产物
  container: Container,  // 一个HTML标签
  callback: ?Function,  // 回调函数，首次渲染完成时调用，很少用到
) {
  // ...警告提示

  // 这里通过nodeType来检查HTML标签是否合法
  if (!isValidContainerLegacy(container)) {
    throw new Error('Target container is not a DOM element.');
  }
  
  return legacyRenderSubtreeIntoContainer(
    null,
    element,
    container,
    false,
    callback,
  );
}
```

`legacy`这个单词可以理解为是“旧的、传统的、以前的”，接下来调用的这个函数会创建（或获取）一个`FiberRoot`：

```flow js
function legacyRenderSubtreeIntoContainer(
  x,
  children: ReactNodeList,  // 注意这个children对应的是前面的element，也就是我们createElement创建的元素树
  container: Container,  // DOM对象
  x,
  x,
) {
  // ...

  const maybeRoot = container._reactRootContainer;  // 注意这里，fiberRoot是直接挂在DOM对象上的
  let root: FiberRoot;
  if (!maybeRoot) {  // 情况一：新建
    root = legacyCreateRootFromDOMContainer(
      container,
      children,
      x,
      x,
      x,
    );
  } else {  // 情况二：已有的更新
    root = maybeRoot;
    updateContainer(children, root, parentComponent, callback);
  }
  return getPublicRootInstance(root);
}
```

在上面的代码中，`_reactRootContainer`这个属性非常有趣，它居然是直接挂在DOM对象上的！也就是说，我们可以试试，在一个react应用页面上（例如我这个博客）取一个根DOM元素然后在控制台里查询这个属性，会发现真的有东西哦！

这个函数调用了三个函数，我们分别来看：

```flow js
function legacyCreateRootFromDOMContainer(
  container: Container,
  initialChildren: ReactNodeList,  // createElement的元素树
  x,
  x,
  isHydrationContainer: boolean,  // false
): FiberRoot {
  if (isHydrationContainer) {
    // ...Hydry模式，不看
  } else {
    // ...先把DOM元素中已有的Child全部删掉
    // ...

    const root = createContainer(  // new一个对象
      container,
      LegacyRoot, // 这是一个常量
      ...x,
    );
    container._reactRootContainer = root;
    markContainerAsRoot(root.current, container);  // 这里又使用了一个叫 __reactContainer$xxx 的属性

    // ...监听container的事件（基于DOM的事件上浮机制），不看

    flushSync(() => {
      updateContainer(initialChildren, root, parentComponent, callback);
    });

    return root;
  }
}
```

`flushSync`这个函数会在它里面再次调用传入的箭头函数，我们先暂时不管它的细节，继续看……

`createContainer`这个函数最主要的就是做了`new FiberRootNode()`这个事情，我们也先暂时不管它的细节，继续看……

`updateContainer`这个函数再次出现了！它的功能就是（），来看看：

```flow js
export function updateContainer(
  element: ReactNodeList,
  container: OpaqueRoot,
  x,
  xx,
): Lane {
  const current = container.current;  // FiberRoot有两个Fiber两个互为替代
  const eventTime = requestEventTime();  // 小细节，这里的时间用的是高精度时间
  const lane = requestUpdateLane(current);  // number

  // 这个container不是DOM了，而是FiberRoot，也就是reactRootContainer这个东西
  container.context = getContextForSubtree(parentComponent);
  
  // 这里创建了一个“任务”对象
  const update = createUpdate(eventTime, lane);
  update.payload = {element};

  // 随后把“任务”对象丢进队列里统一处理
  // 这个队列是current的队列，但是队列下面又有一个shared共享队列
  enqueueUpdate(current, update, lane);
    
  // 把队列里刚刚加入的任务调度起来，作为一个起始
  const root = scheduleUpdateOnFiber(current, lane, eventTime);
  if (root !== null) {
    entangleTransitions(root, current, lane);
  }

  return lane;
}
```

上面这个函数，首先注意到它的返回值是`Lane`，它是一个`number`类型，实质上是一些位掩码(bit mask)，用于区分fiber任务的种类以及在此基础上的优先级。 [参考](https://dev.to/okmttdhr/what-is-lane-in-react-4np7)

非常值得一提的是，在`scheduleUpdateOnFiber()`中的`markUpdateLaneFromFiberToRoot()`函数会将当前fiber的`Lane`一直同步到`FiberRoot`上去，（通过`fiber.return`向上回溯），这个过程很疑惑。

### 小结

在这一步中，我们传入了一个`element`（即VDOM树），然后生成了一个`FiberRoot`（同样也是树形结构），还看到它加入了某个队列中（`enqueueUpdate`）。

接下来需要解决的疑问：

1. Fiber是在什么时候渲染成为DOM的？
2. Fiber的调度逻辑是怎样的？

## 4. MessageChannel

回忆一下，Fiber诞生的初衷，就是为了防止掉帧。所以它的核心逻辑就是当判断到时间不够之后主动让出线程，等浏览器渲染完毕后再继续之前的任务。

如何继续，或者说，该等到什么时候才可以继续下一步呢？所有的调度逻辑都在`Scheduler`这个模块里，以前的很经典的实现是使用`requestIdleCallback`那一套，从19年开始使用了`MessageChannel`，详细内容可以参考 [这篇](https://segmentfault.com/a/1190000022942008) 或者[这篇](https://juejin.cn/post/7020220688719937573#heading-4) （感觉讲的比我好多了……）

（未完待续）
