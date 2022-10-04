```yaml lw-blog-meta
title: "[译]Using the Screen Capture API"
date: "2022-09-22"
brev: "简单玩一下，webRTC能力之一"
tags: ["前端"]
```

## 译者注

我第一次接触到web端的『屏幕捕获』能力，是在牛客网上做笔试题的时候。当时觉着眼前一亮，HTML居然还有这样的能力，我居然从未听说过。

前几天有个后辈折腾过这些东西，他在前端的部分走通了，但是在“后端转发并在其他前端上展示”这个环节上卡住了。

所以，也是作为『视频』相关能力调研的一部分，今天我亲自来尝试一下。

## 原文信息

[Using the Screen Capture API](https://developer.mozilla.org/en-US/docs/Web/API/Screen_Capture_API/Using_Screen_Capture)

翻译时间：2022年9月22日

翻译内容根据译者自身的知识水平做了一定的增删改，但是也尽可能地保留了原文含义。

## 捕获屏幕内容

通过调用`navigator.mediaDevices.getDisplayMedia()`来将屏幕内容捕获为一个实时的『媒体流`MediaStream`』，它的返回值被包在一个Promise里。

基本用法（除了下面展示的异步写法，还可以改为await/async写法）：

```js
// 译者注：参数可以传null
function startCapture(constraints) {
 return navigator.mediaDevices.getDisplayMedia(constraints)
    .catch((err) => { console.error(`Error:${err}`); return null; });
}
```

> 译者注：`.getDisplayMedia`这个API，必须要更新`typescript`到`4.4`以上的版本才能够支持，否则会报错，[参考](https://stackoverflow.com/questions/65123841/getting-property-getdisplaymedia-does-not-exist-on-type-mediadevices-in-an)

执行上面的代码后，用户代理（user-agent，即指浏览器）会显示一个用户界面来提示用户去选择一个要分享的屏幕区域。

### 配置参数 Options and constraints

[参阅](https://developer.mozilla.org/en-US/docs/Web/API/MediaTrackConstraints#properties_of_shared_screen_tracks) 查看约束可用的属性值。（译者注：或者直接从typescript的定义去看）

与其他媒体API的约束不同，这里的约束（constraints，即传入`getDisplayMedia`的参数）它仅用于定义流配置，而不是过滤可用的选择。在选择要捕获的内容之前，不会以任何方式生效任何约束。约束会改变您在结果流中看到的内容。

例如，假如你指定了一个`width`参数，它只会将用户选择的屏幕区域缩放到这个指定的宽度，而不会限制用户只能选择这个宽度的区域，也不会影响用户可用的选择项。（防止应用通过只让用户有一个选择来诱导用户进行选择）

当屏幕捕获正在运行的时候，设备上会显示一些内容以提示用户。另外考虑到安全和隐私因素，`enumerateDevices()`不会枚举屏幕录制资源，`devicechange`事件也不会由`getDisplayMedia()`触发。

### 可见界面与逻辑界面

『显示面`display surface`』指的是任何可以被屏幕共享API选择作为内容的东西。包括一个浏览器Tab，一个完整的窗口，一个应用的多个窗口，一个显示器，一组显示器。

显示面有两种类型。

一种是『可见显示面』，例如一个最前端的窗口，或者整个屏幕。

一种是『逻辑显示面』，它可能全部或者部分不可见。对这类显示面的处理可能会有不同的实现。一般来说，浏览器会在不可见区域用图片（模糊、色块、花纹等）遮挡起来。这是出于安全考虑的。用户代理（浏览器）在用户同意之后也可以提供被遮挡的逻辑显示面区域。

### 捕获音频

`getDisplayMedia()`一般用来捕获画面内容（录像），但是呢，用户代理（浏览器）也可以允许录制音频。

音频的来源，可以是被选择的窗口（程序）、整个电脑的音频系统、用户麦克风，或者上述来源的组合。

示例用法：

```js
const constraints = { video: true, audio: true }
```
```js
const constraints = {
  video: {
    cursor: "always"
  },
  audio: {
    echoCancellation: true,
    noiseSuppression: true,
    sampleRate: 44100
  }
}
```

## 使用捕获到的流

拿到`MediaStream`就去用吧。

### 潜在的风险

围绕着屏幕共享而产生的隐私和安全问题大体来说算不上非常严重，但它们确实存在。最大的潜在问题是用户无意中分享了他们不希望分享的内容。

一个严谨的用户代理（浏览器）应当默认遮挡屏幕上不可见的区域。

## 示例：简单捕获并展示

把流放进一个`<video>`标签中进行展示。

代码并不算复杂，而且如果你曾经用过`getUserMedia()`来获取摄像头数据的话，那你会感觉`getDisplayMedia()`非常熟悉。

> 译者注：这里我直接把它改写为React了。关键代码在于`srcObject`这个属性，它并不被React所支持，需要手动赋值一下。

```tsx
export function App(): JSX.Element {
  const handleStart = useCallback(() => {
    navigator.mediaDevices
      .getDisplayMedia({ video: true, audio: true })
      .then((stream) => {
        videoElement.srcObject = stream;
      })
      .catch((err) => {
        console.error(err);
      });
  }, []);
  const handleStop = useCallback(() => {
    (videoElement?.srcObject as MediaStream)?.getTracks().forEach((track) => track.stop());
    videoElement.srcObject = null;
  }, []);

  return (
    <div id={'app'}>
      <button onClick={handleStart}>开始</button>
      <button onClick={handleStop}>结束</button>
      <video ref={(v) => (videoElement = v)} width={1080} height={720} autoPlay controls />
    </div>
  );
}
```
