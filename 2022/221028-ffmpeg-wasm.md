```yaml lw-blog-meta
title: "ffmpeg.wasm 踩坑体验"
date: "2022-10-28"
brev: "ffmpeg.wasm 的基本使用，相关配置，定制编译，以及前端使用时的一些坑"
tags: ["前端","音视频"]
description: "ffmpeg.wasm 的基本使用，相关配置，定制编译，以及前端使用时的一些坑"
keywords: "ffmpeg.wasm,js"
```

## 背景

都说在`npm`的加持下，前端js几乎可以做任何事情。"视频格式转化"这种在以往印象中很重、很专业的操作也可以开始考虑了。

在视频处理这个领域，最权威的莫过于`ffmpeg`了。而`ffmpeg.wasm`这个项目则把原本在终端中运行的它搬到了浏览器中。

入门参考阅读：[前端webassembly+ffmpeg+web worker视频抽帧](https://juejin.cn/post/6998876488451751973) ，这篇文章并没有写出我心中完美的解决方案，但是依然值得参考。

## 基本使用

照惯例，首先看看它的 [Github仓库](https://github.com/ffmpegwasm/ffmpeg.wasm)

首先安装依赖：

```shell
yarn add @ffmpeg/ffmpeg @ffmpeg/core
```

怎么理解上面这两个库呢，

`@ffmpeg/ffmpeg`是上层的封装，我们日常写代码使用的API都在这个库里。

而`@ffmpeg/core`负责底层与`wasm`的交互的部分。它这个仓库是从`FFmpeg/FFmpeg`这个C语言主仓库Fork出来的，它的工作就是将C代码编译成`wasm`代码，并且附带一些JS胶水便于上层的调用。

由一段代码来看看基本用法：

```ts
const ffmpeg = FFmpeg.createFFmpeg({
  mainName: 'main',
  // corePath: 'http://localhost:8080/static/ffmpeg-core.js',
});

async function flvToMp4(flv: ArrayBuffer): Promise<ArrayBuffer> {
    await ffmpeg.load();
    await ffmpeg.FS('writeFile', 'input.flv', new Uint8Array(buf));
    await ffmpeg.run('-i', 'input.flv', 'output.mp4');
    const mp4 = await ffmpeg.FS('readFile', 'output.mp4');

    return mp4.buffer;
}
```

先理解一下`FS`这个东西。由于`ffmpeg`（的cli）本身是在命令行中运行的程序，它原本处理的输入输出也都是本地文件系统中的文件。而在浏览器运行环境下，`FS`这个东西就是用来模拟本地文件系统的。上面的代码，顾名思义，先写入了一个`input.flv`文件，然后调用`ffmpeg.run`将它转化为了`output.mp4`文件，最后将MP4文件读取出来并返回。

`ffmpeg.run`的参数的写法，也与`cli`完全一致。（有这个特性，我们可以很方便地在多个运行环境中复用同一套参数。）

但是上面的代码，你可能第一次运行是会失败的，因为需要一些额外的配置：

### SharedArrayBuffer所需配置

你可能会见到如下报错：

```text
Uncaught (in promise) ReferenceError: SharedArrayBuffer is not defined
```

原因是在`@ffmpeg/core`里调用了一个叫做`SharedArrayBuffer`的东西，而参考 [SharedArrayBuffer - MDN](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/SharedArrayBuffer) 这个东西由于具有一定的危险性，因此需要 HTML文件 指定额外的 HTTP头 才能使用。

后端配置我不讲，前端devServer的配置如下：

```js
module.exports ={
    devServer: {
        headers: {
            'Cross-Origin-Opener-Policy': 'same-origin',
            'Cross-Origin-Embedder-Policy': 'require-corp',
        },
    }
}    
```

### 不用SharedArrayBuffer

上面说到，既然`SharedArrayBuffer`这个东西有危险性，而且修改运维配置可能还会涉及不少麻烦，那我们能不能不用它呢？

可以不用，但是代价是性能更低。要理解`SharedArrayBuffer`这个东西是用来在多线程之间共享内存空间的，如果不用的话，那么`ffmpeg.wasm`就只能工作在单线程的环境下了。

一边是高性能，一边是兼容性+安全性，我选择后者。为此，我们需要亲自构建`@ffmpeg/core`。

首先需要clone [ffmpegwasm/ffmpeg.wasm-core](https://github.com/ffmpegwasm/ffmpeg.wasm-core) 这个仓库，然后开始编译：

```shell
git submodule update --init --recursive
bash build-with-docker.sh  # 需要docker访问权限，在windows系统可能要在WSL中运行
```

这个编译脚本真的可以说是非常先进了，整个构建环境都是在docker中准备好的，一次跑通，没有任何痛苦。除了缓存方面有一丝丝槽点、以及一次构建需要30分钟之久的等待，之外，没有任何缺点。

对了，上面的命令构建出来的、默认参数版本的core，与NPM提供的内容其实是一样的；而我们需要单线程版本的core的话，编译命令需要加一个环境变量：

```shell
# ST 即 SingleThreading，单线程的意思
export FFMPEG_ST=yes; bash build-with-docker.sh
```

等待大约30分钟，然后我们可以在`wasm/packages/core-st/dist`路径下找到编译产物，复制到我们的前端项目中去，即可使用了。

## ffmpeg与MediaSource的配合

`MediaSource`这个东西非常的脆弱，一言不合就抛出异常，然后异常信息也毫无参考价值，非常难以debug 。

有个问题卡了我半天，我尝试用ffmpeg转化出来的视频片段二进制数据，放进`video`标签可以正常解析，但是放进`MediaSource`就会解析失败。

参考：[ffmpeg encode mp4 for HTML MediaSource stream](https://stackoverflow.com/questions/57350018/ffmpeg-encode-mp4-for-html-mediasource-stream)，解决方案是：

- 要指定`-movflags`为`empty_moov+default_base_moof+frag_keyframe`，生成纯正的MOOF格式
- 必需严格把视频和音频分离开，即用`-an`去除音频，用`-vn`去除视频；分离后的纯视频、纯音频轨道分别放入一个`SourceBuffer`里，并且`codecs`参数一定要写对。

此外还要注意一个很重要的东西：[SourceBuffer.mode](https://developer.mozilla.org/en-US/docs/Web/API/SourceBuffer/mode)

- `segments`(默认)：即根据视频片段中内置的`timestamp`来决定它的播放顺序；这种模式下，可以以任意顺序传入视频数据，播放时依然会以正确顺序播放。但是重点是，视频内的时间戳必须完好，会导致后面的片段覆盖前面的片段。
- `sequence`：仅根据传入视频片段的顺序来决定播放顺序。适用于视频内的时间戳已经错乱的情况。

## webpack构建优化

优化的重点，一方面在`Web Worker`这个东西的入口文件写法。在`Webpack5`中已经[内置了 worker-loader](https://webpack.js.org/guides/web-workers/)，开箱即用。

另一个方面，在于如何优化体积庞大的`ffmpeg.wasm`。具体做法无非就是懒加载、CDN、HTTP缓存这些老生常谈的东西，还有`webpack`打包时的 tree-shaking 似乎也没那么简单。

（以后填坑）

## wasm兼容性

[总体来看还是不错的](https://caniuse.com/?search=wasm)，从chrome57开始支持。

具体对于ffmpeg这个相对比较专业的工具来说，其实我觉得大胆一点——强迫用户使用受兼容的浏览器——应该也问题不大。
