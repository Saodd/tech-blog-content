```yaml lw-blog-meta
title: "HTML Canvas 入门教程"
date: "2021-10-21"
brev: "在前端实现高级图像处理功能就得靠它"
tags: ["前端"]
```

## 背景

其实之前学的`Pixi.js`就是基于canvas的，但终归还是要回归底层，直接玩一下原生接口才行。

其实这篇文章在8月21号的时候就写了一半了，因为太繁琐了，中间鸽了两个月才有勇气继续写下去。

本文翻译、摘录自 [MDN文档](https://developer.mozilla.org/en-US/docs/Web/API/Canvas_API/Tutorial) ，并结合自己的理解做了改进。

代码实现可以参考 [我的github仓库](https://github.com/Saodd/learn-canvas)

## Step1: 基本用法

```tsx
export function App(): JSX.Element {
  return (
    <div>
        <canvas width={800} height={800} ></canvas>
    </div>
  );
}
```

`canvas`标签，用起来其实就是个普通的HTML元素罢了。它只有两个特殊属性，宽度和高度。

你可以通过css去设置它的任意外观，例如宽度、高度、背景图、边框等等，但是要清楚，这与它内部的渲染是完全分开的。

你可以在`canvas`内部添加children，可以起到 Fallback 的作用，那些不支持canvas的旧浏览器会显示它内部的内容作为替代。为了保证兼容，你最好一定要写上`</canvas>`闭合标签。

canvas元素提供了一个固定标准的接口来暴露「渲染上下文`rendering context`」。在本文中，我们只讲2D，如果你想了解3D可以去学习一下WebGL。

你的(JS)程序首先需要获取到一个渲染上下文，才能在canvas中进行绘制。本文用到的功能，你只需要执行`.getContext('2d')`，就能获得一个`CanvasRenderingContext2D`对象。（你可以检查这个dom元素是否有`getContext`方法，如果没有则说明浏览器不支持canvas）

一个简单的例子，画两个正方形：

```tsx
  React.useEffect(() => {
    const ctx = cRef.current.getContext('2d');

    ctx.fillStyle = 'red';
    ctx.fillRect(10, 10, 50, 50);

    ctx.fillStyle = 'rgba(0, 0, 200, 0.5)';
    ctx.fillRect(30, 30, 50, 50);
  }, []);
```

## Step2: 绘制形状 Drawing shapes with canvas

上面`filrRect()`方法的四个参数分别是 x, y, height, width, 这就涉及到坐标系：

![canvas_default_grid](https://developer.mozilla.org/en-US/docs/Web/API/Canvas_API/Tutorial/Drawing_shapes/canvas_default_grid.png)

跟`SVG`一样，canvas只有两种初始形状：「矩形`rectangle`」和「路径`path`」，其他形状必须通过这两者组合出来。

矩形有三个方法进行绘制：

- fillRect(x, y, width, height): Draws a filled rectangle.
- strokeRect(x, y, width, height): Draws a rectangular outline.
- clearRect(x, y, width, height): Clears the specified rectangular area, making it fully transparent.

矩形都是立即绘制的。而路径就麻烦一些。

> 注：或许，把 `Path` 翻译为 `矢量路径` 可能会更利于理解。它是一种抽象的线条，没有宽度。

路径是由一个`点`的数组 ，点之间会用线段连接，可以闭合，这样组成各种形状、曲线。创建一条路径我们需要：

1. 创建路径数据。
2. 用绘图命令画上去。
3. 描线（stroke）或者填充（fill）等动作。

```tsx
ctx.beginPath();
ctx.moveTo(75, 50);
ctx.lineTo(100, 75);
ctx.lineTo(100, 25);
ctx.fill();
```

![triangle](https://developer.mozilla.org/en-US/docs/Web/API/Canvas_API/Tutorial/Drawing_shapes/triangle.png)

顺带一提，如果要画一个三角形轮廓，那么需要`.closePath()`然后`.stroke()`。

接下来画个更复杂的图形，用到圆形路径工具`.arc()`，画一个笑脸：

```tsx
ctx.beginPath();
ctx.arc(75, 75, 50, 0, Math.PI * 2, true); // Outer circle
ctx.moveTo(110, 75);
ctx.arc(75, 75, 35, 0, Math.PI, false);  // Mouth (clockwise)
ctx.moveTo(65, 65);
ctx.arc(60, 65, 5, 0, Math.PI * 2, true);  // Left eye
ctx.moveTo(95, 65);
ctx.arc(90, 65, 5, 0, Math.PI * 2, true);  // Right eye
ctx.stroke();
```

![canvas_smiley](https://developer.mozilla.org/en-US/docs/Web/API/Canvas_API/Tutorial/Drawing_shapes/canvas_smiley.png)

顺带一提，最后一个参数 counterclockwise 是顺时针方向的意思。

- 二次方曲线 quadraticCurveTo(cp1x, cp1y, x, y)
  - 从当前点划到目标点(x,y)，以(cp1x,cp1y)作为控制点。（二次函数有三个未知量，所以需要三个点来确定）
- 贝塞尔曲线 bezierCurveTo(cp1x, cp1y, cp2x, cp2y, x, y)
  - 同理，只不过有两个控制点

使用这两种曲线可能会很难，因为我们在找点坐标的时候没有视觉上的反馈。（所以这两种曲线可能更适合于从别处导入，而不是人肉编写）

![canvas_curves](https://developer.mozilla.org/en-US/docs/Web/API/Canvas_API/Tutorial/Drawing_shapes/canvas_curves.png)

最后，在路径中也可以添加矩形（与直接画矩形是不同的），使用`.rect()`方法。

### Path2D 对象

上面的例子中，看起来要画一个图形需要好多好多代码……用`Path2D`来封装一组绘图命令。

```tsx
const rectangle = new Path2D();
rectangle.rect(10, 10, 50, 50);

const circle = new Path2D();
circle.arc(100, 35, 25, 0, 2 * Math.PI);

ctx.stroke(rectangle);
ctx.fill(circle);
```

![path2d](https://developer.mozilla.org/en-US/docs/Web/API/Canvas_API/Tutorial/Drawing_shapes/path2d.png)

## Step3: 样式和颜色 Applying styles and colors

### 颜色

设置颜色使用`.fillStyle`和`.strokeStyle`。写法是css风格。

![](https://developer.mozilla.org/en-US/docs/Web/API/Canvas_API/Tutorial/Applying_styles_and_colors/canvas_fillstyle.png)

![](https://developer.mozilla.org/en-US/docs/Web/API/Canvas_API/Tutorial/Applying_styles_and_colors/canvas_strokestyle.png)

设置透明度，可以用`.globalAlpha`，或者直接设定一个带透明度的颜色数值。

### 线条样式

- lineWidth = value
  - Sets the width of lines drawn in the future.
- lineCap = type
  - Sets the appearance of the ends of lines.
- lineJoin = type
  - Sets the appearance of the "corners" where lines meet.
- miterLimit = value
  - Establishes a limit on the miter when two lines join at a sharp angle, to let you control how thick the junction becomes.
- getLineDash()
  - Returns the current line dash pattern array containing an even number of non-negative numbers.
- setLineDash(segments)
  - Sets the current line dash pattern.
- lineDashOffset = value
  - Specifies where to start a dash array on a line.

关于线条宽度，是以路径为中心的，也就是说，两边的宽度各一半。因为canvas的坐标单位并不是像素，所以在绘制垂直、水平线条的时候要额外注意。

![](https://developer.mozilla.org/en-US/docs/Web/API/Canvas_API/Tutorial/Applying_styles_and_colors/canvas_linewidth.png)

上面的十条竖线，宽度从1到10，可以观察到，所有奇数宽度的线条都看起来不清晰。这个是小数的近似处理导致的：

![](https://developer.mozilla.org/en-US/docs/Web/API/Canvas_API/Tutorial/Applying_styles_and_colors/canvas-grid.png)

这种像素近似问题，会发生在各种地方，多多注意。

### 渐变 Gradients

有线性渐变、径向渐变、圆锥渐变(切向渐变)三种。

先创建渐变的位置，然后再设置颜色：

```tsx
const g1 = ctx.createLinearGradient(0, 0, 0, 150);
g1.addColorStop(0, '#00ABEB');
g1.addColorStop(0.5, '#fff');
g1.addColorStop(0.5, '#26C000');
g1.addColorStop(1, '#fff');
ctx.fillStyle = g1;
ctx.fillRect(10, 10, 130, 130);
```

![](https://developer.mozilla.org/en-US/docs/Web/API/Canvas_API/Tutorial/Applying_styles_and_colors/canvas_lineargradient.png)
![](https://developer.mozilla.org/en-US/docs/Web/API/Canvas_API/Tutorial/Applying_styles_and_colors/canvas_radialgradient.png)
![](https://developer.mozilla.org/en-US/docs/Web/API/Canvas_API/Tutorial/Applying_styles_and_colors/canvas_conicgrad.png)

### 花纹 Patterns

> 让我回想起玩PhotoShop和SAI的那些年…… 这个术语在专业绘图软件中可能是被直译为「图案」或者「填充图形」的，但我觉得「花纹」会更达意。

总之就是指定某个图案单元的重复填充规则。

### 阴影

- shadowOffsetX = float
- shadowOffsetY = float
- shadowBlur = float
- shadowColor = color

大概跟css的阴影用法一致，跳过。

### 填充规则

指定为`evenodd`时，简单说，就是被包围了两次（偶数次）的区域会抵消掉，不填充。在绘制环形形状的时候很有用。

```tsx
ctx.beginPath();
ctx.arc(50, 50, 30, 0, Math.PI * 2, true);  // 一个大圆
ctx.arc(50, 50, 15, 0, Math.PI * 2, true);  // 同心小圆
ctx.fill('evenodd');
```

![](https://developer.mozilla.org/en-US/docs/Web/API/Canvas_API/Tutorial/Applying_styles_and_colors/fill-rule.png)

## Step4: 文本 Drawing text

### 绘制文本

- `fillText(text, x, y [, maxWidth])`
- `strokeText(text, x, y [, maxWidth])`

```tsx
ctx.font = '48px serif';
ctx.fillText('Hello world', 10, 50);
ctx.strokeText('Hello world', 10, 100);
```

简而言之，就是先在`ctx`上设置好字体属性，然后往上画。

### 文本样式

基本上都等同于在css中的用法。（译者注：在实际使用中，`textBaseline`的坑很深，而且字体资源本身的加载的这件事也是很深的坑，总之字体相关的事情都很坑）

- `font`
- `textAlign`
- `textBaseline` 
- `direction`

关于`textBaseline`，这里引用一张图参考一下：

![](https://developer.mozilla.org/en-US/docs/Web/API/Canvas_API/Tutorial/Drawing_text/baselines.png)

### 测量工具

- `measureText()` 返回一个对象，能够描述在当前的配置下如果要绘制文字会得到什么样的结果。不会真的绘图。

## Step5: 图片

把图片放入canvas，有两个步骤。第一步，如果要从页面内的内容引用，那就先获取一个对应元素的引用，或者直接用一个url；第二步，把它画到canvas上去。

### 获取图片

可以利用下列图片资源，这些资源都整合在一个类型`CanvasImageSource`中了：

- `HTMLImageElement`: `Image()`或者`<img>`
- `SVGImageElement`: `<image>`
- `HTMLVideoElement`: `<video>`元素，会抓取当前帧。
- `HTMLCanvasElement`: 另一个canvas

当你在使用跨域资源（CORS）图片时，你要在`<img>`元素上添加`crossorigin`属性，并且服务端要同意跨域请求，才能正常使用；否则，会「污染`taint`」这个canvas（即导致无法将canvas中的内容导出为图片）

使用另一个canvas作为图片源，一个典型场景是，展示缩略图(thumbnail)。

当直接使用URL时，可以借助`Image()`，但是要记得等图片加载完毕之后再去使用，否则不会产生任何效果。也可以用`data:URL`，也就是将data协议数据字符串传入`img.src`即可。

### 绘制图片

`drawImage(image, x, y)` 这个函数有多个重载的版本，这是它最基础的用法。

结合前面使用URL加载图片的例子看一下：

```tsx
const img = new Image();
img.onload = function() {
  ctx.drawImage(img, 10, 100);
};
img.src = '/favicon.ico';
```

`drawImage(image, x, y, width, height)` 接下来看后面一组参数，是图片的缩放大小。例如原图尺寸64x64，可以通过这两个参数去缩放。举个例子，缩放并填充：

```tsx
const img = new Image();
img.onload = function () {
  for (let x = 0; x < 4; x++) {
    for (let y = 0; y < 3; y++) {
      ctx.drawImage(img, x * 16, y * 16, 16, 16);
    }
  }
};
img.src = '/favicon.ico';
```

`drawImage(image, sx, sy, sWidth, sHeight, dx, dy, dWidth, dHeight)` 这种重载形式，支持「切图 slice」，也就是取出图片的一部分。前四个参数用于切割，后四个参数用于绘制。代码略。

## Step6: 变形 Transformations

### 保存状态

先介绍两个API，他们可以将当前canvas的「绘画状态 drawing state」保存和恢复（用白话翻译就是当前的配置上下文，不包括已经绘制的图像）。状态是保存在一个栈上的（后进先出）。

- `save()`
- `restore()`

### 转换 Translating

> 注意 transform 和 translate 是不同的。

这里指的是转换canvas的当前坐标系，用物理坐标系来说就是转换参考坐标系。

一个典型应用是在批量绘制图形的时候，之前是根据行列数来计算便宜的x/y坐标值；现在可以用`translate()`直接移动坐标系。（最大的区别是，坐标系的移动可以是相对值，不用去计算绝对值了）

```tsx
// step6-1
ctx.save();
for (let i = 0; i < 3; i++) {
  ctx.translate(0, 50); // 坐标系相对向下移动50
  ctx.save();
  for (let j = 0; j < 3; j++) {
    ctx.translate(50, 0); // 坐标系相对向右移动50
    ctx.fillRect(0, 0, 25, 25);
  }
  ctx.restore();
}
ctx.restore();
```

### 旋转 Rotate

也是同理，就是旋转当前的参考坐标系。注意是顺时针，然后要用`Math.PI`去计算。

### 缩放 Scale

缩放整个参考坐标系的单位尺度。两个参数分别对x、y轴的缩放尺度，`1.0`则表示不变。

这里要注意的是，缩放的是整个坐标系轴，而不仅仅是绘制的图像的尺寸。

```tsx
// step6-2
ctx.save();
ctx.translate(200, 200);
for (let i = 0; i < 8; i++) {
  ctx.fillRect(50, 50, 25, 25);
  ctx.rotate((Math.PI / 180) * 45);  // 顺时针旋转45度
  ctx.scale(1.1, 1.1);  // 坐标轴放大1.1倍
}
ctx.restore();
```

### 变形 Transform

`transform(a, b, c, d, e, f)` 有个东西叫做「变形矩阵  transformation matrix」，所有的变形操作都是基于这个矩阵去计算的。下面的例子中，强行用变形去模拟了旋转：

```tsx
const sin = Math.sin(Math.PI / 6);
const cos = Math.cos(Math.PI / 6);
ctx.translate(100, 100);
for (let i = 0; i < 12; i++) {
  const c = Math.floor(255 / 13 * i);
  ctx.fillStyle = `rgb(${c},${c},${c})`;
  ctx.fillRect(0, 0, 100, 10);
  ctx.transform(cos, sin, -sin, cos, 0, 0);
}
```

## Step7: 合成与剪辑 Compositing and Clipping

(TODO)
