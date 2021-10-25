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

代码实现可以参考 [我的github仓库](https://github.com/Saodd/learn-canvas) 。如果你打算以本文作为教程来入门，强烈建议你clone这个代码仓库并跟着进度切换代码去理解。

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

上面`fillRect()`方法的四个参数分别是 x, y, height, width, 这就涉及到坐标系：

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

### 合成

`globalCompositeOperation` 这个属性决定了新的图像合成进来的方式。（译者注：在Photoshop里是叫图像混合啥的来着）

具体的可选项以及相应的效果预览请前往MDN页面。核心代码如下：

```tsx
// step7-1
ctx2.drawImage(canvas0, 0, 0);
ctx2.globalCompositeOperation = 'screen'; // 调整这里，观察效果
ctx2.drawImage(canvas1, 0, 0);
```

### 剪辑路径 Clipping paths

一个剪辑路径就像是一个普通的图形(shape)，但是它起到的作用是作为一个蒙版(mask)去从另一个图形（图层）上选出部分区域。（译者注：在Photoshop里是叫图层蒙版，套索，选区之类的概念）

表面上看起来可能跟前一节的`composite`的`source-in`模式类似，但区别是，剪辑路径并没有真的画在canvas上，并且它也不受下面的图形的影响。

在前面的章节中我们介绍过使用路径(path)的函数`stroke()`和`fill()`，现在介绍第三个函数`clip()`。你用它代替`closePath()`去闭合一个路径并将其转化为一个剪切路径。

默认情况下，canvas上已经有一个剪切路径，它就是canvas本身当前的形状（译者注：大概就是视窗的意思）。

```tsx
// step7-2

// 先画剪辑路径
ctx.beginPath();
ctx.arc(100, 100, 60, 0, Math.PI * 2, true);
ctx.clip();
// 再画被剪辑的图形
ctx.fillStyle = 'red';
ctx.fillRect(10, 10, 120, 120);
```

> 在我的代码仓库的 「step7-2-extra: 星空」 这个commit里，画出来的效果，有点好看。

## Step8: 基础动画 Basic animations

既然我们可以通过JS来控制`<canvas>`元素，那么也就能够很轻易地实现（交互式）动画效果。

不过最大的问题在于，（canvas本身只是一个简单的画板），上面画了什么就是什么，如果你想要移动一个元素，对不起，你必须重新绘制整个画板上的所有东西，所以它对电脑性能要求很高。

代码「step8-1」中绘制了一个简易的太阳、地球、月球绕轨道旋转的动画，其核心逻辑如下：

```tsx
// step8-1 核心逻辑
let ctx: CanvasRenderingContext2D = canvas.getContext('2d');
function draw() {
  // ctx.drawImage(...)
  window.requestAnimationFrame(draw)
}
draw()
```

如果有一点点动画基础（小时候应该都学过Flash吧），应该知道「帧 frame」这个概念。

`window.requestAnimationFrame`这个函数，顾名思义就是针对动画设置的，就是在浏览器准备好、有空渲染下一帧动画的时候，执行其中的回调函数，也就是我们定义好的绘图函数draw，它会有一些优化，例如最多每秒60帧，例如页面切到后台的时候停止执行等。其他需求场景也可以用`setInterval`或者`setTimeout`。

代码「step8-2」中实现了一个简陋的鼠标轨迹。

核心原理就是，在每一帧都在整个画面上盖一层`rgba(0,0,0,0.05)`的图层，这样就模拟了逐渐消失的效果，然后再绘制当前帧对应的新的轨迹。另一种思路是保存一定数量的历史坐标，每次都重新绘制。

```tsx
function init() {
  window.addEventListener('mousemove', (e) => {
    x = e.clientX;
    y = e.clientY;
  });
}

function draw() {
  ctx.save();
  {
    ctx.fillStyle = 'rgba(0,0,0,0.05)';
    ctx.fillRect(0, 0, 800, 800);
    ctx.fillStyle = 'white';
    ctx.translate(x, y);
    ctx.fillRect(-5, -5, 5, 5);
  }
  ctx.restore();
  window.requestAnimationFrame(draw);
}
```

最后还展示了一个贪吃蛇的例子，实现得比较原始，没有太大参考价值吧，反正我学过PIXI之后我觉得没必要用canvas去撸游戏。

## Step9: 高级动画

正如前面说的，有了PIXI这类框架，我们不需要用原始的手段去模拟。一定要模拟的话，也要添加抽象，~~然后做着做着就做成另一个PIXI了~~

一些常见的抽象：速度、边界（及碰撞检测）、加速度、轨迹、键鼠交互等。

## Step10: 像素操作

`ImageData` 借助它，可以设置/读取底层的像素数据。

然后可以实现一些高级功能，例如「拾取颜色 color picker」、「灰度 Grayscale」、「反相 Inverted」、「缩放 Zoom」等等。（看了下灰度和反相这些颜色算法，其实还意外地简单）

最后有个很实用的功能是导出图片，用`toDataURL('image/png')`和`toDataURL('image/jpeg', quality)`支持两种格式。还有`toBlob()`。

## Step11: 优化

这里有些针对web游戏的 [优化建议](https://developer.mozilla.org/en-US/docs/Games/Techniques/Efficient_animation_for_web_games)

然后对于canvas本身，有如下建议：

1. 对于一些反复渲染且不太变化的图像，可以考虑渲染在一个单独的（不在屏幕内的）canvas中，然后每次drawImage进来。
2. 避免浮点数，尽量用整数。（这可以让浏览器不用处理抗锯齿，而且也有利于提升清晰度）
3. 不要在`drawImage`里缩放图像（参考第1条的缓存方式）
4. 考虑使用多层canvas，典型场景是在游戏中你可能需要背景、舞台、UI三层，你只需要三个绝对定位的canvas叠在一起就可以做到"动静分离"。
5. 如果背景是静态的，可以考虑放在底层的一个div标签中。
6. 如果要缩放整个canvas，借助css去做，因为css使用GPU。
7. 可以考虑关闭透明度。
8. 减少文字渲染。

（简而言之，就是要清楚，对于动画来说，每一个tick都需要重新渲染整个canvas，所以就要做好动静分离，然后尽量关掉那些用不着的特性。）

## 总结

目前能够想到canvas应该就是两个应用，一是网页游戏，二是图像编辑器（类Photoshop）。

但其实也并不是只能用canvas，直接操作DOM也不是不可以。值得一提的是，二者之间的转换兼容性很不好，所以要做项目的话可能要在一开始就决定一条路线，别想脚踏两条船。

根据我的认知、以及对一些产品的观察来看，目前基本上还是以canvas为主流。
