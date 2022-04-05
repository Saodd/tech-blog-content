```yaml lw-blog-meta
title: "前端 mono repo 实战"
date: "2022-04-05"
brev: "npm workspace"
tags: ["前端"]
```

## 背景

大概从半年前开始，我们项目组内部孵化了一个创新项目，从技术上来说主要就是做一个浏览器插件帮用户做一些操作，前后端都由我一手操刀。从客观结果来说，我们在没有任何运营投入的情况下获得了稳定的自然用户增长，从这个角度可以说我们的产品是成功的。

现在我所在的公司的经营状况不断加速下坡，于是公司划分出了独立的探索事业群，全部精力投入探索拯救公司的道路。可是现在都已经2202年了，在这个互联网早就被玩烂了的时代，没有前期的铺垫，想要空手创业谈何容易？于是我们这个新项目就成为了整个部门的独苗，全村唯一的希望。

说到这里我得发发牢骚。想当初我们刚起步的时候经历了多少来自管理层的质疑？全靠我们产品经理姐姐抗下了压力，做了各种文档和汇报忽悠住了领导；再加上我在较少投入的情况下完成了关键技术突破并提供巨量的产出，才让这颗小树苗活了下来，得到了茁壮生长的机会。就这样还没完，这颗小树苗还一度被分到其他项目组，差点被其他项目组搞死了，最后我有点生气了才很强硬地抢回来。

只能说，冰冻三尺，非一日之寒。

这个项目在我们的推动下终于走上了正轨，下一阶段的目标就是正式产品化，从技术上来说就是在原来一个浏览器插件的基础上再做一个B端网页，拓展更丰富的能力。

那么问题来了：一端的代码要在两端复用，我该如何处理其中的工程化问题？

## 什么是 Mono Repo

`mono`是一个词根前缀，表示`单个，一个`，典型例子有：monarch 君主，monopoly 垄断。

`mono repo`的意思就是一个代码仓库，指多个项目的代码放在一个(git)仓库里进行管理。

典型例子，例如有一个全栈项目 web + nodejs 两端的代码就可以放在一个仓库里；再例如我这个 web + 插件 两端的代码也可以放在一起。

选择它，最显而易见的理由就是为了**代码复用**，其次还能优化一些运维上的工作流程。

与MonoRepo相对的就是`MultiRepo`了，顾名思义，就是把代码放在多个代码仓库里，通过其他包管理工具（例如`npm`）来互相引用。

『核心区别是在于你相信什么样的代码结构可以让你的团队拥有最高的效率。』

## MonoRepo的解决方案

### 一个仓库内划分目录

一个最简单的想法，既然多个项目都是相同的技术栈，那我就把代码放在一起就好了，不需要额外配置什么。

甚至连代码目录都未必需要分开。

例如，在使用`webpack`的场景下，我只需要配置多个`entry`就行了；或者需要分开打包的话，我写多个`webpack.config`然后根据需要选择一个运行即可。

这种方式简单而有效。缺点就是项目之间会有更多的耦合。我认为对于一些复用程度非常高的项目（例如模板化的前端工程复用）用这一套会比较合适。在实践中，我也见到我们公司有团队使用这种方案，没什么毛病。

### Lerna

