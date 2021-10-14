```yaml lw-blog-meta
title: "Redux v.s. Mobx"
date: "2021-10-14"
brev: "React世界内两大主流的状态管理库"
tags: ["前端"]
```

## 背景

我不是一个典型的前端工程师，我并没有"随大流"先去学`Redux`；或者更详细地说，我现在公司的项目中就没见到过`Redux`，所以我直接上手就是`Mobx`。

从我目前的开发主观体验来说，我认为`Mobx`基本上是完美的，没有完全跨不过去的坑；就公司项目来说，最近重构的一个极度复杂的前端编辑器项目也是用Mobx实现的，因此从结果来说它是足够支撑商业级的复杂项目的。其实完全没必要再去学一个新的框架。

但是大道理谁都懂，不能沉浸在自己的信息茧房里，所以今天还是来看看`Redux`的用法和特性。

## TL;DR

`Redux` 由于将所有的state和action都整合在一个对象中，因此仅适用于状态不复杂的偏展示类的前端项目。

`Mobx` 适用于所有的项目。

## 参考阅读

- [我为什么从Redux迁移到了Mobx](https://tech.youzan.com/mobx_vs_redux/)
- [你需要Mobx还是Redux？](https://juejin.cn/post/6844903562095362056)
- [Redux 入门教程（三）：React-Redux 的用法](https://www.ruanyifeng.com/blog/2016/09/redux_tutorial_part_three_react-redux.html)

## Redux基本用法

参考阮一峰的教程，然后结合之前Mobx的使用经验，大概是这样的：

`Redux` 只是一个状态管理库，它并没有限定底层的UI库，但大家一般都结合React用，所以要再引用一个`react-redux`库。（就像`mobx`要配合`mobx-react-lite`一起使用）

在它的世界中，将组件分为了两个概念：UI组件，和，容器组件。

前者是个完全无状态的组件，所有数据来自于`props`；所有的状态都放在容器组件中。我们不直接写容器组件，只写UI组件，然后传入一些处理函数生成出容器组件。（就像`mobx`用`obeserver()`把组件包装起来）

看一个例子，在下面的四块代码中，我们在APP里做一个计数器，它每点击一次、数字加一。

```tsx
// 这是一个UI组件，无状态
function Counter(props): JSX.Element {
  const { value, onIncreaseClick } = props;
  return (
    <div>
      <span>{value}</span>
      <button onClick={onIncreaseClick}>Increase</button>
    </div>
  );
}
```

```tsx
// 将UI组件封装为容器组件
export const CounterWrapper = connect(
  (state) => ({
    value: state.count,
  }),
  (dispatch) => ({
    onIncreaseClick: () => dispatch({ type: 'increase' }),
  }),
)(Counter);
```

这样，我们就写好一个组件。后续使用时直接用容器组件。

接着创建Redux对象：

```typescript
// Reducer函数
function counterReducer(state = { count: 0 }, action) {
  const count = state.count;
  switch (action.type) {
    case 'increase':
      return { count: count + 1 };
    default:
      return state;
  }
}
// store实例
const store = createStore(counterReducer, { count: 0 });
```

然后装进App组件里试用一下：

```tsx
export function App(): JSX.Element {
  return (
    <div id={'app'}>
      <Provider store={store}>
        <A />
      </Provider>
    </div>
  );
}

function A(props): JSX.Element {
  return (
    <div>
      <CounterWrapper />
      <CounterWrapper />
    </div>
  );
}
```

> UI组件可以复用，例如在本例中，可以写一个 CounterWrapper2 ，将其他state或者action传递进去。

## Redux简析

### 核心原理

`Redux`的核心原理，很像JS中的`reduce()`函数，每次传入 上次的state + 本次变化量action ，返回一个新的state对象。

即使你只改了1个状态量，你也要将其他所有状态量浅拷贝一遍。这可以是个优势，方便你追踪状态的变化过程，甚至记录历史状态随时倒回；也可以是个劣势，因为产生了很多无用的拷贝和脏检查。

action的标识符是字符串！用字符串来做路由真的是个很糟糕的事情，会让IDE的代码跳转功能失效。

然后这里还有一个无法回避的巨坑：`dispatch()`函数是同步的。因此如果要做一个异步的action（业务中常见的从后端取数据），那就需要借助一定的手段来实现。（不过这个问题Mobx也有，都是通过回调其他action来实现）

### 响应式原理

`react-redux`的原理也能够想象了。第一，它只将state中的一部分取出来，作为props传给UI组件，这样UI组件不会因为无关的状态量的变化而更新；第二，它将dispatch的参数包装成了函数，并结合容器组件中的状态去实现真正的action逻辑。

这个部分的最大问题，就是在`connect()`这个中间层额外定义了一些东西，而这些定义的东西的类型信息没办法支持 typescript 传递到props中去，就算我们手动地添加类型声明，也会失去类型校验的效果。 

### 注入原理

在Redux的世界里，只有一个全局的store，并且只有一个`Provider`将其注入到应用中去。

这意味着必须把所有的东西都塞在一起，这个问题很严重。

举个例子，我们一个项目组的一个项目的一个页面，就可能有 十几个state + 几十个action ；照此估算，如果全公司的前端都用Redux来写，项目组之间的代码冲突先不说，光是这个代码文件恐怕就得几千几万行。

作为对比，Mobx的注入方式，是可以有多个store，每个store有自己的Provider，只注入自己；后代通过`useContext`取出对应的store，过程很透明，类型提示也完美。

## 总结

从原理上来说，二者最大的区别，应该就是`Redux`的状态对象是不可变的，每次都返回一个新的状态树。

但实际上，在`Reducer`函数中，还是靠人肉做浅拷贝的。因此如果想要在`Mobx`中利用不可变对象做一些事情（例如状态可视化、历史状态保存等），其实做起来是一样的。

所以总的来说，在我能想象到的场景中，`Mobx`大于等于`Redux`。

这是我的个人见解，不过，我目前看过的博客文章里大多数人也都是支持Mobx的，我们公司项目选型也是Mobx，所以这个结论还是比较可信的。
