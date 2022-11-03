```yaml lw-blog-meta
title: "[译] Digital audio concepts"
date: "2022-10-31"
brev: "译自MDN文章，详细介绍了数字音频相关理论"
tags: ["音视频"]
description: "译自MDN文章，详细介绍了数字音频相关理论"
keywords: "Digital,audio,concept,音频,理论"
```

## 译者按

我们知道，视频其实是由一帧一帧的图片组成的，只不过是由于快速连续播放而产生了"动画"的效果。因此，在某一个特定的"时刻"，这个视频对应的图像是可以被精确定位出来的。

那么音频又是怎么一回事呢？仔细想一想，声音应该是无法被静止下来的，即，我们无法想象一个"静止"的声音应该是什么样子的。

有一些概念的同学可能会知道，声音其实也就是"波形"，然后把连续的声音波形转化为数字化的"音阶"，就得到了数字音频数据。但是要注意的是，一个"静止"的声音，并不是连续播放某个等级的音阶，因为声音是振动，要能被听到那么它一定就是正在变化的，如果不变的话，那就什么也听不到了，也就没有意义了。

接下来看看这篇MDN的文章，是否能够解答我们的疑惑。

## 原文信息

[Digital audio concepts](https://developer.mozilla.org/en-US/docs/Web/Media/Formats/Audio_concepts)

译文有一定的增删改，以及附加的引用、解释，但尽量保留了原意。

"把声音以数字形式来表达"这件事涉及了许多步骤和处理过程，特别是在web中还存在着多种格式来表达『原始音频 raw audio』和『编码音频』。这篇文章大概介绍了在web中声音是如何表达、编码、解码的。

## Sampling audio 采样

音频是自然世界固有的模拟特征。当一个物体（微粒）振动的时候，它会引发周围的物质微粒也一起振动；这些微粒又引发它周围的微粒振动，如此不断传递，就在物体周围形成了一种形式的『波 wave』，从源向外传播振动，直到波的振幅随距离增加而逐渐消失。因此，声波的『粒度 granularity』其实也就是它传播『介质 medium』的微粒的间隔。

一般来说，日常生活中的声音基本都是通过空气来传播的。

人们听到的声音，实际上是空气微粒的振动引发了耳朵内部（耳膜）的工作。空气微粒在振动时移动的幅度越大，『波幅 amplitude』就越大，也就是音量越大。微粒振动得越快，声波的频率就越高。

![sound wave](https://developer.mozilla.org/en-US/docs/Web/Media/Formats/Audio_concepts/audio-waveform.svg)

然而，电脑是数字化的。因此需要『模数变换 analog to digital conversion』（简称 A/D）的过程来把声音转化为计算机可以处理的数字形式。

影响音频采样的『保真度 fidelity』的第一个因素是『音频带宽 audio bandwidth』，即模数变换过程中能够容纳和捕获的频率范围，这个会受具体的『编解码器 codec』的影响，有的codec会把不能接受的信息丢弃掉。

声音通过麦克风等设备进入电脑的时候，是连续的电压信号变化的形式，电压代表振幅（即音量）。然后，这个模拟信号被电路转换成数字形式，以指定的间隔，将数据转换成音频记录系统可以理解的数字形式。每个被捕获的时间段（moment）被称为『样本 sample』。

![sound sample](https://developer.mozilla.org/en-US/docs/Web/Media/Formats/Audio_concepts/audio-waveform-samples1.svg)

在上图的例子中，蓝色代表采样，黑色代表原始音频波形。在一个特性间隔里（蓝色的每个横线部分），模数转换器必须从波形中选择一个值来作为当前采样的值（来代表整个采样周期内的声音）。算法可以有多种，上图采用的是时间中点对应的值，此外还可以考虑算平均值。

等电脑需要播放音频数据的时候，实际被播放的是蓝色的数字波形，它只是对原始音频的一种粗糙的模仿（也就是说这个过程中有失真）。

如果你的采样间隔越短，你就能得到更加接近原始波形的数字信号。每秒采样的次数称为『采样率 sample rate』。

### Audio data format and structure

从技术底层来看，音频是由一系列采样数据组成的，每个采样代表一个采样周期内的振幅。

> 译者注：一个sample其实也就是一个数字，例如uint16，因此一段音频数据实质上就是几千、几万、几百万个(uint16)数字连续排列所构成的。

采样的格式有多种，大多数文件使用`int16`来表示一个采样，此外也有`float32`, `int24`, `int32`等其他形式，以及现在web中已经不再使用的`int8`格式。每个采样的大小（即bit数量）称为『采样大小 sample size』。

每个声源在音频信号中的位置称为『声道 Channel』。音频数据可以包含多个声道，声道的数量称为『Channel count』，例如『立体声 stereo』就有左、右两个声道。在某个特定时间（采样周期）内，每个声道上分别会有一个采样数据(sample)。

当生成多声道的音频文件时，声道会被组织成『音频帧 audio frame』，每一帧里包含每个声道的一个采样。

目前网络上主流的音频参数是：双声道 + 16bit采样尺寸 + 48kHz采样频率（即每秒48000次采样），这样计算下来，每秒钟的音频数据是 192kB ，一首典型的3分钟的歌曲则需要 34.5MB 的空间。因此音频数据需要压缩。

> 译者注：原文中似乎把 "sample" 与 "audio frame" 两个术语混用了。

声音的压缩和解压的过程，用到的工具就是所谓的『编解码器 codec』。为了了解web开发中常见的codecs，参考阅读：[Guide to audio codecs used on the web](https://developer.mozilla.org/en-US/docs/Web/Media/Formats/Audio_codecs) 。

## Audio channels and frames

声道有两种类型。标准声道，用于呈现大部分可听到的声音，例如左右立体声。另一种特殊的『低频增强 Low Frequency Enhancement (LFE)』声道，给特殊的播放设备使用，例如低音炮。

『单声道 Monophonic』有一个声道， 『立体声 stereo』有两个，『5.1环绕』有六个声道（5个标准声道+LFE声道）。一个 16bit 双声道音频的一帧(frame)的尺寸是 32bit ，即 4byte 。

注：某些编码器可能会将多个声道的数据分开储存，即，在物理上多个声道的采样数据并不是被放在一起的，但是在逻辑上依然是每个声道的采样集合起来才叫一个帧。

常见的采样率有：

- 8000 Hz: 国际标准[G.711](https://zh.wikipedia.org/wiki/G.711)定义了电话通信使用的采样频率，这个频率足够分辨人类语音。
- 44100 Hz: CD的采样率。CD记录的是未被压缩的 44.1kHz 16bit 立体声 。这也是计算机默认使用的频率。
- 48000 Hz: DVD的采样率。计算机也经常会用这个频率。
- 96000 Hz: 高解析音质
- 192000 Hz: 超高分辨率音频。目前尚未大规模使用，但这可能会随着技术的发展而改变。

44.1kHz 也被称为"最低的高保真采样频率"，它有背后的理论支持。首先根据[采样定理](https://zh.wikipedia.org/wiki/%E9%87%87%E6%A0%B7%E5%AE%9A%E7%90%86)，数字信号的频率至少是模拟信号的两倍才能准确地再现模拟信号，考虑到人类能听到的频率是 20~20000Hz ，计算得到最低高保真频率应该是 40kHz 。但是为了给[低通滤波器](https://zh.wikipedia.org/wiki/%E4%BD%8E%E9%80%9A%E6%BB%A4%E6%B3%A2%E5%99%A8)提供额外的空间，以避免由[混叠](https://zh.wikipedia.org/wiki/%E6%B7%B7%E7%96%8A)引起的失真，于是增加了2.05 kHz 的[过渡带](https://en.wikipedia.org/wiki/Transition_band)，最终计算得到 44.1kHz.

## Audio compression basics

与文本等其他类型的数据不同，音频数据往往是"嘈杂的(noisy)"，或者说是"没有规律的"，即在 byte, bit 这种微观层面上，很少会出现几段完全相同的字节数据段落。也就是说，传统的压缩算法很难处理音频数据。例如`zip`，它的工作原理是把重复出现的段落用一个更短的代号来表示。

大多数 codec 经常用到几种常用的压缩技术。

例如一种最简单的方式，你可以使用一个过滤器，将所有静音的片段移除掉（并用一些记号来代替它们）。同理可以用于一些重复的、或者近似于重复的片段。

另一种简单的方式，你可以用一个过滤器来降低音频带宽（采样率、声道、位宽），这在语音通话场景非常有用（前面提过最低8kHz即可听清语音）。

### psychoacoustics 心理声学

如果你知道你正在处理的是什么类型的声音，你可能找到适合这种类型的声音的某种特别的技术，以此来优化编码算法。

最常用的压缩算法，应用的是[心理声学(psychoacoustics)](https://zh.wikipedia.org/wiki/%E5%BF%83%E7%90%86%E5%A3%B0%E5%AD%A6) 的理论。这门科学主要研究人类是怎样理解声音的（例如哪段声音频率对人类来说很重要），以及人类会对声音做出怎样的反应（基于环境和声音内容）。

基于心里声学，有可能设计出一种压缩算法，能够最小化压缩体积并且尽可能保留最大的『听觉保真度 perceived fidelity』

### Lossy vs lossless compression

看压缩目标的要求，根据是否丢失部分细节数据，可以分为『有损 lossy』和『无损 lossless』压缩两类。

大多数编码都是有损算法，可以压缩到原始体积的5-20%；现代的无损压缩则可以压缩到40-50%的原始体积。

这个研究领域依然活跃，隔一段时间就可能有新的算法被提出。

## Psychoacoustics 101

> 译者注："101"是"入门教程"的意思

心理声学的细节在本文中不展开讨论，但是简单了解个大概，将有助于你选择更适合的codec

以人类语音通话的场景为例。人类语言的频率在300-18000Hz范围，但是，大多数人日常说话的频率在500-3000Hz这个小范围内，因此你可以过滤掉这个范围以外的采样数据，而依然保证语音能够被听清楚。由于这个和一些其他的因素，语音通话这个场景因此能够在低码率的情况下达到"高保真"效果（"足够高保真"效果）

![Human Speech](https://developer.mozilla.org/en-US/docs/Web/Media/Formats/Audio_concepts/human-hearing-range.svg)

> 译者注：参考阅读：[人声的频率到底范围在多少？](https://www.zhihu.com/question/264642786)

更多关于如何选择codec的知识，请参考[Web audio codec guide](https://developer.mozilla.org/en-US/docs/Web/Media/Formats/Audio_codecs)

## Lossless encoder parameters

## Lossy encoder parameters

比特率：平均比特率 (Average bit rate), 可变比特率 (Variable bit rate)

频率带宽 (Audio frequency bandwidth)

联合立体声 (Joint stereo)。普通的简单立体声(simple stereo)是分别储存两个声道的采样数据。而联合立体声，考虑到左右声道的音频是非常相似的，则使用一个"基础声道"，外加一个更小尺寸的"辅助声道"。联合立体声有两种处理方式，(Mid-side stereo coding) 和 (Intensity stereo coding)
