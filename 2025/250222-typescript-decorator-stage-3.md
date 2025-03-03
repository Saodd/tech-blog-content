```yaml lw-blog-meta
title: "Typescript Decorator Stage3"
date: "2025-02-22"
brev: "2023年开始 Typescript5.0 版本开始支持最新的装饰器语法 2022.3/Stage3"
tags: ["前端"]
```

## 背景

在[《控制反转、依赖注入与Nestjs》](../2024/241018-ioc.md)这篇文章中我提到过，这段时间我一直在研究设计模式及其在前端中的运用。我在Nest.js框架中见到了很多装饰器的用法，必须承认，这种语法糖确实对简化代码有比较明显的帮助。因此，今天当我想要尝试把相同的语法搬到前端项目中的时候，我也想自己试一试装饰器的用法。

关于装饰器本身的语法我不做详细解释了，它也不是JS/TS独创的概念，早在N年前我刚入行的时候就已经在Python语言中做过很多实战了。简单说它就是个闭包而已，或者用前端熟悉的话来说叫HOC高阶函数。

不过说实话我还从未在JS项目中亲自实现过一个装饰器。因此我需要复习一下语法。因此我看到了这篇文章：[TypeScript 装饰器](https://typescript.p6p.net/typescript-tutorial/decorator.html) 。（这里先吐槽一下，这个网站应该是盗版的，可是它却出现在bing搜索的第二名，在google上更是第一页搜不到，也是有点搞笑，盗版比正版运营得更好是吧。）

我花了一个蹲坑的时间快速地看了一遍，嗯嗯很简单嘛，我会了！出来就开始写，但是，很快我就懵逼了，编译没有报错，可是运行时怎么第二个参数`context`的值是`undefined`啊？！

于是我做了亿点点调查，记录如下。

## 旧版本装饰器

以前（准确地说是2023年以前，TS4.0及以前）我们是怎么用装饰器的呢？

首先需要配置`tsconfig.json`，这样在开发过程中，tsc编译器就不会报错了，配置如下：

```json
{
  "compilerOptions": {
    "experimentalDecorators": true
  }
}
```

然后对于前端项目，我们需要对 webpack/rspack/vite 等构建工具进行配置，这样打包后的js代码也能正确运行了，以下示例 rspack 配置：

```js
module.exports.module.rules = [
    {
        test: /\.(ts|tsx)$/,
        use: {
            loader: 'builtin:swc-loader',
            options: {
                jsc: {
                    parser: {
                        syntax: 'typescript',
                        tsx: true,
                        decorators: true,
                    },
                },
            },
        },
        type: 'javascript/auto',
    }
]
```

配置完成，这样我们就可以写装饰器语法的代码了，示例代码如下：

```ts
// 定义一个类装饰器
function logClass(constructor: Function) {
    console.log(`Class ${constructor.name} has been created.`);
}

// 使用装饰器
@logClass
class MyClass {
    constructor() {
        console.log('MyClass instance created.');
    }
}

// 创建类的实例
const instance = new MyClass();
```

注意，装饰器只有一个参数。

## 新版本装饰器

> 参考阅读：[Announcing TypeScript 5.0](https://devblogs.microsoft.com/typescript/announcing-typescript-5-0/#decorators)

从2023年发布 TypeScript 5.0 开始，最新版本的装饰器语法（完整名称为"2022.3/Stage 3"，以下简称为新版）已经实装。

简单概括一下，语法上的最大变化，就是装饰器函数提供了第二个参数：

```ts
// 定义一个带有上下文参数的类装饰器
function logClassWithContext(constructor: Function, context: ClassDecoratorContext) {  // 这里提供了第二个参数
    console.log(`Class ${constructor.name} has been decorated at ${context.name}.`);
}

// 使用装饰器
@logClassWithContext
class MyNewClass {
    constructor() {
        console.log('MyNewClass instance created.');
    }
}

// 创建类的实例
const newInstance = new MyNewClass();
```

我们可以在`lib.decorators.d.ts`文件中找到相关的定义，这里节选其中一个接口作为例子：

```ts
interface ClassDecoratorContext<
    Class extends abstract new (...args: any) => any = abstract new (...args: any) => any,
> {
    readonly kind: "class";
    readonly name: string | undefined;
    addInitializer(initializer: (this: Class) => void): void;
    readonly metadata: DecoratorMetadata;
}
```

为了开启新版装饰器语法，我们需要配置编译时`tsconfig.json`：

```json
{
  "compilerOptions": {
    "experimentalDecorators": false /* or just remove the flag */
  }
}
```

还需要配置运行时`rspack.config.js`：

```js
module.exports.module.rules = [
    {
        test: /\.(ts|tsx)$/,
        exclude: /[\\/]node_modules[\\/]/,
        use: {
            loader: 'builtin:swc-loader',
            options: {
                jsc: {
                    parser: {
                        syntax: 'typescript',
                        tsx: true,
                        decorators: true,
                    },
                    transform: {
                        decoratorMetadata: true,
                        decoratorVersion: '2022-03',  // 开启此项
                    },
                },
            },
        },
        type: 'javascript/auto',
    },
]
```

关于`swc-loader`的详细配置，可以在[schema.json](https://swc.rs/schema.json)或者[文档](https://swc.rs/docs/configuration/compilation#jsctransformdecoratormetadata)中查询。

如果使用`babel`，则配置如下：

```json
{
    "plugins": [
        [
            "@babel/plugin-proposal-decorators",
            {
                "version": "2023-05"
            }
        ]
    ]
}
```


## 新旧版本的区别

在TS官方文档中是这样说的：

1. 需要关闭`--experimentalDecorators`，默认已开启新版语法支持。
2. 不兼容`--emitDecoratorMetadata`，不支持参数装饰器`decorating parameters`，未来的提案也许会补充支持。

但是在实际使用中，由于语法的变化，实际体验还是有区别的。

现在新版语法可以以更优雅的方式来做一些事情，例如我觉得有个很有趣的例子如下：

```ts
function twice() {
  return (initialValue) => initialValue * 2;
}

class C {
  @twice
  field = 3;
}

const inst = new C();
inst.field; // 6
```

也有一些功能由于缺乏`--emitDecoratorMetadata`而导致无法实现了，例如下面这个运行时类型检测器：

```ts
import "reflect-metadata";
 
class Point {
  constructor(public x: number, public y: number) {}
}
 
class Line {
  private _end: Point;

  @validate
  set end(value: Point) {
    this._end = value;
  }
}
 
function validate<T>(target: any, propertyKey: string, descriptor: TypedPropertyDescriptor<T>) {
  let set = descriptor.set!;
  
  descriptor.set = function (value: T) {
    // 这里在运行时得到了 Point 这个类型，随后可用 instanceof 进行检测
    let type = Reflect.getMetadata("design:type", target, propertyKey);
 
    if (!(value instanceof type)) {
      throw new TypeError(`Invalid type, got ${typeof value} not ${type.name}.`);
    }
 
    set.call(this, value);
  };
}
 
const line = new Line()
line.start = new Point(0, 0)
 
line.end = {}
// 运行报错：
// > Invalid type, got object not Point
```

一些基于旧版装饰器实现的三方库，不能直接兼容新版装饰器语法了。例如`mobx`这个库（[文档](https://mobx.js.org/enabling-decorators.html)和[讨论](https://github.com/mobxjs/mobx/discussions/3373)），示例代码如下：

```ts
class HelloStore {
  constructor() {
    makeObservable(this);
  }

  @observable count: number = 0;
  @action setCount = (v: number): void => {
    this.count = v;
  };
}

// 运行时报错：
// [MobX] Please use `@observable accessor count` instead of `@observable count`
```

看起来只需要添加一个`accessor`就可以从旧版迁移到新版语法，（实际已经挺讨厌的了），但mobx文档中也提到了这个关键词会将 旧版的`property` 改变成 新版的`get/set method`，可能会产生一些额外的影响，需要谨慎迁移。

## 总结

我用新版装饰器写了一些Demo，体验下来感觉不错，特别是Typescript类型支持很完整，没遇到什么问题。

总体来说，对装饰器的使用者，也就是大多数日常业务场景，有一些影响，但是影响不大。现阶段主要影响的是装饰器的实现者，也就是一些第三方库的作者，需要尽快跟进兼容。

考虑到 ECMAScript标准 的工作流程，当一个语言特性进入 Stage3 意味着已经几乎非常稳定了，不太可能会有颠覆性的变化了，因此这里建议大家，可以开始考虑迁移或者在新项目中采用新的装饰器语法了。
