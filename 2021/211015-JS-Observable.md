```yaml lw-blog-meta
title: "JS撸一个Observable对象"
date: "2021-10-15"
brev: "理解Proxy"
tags: ["前端"]
```

## 基础知识：三剑客

> 本章内容翻译并摘抄自 [MDN](https://developer.mozilla.org/)

### Symbol

顾名思义，它就是「标志符」。它的作用，是在JS运行时全局创建一个唯一的标识符。

它是JS语言中的七大「原始类型 [primitive](https://developer.mozilla.org/en-US/docs/Glossary/Primitive) 」之一。

也就是说，它是「不可变的 immutable」。

也就是说，可以用typeof：

```js
typeof Symbol()  // 'symbol'
```

之前我们在[React Fiber](../2021/211011-React-Fiber.md#准备知识：术语) 关于Element的部分中见过这个用法。再扯远一点，更早之前的 [Go源码之context](../2019/191126-Go源码之context.md#4-方法四：withvalue) 中提过，`WithValue()`方法最好使用一个独特的唯一值，例如`new(int)`，就等同于这里Symbol的效果。

注意它不能用`new`关键字，而是直接调用创建`Symbol()`。它可以接受参数，但是这个参数没有实际意义，大概只会在控制台Debug的时候看到输入值。每次调用它创建一个新Symbol对象，都是全局唯一且互不相等的。

```js
Symbol()     // Symbol()
Symbol('a')  // Symbol(a)
Symbol('a') === Symbol('a')  // false
```

但是还有某个全局的储存空间，你可以用 `Symbol.for()` and `Symbol.keyFor()` 来设置和查询可以通过key来复用的Symbol ，就像是在用一个Map一样。

```js
Symbol.for('a') === Symbol.for('a')  // true
```

### Reflect

顾名思义，「反射」，就是用来检查底层对象的工具。

它是一个JS的内建对象，就像`Math`那样用。它基本上就配合`Proxy`来使用。

### Proxy

顾名思义，「代理」，它接受一个对象，然后返回一个经过包装后的新对象。

```js
new Proxy(target, handler)
```

参数一`target`是需要被包装的对象，可以是任何值、可以是另一个Proxy。参数二`handler`是指定哪些方法需要被代理。

也许你可能会像我一样，想到Python的魔法方法，然后觉得我自己写一个封装类也可以实现呀，为啥要内建这个对象呢？——事实是，JS并没有像Python那样提供整套的魔法方法，至少对于一些操作符来说，你只能通过Proxy来实现，典型的就是getter和setter 。

```js
const handler = {
  get: function(target, prop, receiver) {
    if (prop === "proxied") {
      return "replaced value";
    }
    return Reflect.get(...arguments);
  }
};
```

如果你也是Mobx或者其他一些响应式状态框架的使用者，那你应该也经常见到Proxy这个东西，当你尝试通过`console.log()`打印某个状态量的时候很可能就会看见它……（好烦！）

## 手撸一个Observable

> 本章内容参考自 [MOBX原理与实践](https://juejin.cn/post/6850418118968377357)

先理解一段代码，它体现了Mobx的核心用法：

```typescript
const obj = observable({
    a: 1,
    b: 2
})

// autorun 函数是来运行状态后更新后引发的 reaction
autoRun(() => {
    console.log(obj.a)
})

obj.b = 3 // 什么都没有发生
obj.a = 2 // observe 函数的回调触发了，控制台输出：2
```

也就是说，`autoRun()`可以自动识别出自己监听了哪些状态量，或者准确来说应该反过来，是`observable`知道自己哪些属性被谁监听着。

前面已经介绍过了`Proxy`，现在我们可以想象一下它是如何实现的：

- `obj.a`这个行为，会触发Proxy的getter方法
- 但是在执行`obj.a`的时候，并没有传入函数本身，所以我们需要做一个「陷阱 trap」，能够把外层的函数捕获到。
- 把捕获到的、使用了`obj.a`的函数，在Proxy中，与`a`属性关联起来。
- 当遇到setter方法的时候，重新调用那个函数

按照测试驱动开发的原则（狗头），我们先明确最终目标：

```typescript
const store = new Proxy(
  { a: 0, b: 0 },
  {}
)
function autoRun(view: () => void): void {
}

autoRun(() => {
  console.log('the value of a is:', store.a);
})
store.b = 1
store.a = 2
```

第一步，为了捕获到读取了store的属性的函数，我们需要把`autoRun`传入的函数临时挂载在某个地方，思路很明确，把函数本体保存起来，然后开始执行函数，执行过程中访问了哪些属性就都能知道了。

```typescript
let trapping: () => void = null
function autoRun(view: () => void): void {
  trapping = view
  view()
  trapping = null
}
```

这会导致`autoRun()`在声明时就先执行一次，这是符合预期的。

第二步，我们需要一个东西，保存每个属性捕获到的函数，这里直接就用Map来实现：

```typescript
const obMap = new Map<string | symbol, Set<() => void>>()
```

第三步，我们在Proxy里拦截getter，也就是前端同学们喜欢说的所谓加钩子：

```typescript
const store = new Proxy(
  { a: 0, b: 0 },
  {
    get: function(target, prop, receiver) {
      if (trapping) {
        if (!obMap.has(prop)) obMap.set(prop, new Set<() => void>());
        obMap.get(prop).add(trapping)
      }
      return Reflect.get(...arguments);
    }
  }
)
```

在上面代码中，通过`if (tapping)`，把初始化阶段（捕获阶段）的行为 与 后续正常使用的行为 区分开来。

到这里可以简单运行验证一下，打印出`obMap`对象出来看看，确认正确捕获。

第四步，向setter里加钩子：

```typescript
{
  set: function(target, prop, value) {
    const res = Reflect.set(...arguments);
    if (!trapping) {
      obMap.get(prop)?.forEach((view) => view());
    }
    return res;
  }
}
```

这里有个小坑，`Reflect.set`是有返回值的，所以要把返回值原样扔回去。

完成！可以用各种姿势去测试（~~好像有点意犹未尽是怎么回事……~~

主要原理就是这样了，在此基础上再做一些封装（多实例、装饰器、兼容性等），再写很多脏代码（去除重复等），应该就能做成Mobx的样子了。

> 其实，根据文档所说，Mobx从5.0开始才正式使用Proxy 。
