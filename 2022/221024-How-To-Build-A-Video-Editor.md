```yaml lw-blog-meta
title: "[译] How to build a video editor"
date: "2022-10-24"
brev: "VEED.IO自述：如何打造一个视频编辑器"
tags: ["前端","音视频"]
description: "VEED.IO自述：如何打造一个视频编辑器"
keywords: "video,editor,视频编辑器,web,browser"
```

## 原文信息

[How to build a video editor](https://www.veed.io/blog/how-to-build-a-video-editor/)

作者： [SABBA KEYNEJAD](https://www.veed.io/blog/author/sabba/) from veed.io

发布时间：2020-12-21

翻译内容有一定的精简。

## 简介

我们的在线视频编辑器 VEED.IO 是我们完全自研的产品。我们团队的研发工程师总共不到10人。在过去两年的时间里，虽然有些地方我们没有做好，但我们依然打造了一个我们曾以为不可能的产品——一个基于云的视频编辑器。

一路走来，我们遇到了许多障碍，并且多次把代码推翻重写，最终才找到了我们觉得能行的路子。

今天我们将与你分享我们在研发过程中的故事，希望能给你一个关于在线视频编辑器的技术选型的大概印象。

## 一些历史

（译者注：他们的团队成员应该是有大量的视频内容产出工作经历的）在使用那些复杂臃肿的软件来进行视频编辑，长达数千个小时之后，我们厌烦了。我们就疑惑了，为什么市面上的这些编辑工具，要附带如此多的不必要的复杂功能，最后却拿来去做简单的事情。

所以大概2.5年以前（2018年），我们开始打造自己的简单的在线视频编辑器。

大多数重量级的编辑工具，都是被设计为用来产出"好莱坞级别"的大片的，而不是我们日常生活中的社交媒体中的内容。因此我们意识到，"一个简单而有力的在线视频编辑器"的需求应当是存在的——一个任何人都能在几分钟内上手的工具。我们没在市场上找到符合我们需求的软件，因此我们"愚蠢地"决定自己造一个。

在研发过程中，我们踩了许多的坑，这些坑浪费了我们数百、上千个小时的工作时间。

## 第一步：基本知识

首先需要明确"视频创作"(video creation)这个概念它的核心基础到底是什么，然后是哪些技术可以采用。

什么是视频？很简单：视频是一连串的图像（所谓的"frames"，帧）以一定的速率（所谓的"frame rate"，帧率）连续地向你展示，这个过程中会（在你的视网膜上）营造出"动画"的幻觉。

![Video Frames / Frame Rate](https://ghost-veed-blog.s3.eu-west-2.amazonaws.com/2020/12/videoFrames.png)

如果把视频速度降低到一定程度（小于24帧每秒），我们看到的就不再是"视频video"了，而是"幻灯片"了（也就是所谓的"卡成PPT"）。

### 视频剪辑所需的功能

最重要的功能之一是『Trim （剪辑）』。当你想移除视频中的某一段的时候，技术上来说需要做的就是把那一段时间里的『frames 帧』给删除掉。反过来，增加一些frames就是视频拼接的效果了。

> 译者注：我们日常语境中说的"视频"，一般是包含了 图像+声音 的、从技术上来说是有多个轨道的视频文件。在专业术语中，"video"指的是图像部分，"audio"指的是声音部分。

讨论另一种情况，我们希望在一段视频内部添加一些东西，例如一个图片或者字幕文本。由于视频仅仅只是一些"帧"，我们需要做的只是把需要添加的元素盖在帧图像的上方，然后渲染出新的"帧"来。

这就是"视频剪辑"的基本逻辑了！但在实际中，事情没有那么简单。不同的视频之间差别巨大，例如 宽高比(aspect ratios)、色彩表示(color representation)等等，有不同的 编码(codecs)和格式(formats) 。

为了兼容这些各种各样的差异，代码会非常难写。所以"理解我们应该如何处理这些边缘场景"这个事情非常重要。

## 技术实现

我们都虔诚地相信：在浏览器中打开一个网页就能开始视频编辑、这远比你先下载10+GB的巨型软件并花几个小时去学习才能开始视频编辑，要友好太多了。刚好我们就是web开发工程师，因此我们就开始了开发。

然后我们有两个巨大的挑战：首先我们需要一个后端（不一定是后端服务器、也可以是后台进程），这个后端能够生成高质量的视频；其次我们需要一个前端GUI，这个前端能够准确地模拟后端渲染的结果。

为了简化，我们打算先从前端搞起。有开源技术可以做这个事吗？我们决定在还没有人尝试过的情况下，先不要"过度设计"(overengineering)我们的技术栈。先做个MVP。

### Processing 框架

[Processing](https://processing.org/) 框架，是一个开源的、Java写的、具有创造性的编码工具包，非常易用，它看起来像这样：

![Building a Online Video Editor - Rendering With Processing](https://ghost-veed-blog.s3.eu-west-2.amazonaws.com/2020/12/Screenshot-2020-12-21-at-15.25.32.png)

这个开源项目的宗旨是，让人们通过做"视觉艺术"(visual arts)来接触编程。它的另一个好处是只需要编写脚本即可运行，对我们来说很方便；随后我们着迷于它海量的、开箱即用的工具库和易用性。它甚至还有一个`record`函数来直接把动画录制下来，而不需要我们自己写编码工具。使用它，我们很快就能搭建出一个超级简单的视频编辑器。

但它的问题在于，处理时间太长了。几秒钟的视频需要5分钟才能处理完。

此外，它还不能保留音频。虽然我们可以用另外的工具来处理音频，但是这样就不能与GUI无缝结合了。

### Phantom.js

然后我们想，既然想要让我们的系统与GUI无缝结合，那么我们可以考虑"在后端也运行前端代码"。

只需要在后端运行无头浏览器，把用户的指令重新运行一遍，然后录制下来就可以了。

无头浏览器，我们选择了 [Phantom.js](https://phantomjs.org/) （译者注：现在这个项目已经停止迭代了，可以选择其他的还活着的开源项目替代）。

使用无头浏览器的好处之一，是我们只需要写一份代码即可（前后端通用），并且由于它的API简单，我们很快就做出了一些功能来。

![Phantom.js](https://ghost-veed-blog.s3.eu-west-2.amazonaws.com/2020/12/carbon.png)

很快我们遭遇了Phantom.js的瓶颈：它不能提供对视频的细粒度的（基于帧的）控制，具体来说，Phantom最多只能处理1秒间隔的视频。

另一个重要的缺陷是，既然它在后端是回放，我们不得不等待回放播放完毕，这在视频时长很长的情况下就很糟糕了。

我们单纯地以为这些问题可以通过技术手段来解决，但很显然我们低估了它的难度。在与潜在用户沟通之后，我们意识到这样的产品是完全不够的。

### Adobe After Effects Render

我们注意到，已经有一些视频编辑网站，它们能够轻易地提供高质量的视频产出，甚至比今天的 VEED.IO 做得还好。唯一的问题就是模板很难修改，并且运行费用也难以控制。

这些网站使用的是 AE (Adobe After Effects Render) 或者 PR (Adobe Premiere Pro) 的SDK 。他们预设好了一批高质量的视频模板，用户只需要提供素材填入，即可产出视频。

这个方案的缺陷很明显：你只能接受这些模板，不能进行其他的修改。这样会限制用户的创意，这不是我们的初衷。更何况我们团队本身就是"内容创作者"(content creators)

### FFmpeg 与 C++

如果你是一名开发者并且对视频领域感兴趣，那你一定听过 [FFmpeg](https://ffmpeg.org/) ，它是一个功能完备的、跨平台的解决方案，可以录制、转化、流式处理(stream)音视频资源。（可以说，）如果没有它的存在，绝大多数基于浏览器的视频编辑工具都不会存在。它真是一个"天赐之物"(godsend piece)，类似"瑞士军刀"的存在，它用它丰富的API催生出了数十亿美金的产业。

我们必须赞美它的制作者， Fabrice Bellard 和 Michael Niedermayer ，如果你们碰巧正在读这篇文章，请接受我们的感谢，感谢你们所做的一切。

其实此时我们还尝试了另一个选择，[moviepy](https://zulko.github.io/moviepy/) 。但是经过多次失败的尝试之后，我们放弃了。事后回顾，幸好我们当时放弃了，否则很快我们又会遇到瓶颈了。

最后，我们的得出了一个艰难的结论：如果我们想要打造一个高质量的、足以与当前领域中的巨头进行竞争的 视频编辑软件，这个软件必须能够给我们足够的自由度去修改和拓展功能。这意味着我们需要"卷起袖子"(roll up our sleeves round)并且准备进入一个长时期的开发循环——这对于创业公司来说可是噩梦啊。

我们所做的，就是选择 C++ 作为编程语言来构建渲染逻辑。在这个过程中，我们深入挖掘了许多底层协议，这是我们以前从未接触过的。换句话说，使用 C++，我们现在有能力直接对接 `libavcodec` 和 `FFmpeg` 的C库，免去了cli作为中介。

在这个时间点，我们真心希望，我们现在放慢脚步，可以给未来培育出丰盛的果实。

### OpenCV

其实我们还还还尝试了另一个工具，[OpenCV](https://opencv.org/) ，它是最流行的开源计算机视觉库之一。使用它，我们可以用最少的代码来生成视频。

OpenCV 其实在底层也是在调用 `libavcodec` 和 `FFmpeg` ，换句话说，虽然它的初衷不是用来视频编辑，但它依然有大量优秀的能力可以用来视频编辑，而且开箱即用。

我们用它用了4-5-个月，但最后还是放弃了。

原因之一，我们这种云端的视频编辑器产品，非常依赖云端资源，我们必须密切关注资源成本。随着我们用户数量、服务器成本增加，我们决定放弃 OpenCV ，直接使用 FFmpeg 和 libavcodec 才会是最佳解决方案。

放弃OpenCV带来的挑战是，它会明显地增加我们的代码复杂度。但我们不得不做。因为我们确实需要能够直接处理所有不同格式的视频，这些格式与 libavcodec 稍有不同。

这也是为什么，在尺长上，没有其他竞品选择与我们相同的方案，至少我们没听说过有。为了初始化 libavcodec ，我们需要写大量的重复枯燥的代码，而且相关文档还很匮乏（大多数时候我们最终直接阅读源码了）。除了视频之外，音频也是与视频同样复杂的（特别是还想要将它"可视化"(visualizing)的时候），我们必须将它转化为AAC格式，这是我们的输出的标准编码格式。谢天谢地，此时 ffmpeg 也帮助我们同时使用 libavfilter ，它可以用来对音频做混合(mix)、切割(cut)和转化(convert)。AAC的严格控制，导致了我们的产品在渲染视频阶段超过50%的BUG和崩溃，但随着我们慢慢优化 libavfilter 的参数，我们最终能够处理几乎所有类型的文件了，甚至有时文件本身是损坏的时候我们也能够处理（这个场景其实不少）。

### 构建GUI (OpenGL 与 WebGL)

> 译者注： [OpenCV 与 OpenGL 的关系是什么？](https://www.zhihu.com/question/20212016) 注意理解 vision 与 graphics 的区别。

最后，我们做GUI了 。一个可能表面上看不出来的事情是，我们需要有2个视频渲染器同时运行，一个在前端提供预览，另一个在后端渲染出最终产出。最难的工作也就是如何让这两个部分无缝协同工作，即，所见即所得。

这个设计背后的主要概念有：

1. 用户仅在前端操作
2. 如果添加了图片或者其他附件，我们会把它们收集到我们的存储空间中
3. 用户的编辑操作，会转化为"指令集"(recipe of instructions)，这份数据会被传输到后端渲染器中去执行。
4. （后端的）C++渲染器 根据指令集来渲染出结果
5. 用户从前端获取渲染成品视频

实际上，做出这个东西来远比想象中要难。

最难的依然是如何保证"所见即所得"。我们稍微作弊了一下。首先是关于字体部分，我们把前端的字体作为图片附件上传。另外关于视频特效方面，我们利用了一部分 OpenGL/WebGL 的能力，虽然它们在前后端的实际效果可能会 有细微的差别，而这种差别往往是由于解码过程(decoding)而不是渲染器(renderer)的差异造成的。

> 译者注：作为我们公司产品『美折』旗下的"水印编辑器"模块的全栈研发、而且是其中字体模块的核心研发人员，我对这部分"作弊"的做法表示非常赞同。这确实是一个不错的思路。

[OpenGL](https://en.wikipedia.org/wiki/OpenGL) 是一个跨语言、跨平台的API，用于渲染2D和3D向量图形。[WebGL](https://developer.mozilla.org/en-US/docs/Web/API/WebGL_API)是同一个东西，它是给浏览器中的web应用使用的接口。我们使用这两项技术来渲染2D图形，或者更确切地说，渲染你的视频的"帧"。如果你熟悉 [Shaders](https://developer.mozilla.org/en-US/docs/Web/API/WebGL_API/Tutorial/Using_shaders_to_apply_color_in_WebGL) ，它可以用来给你的视频添加额外的"风格"（flavour, 或者说"特效"）。

"图形学"(Graphics)它本身是一门科学(science)，如果你对 shaders 和 Graphics 感兴趣的话，可以去 [Shader Toy](https://www.shadertoy.com/) 快速体验一下。

然后我们面临一个大难题：如何将大量的 webGL 逻辑与 React 框架结合在一起工作。实际上，我们的前端渲染器是我们的软件中最复杂的部分，甚至超过了 C++渲染器。这是因为前端有大量的时刻变化的元素，而且React本身其实并不适合这样的场景。

我们未来可能会在另一篇文章中介绍我们如何构建了前端部分的工作。

## 创意栈

虽然到目前为止，可以说我们打造的这个视频编辑器已经非常强大了，但其实我们依然偷了很多懒(shortcuts)，并且整个系统其实还并没有发挥出它的全部潜力。

我们依然持续地在优化渲染器的性能，来减少用户的等待时间。我们做了很多事情，例如：把C++渲染器作为独立节点运行并且每分钟动态扩容；使用更智能的CDN技术来减少前端页面的加载时间；以及和用户交谈，收集故障反馈和建议。

我们还有无数的想法等待尝试。例如，也许我们未来又会使用无头浏览器来渲染字体，或者我们能够用GPU来加速渲染过程，或者使用更加智能的图形引擎例如 [Vulkan](https://www.vulkan.org/) .

## 结语

我们相信，"视频编辑"这个领域的未来一定会在"云"上。经过两年的研发，我们的团队提供了强大的 video API 来帮助数百万的用户轻松创作和发布他们的视频内容。

另外，从一开始我们就一直热衷于与世界分享我们的最新发现和知识。我们也可以激动地宣布，如果你是开发者，你可以借鉴我们的技术和架构来创建自己的视频编辑工具，而不用再去踩那些曾经坑了我们很久的大坑。

（致谢）

## 译者：结语

虽然在现在这个时间点（2022年）来看，市面上已经有很多大厂做出了一些轻量级的剪辑工具（例如剪映、必剪）等，但是，这些产品都是app的形式存在的（虽然可能是浏览器套壳），而正经的像VEED.IO这样纯浏览器的解决方案的，还是非常少的。而且，回溯到2018年那个时间点去看，那时候我剪视频都还是在用Pr这类传统的重量级软件，那时甚至完全都不能想象这样的能力（的一部分）居然有朝一日能够只靠浏览器就能完成，从这个角度来看，VEED.IO 这个团队真的非常值得尊敬。

当然，他们的研发经历也有他们的局限。例如，选择Java体系的`Processing`和Python体系的`moviepy`，在我看来其实可以算是"外行"的；选择`Phantom.js`来控制无头浏览器，也远比不上`puppeteer`或者`playwright`来的专业；关于`ffmpeg`，如果放在现在，我会直接考虑`ffmpeg.wasm`，而这个东西是2019年才出现的，比VEED.IO创建的时间还更晚，也因此他们不得不选择了后端渲染方案。

我和我的团队，现在确实可能存在弯道超车的机会。不过我还是必须得感谢他们无私分享的精神，真的非常值得尊敬。我会争取以他们作为榜样，继续做更多的分享。

多说一句，他们这样无私奉献的工匠精神，与我们国内目前互联网行业的跑马圈地、利益至上的思想风气是截然相反的，看完这篇博客之后，我甚至有些"心灵受到净化"的感觉。这一点也真的非常值得我们这些互联网从业者去反思啊。
