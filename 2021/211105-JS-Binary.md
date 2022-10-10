```yaml lw-blog-meta
title: "JS二进制数据操作"
date: "2021-11-05"
brev: "关键字: ArrayBuffer Blob"
tags: ["前端"]
```

## 背景

如果只是个 CRUD Boy ，那其实是不用在乎二进制的，全部用语言运行时提供的常规类型就足够完成产品需求了；但一旦开始搞一些高端应用或者性能优化领域，那么直接操作二进制肯定是十有八九的事情。

本文稍微深入理解一下JS中的常规二进制操作。

## 字节序

> 参考阅读： [Endianness - MDN](https://developer.mozilla.org/en-US/docs/Glossary/Endianness) 

我们知道计算机的数据流都是二进制是吧，010011100这个样子的。在二进制之上，有「字节`byte`」的概念，一个字节是 8bit 。在字节之上，有了我们常见的各种数据类型，例如`int`，现在一般情况都是`int64`，意思是有 64bit，也就是 8字节 。

然后再回顾一下十六进制表示法。一个十六进制数，是 4bit 对吧，那么两个十六进制数可以表示为 1字节，那么`int32`则可以用8个十六进制数表示。

举个例子，我们用十六进制数声明一个数字变量：`var a int32 = 0x10203040`。（这个数字用十进制表示是`270544960`）

那么这个数字在内存中是如何存放的？想象一下，在一堆二进制（十六进制表示）数据流中，它会是`..` `..` `10` `20` `30` `40` `..` `..`这样排列的吗？

像上面这种，从高位写到低位的写法、也是符合人类直觉的写法，叫做「大端序`big-endian`」。

但它并不符合计算机的直觉。因为这个变量中的数据`0x10203040`在内存中只是一堆二进制中的一部分而已；在用某种规则去分析之前，计算机并不知道，`10`这个字节，是要跟后面三个字节组合起来才有意义呢，还是跟后面二个字节、还是一个字节组合起来、还是就它本身自己这个字节才有意义呢？

所以，对计算机来说效率最高的是从低位读到高位，即`40` `30` `20` `10` 这个顺序。这叫「小端序`little-endian`」。反直觉，不过这才是数组在内存中应有的顺序。

（注意，字节是一个最小单位，一个字节(byte)内部的位(bit)的顺序是不会打乱的。所以`Endianness`也被是翻译为字节序而不是位序）

某台设备到底大端序还是小端序，取决于它的硬件平台，跟软件无关（当然软件层面模拟一下也可以）。虽然目前我们主流的 x86, ARM 等芯片都是**小端序**，但是并不能完全保证一定有效。

什么时候会遇到大端序？可能在嵌入式、网络协议、flv视频协议等中遇到。

我现在使用的机器是 MBP 2018 Intel Core i5 。尝试在JS（浏览器环境）中验证一下字节序：

```javascript
const buffer = new ArrayBuffer(4);
(new Int32Array(buffer))[0] = 0x10203040;
String(new Int8Array(buffer));  // '64,48,32,16'
```

尝试在Golang中验证一下字节序：

```go
func main() {
	var num int32 = 0x10203040
	var array = (*[4]byte)(unsafe.Pointer(&num))  // 将int32强制转化为[4]byte
	fmt.Println(array)  // &[64 48 32 16]
}
```

### JS中处理大端序

如果在某些场景下，前端就是拿到了一段大端序的二进制数据，那么在"JS字节序取决于硬件"的这样一个不确定的情况下，如何稳定地以大端序解析二进制数据？

首先我们需要一个判断，判断起来也很简单，就看看int32的数据会长什么样子：

```ts
// 判断当前运行环境的JS是不是大端序
function osIsBigEndian():boolean {
  const buffer = new ArrayBuffer(4);
  new Uint32Array(buffer)[0] = 0x1;  // 以int32格式写入一个数字
  const view = new Uint8Array(buffer);
  return view[0] > 1;  // 看看数字在不在左边（即大端）
}
```

大多数情况下，得到的判断结果都会是`false`，即当前运行环境一般是小端序。

在小端序的运行环境下，我们需要借助`DataView`来操作`ArrayBuffer`中的数据进行反转，代码如下：

```ts
// 传入的view是长度为4的大端序数据
const parseUint32 = (view: Uint8Array): number => {
    if (!isBigEndian) view.reverse();
    return new Uint32Array(view.buffer)[0];
};
```

顺带一提，在Golang中，标准库`encoding/binary`是可以显式指定字节序的，不需要用户自己操心如何判断和转换。

## ArrayBuffer家族

从底层向高层逐步介绍。

### ArrayBuffer

> 本节参考自 [ArrayBuffer - MDN](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/ArrayBuffer) 

`ArrayBuffer`对象代表着一个通用的、固定长度的、二进制数据缓冲区。

它是一个"字节组成的数组"，在其他语言中一般被称为「字节数组`byte array`」（在Golang就是`[n]byte`）。但是你不能直接用这个对象去操作底层的那个二进制缓冲区，你只能用 `TypedArray`或者`DataView`去操作。

构建函数`ArrayBuffer()`可以创建一个对象。也可以从某些数据结构中生成出来，例如`Blob`。

实例方法`.slice()`会切片并且复制出一个新的二进制缓冲区。

```javascript
const buffer = new ArrayBuffer(4);
const buffer2 = buffer.slice();
(new Int32Array(buffer))[0] = 0x10203040;
console.log(String(new Int8Array(buffer)), String(new Int8Array(buffer2)))
// 64,48,32,16    0,0,0,0
```

### DataView

> 参考 [DataView - MDN](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/DataView)

ArrayBuffer不能直接操作，`DataView`就是专门用来操作的，一个比较底层比较原始的工具。

它的最大特性是可以指定大端序还是小端序，并且保证不同平台上的一致性。

```javascript
const buffer = new ArrayBuffer(4);
new DataView(buffer).setInt32(0, 0x10203040, true);
console.log(String(new Int8Array(buffer)));
// 64,48,32,16
new DataView(buffer).setInt32(0, 0x10203040, false);
console.log(String(new Int8Array(buffer)));
// 16,32,48,64
```

但是要注意，大小端序可以模拟，但是`64位`这个不好模拟。对于32位硬件平台，要使用64位数字的话，建议使用`BigInt`数据类型，兼容是没问题，但要知道它的处理效率很低。

它的实例方法，都是 get/setIntXXX ，第一个参数是偏移量`offset`。也就是说都以数字的形式进行读写，比较原始。

### TypedArray

> 参考 [TypedArray - MDN](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/TypedArray)

它是另一种对`ArrayBuffer`进行读写操作的工具类。ES2015定义了`TypedArray`作为原型，但是它本身并没有直接暴露给用户。我们写代码的时候只能选择更具体的类型的构建函数，例如`Int8Array`。

它的构建函数有多种重载形式，例如我们看`Uint8Array`的：

```javascript
// es5 lib
new(length: number): Uint8Array;
new(array: ArrayLike<number> | ArrayBufferLike): Uint8Array;
new(buffer: ArrayBufferLike, byteOffset?: number, length?: number): Uint8Array;
```

如果传入一个数字，或者不传参数，它内部会创建一个指定长度的、私有的`ArrayBuffer`。

如果传入一个`object`，那么等同于调用`.from()`方法进行实例化。如果是`ArrayBuffer`，那就那么就会以这个buffer为基础进行读写操作；此外还能接受number数组或者另一个TypedArray，逐项转化类型并写入底层二进制缓冲区（注意是上层类型长度相同而不是底层二进制缓冲区长度相同）。

```javascript
const a32 = new Int32Array(2);
a32[0] = 0x10203040;
const a8 = new Int8Array(a32);

console.log(String(a32), String(a8));  // 270544960,0  64,0  因为Int8只能读取最后一个字节0x40
a8[0] = 0;
console.log(String(a32), String(a8));  // 270544960,0  0,0   复制后创建了新的缓冲区，不会互相影响
```

### Buffer

> 参考 [Buffer - Node.js](https://nodejs.org/api/buffer.html)

`Buffer`是`Node.js`的实现（也就是说它不属于`Javascript`）；它是`Uint8Array`的子类，也就是说它是一种TypedArray，是一种很高层的API类。

## ArrayBuffer v.s. Blob

> 参考 [stackoverflow](https://stackoverflow.com/questions/11821096/what-is-the-difference-between-an-arraybuffer-and-a-blob/39951543)

相同点：他们都是代表着二进制数据。

不同点：

- 可变性：Blob 是只读的。
- 源码层面：ArrayBuffer 存活于内存中，随时可以操作；Blob 可以在内存、缓存、硬盘上，有可能暂时没准备好（所以会涉及到Promise）。
- API层：ArrayBuffer 必须借助 TypedArray 进行操作；Blob 本身就直接可以用在一些地方，也可以借助类似的工具类例如 FileReader 进行操作。
- 转化：二者可以互相转化
- 三方库的支持：例如 jzZip 是同时支持二者的；其他库应该也都类似。
- 协议层API支持：WebSocket 和 XHR 都可以指定响应格式是 ArrayBuffer 还是 Blob

## Blob家族

### Blob

> 参考 [Blob - MDN](https://developer.mozilla.org/en-US/docs/Web/API/Blob)

「blob」这个单词本意是"一团"，在计算机领域中就是"二进制大对象"。注意这个"一团"，其实就隐含了"不可变"的意思。

`Blob`就是代表一个二进制对象，它是不可变的、原始的数据。它可以以文本或者二进制或者`ReadableStream`的形式去读取。它也可以代表一些非JS原生的数据格式，例如`File`。

Blob是个比较底层的东西，它的API也总共只有6个而已。我们直接来看一下用法。

从JS中手动创建Blob意义不大，我这里借助`xhr`下载一张图片来看看：

```javascript
async function main() {
  const req = new XMLHttpRequest();
  req.onload = function () {
    console.log(this.response instanceof Blob); // true
    const resp: Blob = this.response;
    console.log(resp); // Blob {size: 1901, type: 'image/vnd.microsoft.icon'}
  };
  req.responseType = 'blob';
  req.open('GET', '/favicon.ico');  // 图片地址
  req.send();
}
```

或者用`fetch`会清爽很多，效果一样：

```javascript
async function main() {
  const resp = await fetch('/favicon.ico');
  console.log(await resp.blob());
}
```

在上面的代码中，`resp`是一个Blob对象，它保存着一张图片的原始二进制数据。

它还有两个属性，`size`表示二进制数据的字节数，`type`表示它被标记的MIME类型。

它有四个方法：可以尝试`await resp.text()`会发现得到一堆乱码，因为这个方法尝试用`utf-8`去解析二进制数据。`arrayBuffer()`方法可以导出一个新的`ArrayBuffer`对象，`slice()`则是切片（并复制）一个新的`Blob`对象，`stream()`则是导出为一个`ReadableStream`对象。注意Promise哦。

### URL.createObjectURL

之前我稍微研究了一下B站的Web端实现，其中发现它的视频资源地址都是`blob:http`协议。经过我的一番研究（ [参考](https://segmentfault.com/a/1190000021724570) ），明确了，它就是利用`URL.createObjectURL`这个功能，先从某种渠道获取了`Blob`数据，然后转化为一个url塞到`<video>`里去就行了。

这个函数只能接受`Blob`和`File`，以及不太兼容的`MediaSource`。我们知道File就是Blob是吧，所以可以说这个方法就是Blob专用的。

简单看一下用法：

```javascript
async function main() {
  const resp = await fetch('/favicon.ico');

  const u = URL.createObjectURL(await resp.blob());
  console.log(u); // blob:http://localhost:7000/98774777-6714-4606-9144-c3918fa95625

  const img = new Image();
  img.src = u;
  document.body.appendChild(img);
}
```

### File

> [MDN](https://developer.mozilla.org/en-US/docs/Web/API/File)

它是一种特殊的`Blob`。它一般产生于用户在`<input>`标签中选择文件得到的`FileList`对象，或者drag/drop操作的`DataTransfer`对象，或者Canvas的某个方法。

它基本上没有什么额外的属性和方法。唯一比较有用的是`name`这个字段，顾名思义——文件的名字。

在业务上经常会用到这个东西，不过用来用去也就那么一个套路，到处复制罢了……等下次我做拖拽专题的时候再提一下吧。

### FileReader

> [MDN](https://developer.mozilla.org/en-US/docs/Web/API/FileReader)

它是用来读取`Blob`（包括`File`）的方式之一。用法基本上与xhr差不多，通过处理事件来完成功能：

```javascript
async function main(file: File) {
  const reader = new FileReader();
  reader.onload = (e: ProgressEvent<FileReader>) => {
    const res: string|ArrayBuffer = e.target.result;
  };
  reader.readAsArrayBuffer(file);
}
```

它的`.target.result`的类型，会根据下面具体的执行函数（这里是`.readAsArrayBuffer()`）而发生变化。

其实吧，如果有了`File`对象，直接调用它自己的API就能达到一样的效果……：

```javascript
async function main(file: File) {
  const res: ArrayBuffer = await file.arrayBuffer();
}
```

> 2021年12月17日踩坑记：这个`file.arrayBuffer`方法兼容版本是chrome76+，第一次用这个API就被反馈说没反应，好尴尬……  
> 吃一堑长一智，以后学新东西学归学，到线上使用的时候还是要谨慎一些。

## Streams家族

> [MDN](https://developer.mozilla.org/en-US/docs/Web/API/Streams_API)

「Stream」这个单词就是"流"的意思。大家一定都听说过"流式数据处理"这样的术语吧。它的核心理念就是，接受到部分数据就可以立刻开始处理（有时候是网络波动导致的延迟，有时候可能是故意的——即KeepAlive的思路），而不是等待所有数据都加载完成再处理。

其实浏览器就是这样工作的，例如图片、视频我们可以看到它们一点一点地加载出来。只是之前没有在JS里暴露出来给开发者使用而已。

（大概看了一下，感觉意义不大，先放着，以后遇到使用场景了再回来补充吧）

## 小结

整体梳理了一遍，然后对照一下自己在公司项目上写的代码，发现我之前的用法挺丑的，绕了一些弯路。嗯，不过以后我可以以最优雅的方式去实现了。

所以这次收获也是不小的。
