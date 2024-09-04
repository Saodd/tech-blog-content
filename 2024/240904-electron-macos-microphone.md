```yaml lw-blog-meta
title: "electron 在 macOS 上启用麦克风录音"
date: "2024-09-04"
brev: "苹果真的难搞"
tags: ["前端"]
```

## 背景

业务需求在前端使用麦克风进行录音，我直接选用web技术，即主要是 `navigator.mediaDevices.getUserMedia` 和 `MediaRecorder` 这两个API ，这两个API虽然以前没怎么接触过，不过实际上对于chromium来说已经非常成熟了，直接用即可。

在浏览器端和windows客户端运行都表现正常，然后测试的时候发现，macOS不行。表现为：看起来是在录音了、但停止之后发现录下来的音频数据是空的（有size但是没有声音），且控制台没有任何报错。

然后在macOS上进行调试，发现chrome在启用录音的时候会弹出用户确认框、并且在任务栏显示一个黄色小点（与windows任务栏显示一个麦克风icon同理），但是在electron客户端中没有按预期出现黄色小点。

## 解决

主要参考这篇文章进行解决：[Electron app not asking for Camera and Microphone permission on macOS Monterey](https://stackoverflow.com/questions/72024011/electron-app-not-asking-for-camera-and-microphone-permission-on-macos-monterey) 

不过在实践过程中踩了很久的坑，差点放弃了，最后终于成功。

简而言之，要对`electron-builder`改两个文件：

第一个是`entitlements.mac.plist`文件中，必须添加布尔值的`entitlements`声明，如下：

```text
<key>com.apple.security.device.audio-input</key>
<true/>
<key>com.apple.security.device.camera</key>
<true/>
```

第二个是`electron-builder.js`文件中，在`mac`字段下添加如下字段：

```text
"extendInfo": {
    "NSMicrophoneUsageDescription": "Please give us access to your microphone",
    "NSCameraUsageDescription": "Please give us access to your camera",
    "com.apple.security.device.audio-input": true,
    "com.apple.security.device.camera": true
  },
```

然后在业务代码中，既可以直接在渲染页面中通过上述的web-API直接调用，也可以在主进程中对用户[发起确认](https://www.electronjs.org/docs/latest/api/system-preferences)，如下：

```ts
const { systemPreferences } = require('electron')

const microphone: Promise<boolean> = systemPreferences.askForMediaAccess('microphone');
```

## 可能闪退

如果配置错误，那有可能完全没有效果（继续像之前一样看似在录音但实际上没录到声音），也有可能会闪退并弹出macOS的崩溃信息弹窗，核心内容如下：

```text
…… The app's Info.plist must contain an com.apple.security.device.audio-input key with a string value. ……
```

以上报错是因为只设置了一个文件，另一个文件没有配置，只要完全按上述方法进行配置即可解决。
