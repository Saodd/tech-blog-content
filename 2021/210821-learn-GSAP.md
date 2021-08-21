```yaml lw-blog-meta
title: "GSAP.js学习笔记"
date: "2021-08-21"
brev: "抄一个B站活动页"
tags: ["前端"]
```

## 背景

作为一个马上要升6级的老（大）会员，对B站我是非常熟悉了。

随着最近前端学习的深入，我发现，原本熟悉的B站页面，在我眼里开始变得支离破碎 —— 扯一句鬼话，「看山是山，看山不是山，看山还是山」这三重境界，我已经进入了第二重。（贴一个 [我认可的解释](https://www.zhihu.com/question/20146527/answer/22293676) ）

前阵子偶然点进了一个 [活动页](https://www.bilibili.com/blackboard/activity-yellowVSgreen7th.html) ，叫什么黄绿合战，虽然我对活动本身完全不感兴趣，但是这个页面的动画特效却吸引了我的注意。

稍微调查一下，wow，原来动画是用的 GSAP+视频循环 来实现的。那这个简单，我肯定能抄一个。

那就抄！

## GSAP

[Getting Started](https://greensock.com/get-started/) ，别看这个，直接看React版本的 [Getting Started](https://greensock.com/react) .

这是一个js动画框架。自称能让JS能触及的一切东西动画化。

它说，动画的本质其实就是不断地改变元素属性。我之前在`Pixi.js`的文章中说过，动画是一帧一帧画出来的，同理。所以GSAP这个框架做的事情也就是很频繁地修改元素的属性。

> 注：下面的每个小节分别对应 [我github仓库](https://github.com/Saodd/learn-gsap) 中的一个commit 。

### React1: 捕获目标 Targeting elements

先看一段入门代码：

```typescript jsx
import * as React from 'react';
import gsap from 'gsap'
import styles from './index.scss'

export function App(): JSX.Element {
    const boxRef = React.useRef();

    React.useEffect(() => {
        gsap.to(boxRef.current, {
            duration: 5,
            rotation: "+=360",
        });
    });

    return (
        <div className={styles.App}>
            <div className={styles.box} ref={boxRef}>Hello</div>
        </div>
    );
}
```

- 使用`gsap.to()`方法来控制元素。
- 然后在React中，使用`useRef`来获取元素的引用。
- 使用`useEffect`的目的是避免操作到空引用。

### React2: 后代选择器 Targeting descendant elements

提供一种批量选择引用的思路。并且它不仅仅是选择，还能基于组内顺序来实现一些递增计算。

```typescript jsx
import * as React from 'react';
import gsap from 'gsap'
import styles from './index.scss'


function Box(props: { children: React.ReactNode }) {
    const {children} = props
    return <div className={styles.box}>{children}</div>;
}

function Container() {
    return <div><Box>Nested Box</Box></div>;
}


export function App(): JSX.Element {
    const appRef = React.useRef();
    const q = gsap.utils.selector(appRef)

    React.useEffect(() => {
        gsap.to(q("." + styles.box), {
            x: 100,
            stagger: 0.33,
            repeat: -1,
            repeatDelay: 1,
            yoyo: true
        });
    });

    return (
        <div className={styles.App} ref={appRef}>
            <Box>Box1</Box>
            <Container/>
            <Box>Box2</Box>
        </div>
    );
}
```

- 类选择器可以与scss共同工作，但是记得前面加`.`号。
- `stagger`属性指定了多个子元素是如何错开的。

`gsap.utils.selector()`方法会对所有符合条件的后代进行筛选。如果你需要更细粒度的控制，那你可能要多写几个css类，或者，给每个后代元素分别创建引用，然后给`q()`传入一个引用数组。

### React3: 控制时间线 Creating and controlling timelines

在React中，`Ref`是独立于Render循环之外的，所以我们应当把时间线对象放进去。

```typescript jsx
import * as React from 'react';
import gsap from 'gsap'
import styles from './index.scss'


export function App(): JSX.Element {
    const appRef = React.useRef();
    const q = gsap.utils.selector(appRef)

    const tlRef = React.useRef<gsap.core.Timeline>();
    const [reversed, setReversed] = React.useState(false);

    React.useEffect(() => {
        tlRef.current = gsap.timeline()
            .to(q("." + styles.box), {
                rotate: 360,
            })
            .to(q("." + styles.circle), {
                x: 100,
            })
    }, []);
    React.useEffect(() => {
        tlRef.current.reversed(reversed);
    }, [reversed]);

    return (
        <div className={styles.App} ref={appRef}>
            <button onClick={() => setReversed(!reversed)}>点击我</button>
            <div className={styles.box}>Box</div>
            <div className={styles.circle}>Circle</div>
        </div>
    );
}
```

- 创建timeline的那个`useEffect`必须要加`[]`参数，确保只执行一次。
- 多个`.to()`的写法，会按顺序执行。
- 时间线这个东西挺神奇的，它走到一半的时候也可以进行操作，做出完美的倒流效果。

### React4: 交互动画 Animating on interaction 

其实对React来说，只是函数触发的方式不同罢了。这里再展示一个与hover相关的动画：

```typescript jsx
export function App(): JSX.Element {
    const appRef = React.useRef();
    const q = gsap.utils.selector(appRef)

    const handleEnter = React.useCallback(() => {
        gsap.to(q("." + styles.box), {
            scale: 1.5,
        })
    }, [])
    const handleLeave = React.useCallback(() => {
        gsap.to(q("." + styles.box), {
            scale: 1,
        })
    }, [])

    return (
        <div className={styles.App} ref={appRef}>
            <div className={styles.box} onMouseEnter={handleEnter} onMouseLeave={handleLeave}>Box</div>
        </div>
    );
}
```

### React5: 起止帧 与 防闪烁 Avoiding flash of unstyled content

`gsap.fromTo()`方法可以分别指定动画的起、止状态。

而之前的`gsap.to()`仅仅指定结束状态。因此，额外指定的起始状态，可能会造成页面元素的闪烁（即css样式与起始状态样式不同）。为了避免闪烁，我们可以用`useLayoutEffect`来代替`useEffect`，前者是在组件渲染之前执行的。

> 注：另一种思路是将css的状态设为与初始状态相同。

```typescript jsx
export function App(): JSX.Element {
    const appRef = React.useRef();
    const q = gsap.utils.selector(appRef)

    // 此处改为useEffect即可观察到闪烁。
    React.useLayoutEffect(() => {
        gsap.fromTo(q('.' + styles.box), {
            opacity: 0,
        }, {
            opacity: 1,
            duration: 1,
            stagger: 0.2,
        })
    }, [])

    return (
        <div className={styles.App} ref={appRef}>
            <div className={styles.box}>Box1</div>
            <div className={styles.box}>Box2</div>
            <div className={styles.box}>Box3</div>
        </div>
    );
}
```

### React6: 内存回收 Cleaning Up

gsap的动画对象是独立于React组件生命周期的，因此如果你的网页是个SPA，请务必记得回收资源。在React中，利用`useEffect`的返回值来做这个事情：

```typescript jsx
React.useLayoutEffect(() => {
    const animation1 = gsap.fromTo(q('.' + styles.box), {
        opacity: 0,
    }, {
        opacity: 1,
        duration: 1,
        stagger: 0.2,
    })
    return () => {
        animation1.kill()
    }
}, [])
```

### GSAP-1: 修改自定义对象

首先要明确，GSAP并不是一个UI动画库，它所做的仅仅只是「频繁地修改属性」这一件事情。因此，我们试着用它来修改一个自定义对象：

```typescript jsx
const objRef = React.useRef({count: 0});
React.useEffect(() => {
    gsap.to(objRef.current, {
        count: 200,
        duration: 1,
        onUpdate: () => {
            console.log(objRef.current.count)
        }
    })
}, [])
```

咦，既然gasp改的是对象的属性，那对于DOM元素来说，改变那些看起来像css的属性，是如何实现的？

这就要说到gsap的插件了。

### 插件

为了保持GSAP库核心的简洁性，我们交给插件来实现各种丰富的功能。常用的插件有：

- `SplitText`: Splits text blocks into lines, words, and characters and enables you to easily animate each part.
- `Draggable`: Adds the ability to drag and drop any element.
- `MorphSVGPlugin`: Smooth morphing of complex SVG paths.
- `DrawSVGPlugin`: Animates the length and position of SVG strokes.
- `MotionPathPlugin`: Animates any element along a path.

在GSAP核心库中内置的插件，叫做`CSSPlugin`。它会判断目标是否是DOM元素，如果是，则会修改它的内联样式(inline-style)。

CSSPlugin很强啦，balabala……

其实讲道理，CSSPlugin只是众多插件中的一个。不过动画化这件事情中修改css样式太普遍了，所以我们直接把它作为默认行为了，你不用把它像这样`css:{}`装起来。让你少写很多代码。不用谢~

CSSPlugin对所有的`transform`样式都能处理，而且性能比原生的更好。

性能提示：浏览器处理`x`和`y`的变形，比起`top`和`left`来说性能更好，所以尽量使用前者。

### from()方法

你可以在css里写好最终样式，而把起始样式交给GSAP来处理，有时候这会非常方便！

### set()方法

它会不带动画地立即生效。为啥需要这个？因为它接受`to()`和`from()`一样的参数。

### 常用的特殊属性

- `duration`: 动画持续时间。
- `delay`： The delay before starting an animation.
- `onComplete`： A callback that should be invoked when the animation finishes.
- `onUpdate`： A callback that should be invoked every time the animation updates/renders.
- `ease`： The ease that should be used (like "power2.inOut").
- `stagger`： Staggers the starting time for each target/element animation.

### 变速 ease

作为特殊属性之一，它实质上是一些数学函数，用于计算「时间-位移」，从而实现变速运动效果。

GSAP提供了一批内置设定，你也可以根据需求自己拟合一个。

### 梯次 Staggers

它不仅仅接受一个数值，甚至也接受一个对象，可以定义很多副属性。

你甚至可以用它实现二维特效。

### 回调 Callbacks

- `onComplete`: invoked when the animation has completed.
- `onStart`: invoked when the animation begins
- `onUpdate`: invoked every time the animation updates (on every frame while the animation is active).
- `onRepeat`: invoked each time the animation repeats.
- `onReverseComplete`: invoked when the animation has reached its beginning again when reversed.

回调函数可以传参数哦！

### 控制动画 

可以对创建动画时返回的对象进行操作：

```typescript jsx
//create a reference to the animation
var tween = gsap.to("#logo", {duration: 1, x: 100});

//pause
tween.pause();

//resume (honors direction - reversed or not)
tween.resume();

//reverse (always goes back towards the beginning)
tween.reverse();

//jump to exactly 0.5 seconds into the tween
tween.seek(0.5);

//jump to exacty 1/4th into the tween's progress:
tween.progress(0.25);

//make the tween go half-speed
tween.timeScale(0.5);

//make the tween go double-speed
tween.timeScale(2);

//immediately kill the tween and make it eligible for garbage collection
tween.kill();
```

### 时间线 Timeline

Timeline 其实是 动画对象 的 **容器**。

- 同时控制多个动画
- 便捷地实现动画顺序。（不需要你费劲去算各种delay了）
- 有助于代码模块化，从而写出 [更复杂的舞台编排](https://css-tricks.com/writing-smarter-animation-code/) 。
- 对一组动画执行统一的回调函数。

对Timeline对象连续调用多个`.to()`，会创建多个动画，并且它们将按顺序执行。

在创建Timeline对象时，可以指定 `default` 属性，它将会被所有动画继承。

### GSAP-2: 位置参数 Position Parameter

Timeline对象的`.to()` `.from()` `.fromTo()` 三个动画方法，都接收一个额外的参数`position`，它可以调整时间线上的执行顺序。

其中，`.addLable()`可以创建一个时间点，让多个动画同时开始。

### Getter / Setter 方法

动画(`Tween`)和时间线(`Timeline`)对象的属性，都可以动态调整。使用setter方法。

### GSAP-3: 获取元素当前属性

在`function`常规函数中，用`this.targets()`可以访问到当前元素(们).

### Club GreenSock

这似乎是这个框架背后团队的一个商业会员组织。有一些商业版的插件可以使用。

## 需求分解

> 版权警告：接下来所使用的美术资源均属于版权方所有，本文仅作交流学习使用。

好，框架学完了，接下来回到主题，今天我们要抄B站的一个活动页来着。

把需求简化一下，先把美术资源提取出来。

第一步：Loading页面。

- [GIF动图](https://i0.hdslb.com/bfs/activity-plat/static/20201212/256a1a14b990ce65d4a3168e1090a5f7/TWooRk9HO.gif)
- 等待视频资源加载完毕。

第二步：开幕动画

- [mp4](https://activity.hdslb.com/blackboard/static/20210816/70bb422013eb4692db31a31f5240742d/DwhRmMyI0P.mp4)
- 视频播放完毕后自动跳转。

第三步：主屏幕一

- 顶部导航栏
- [mp4](https://activity.hdslb.com/blackboard/static/20210816/70bb422013eb4692db31a31f5240742d/gZmugR6GWm.mp4) 作为背景，循环播放。
- 中间放一个 [固定图片](https://i0.hdslb.com/bfs/activity-plat/static/20210721/70bb422013eb4692db31a31f5240742d/4ar7CINGre.png)
- 底部一个 [上下移动的箭头](https://i0.hdslb.com/bfs/activity-plat/static/20210714/70bb422013eb4692db31a31f5240742d/c7Zab3R4Sb.png)
- 向下滚动则切换第四步，向上滚动则切回第三步。

第四步：主屏幕二

- 中央部分元素由底部向上滚动出现。
- 顶部导航栏、左右悬浮图片，自由发挥。

不需要移动端适配，简单糊弄一下屏幕宽度适配即可。B站做法是由js判断客户端之后跳转移动端专用url，而不是在本页内适配。

## Step1: Loading & Start

前两步合在一起做吧，没几行代码。

首先我们在某个顶层组件中，要同时摆上 loading图 和 开幕video ，这样后者才能开始加载。（当然也有其他的预加载方式，这里暂不深究。）所以我们的顶层组件大概长这样：

```typescript jsx
export function App(): JSX.Element {
    return (
        <div className={styles.app}>
            <div className={styles.loadingContainer} style={{ backgroundImage: `url(${imageLoading})` }} />

            <video>
                <source src={videoStart} type="video/mp4" />
            </video>
        </div>
    );
}
```

然后我们要一个`loading`的state，然后监听`video`元素的数据加载完成事件，加载完成后设置loading为false，隐藏loading图并展示video元素。不要忘了还要开始播放哦。

```typescript jsx
export function App(): JSX.Element {
  const [loading, setLoading] = React.useState(true);

  const ref1 = React.useRef<HTMLVideoElement>();
  const handleLoad1 = React.useCallback(() => {
    setLoading(false);
    ref1.current.play();
  }, []);

  return (
    <div className={styles.app}>
      <div
        className={classNames(styles.loadingContainer, loading || styles.hide)}
        style={{ backgroundImage: `url(${imageLoading})` }}
      />
      <video className={classNames(styles.videoStart)} muted preload="auto" ref={ref1} onLoadedData={handleLoad1}>
        <source src={videoStart} type="video/mp4" />
      </video>
    </div>
  );
}
```

接下来我们就要准备第二段视频了。因此：

- 首先，`loading`如果还是boolean类型的话那就不够用了，要么我们准备一个`loading2`，要么把它改造成number用位运算。
- 第一段视频加载完成之后，不能自己开始播放了，要等第二段也完成，loading状态彻底结束才可以开始播放。
- 要监听第一段视频播放结束的事件。

代码就不贴了，请参考 我的github 。

## Step2: 主屏幕一

咦，在前面的步骤中，顶上居然也有导航栏的吗。那要稍微调整一下布局。

记得导航栏在后面也要跟随页面滚动。在顶部插入一个宽度100%，高度固定的div作为导航栏，然后把前面一个步骤中写的几个背景元素装到另一个div中，这个div设置一个最小高度`calc(100vh - ??)`。

然后把页面上两张图片装进去。注意这里图片尺寸会响应页面**高度**，所以要设置`height: ??vh`，具体的数字还不好直接抄，慢慢调整一组看起来顺眼的数字就好。

其中下面那个箭头，嗯，要写动画了，来现学现卖写个gsap：

```typescript jsx
React.useEffect(() => {
    const tween = gsap.to(ref.current, {
        y: '2vh',
        repeat: -1,
        duration: 1,
        ease: "power1.inOut",
        yoyo: true,
    });
    return ()=>{
        tween.kill()
    }
}, []);
```

关于`ease`算法的选择，官方提供了一个 [可视化选择器](https://greensock.com/docs/v3/Eases) 还挺好用的。 

实现效果不错，作为验证我还是写了一份css的，肉眼观察效果是完全一样的：

```css
img{
    animation: bounce-down 1s ease-in-out infinite alternate;
}
@keyframes bounce-down {
    to{bottom: 3vh}
}
```

## Step3: 主屏幕二

难点主要是滚动切换，要看看怎么处理滚动事件。

我的思路，把两个屏视为两个独立的组件，然后通过状态提升，把他们的切换条件放在父组件中。这里由于需要淡出淡入，所以没有直接把组件干掉，而是传入`visible`让组件自己处理自己的动画。

```typescript jsx
function Main(props: { visible: boolean }): JSX.Element {
    const { visible } = props;
    const [page, setPage] = React.useState(1);
    return (
        <>
            <MainPage1 handover={() => setPage(2)} visible={visible && page === 1} />
            <MainPage2 handover={() => setPage(1)} visible={visible && page === 2} />
        </>
    );
}
```

在每个组件中，分别监听`onWheel`事件而不是`onScroll`事件。对于上面这个屏，向下滚动则切换；对于下面这个屏，滚动到了负数则切换。然后两个组件通过切换`z-index`来避免互相干扰。

像这样由两个组件分别监听滚动事件，有个缺陷，是在快速上下滚动连续切换时，滚动的目标组件会不能正确切换，要移动一下鼠标才能滚动到正确位置。不过我看了下B站的实现也有这个问题，所以就不优化了。

最终代码效果请见 我的github仓库。

## 结语

好家伙，虽然说是学习`GSAP`，可是最后还是用最顺手的原生css(3)来解决了问题。

不过，应该不能说是GSAP太烂，而大概是还没有到那么复杂的场景，暂时还用不上它。今天一天大概学了五六个小时，体验下来，总体体验还是不错的，文档也非常齐全，而且在学习过程中还了解了不少的额外知识。所以我觉得对于前端同学来说，这个库非常值得一学。

然后最后抄出来的这个页面呢，可以在 [这里预览](https://saodd.github.io/learn-gsap/) ，看着是挺炫酷的，不过其实大部分都是已经做好的视频和图片素材了哈哈哈，真正写的代码也就 一百多行ts + 一百多行scss ，真的不算多，花的时间也不多，大概就四五个小时吧。

这下差不多可以自称是一个会写动画的前端程序员了吧哈哈哈哈！

叉会腰！
