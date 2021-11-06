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

某台设备到底大端序还是小端序，取决于它的硬件平台，跟软件无关（当然操作系统硬要模拟一下也不是不行）。而目前我们主流的 x86, ARM 等芯片都是**小端序**，所以至少在大前端开发的领域，可以默认都是小端序，不需要额外的兼容逻辑。

什么时候会碰到大端序？大概只有在某些嵌入式开发领域，以及网络通信协议中才会有。

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

(TODO)
