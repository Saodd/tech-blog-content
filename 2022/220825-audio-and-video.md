```yaml lw-blog-meta
title: "[译]Audio and video manipulation"
date: "2022-08-25"
brev: "web中的音视频操作技术"
tags: ["前端", "音视频"]
```

## 原文信息

[Audio and video manipulation](https://developer.mozilla.org/en-US/docs/Web/Guide/Audio_and_video_manipulation)

翻译时间：2022年8月25日

翻译内容根据译者自身的知识水平做了一定的增删改，但是也尽可能地保留了原文含义。

## 前言

web技术迷人的地方在于，你可以将旧的技术组合起来，做成一些新的东西。在浏览器中拥有了原生的音视频（能力）之后，我们可以用这些数据流与传统的技术，例如`canvas`, `WebGL`或者`Web Audio API`，结合起来，以此达到直接修改音视频的能力。例如，给音频添加混响（reverb）或者压缩（compression）特效，给视频添加灰度（grayscale）或者老照片（sepia）特效，等等。这篇文章会告诉你应该怎么做。

## 视频操作

能从视频中读取每一`帧`（frame）的`像素值`（pixel values）的能力会是非常有用的。

### video 与 canvas

`<canvas>`元素提供了一个让人可以在web页面上`绘制图形`（drawing graphics）的能力。它非常强力并且可以紧密地与视频相结合。

一般做法是：

1. 从`<video>`元素中取出一帧，传入`<canvas>`元素。
2. 从`<canvas>`元素中读取数据并进行修改。
3. 将修改后的数据传入“显示用的”`<canvas>`元素中（从效果上来说可以是同一个元素）
4. 暂停或者重复上述操作。

举个例子，我们尝试将一个视频以灰度的形式进行显示。在这个例子中，我们将会同时显示原始视频以及经过灰度处理之后的视频（的每一帧）。不过一般来说，如果你正在实现一个“以灰度模式播放视频”的功能，那你应该会需要给`<video>`元素添加一个`display:none`属性，来隐藏原始视频并只显示经过转化后的视频（的帧），即那个`<canvas>`元素。

> 译者注：原文提供的示例代码是原生HTML+js，我这里直接改写为React了。

```tsx
const MyVideo: React.FC = () => {
  const videoRef = useRef<HTMLVideoElement>();
  const canvasRef = useRef<HTMLCanvasElement>();

  const handlePlay = useCallback(() => {
    const video = videoRef.current;
    const canvas = canvasRef.current;

    const context = canvas.getContext('2d');
    const width = video.width;
    const height = video.height;

    const computeFrame = () => {
      context.drawImage(video, 0, 0, width, height);
      const frame = context.getImageData(0, 0, width, height);
      for (let i = 0; i < frame.data.length; i += 4) {
        const grey = (frame.data[i + 0] + frame.data[i + 1] + frame.data[i + 2]) / 3;
        frame.data[i + 0] = grey;
        frame.data[i + 1] = grey;
        frame.data[i + 2] = grey;
      }
      context.putImageData(frame, 0, 0);
    };

    const timerCallback = () => {
      if (video.paused || video.ended) {
        return;
      }
      computeFrame();
      setTimeout(() => {
        timerCallback();
      }, 16); // roughly 60 frames per second
    };
    
    timerCallback();
  }, []);

  return (
    <div>
      <video ref={videoRef} controls width="480" height="270" crossOrigin="anonymous" onPlay={handlePlay}>
        <source src="https://jplayer.org/video/webm/Big_Buck_Bunny_Trailer.webm" type="video/webm" />
        <source src="https://jplayer.org/video/m4v/Big_Buck_Bunny_Trailer.m4v" type="video/mp4" />
      </video>

      <canvas ref={canvasRef} width="480" height="270"></canvas>
    </div>
  );
};
```

（预览效果略）

这个简单的例子展示了如何通过一个canvas来操作视频帧。一个小的性能优化是，你可以考虑用`requestAnimationFrame()`来代替`setTimeout()`。

当然，你也可以简单地通过给`<video>`元素设置`grayscale()`这个CSS属性来达到同样的效果。（不过这就不是本文讨论的内容了）

### video 与 WebGL

WebGL是一种强力的API，它依然使用canvas，但它可以使用硬件加速来绘制3D或者2D场景。你可以将WebGL与`<video>`标签结合起来创建视频纹理，也就是说你可以把视频放进3D场景中去。

参考：[源代码](https://github.com/mdn/dom-examples/tree/master/webgl-examples/tutorial/sample8)

### 播放速率

我们可以通过`<audio>`和`<video>`标签的`playbackRate`属性来调整音频、视频的播放速率。

注意`playbackRate`这个属性，它只能修改播放速率（playback speed）而不会修改音调（pitch）。如果需要修改音调的话需要使用 Web Audio API，参考[AudioBufferSourceNode.playbackRate](https://developer.mozilla.org/en-US/docs/Web/API/AudioBufferSourceNode/playbackRate).

## 音频操作

包括`playbackRate`在内，一般来说，操作音频你需要用到[Web Audio API](https://developer.mozilla.org/en-US/docs/Web/API/Web_Audio_API)

### 不同的音频源

Web Audio API 可以接受多种类型的音频源，它会处理原始数据然后将其发送到一个`AudioDestinationNode`对象中，这个对象代表着输出设备。

| 如果音频源是……                            | 使用这种类型                      |
|-------------------------------------|-----------------------------|
| `<audio>`或者`<video>`元素中的一条音轨（track） | MediaElementAudioSourceNode |
| 一份原始的音频数据缓冲区                        | AudioBufferSourceNode       |
| 波形（oscillator）                      | OscillatorNode              |
| WebRTC音轨                            | MediaStreamAudioSourceNode  |

### 音频过滤器

Web Audio API 有很多不同的过滤器（filter）和特效（effects）。例如`BiquadFilterNode`（二阶滤波器）。

> 译者注：原文提供的示例代码是原生HTML+js，我这里直接改写为React了。

```tsx
const MyVideo: React.FC = () => {
  const videoRef = useRef<HTMLVideoElement>();

  useEffect(() => {
    const context = new AudioContext();
    const audioSource = context.createMediaElementSource(videoRef.current);
    const filter = context.createBiquadFilter();
    audioSource.connect(filter);
    filter.connect(context.destination);

    filter.type = 'lowshelf';
    filter.frequency.value = 1000;
    filter.gain.value = 25;
  }, []);

  return (
    <div>
      <video ref={videoRef} controls width="480" height="270" crossOrigin="anonymous">
        <source src="https://jplayer.org/video/webm/Big_Buck_Bunny_Trailer.webm" type="video/webm" />
        <source src="https://jplayer.org/video/m4v/Big_Buck_Bunny_Trailer.m4v" type="video/mp4" />
      </video>
    </div>
  );
};
```

常见的音频过滤器：

- Low Pass: 允许低于cutoff频率的的频率通过，高于的减弱
- High Pass: 允许高于cutoff频率的的频率通过，低于的减弱
- Band Pass: 允许一个范围内的频率通过，范围外的频率减弱
- Low Shelf: 增强或者减弱较低的频率
- High Shelf: 增强或者减弱较高的频率
- Peaking: 增强或者减弱一个范围内的频率
- Notch: 屏蔽一部分频率
- Allpass: 改变不同频率之间的相位关系

### 卷积与脉冲

> 译者注：关于这两个音频术语，可以参考[卷积混响 - FL Studio 20 参考手册](https://www.image-line.com/fl-studio-learning/fl-studio-online-manual-zh/html/plugins/editortool_reverb.htm) 或者 [卷积混响 - 虚幻4 文档](https://docs.unrealengine.com/4.26/zh-CN/WorkingWithMedia/Audio/ConvolutionReverb/)

卷积（Convolutions）与脉冲（impulses）

使用`ConvolverNode`可以给音频施加脉冲响应。『脉冲响应』是在短暂的脉冲声（如拍手）之后产生的声音，它可以代表当时创建那个脉冲声音的环境（例如在隧道中鼓掌的回声）。

（示例代码略，因为缺少脉冲数据）

### 立体声

使用`PannerNode`可以给声音定位。它允许我们定义一个『源音锥』（source cone）以及位置、方向等元素，所有这些都定义在3D坐标系所描述的3D空间中。

```tsx
const MyVideo: React.FC = () => {
  const videoRef = useRef<HTMLVideoElement>();

  useEffect(() => {
    const context = new AudioContext();
    const audioSource = context.createMediaElementSource(videoRef.current);
    const panner = context.createPanner();
    panner.coneOuterGain = 0.2;
    panner.coneOuterAngle = 120;
    panner.coneInnerAngle = 0;

    panner.connect(context.destination);
    audioSource.connect(panner);

    // 把监听位置放在音源的右侧，此时通过耳机可以听到声音从左侧传来
    context.listener.setPosition(0.2, 0, 0);
  }, []);

  return (
    <div>
      <video ref={videoRef} controls width="480" height="270" crossOrigin="anonymous">
        <source src="https://jplayer.org/video/webm/Big_Buck_Bunny_Trailer.webm" type="video/webm" />
        <source src="https://jplayer.org/video/m4v/Big_Buck_Bunny_Trailer.m4v" type="video/mp4" />
      </video>
    </div>
  );
};
```

### JS解码器

在js环境中，在底层对音频进行直接操作也是可行的。如果你想自己实现一个解码器（codecs）的话这就有用处了。

## 参考阅读

教程（略）

参考（略）
