```yaml lw-blog-meta
title: "React Fiber 基本概念"
date: "2021-10-11"
brev: "React16底层的diff算法"
tags: ["前端"]
```

## 引子

原文： [Virtual DOM and Internals](https://reactjs.org/docs/faq-internals.html)

Virtual DOM 是一个编程概念，「虚拟`virtual`」代表着UI保存在内存中，并通过（ReactDOM等库）与"实际"的DOM同步。

这种理念，使得「声明式的`declarative` API」在React中得以实现：你只需要告诉React某个UI当前的状态，React会保证DOM符合你指定的状态。通过这样，把属性、事件处理、DOM更新等麻烦操作都抽象掉了。

因为「virtual DOM」更多的是一种模式而不是一种特定的技术，所以人们有时也用它来指代不同的东西。在React的世界里，一般指的是`React Elements`，因为它是代表着用户接口的对象。同时，在React内部，使用一种叫`Fiber`的对象来保存组件树的数据，它也算是React的VDOM的一部分。

## 准备知识：术语

原文： [React Components, Elements, and Instances](https://reactjs.org/blog/2015/12/18/react-components-elements-and-instances.html) ，准确区分三个术语的定义。

先想象一下，在传统的面向对象UI编程中，我们定义一个组件，需要保存底层DOM的引用、所有后代的引用，然后每次状态更新的时候，要判断状态的情况然后逐个去修改DOM或者后代。这会使得代码非常臃肿。

在React里，「元素`Element`」是一个简单的对象(object)，只含有一些用来描述底层DOM的数据 (An element is a plain object describing a component instance or DOM node and its desired properties)，看个例子就懂了：

```tsx
export function App(): JSX.Element {
  return (
    {
      type: 'b',
      props: {
        children: 'OK!'
      },
      // key:'sample',
      // $$typeof: Symbol.for('react.element'),
      // ref: null,
    }
  );
}
```

> 上面这个例子是可以运行的！还原注释的部分代码就可，不需要 JSX ！

在上面的例子中，`App`是一个组件(Component)，它可以接收一些属性(props)，它的返回值是一个元素(Element)。这个元素代表了一个`<b></b>`标签，含有一个字符串作为后代元素。

Element只是对于组件的一个描述，它并不真实对应任何底层的对象(are just descriptions and not the actual instances)。

这个`type`，既可以是真实的底层HTML标签类型（a、img、div等），也可以是React组件（即组件嵌套）。如果是组件，那么React会递归这个组件去获得它对应的元素，直到所有的元素都是底层类型。（An element describing a component is also an element, just like an element describing the DOM node. They can be nested and mixed with each other.） 

一个更完整的例子：

```tsx
const DangerButton = ({ children }) => ({
  type: Button,
  props: {
    color: 'red',
    children: children,
  },
});
const DeleteAccount = () => ({
  type: 'div',
  props: {
    children: [
      {
        type: 'p',
        props: {
          children: 'Are you sure?',
        },
      },
      {
        type: DangerButton,
        props: {
          children: 'Yep',
        },
      },
      {
        type: Button,
        props: {
          color: 'blue',
          children: 'Cancel',
        },
      },
    ],
  },
});
```

上面展示的是用函数(Function)来作为组件(Component)，更经典的用法是用类(Class)来做。

用Class会稍微强大一点点，因为它会在React底层创建并维护一个实例(Instance)，It can store some local state and perform custom logic when the corresponding DOM node is created or destroyed. 用Function更简洁，如非必要，我们鼓励你使用函数组件。

最后补充一点，元素(Element)是不可变的(Once an element is created, it is never mutated) ，因此每次`render()`的时候，会生成一个新的Element树，React对新旧两棵树做比较算法(Reconciliation)，从而得知哪些组件发生了变化，从而依此去更新底层的DOM。

总结：

- `Element` 是简单的数据结构体对象，仅仅描述DOM或者组件，可以嵌套成树。
- `Component` 可以是函数或者类，它输入属性(props)，输出元素(Element)。
- `Instance` 就是类组件中的`this`，React会替你维护好。

## 准备知识：Reconciliation

原文： [Reconciliation](https://reactjs.org/docs/reconciliation.html)

这个单词翻译为「协调」，理解为是React组件树的 比较算法 。

### 动机

使用React的时候，`render()`会创建一个React元素(Element)组成的树。每当你更新属性(props)或者状态(state)的时候，`render()`都会返回一个不同的树。因此React需要高效地将最新的元素树反映在UI上。

如果作为普通的「树」，两个树的比较算法，即使是最先进的算法也需要`O(n^3)`的复杂度。

React根据实际情况作了一些假设，使得算法复杂度降为`O(n)`：

1. 不同类型(type)的元素(Element)会产生完全不同的树。
2. 开发者可以通过指定子元素的`key`来为算法提供提示。

### 不同类型的元素

例如，将`<a></a>`标签替换为`<img></img>`标签。

会进行完整渲染，即销毁旧的DOM（触发`componentWillUnmount()`）、创建新的DOM（触发`UNSAFE_componentWillMount()`和`componentDidMount()`）。

### 相同类型的元素

React仅仅比较它们的属性，然后更新发生了变化的那些属性。不会替换底层的DOM。

在这个过程中，组件`component`对象依然保留，因此状态`state`会被保留、不会因为`render()`而被重置。React只会更新组件对象的属性，这个过程中触发 `UNSAFE_componentWillReceiveProps()`, `UNSAFE_componentWillUpdate()` 和 `componentDidUpdate()`. 然后，调用`render()`得到一个新的树并使用比较算法去递归地比较前后两棵树。

### 递归子节点

默认情况下，React只是简单地递归比较所有子节点。

因此，在某些场景下性能很糟糕，例如在最前面插入一个新的子节点，会导致所有子节点都受到影响。

为了解决这个性能问题，开发者可以指定`Key`，这样React只会去比较Key相同的节点。

在实践中，找一个key并不难。就算你的数据模型中没有id，你也可以做一下hash来得到一个值来作为key 。key只需要在同辈(sibling)之间保持唯一即可，不需要全局唯一。

## 准备知识：React设计原则

原文： [Design Principles](https://reactjs.org/docs/design-principles.html)

组合(Composition)： React的核心特性是组件组合(composition of components)（译者注：组合与继承对立），这允许你修改一个组件而不影响其他不相关的组件。

通用抽象(Common Abstraction)：正常情况下我们拒绝向React中添加特性。但如果我们观察到某些很常用的特性，在我们认为融入React会对整体生态有益的话，我们还是会加的。

逃生舱(Escape Hatches)：我们遵循实用主义，尽量保持易用性，特性的增减都会留下后路，并持续与社区保持沟通。

可靠性(Stability)：我们在Facebook内部带头检验React，因此正常情况下其他用户不太会遇到我们未曾设想过的场景。在废弃API时，我们会提供充足的缓冲、指引甚至代码编辑脚本。

互用性(Interoperability)：保持与其他库的兼容性。

调度(Scheduling)：

- React负责调用开发者编写的函数或者类组件，因此也有机会选择执行的时间。目前会在一帧(tick)之内更新所有组件，但是后续版本可能会延迟部分更新操作，以避免掉帧(dropping frames).
- 有些库会用`push`策略来更新数据，但是React坚持`pull`策略（即懒加载）
- 我们内部有个笑话：React不如改名为`Schedule`吧，因为React并不完全`reactive`(响应式)了

开发体验(Developer Experience)：我们为开发者做了很多事情，例如 React DevTools插件，例如开发模式的React库会提供很多额外的警告和错误信息。

调试(Debugging)：我们在报错信息中添加了面包屑(breadcrumbs)，这样借助DevTools你可以更加轻松地定位问题。

配置(Configuration)：给React做个全局配置会带来很多问题，所以我们暂时没有提供。但是提供了一些其他的东西。

不局限于DOM(Beyond the DOM)：React Native同样重要。所以我们坚持让React做到渲染器无关(renderer-agnostic)，这样我们可以保持单一的编程模型，这使得我们可以以产品为单位来组织人力资源，而不是以平台来区分。

代码实现(Implementation)：我们优先考虑让API优雅，而不是让React内部代码优雅。我们喜欢简单可靠的代码，即使它显得枯燥啰嗦。(We prefer boring code to clever code.)

为工具优化(Optimized for Tooling)：有些API的命名是故意为了提升可读性和可搜索性而定下的。JSX很重要，为开发/编译期间的工具提供了巨大的帮助。

内部试用(Dogfooding)：React主要为Facebook内部解决问题，同时也兼顾一些社区的需求。到目前为止，不论是质量还是需求满足度应该都是不错的，大家可以放心押宝React。

## React Fiber 架构

原文： [react-fiber-architecture](https://github.com/acdlite/react-fiber-architecture) ，最后更新于2016年，而搭载了`Fiber`的`React16`是在2017年发布的，所以本文是一种提案性质的语境。

作者说，本文不是官方出品，只是他个人的一些笔记整理。但是也经过官方团队的review，并且目前直接推荐在教程页面上，因此还是足够权威的。

### 介绍

Fiber 是React内核算法的一次重构。

Fiber的目标，是提升对于特定领域的适用性，例如动画、布局(layout)、手势(gestures)等。

Fiber的核心特性是「增量渲染`incremental renderding`」：有能力把渲染工作任务分片并分配到多个帧(frames)中去。

还有一些额外特性，例如暂停、放弃、复用渲染任务，例如指定优先级，例如新的并发原语(primitives)。

### 回顾reconciliation

每次状态更新的时候，React都会重新渲染整个app 。具体一点，就是重新执行`render()`，得到一个新的完整的元素树(Element)。

这样做的代价很大，所以`reconciliation`就是一种比较算法，用来找出那些变化的部分去重新渲染或者更新，而不是整个app 。

尽管 Fiber 是完全重构的新的 reconciliation算法，但宏观上来看还是差不多的，重点依然是：

- 不同类型的组件不会比较，会直接替换。
- 对于列表会使用key

协调(reconciliation) 和 渲染(rendering) 是独立的两个步骤。Fiber 就是一种 协调器(reconciler) ，先比较新旧两棵树，找出变化的部分，然后在一个合适的时间将其交给底层的渲染框架（例如ReactDOM）去执行。

### 回顾Scheduling

调度(Scheduling): 指决定一项工作应该在什么时候完成的一个过程。

工作(work): 指任何需要执行的计算任务。在这个语境下，通常指的是一个更新(update)的结果（例如`setState`）

关键点：

- 对于UI来说，并不是每个更新都需要立即执行；
- 不同的更新会有不同的优先级。例如，动画(animation)更新往往需要立即执行，而加载数据可以稍缓。
- 基于`push`策略的应用需要开发者自己决定如何调度；而基于`pull`策略的应用，例如拥有Fiber之后的React，可以替你实现调度。

### 什么是Fiber？

我们已经清楚了，Fiber的目标是让React能够利用调度的优势。具体地说，我们需要能够：

- 暂停一项工作并让它稍后执行；
- 分配优先级
- 复用前面已经执行过的任务（的结果）；
- 放弃任务

为了实现任何一个目标，我们都首先需要能够将任务(work)分解为任务单元(units)。某种角度来说，Fiber就是这么一个东西：一个Fiber就是一个单元任务(unit of work).

先理解一个概念，「渲染一个React应用相当于就是在执行一个函数」。所以Fiber也可以理解为是重构了函数调用栈(call stack)，或者说是一种「虚拟栈帧 virtual stack frame」，借助它你可以决定何时执行栈帧。

虚拟栈帧还带来其他好处：并发(concurrency)、错误处理(error boundaries)等。

### Fiber的结构

具体而言，一个Fiber就是一个JS对象，它包含着关于组件(component)的信息以及入参和返回值。

一个Fiber对应一个栈帧(stack frame)，同时也对应着一个组件实例(instance)。

以下列举Fiber的部分重要字段：

`type`和`key`: 从Element中复制而来。

`child`和`sibling`: 后者是比Element多出来的部分。sibling是一个单向链表，头结点是第一个子元素。例如在下面的例子中，`Parent`的child是`Child1`，`Child1`的sibling是`Child2`。

`return`: 是一个Fiber的返回点，一般就是它的父节点。

`pendingProps` and `memoizedProps`: 在Fiber执行之前，会将props设置到pendingProps中，执行完毕后会保存到memoizedProps中。如果下一次更新时新的props等于pendingProps，那么意味着这个Fiber可以复用而不需要重新执行。

`pendingWorkPriority`: 表示优先级，数字越大、优先级越低。特例是`NoWork`表示为0.

`alternate`: 在任意时刻，每个组件实例最多对应着2个Fiber，一个是当前已经提交(Flush)到渲染框架暴露在UI上的Fiber，一个是正在处理中的Fiber 。为了避免GC，两个Fiber会依次替换对方。所以它们两个互相将对方储存在`alternate`字段中。

`output`: 是函数的返回值（前面说的`return`是返回点而不是返回值）。每个Fiber最终都有一个`output`，但是只有叶子节点，即宿主组件(host component)才会最终`output`，它们创建的`output`将会沿着树向上传递。（译者注：即叶子节点会创建一个真正的VDOM，然后这个VDOM引用会向上传递；然后ouput交给渲染器之后转化为真实的DOM）

## 理解

好家伙，听君一席话胜听一席话。果然是一些零散笔记的集合体……

如果仅仅只是了解Fiber的一些特性和流程，并没有什么意义，就是八股文而已。所以接下来再结合一篇 [中文博客](http://www.ayqy.net/blog/dive-into-react-fiber/) 来进一步理解Fiber带来的影响和应对。

用一句话概括，就是：由于旧版React的vDOM机制会无差别的检查所有vDOM，在极端情况下性能很差造成掉帧，所以做一个能够实现优先级的东西，在性能紧张的时候优先执行部分vDOM更新；Fiber就是为了实现调度功能而创造出来的新的数据结构体（及配套逻辑）。

在旧版React中(Stack Reconciler)渲染步骤是「element->vDOM(Instances)->DOM」，在 React Fiver 中则是「element->fiber->effect->DOM」。

React Fiber 实质上就是把 原来必须一口气执行完毕的递归函数 转换为了 有优先级和时间限制的循环函数 。

一个`fiber`是一个函数执行栈，也是一个调度单元，也对应一个组件。所以这个「循环」大概就是，执行一个fiber，观察一下是否需要中断并返回主线程，不需要的话接着执行下一个fiber。

整个协调过程分为两个阶段，阶段一，通过上述循环执行一定数量的fibers并得出一个`effect list`即需要被渲染的DOM的变化量，这个阶段可以被中断/可以被调度/可以插队；阶段二，将本次得出的`effect list`交给底层的渲染器去执行，反映在DOM上。

## 启示

回忆一下最初的目标，Fiber是为了优化那些高优先级的计算任务而生的。所以我们在编程的时候也要充分利用两个阶段的特性。简而言之，就是别在不可中断的阶段二里做很重的事情。

那些生命周期函数也划分到两个阶段中：

```text
// 第1阶段 render/reconciliation
componentWillMount
componentWillReceiveProps
shouldComponentUpdate
componentWillUpdate

// 第2阶段 commit
componentDidMount
componentDidUpdate
componentWillUnmount
```

除此之外还要知道Fiber带来的其他影响：

- 生命周期函数可以被插队，这样导致触发顺序、次数没有保证。
- 低优先级饿死。
