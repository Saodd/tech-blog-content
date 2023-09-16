```yaml lw-blog-meta
title: "js的类能否继承自null？"
date: "2023-09-15"
brev: "一个有趣的话题"
description: "js class extends null"
tags: ["前端"]
```

# 背景

`class`关键字以及相关的面向对象的编程模式，是es6引入的典型语法之一，这个应该大部分同学应该会用，或者至少八股是背过的，应该没有什么疑问。

但是今天我突发奇想：如果我想要一个“干净”的类，即排除掉`Object`类所附带的那些`toString()`、`isPrototypeOf()`等方法的类，那我应该怎么写这个类？

试着写了一下`class A extends null`，嗯，编译器没有报错；但是深究一下，我发现事情没有想象得那么简单。

参考文章： [How and why would I write a class that extends null?](https://stackoverflow.com/questions/41189190/how-and-why-would-i-write-a-class-that-extends-null)

# 哪些能写、哪些不能写？

首先是最基础的写法，没有任何花里胡哨：

```ts
class A extends null  {}
```

这个类本身被正常地声明了，但是它却无法被实例化为对象：

```ts
a = new A()
/**
 * chrome 和 node.js 中的运行结果：
 VM524:1 Uncaught TypeError: Super constructor null of A is not a constructor
   at new A (<anonymous>:1:1)
   at <anonymous>:1:5
 */
```

看起来是对象实例化的时候，`super()`遇到了问题，可是我们又不能强制不写super，像下面的写法依然无法运行：

```ts
class A extends null {
    constructor() {}
}

a = new A()
/**
 VM591:2 Uncaught ReferenceError: Must call super constructor in derived class before accessing 'this' or returning from derived constructor
   at new A (<anonymous>:2:16)
   at <anonymous>:1:5
 */
```

上面报错的原因是没有追溯到原型链顶层，因此没有挂上`this`，于是我们可以尝试按照报错提示，自己return一个作为this：

```ts
class B extends null {
    constructor() {
        return Object.create(null)
    }
}

b = new B();  // 运行结果：{}
```

ok，像上面的写法得到的`b`对象，确实达到了我们的初衷，也就是得到一个“干净的”对象。但是再深入挖掘一下，会遇到一个很诡异的问题：

```ts
b instanceof B  // false
```

不仅`b`不是`B`的实例，同时`b`也无法访问在`B`中声明的成员函数，这都是原型链不完整所导致的。

为了解决这个问题，我们可以参照原帖中的回答，用一个特别的原型来实例化对象：

```ts
class C extends null {
    constructor() {
        return Object.create(C.prototype)
    }
    
    call() {
        console.log('hello!')
    }
}

c = new C();  // C {}
c instanceof C;  // true
c.call;  // function
c.constructor;  // function
```

此时的`c`对象，只有两个成员方法，一个是`C`类中声明的方法`call`，一个是类的原型所携带的`constructor`，可后者是我们并不想要的东西，是多余的。

底层原因是js的类是由“原型链”的机制来模拟实现的，而不像其他语言那样类本身与对象有本质上的区别（[参考](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide/Using_classes)）。因此不论如何尝试，都不可能做出一个“完全干净”、同时还能保持“面向对象”功能的对象。

# 那么，正确的语法是什么？

参考[es7规范文档](http://www.ecma-international.org/ecma-262/7.0/#sec-class-definitions)，其中定义了，`extends`关键字后面跟随的应当是`LeftHandSideExpression`，中译名**左值表达式**，

熟悉c++的语言的同学应该都了解所谓左值表达式，在js中举个例子，下面的例子会抛出运行时异常：

```js
null = 123;  // Uncaught SyntaxError: Invalid left-hand side in assignment
```

或者：

```js
function A() {return {}};
A() = 123;  // Uncaught SyntaxError: Invalid left-hand side in assignment
```

> 参考阅读：[如何理解左值](https://segmentfault.com/q/1010000012807803)

简而言之，`null`和函数直接的返回值不能作为左值进行赋值，会抛出运行时错误；但是，虽然es7规范中说`extends`右侧只能是左值表达式，chrome的实现却可以用`null`和函数返回值。null的例子前面已经说过了，再看一下函数返回值的写法作为例子：

```js
function A() {return Object};
class B extends A() {};  // 没有抛出异常！
a = new A();  // 没有抛出异常！
```

# 结论

不能。

js类的功能的实现必须依赖一些特殊的成员属性/成员方法，因此无法创造出一个完全干净又带有完整面向对象能力的类。“js的类能否继承自null”这个思路也没有实际价值，有其他更好的思路可以替代。

js规范是不支持继承自null的，但实际chrome引擎的实现是部分允许的。
