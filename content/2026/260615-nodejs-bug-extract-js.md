---
title: 'Nodejs 导致 extract-zip 异常'
description: 'Node.js >=24.16.0、>=26.3.0 存在已知回归，会让 extract-zip 解压时挂起，进程以退出码 0 静默退出，导致 Electron 安装失败'
pubDate: '2026-06-15'
updateDate: ''
tags: ['前端']
keywords: 'Node.js, extract-zip, Electron, yauzl, 回归bug'
---

## 现象

最近我从`yarn`切换到了`pnpm`，当然也踩了很多坑，但是当我一个个排除，最后把`pnpm`推广到团队的时候，意外还是发生了。

出现一个诡异的bug：项目中安装的依赖`electron`无法运行，提示`install failed`。

经过一番排查，发现安装 Electron 时进程没有任何报错，却直接结束，`dist` 目录没有完整解压出来——Electron 装不上。

排查后定位到这是 **Node.js 的一个官方已确认的回归 bug**，与 zip 文件或网络都无关。

- 相关 issue：[Node 24.16.0: extract-zip hangs extracting · nodejs/node#63487](https://github.com/nodejs/node/issues/63487)
- 影响版本：`>=24.16.0`、`>=26.3.0`（`v24.15.0` 正常）
- 影响范围：所有依赖 `extract-zip` / `yauzl` 解压二进制的安装流程，包括 Puppeteer、Mailspring 等。Mailspring 社区也报告了同样问题。

## 验证：完整的证据链

1. **zip 文件本身完好**：压缩包第一个条目是 `d3dcompiler_47.dll`，完整大小应为 `4916712` 字节，而 `dist` 里写出的只有 `4898583` 字节——差约 18KB 没写完，流在中途停了。

2. **用项目里的 extract-zip 复现**：解压到临时目录时稳定复现，卡在第一个文件，Promise 永不 settle，进程以退出码 0 退出。

3. **绕过 extract-zip 直接用 yauzl 测试**：问题出在**读流（解压侧）**——在 `4898583` 字节处停止发出 `data`，`end` 和 `error` 都不触发，写流是无辜的。

4. **用 zlib 单独解压同一份字节**：`zlib.inflateRawSync` 和纯流式 `createInflateRaw` 都能完整解出 `4916712` 字节。说明 zlib 本身没坏，坏的是 **Node 24.16.0 中 yauzl 流管道的交互行为**。

## 两个延伸问题

**`pipeline = promisify(stream.pipeline)` 是什么？**

`stream.pipeline` 原本是回调风格的 API，把多个流串起来，结束或出错时调回调；`promisify` 把它包装成返回 Promise 的版本，等价于 `require('stream/promises').pipeline`。它本身没有问题。

**为什么不抛异常，进程却直接退出？**

Node 中一个悬空的 Promise 并不会维持进程存活——维持进程的是待处理的 I/O、定时器等 libuv 句柄。

这个 bug 触发后，读流停滞，但 Node 认为「没有任何待处理的 I/O」，事件循环排空，进程便以退出码 0 正常退出。`await pipeline(...)` 之后的代码路径永远不会恢复执行，`catch` 也不会进入——因为根本没有 rejection，只有一个永远 pending 的 Promise。

> 在复现脚本里加一个心跳定时器，进程就会一直活到定时器被清除为止，正好印证了这个机制。

## 解决方案

按推荐优先级：

**方案一：降级 Node（根治）**

降到 `24.15.0`（或 26.1.0 我没验证），然后删掉 Electron 的 `dist` 目录重新安装：

```bash
rm -rf node_modules/.pnpm/electron@29.2.0/node_modules/electron/dist
pnpm install
```

**方案二：手动解压打补丁（临时）**

如果暂时不想动 Node 版本：

用 7zip 把缓存里的 electron-v29.2.0-win32-x64.zip 解压到 node_modules\.pnpm\electron@29.2.0\node_modules\electron\dist
在 electron 包根目录写一个 path.txt，内容为 electron.exe
（install.js 的 isInstalled() 正是检查这两样。）缺点是每次重装依赖都要重来一次。

提醒：调试时加在 install.js 和 extract-zip/index.js 里的 console.log 还在 node_modules 中，重装会被覆盖，不影响仓库，忽略即可。

## 参考

- [Node 24.16.0: extract-zip hangs extracting · nodejs/node#63487](https://github.com/nodejs/node/issues/63487)
- [Building is broken with Node.js 26.1.0+ and 24.16.0+ · Mailspring Community](https://github.com/Foundry376/Mailspring)

声明：本文原始内容由AI生成，然后由AI改写为博客文章，最后经过我人工少量修改润色完成。