[官网](https://lerna.js.org/) 它定义自己是：一个多包项目管理器（A tool for managing JavaScript projects with multiple packages），它这样自我介绍：

『把代码划分为多个独立的包，这件事对于代码复用来说是非常有用的。然而，在多个代码仓库之间修改代码的工作流程非常复杂和恶心。为了解决这类问题，有些项目把它们的代码放在包含多个包的一个大仓库内，典型例子有 Babel, React 等等。』

用法上呢，类似`npm`和`webpack`，全局安装使用：

```shell
$ npm install -g lerna
$ lerna init
```

初始化项目后会得到这样一个目录结构：

```text
lerna-repo/
  packages/
  package.json
  lerna.json
```

更多的我就没有尝试了，我只是简单的看了下同事用过，也没有兴趣尝试。而且，现在中文社区里讲MonoRepo一般都是用的这个lerna，相关文章一搜一大把，我不再赘述。

### workspace

如果用英文搜索，那么会发现与中文环境下完全不同——大部分解决方案都是这个`workspace`。

它是`npm`从v7版本开始提供的最新特性。 [官方文档](https://docs.npmjs.com/cli/v8/using-npm/workspaces) （我之前默认安装的npm是v6+，似乎是node.js的v14的默认版本，而当前npm最新版本v8.6）

它的原理与lerna是相同的（[参考](https://classic.yarnpkg.com/lang/en/docs/workspaces/#toc-how-does-it-compare-to-lerna) ），只不过它直接由`npm`提供，更加官方、可靠。

在实际使用中，`yarn`提供的命令与`npm`稍有不同，具体需要看看[文档](https://classic.yarnpkg.com/en/docs/cli/workspace) 。IDE方面，我使用的`Webstorm`可以完美支持，没有遇到任何痛点。

## npm workspace 主要用法

### 1. 初始化

在一个空的仓库下，我们先初始化这个仓库（作为一个根包）（记得要设置为private项目），然后再初始化一个子包：

```shell
$ npm init
...
$ npm init -w ./packages/a
```

于是我们得到一个这样的目录结构：

```text
+-- packages
   +-- a
   |   `-- package.json
+-- package.json
```

在`./package.json`文件中最重要的一个字段，指定了哪些路径需要被识别为这个仓库的子包：

```json
{
  "workspaces": [
    "packages/a"
  ]
}
```

可以通过命令来查看相关信息：

```shell
$ yarn workspaces info
yarn workspaces v1.22.17
{
  "a": {
    "location": "packages/a",
    "workspaceDependencies": [],
    "mismatchedWorkspaceDependencies": []
  }
}
Done in 0.04s.
```

### 2. 添加依赖

我们开始开发，那第一步应该就是添加依赖对吧，这里先手动添加一个依赖作为示例：

```shell
$ yarn workspace a add axios
```

这个命令，会将`axios`这个依赖添加到`a`这个子包里（即写入`packages/a/package.json`中），但是代码则会下载在根目录的`./node_modules`路径下（因此多个子包的相同依赖可以在根目录共享）。

现在我们有了这样的目录结构（省略了部分）：

```text
+-- node_modules
   +-- a
   +-- axois
+-- packages
   +-- a
   |   `-- package.json
`-- package.json
`-- yarn.lock
```

> 这里有个小细节，`node_modules/a`这个东西它不是一个普通的目录，而是一个类似软链接的东西。我之前一直以为在windows平台下是没有Unix中的软链接的概念的，关于它的具体特性以后我再研究一下。

### 3. 添加另一个子包

```shell
$ npm init -w packages/c
```

通过上面的命令我们会创建一个`packages/b`的目录，然后

```shell
$ yarn
```

之后`b`包会出现在`node_modules/b`。（理解为`b`包是根目录这个包的依赖。）

顺带一提，其实根目录的`package.json`可以简化为一个目录加通配符：

```json
{
  "workspaces": [
    "packages/**/*"
  ]
}
```

### 4. 跨包调用

现在a包里有`axios`，而b包里没有，我们可以在b包里调用a包的封装。

先在`a`包里写一个被调用的函数：

```js
// packages/a/main.js
const axios = require('axios');

exports.run = () => axios.get("http://baidu.com");
```

然后在`b`包里去调用它：

```js
// packages/b/main.js
const a = require('a/main');

a.run().then(resp => console.log(resp.data))
```

> 值得一提的是，`packages/b/main.js`既可以在根目录下执行，也可以在`b`目录下执行（由于`node_modules`的向上查找的机制）。

在实际项目中，我们更可能会把多个项目的公共依赖提取出来，放在（一个叫common这类名字的）单独的子包里进行管理。

### 5. 封装命令

我们可以在`b/package.json`里封装好命令：

```json
// packages/b/package.json
{
  "scripts": {
    "run": "node main.js"
  }
}
```

这个命令可以封装在根目录中，直接在根目录调用所有子包的命令会是比较方便的实践：

```json
// package.json
{
  "scripts": {
    "run:a": "npm run -w a run",
    "run:b": "npm run -w b run"
  },
}
```

## 其他一些细节提醒

一定记得升级`npm`到 >=7 ！

一些配置文件，例如`.eslintrc.json`, `.prettierrc`等文件都可以放在根目录下，这些代码风格的东西常常是可以跨项目共享的。

`global.d.ts`这类定义文件也可以放在根目录下，但是相应地，要在`tsconfig.json`里注册这个文件才会让代码检测工具识别到：

```json
// tsconfig.json
{
  "include": [
    "packages/**/*",
    "global.d.ts"
  ]
}
```

跨包引用资源时，webpack配置中也要记得添加对应的包的路径（否则会提示没有对应的loader），例如：

```js
{
  module: {
    rules: [
      {
        test: /\.(js|jsx|ts|tsx)$/,
        include: [path.resolve('src'), path.resolve('../a/src')],
      }
    ]
  }
}
```

推荐搭配`git lfs`工具，仓库更干净！
