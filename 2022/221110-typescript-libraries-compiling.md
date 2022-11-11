```yaml lw-blog-meta
title: "Typescript库的三种编译打包方式"
date: "2022-11-10"
brev: "使用 Typescript 等非原生语言编写的第三方库组件，应该如何编译打包为方便用户使用的样子？ cjs, esm, umd 三种形式。"
tags: ["前端"]
description: "将 Typescript 编写的第三方库编译打包为 cjs, esm 和 umd 三种形式"
keywords: "typescript,library,complie,bundle,cjs,esm,umd"
```

## 背景

最近仔细研读了 [flv.js](https://github.com/Bilibili/flv.js/) 这个库的源代码。说实话，一开始我还觉得挺惊艳的，可是越看越是让我皱眉头。它有一些代码设计上和具体实现上的问题，但是我现在想要专门挑出来讲的，是它不完全的类型体系。

我想，不管是作为读者还是作为项目维护者，一套健康的类型系统都是必要的。而`flv.js`的做法是，单独另外维护了一套`d.ts`文件来定义类型，而具体的代码实现依然是原始的js 。也许有它的历史原因，但是作为一个 Star 21.6k、Used-By 5.4k、google搜索排名前列的流行开源项目，这样的完成度，（感谢归感谢，但是），我觉得本可以做得更好的。

但如果要配上类型系统，例如Typescript，的话，对于这样的开源库来说，其实还是需要考虑更多兼容性因素的。可能有的同学已经注意过，例如在配置通过外部CDN引入`react`库的时候，CDN平台其实是提供了许多文件的，例如`cjs`, `umd`等等，就是为了兼容性的考虑。

参考：[Compiling and bundling TypeScript libraries with Webpack](https://marcobotto.com/blog/compiling-and-bundling-typescript-libraries-with-webpack/) 这篇文章介绍了如何将Typescript编写的项目构建为开发者常用的三种格式。

下文内容有一部分是直接翻译自上述文章，大部分是我参考上述文章后、结合其他开源项目和我司业务的实际情况所做的总结。

## 三种输出

1. 使用`tsc`编译输出的源码 + `*.d.ts`类型声明文件 + sourcemap文件。模块语法将使用`CommonJS`以便于支持大部分的打包工具。（即`cjs`）
2. 与1相同，但是使用`ES6`的模块语法。（即`esm`）
3. 一份`umd`的输出。即编译为标准ES5语法，可以直接在浏览器中工作并且在全局变量(window)上挂载自己。

通常来说`umd`版本的输出并不需要类型定义文件，因为类型定义文件(.ts)并不符合`umd`的定义。但是为了调试方便，我们依然打算输出类型定义文件。

我们使用简单的`tsc`就可以实现前两种输出，再使用`webpack`即可实现第三种。

## 初始化workspace

> 不用 npm workspace 也可以正常做编译配置，不感兴趣的同学可以跳过本章。

`react.js`这个项目，虽然也有一些让我觉得难受的地方，但是依然有很多配置都非常的先进，非常值得参考学习。特别是对`workspace`这个工具的使用。更入门的可以参考[Getting Started with npm Workspaces](https://ruanmartinelli.com/posts/npm-7-workspaces-1)

通过下面的命令可以在当前工程目录下创建一个新的`workspace`：

```shell
npm init -w packages/video
```

配置完成之后，目前工程的核心文件结构如下所示：

```text
.
├── package.json
└── packages
    └── video
        └── package.json
```

注意，在`./package.json`文件中，有一行很重要的配置，它决定了哪些目录要被当作`workspace`来处理：

```json
{
  "workspaces": [
    "packages/*"   // 刚init之后，显示的应该是"packages\\video"
  ]
}
```

此时，使用`yarn workspaces info`命令，可以列举查看所有的workspace的情况。

在后续使用`workspace`的时候，例如`npm run`或者`import from`的时候，用到的名字并不是物理上的目录名称或者目录路径，而是每个`package.json`中所定义的名字。

## 准备一些TS代码

例如，创建`packages/video/src/index.ts`文件，并在其中写一个类、一个方法，记得要标注类型哦。

## 配置tsc

既然要用到`tsc`，那就少不了`tsconfig.json`这个配置文件。

具体的配置项不一一展开，像我这样配置即可输出`esm`版本的构建产物：

```json
{
  "compilerOptions": {
    "module": "es6",
    "target": "es5",
    "lib": ["esnext", "dom"],
    "outDir": "dist/esm",
    "sourceMap": true,
    "declaration": true,
    "skipLibCheck": true
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules"]
}
```

如果给`tsc`指定运行时参数，则可以构建出`cjs`的产物：

```shell
tsc -m es5 --outDir dist/cjs
```

此时我们得到了两套构建产物：

```text
dist
├── cjs
│   ├── index.d.ts
│   ├── index.js
│   └── index.js.map
└── esm
    ├── index.d.ts
    ├── index.js
    └── index.js.map
```

## 被其他包引用

假如，我想在另一个包名为`web`的workspace中调用上面创建的`video`包里的代码。

首先，`video`这个包本身需要声明它的产物要去哪里找。配置方法是在`video/package.json`里去写：

```json
{
  "name": "video",
  "version": "1.0.0",
  "main": "dist/cjs/index.js",
  "module": "dist/esm/index.js",
  "types": "dist/esm/index.d.ts"
}
```

> 上面所展示的三个路径，与前文所说`tsc`的配置是相关联的。此外，"types"配置项也可以不写，这样 typescript 会自己尝试去找`d.ts`类型声明文件。

然后我们去`web`包里安装对`video`包的依赖。

就跟普通的添加依赖的方式一样。不过，`video`包并不需要发布到npm之类的托管环境(registry)上去，直接在本地就可以进行引用，类似`yarn link`的能力。但是在使用过程中要注意，必须指定确定的版本号（否则`yarn`会去registry上去查询当前最新版本列表）。

```shell
yarn workspace web add video@1.0.0
```

随后，我们就能在`web`包中调用`video`包的代码，并且IDE也能正确地提供类型提示、源码跳转功能。

但要注意的是，此时被引用的是构建后的产物，而不是源码本身。因此在后续开发过程中，如果对`video`的源码有了修改，那就需要重新运行`tsc`进行构建。

## 构建umd

安装相关工具：

```shell
yarn workspace @meideng/bo-fe-video add webpack@5.49.0 webpack-cli@4.9.2 ts-loader terser-webpack-plugin uglify-js
```

> 注：根据[webpack5官方文档](https://webpack.js.org/plugins/terser-webpack-plugin/) ，webpack5中已经自带了 TerserWebpackPlugin ，一般情况下可以不需要自行安装、配置。但是在本例中，只希望对`umd/xxx.min.js`做 minify 操作，而希望 `umd/xxx.js` 保留原样，因此这种特殊需求要自行配置才能实现。

直接贴上`webpack.config.js`的完整配置：

```js
const path = require('path');
const TerserPlugin = require('terser-webpack-plugin');

module.exports = {
  mode: 'production',
  devtool: 'source-map',
  entry: {
    'video': './src/index.ts',  // 名字随便起
    'video.min': './src/index.ts',  // 名字随便起
  },
  output: {
    path: path.resolve(__dirname, 'dist/umd'),
    filename: '[name].js',
    libraryTarget: 'umd',
    library: 'MyVideoPackage',  // 名字随便起
    umdNamedDefine: true,
  },
  resolve: {
    extensions: ['.ts', '.tsx', '.js'],
  },
  plugins: [],
  module: {
    rules: [
      {
        test: /\.ts$/,
        loader: 'ts-loader',
        include: [path.resolve('src')],
      },
    ],
  },
  optimization: {
    minimize: true,
    minimizer: [
      new TerserPlugin({
        minify: TerserPlugin.uglifyJsMinify,
        include: /\.min\.js$/,
        terserOptions: {
          sourceMap: true,
        },
      }),
    ],
  },
};
```

用上面的配置构建，会输出四个文件：

```text
video.js
video.js.map
video.min.js
video.min.js.map
```

然后我们就可以试验一下，在一个web项目的HTML中，用常见的CDN的引入方式来引入这个文件：

```html
<script defer crossorigin="anonymous" src="/umd/video.min.js"></script>
```

随后我们就能在`window`上找到`MyVideoPackage`这个对象，能够调用它的方法，并且devtool也能正确地加载sourcemap便于我们调试。
