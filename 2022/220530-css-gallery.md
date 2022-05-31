```yaml lw-blog-meta
title: "CSS: 从轮播图到老虎机"
date: "2022-05-30"
brev: "动画也还挺好玩的"
tags: ["前端"]
```

## 轮播图

众所周知：『一个合格的前端至少也要能够达到会写轮播图的水平吧！』

同时它也是前端应用中非常常见的展示组件。在ant中这个组件的名字叫做[走马灯(carousel)](https://ant.design/components/carousel-cn/) ，虽然听着有点毛骨悚然，不过还是很形象的。

接下来，从简单的架构开始，逐步优化一个轮播图组件吧：

### 1. 轮播图的基本结构

首先，这个组件应当接受一个数组结构的数据，代表着需要轮播展示的内容。比较理想的实现应该是一组相似的组件，以`children`参数的形式传入；我们先简化一下，接受一个包含url的`string[]`结构。

其次，既然要"轮"而且还要能控制、切换，那就需要一个状态量来记录当前展示到哪个元素了。

```tsx
const Gallery: FC<{ urls: string[] }> = ({ urls }) => {
  const [current, setCurrent] = useState(0);

  return (
    <div className={styles.Gallery}>
      <div>
          {/* ... */}
      </div>
      <button onClick={() => setCurrent(current - 1)}>左</button>
      <button onClick={() => setCurrent(current + 1)}>右</button>
    </div>
  );
};
```

### 2. 图片的移动

要让多个图片左右切换，很容易能够想到，我们需要`transform`属性来控制图片的相对位置，然后用`overflow: hidden`来隐藏那些不需要的图片。如果再加上一些`transition`属性的话，滑动就可以变得更加丝滑。

```tsx
<div className={styles.GalleryWrapper}>
  <div
    className={styles.GalleryImages}
    style={{ transform: `translate3d(${-current * 90}px, 0, 0)`, width: `${urls.length * 90}px` }}
  >
    {urls.map((url) => (
      <img src={url} key={url} />
    ))}
  </div>
</div>;
```

```scss
.GalleryWrapper {
  width: 90px;
  height: 120px;
  position: relative;
  overflow: hidden;

  .GalleryImages {
    height: 120px;
    transition: transform 0.5s;
    position: absolute;

    & > img {
      display: inline-block;
      width: 90px;
      height: 120px;
    }
  }
}
```

### 3. 首尾相连

先是索引本身的计算，相对简单，减到负数的时候就切回尾部，加过头了就切回头部。

然后是一个比较麻烦的：DOM元素的动画要如何过渡？——具体来说，我们希望最后一个元素继续向右的时候，第0个元素能够正确地从右方滑动入场，而不是整个列表快速倒退回去。

为此我们可以在最后一个元素后面，再加一个第0元素。即 `[0,1,2,3]` 渲染成 `[0,1,2,3,0]` 。

当越界的时候，等动画播放完毕后，暂时关闭`transition`然后再立即调整`transform`，这样就能在用户没有感知的情况下完成一轮循环。

```typescript
const moveRight = useCallback(() => {
  setCurrent(current + 1); // 正常右移

  if (current === urls.length - 1) {
    // 如果移到了尾部
    setTimeout(() => {
      setTransition('none'); // 关闭动画，然后闪回头部
      setCurrent(0);
      setTimeout(() => {
        setTransition('transform 0.5s'); // 重新开启动画
      }, 200);
    }, 500);
  }
}, [current]);
```

上面的实现其实并不完美，因为我拍脑袋定了一个`200ms`的调整间隔，这个间隔在真实的用户场景下可能是不准确的，也许会发生预想不到的结果（不过这个结果也就仅仅是动画上的，只影响视觉体验，并不影响用户功能）。

对这个间隔的问题就很不好优化了，作为参考，我看了下ant的组件实现得也并不完美。其根本原因在于，我们js里面很难确定css的实际执行状态，因此总是不可避免地要做一些强行设定，导致在某些极端场景下的动画还是不够完美。

如果一定要优化的话，我认为终极解决方案应该是：**由js来控制动画关键帧**！这样我们才能清楚地知道当前动画执行到哪一帧，才能做出最完美的优化。具体实现可以选择`GASP.js`这个框架，顺带一提，B站前端大量使用了这个库。

### 4. 懒加载

如果希望提升加载速度，懒加载可能是一个很常见的选择。即，只加载当前可见的元素和它前后两个元素（为了动画效果），其他元素不加载。

```tsx
const Gallery: FC<{ urls: string[] }> = ({ urls }) => {
  const [current, setCurrent] = useState(urls.length - 1);

  const moveLeft = () => {
    setCurrent(current === 0 ? urls.length - 1 : current - 1);
  };
  const moveRight = () => {
    setCurrent((current + 1) % urls.length);
  };

  const u1 = urls[current];
  const u2 = urls[(current + 1) % urls.length];
  const u3 = urls[(current + 2) % urls.length];

  return (
    <div className={styles.Gallery}>
      <div className={styles.GalleryImages}>
        <img src={u1} key={u1} className={styles.left} />
        <img src={u2} key={u2} className={styles.mid} />
        <img src={u3} key={u3} className={styles.right} />
      </div>
      <button onClick={moveLeft}>左</button>
      <button onClick={moveRight}>右</button>
    </div>
  );
};
```

上面的代码里，要特别注意img标签的`key`属性，必须要有它，React才会帮我们复用同一个DOM、在同一个src上替换它的class以形成动画效果。如果不写的话，那就是只改src而不改class，这将毫无意义。

```scss
.GalleryImages {
  width: 90px;
  height: 120px;
  position: relative;
  overflow: hidden;

  & > img {
    width: 90px;
    height: 120px;
    transition: transform 0.5s;
    position: absolute;

    &.left {
      transform: translate3d(-90px, 0, 0);
    }
    &.mid {
      transform: translate3d(0, 0, 0);
    }
    &.right {
      transform: translate3d(90px, 0, 0);
    }
  }
}
```

![gallery](../pic/2022/220530-gallery.gif)

这个方案也有缺点：只渲染3个DOM，会导致不能快速跳转（例如用户想从第1个跳到第4个）。如果要解决这个问题，可以考虑多渲染几个节点，然后做个自动连续点击的功能来模拟跳转。

### 5. 索引小圆点

上面的实现中，我只简单写了 左、右 两个按钮。在实际中我们可能会需要在下方显示多个圆点来代表图片索引。

这个只是一个简单的固定位置的组件，结合上面已经维护好的`current`状态，很好写，我就不展开讲了。

## 老虎机

本章内容参考了这篇文章：[产品经理：能不能让这串数字滚动起来？](https://juejin.cn/post/6986453616517185567)

老虎机可以看成是一种特殊类型的轮播图，但是又有一些根本上的区别——它的中间状态不需要交互。因此老虎机可以直接简化成两段动画：一段通用的刷屏动画 + 最后定格数字的动画。

### 1. 刷屏数字的基本结构

基本结构与轮播图是类似的：父元素充当遮罩层的作用，关键属性是`overflow: hidden`；子元素是一个纵向的数字列表，从0~9最后再加一个0，一共11个数字，刚好10层偏移高度。

因此一个最基本的框架如下：

```tsx
const Tiger: FC = () => {
  return (
    <div className={styles.Tiger}>
      <div className={styles.TigerNumbers}>
        <div>0</div>
        <div>9</div>
        <div>8</div>
        <div>7</div>
        <div>6</div>
        <div>5</div>
        <div>4</div>
        <div>3</div>
        <div>2</div>
        <div>1</div>
        <div>0</div>
      </div>
    </div>
  );
};
```

```scss
.Tiger {
  width: 1em;
  height: 1.5em;
  position: relative;
  overflow: hidden;

  .TigerNumbers {
    position: absolute;
    animation: NumberRoll 3s infinite linear;

    & > * {
      line-height: 1.5em;
      height: 1.rem;
    }
  }
}

@keyframes NumberRoll {
  0% {
    transform: translate3d(0, -15em, 0);
  }
  100% {
    transform: translate3d(0, 0, 0);
  }
}
```

这样我们就得到了一个无限循环滚动的数字。

### 2. 最后定格数字

很简单来一个回弹动作：

```scss
@keyframes NumberStop {
  0% {
    transform: translate3d(0, -1.5em, 0);
  }
  50% {
    transform: translate3d(0, 0.2em, 0);
  }
  75% {
    transform: translate3d(0, -0.3em, 0);
  }
  100% {
    transform: translate3d(0, 0, 0);
  }
}
```

上面的写法是针对单独一个数字的情况。在本例中，我们要对整个数字列表施加样式，因此在y轴上还要增加一个偏移量。

### 3. css变量

由于接下来的主要内容都是写在`@keyframes`里的，这里的东西用JSX来表达不太现实，因此我们需要引入一个新的工具：[CSS变量](https://developer.mozilla.org/en-US/docs/Web/CSS/Using_CSS_custom_properties)

简而言之，它的用法就是先定义，名称必须以两个减号开头，例如`--num: 9px`，定义之后再取出来使用`var(--num)`，相当于是字符串替换，可以参与`calc`运算。

我们借助css变量来实现前一小节提到的"偏移量"的概念。我将其命名为`--num`，它的计算方式很简单：当我们最后需要停在数字`n`，我们就从底部向上偏移`n`个行高即可。

```tsx
<div style={{ '--num': `${((num || 10) - 10) * 1.5}em` }} />
```

与前面的`NumberStop`动画结合，最后我们得到它的终极样式：

```scss
@keyframes NumberStop {
  0% {
    transform: translate3d(0, calc(var(--num) - 1.5em), 0);
  }
  50% {
    transform: translate3d(0, calc(var(--num) + 0.2em), 0);
  }
  75% {
    transform: translate3d(0, calc(var(--num) - 0.2em), 0);
  }
  100% {
    transform: translate3d(0, var(--num), 0);
  }
}
```

### 4. 结合两种状态

我们用一个状态量来表示当前是正在播放"刷屏动画1"还是"定格动画2"，这个状态量通过`setTimeout`来简单地更新。终极代码：

```tsx
const Tiger: FC<{ num: number; delaySec: number }> = ({ num, delaySec }) => {
  const [rolling, setRolling] = useState(true);
  useEffect(() => {
    setTimeout(() => setRolling(false), delaySec * 1000);
  }, []);

  return (
    <div className={styles.Tiger}>
      <div
        className={`${styles.TigerNumbers} ${rolling ? styles.rolling : styles.stop}`}
        style={{ '--num': `${((num || 10) - 10) * 1.5}em` }}
      >
        <div>0</div>
        <div>9</div>
        <div>8</div>
        <div>7</div>
        <div>6</div>
        <div>5</div>
        <div>4</div>
        <div>3</div>
        <div>2</div>
        <div>1</div>
        <div>0</div>
      </div>
    </div>
  );
};
```

```scss
.Tiger {
  width: 1em;
  height: 1.5em;
  position: relative;
  overflow: hidden;
  background-color: lightgreen;

  .TigerNumbers {
    position: absolute;
    width: 100%;
    text-align: center;

    & > * {
      line-height: 1.5em;
      height: 1.5em;
    }

    &.rolling {
      animation: NumberRoll 0.3s infinite linear;
    }

    &.stop {
      animation: NumberStop 0.6s forwards;
    }
  }
}
```

最后得到的效果：

![tiger-number](../pic/2022/220530-tiger-number.gif)

其实这种方式实现的动画并不算完美，因为当我们切换`rolling`状态的时候，y轴偏移量是有一个突变的，只不过这里动画速度太快，人的肉眼看不出来罢了。

如果要做完美的话，可能需要考虑从终结状态来推导初始状态；但这样可能也会有时间误差，如果要最精确的动画，终究还是需要js控制关键帧。
