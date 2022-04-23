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

注意要link至少3个包：`react`, `react-dom`, `scheduler`

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

`flushSync`这个函数会在它里面再次调用传入的箭头函数，它的作用是立即（同步地）刷新整颗树 [参考](https://github.com/facebook/react/issues/11527#issuecomment-360199710)

`createContainer`这个函数最主要的就是做了`new FiberRootNode()`这个事情。

`updateContainer`这个函数再次出现了！它的功能就是创建"任务"，并把任务添加到任务队列中去，来看看：

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

非常值得一提的是，在`scheduleUpdateOnFiber()`中的`markUpdateLaneFromFiberToRoot()`函数会将当前fiber的`Lane`一直同步到`FiberRoot`上去，（通过`fiber.return`向上回溯）。

### 小结

在这一步中，我们传入了一个`element`（即VDOM树），然后生成了一个`FiberRoot`（同样也是树形结构），还看到它加入了某个队列中（`enqueueUpdate`）。

接下来需要解决的疑问：队列里的任务是如何调度的？

## 4. scheduler.workLoop

回忆一下，Fiber诞生的初衷，就是为了防止掉帧。所以它的核心逻辑就是当判断到时间不够之后主动让出线程，等浏览器渲染完毕后再继续之前的任务。

> 所有的调度逻辑都在`Scheduler`这个模块里，详细内容可以参考 [这篇](https://segmentfault.com/a/1190000022942008) 或者[这篇](https://juejin.cn/post/7020220688719937573#heading-4) （感觉讲的比我好多了……）

> 顺便一提，Scheduler看起来并不是专为react设计的，目前它的逻辑是独立出来作为一个独立的包而存在的。

如何判断“当前剩余时间不够”了？

```js
function workLoop(hasTimeRemaining, initialTime) {
  var currentTime = initialTime;
  advanceTimers(currentTime);
  currentTask = peek(taskQueue);  // 从任务队列中取出第一项

  while (currentTask !== null && !(enableSchedulerDebugging )) {
    if (currentTask.expirationTime > currentTime && (!hasTimeRemaining || shouldYieldToHost())) {
      break;  // 到时间了，不再处理队列中后续的任务
    }
    
    // ...执行任务的步骤，主要就是执行task.callback
    pop(taskQueue);  // 执行完毕后丢掉任务

    currentTask = peek(taskQueue);  // 取出下一项任务
  }
  
  if (currentTask !== null) {
    return true;  // 这里的返回值给回hasMoreWork
  } else {
    return false;
  }
}
```

上面代码的核心逻辑是，从队列里取一个任务，执行，然后检查是否还有剩余时间。

上面的代码中省略了`timerQueue`相关的逻辑，它的作用相当于是`taskQueue`的延迟队列，每次从`taskQueue`里执行一个任务之后，都会检查一下是否有到期的递延任务，如果有，则从`timerQueue`里取出并推入`taskQueue`。

值得一提的是，这两个Queue的数据结构都是`最小堆`。

这个函数的返回值会赋给`hasMoreWork`这个变量里：

```js
var performWorkUntilDeadline = function () {
    var hasMoreWork = true;

    try {
      hasMoreWork = scheduledHostCallback(hasTimeRemaining, currentTime);  // workLoop的返回值在这里
    } finally {
      if (hasMoreWork) {
        schedulePerformWorkUntilDeadline();  // 如果还有剩余工作，则留下一个小尾巴，等待下一次调度
      } else {
          // ...
      }
    }
};
```

如果还有剩余工作，那么会再次调用一次`schedulePerformWorkUntilDeadline()`函数，顾名思义："调度下一次工作"。

```js
if (typeof MessageChannel !== 'undefined') {
  var channel = new MessageChannel();
  var port = channel.port2;
  channel.port1.onmessage = function () {
    performWorkUntilDeadline(arguments)
  };

  schedulePerformWorkUntilDeadline = function () {
    port.postMessage(null);
  };
}
```

关于"调度"这件事情的具体逻辑，以前的很经典的实现是使用`requestIdleCallback`那一套，从19年开始使用了`MessageChannel`，创建过程就如上面所示。

> 放弃`requestIdleCallback`是因为兼容性问题，而且它的标准执行间隔是50ms太慢了；放弃`requestAnimationFrame`是因为它可能受到硬件屏幕刷新率的影响，而且执行顺序不对；放弃`setImmediet`是因为它会浪费4ms；最终`MessageChannel`可以提供更加稳定的机制。 [参考](https://juejin.cn/post/6953804914715803678)

所以调用它一次的效果，实质上就是创建了一个`宏任务`，从而创造一次"将控制权交回浏览器"的机会。

### task的缺陷

听起来一切都很完美，每次完成一个fiber任务都会检查是否跳出循环，掉帧的问题似乎解决了呢？可是"频繁地比较时间"这件事难道不会对性能造成影响吗？

实际上：**一个"任务"并不是一个react节点的更新，而是发生状态变化的节点下面整棵树的更新**。

所以如果一次状态更新就导致了严重的延迟，那么fiber也救不了你（不过我感觉这个应该只是个理论极端情况，不太可能出现在生产环境下）。

> 我借助斐波那契函数来人为制造工作量并发现了上述这个问题。在实验过程中，我发现似乎react对某些简单的状态更新是有优化的，根本不会进入workLoop这个逻辑里（具体优化策略暂不清楚）。

### 小结

现在我们知道了Scheduler是如何在多个任务之间检查并让出主线程的，接下来我们需要知道在单次执行任务的过程中发生了什么。

## 5. ReactDOM.performUnitOfWork

接下来我们需要回到`ReactDOM`的领域内。（可以继续参考[这篇文章](https://indepth.dev/posts/1008/inside-fiber-in-depth-overview-of-the-new-reconciliation-algorithm-in-react#main-steps-of-the-work-loop) ）

我们需要看的是`performUnitOfWork()`这个函数。

如果从它开始向上追溯，可以追到`ensureRootIsScheduled`这里，这个我们在第3步探索`ReactDOM.render`的时候已经见到过了（虽然我没有在本文中贴出相关代码）。

看看代码：

（未完待续）
