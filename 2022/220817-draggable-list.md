```yaml lw-blog-meta
title: "前端组件：可拖拽列表"
date: "2022-08-17"
brev: "产品交互做得好不好，全看它了"
tags: ["前端"]
```

## 背景

只要是列表、有顺序的列表、而且顺序有意义的列表，那么就肯定会有拖拽排序的需求。这个需求很常见。

常有人吐槽，说国外大厂的产品在交互上做得如何如何好。而『可拖拽列表』这个组件由于其很高的复杂性，在产品之间对比的时候总是能见到它作为典型案例出现。

虽然目前也已经有很多成熟的开源库，参考：[10 Best React Drag & Drop List Libraries](https://openbase.com/categories/js/best-react-drag-and-drop-list-libraries)。

但保持我的一贯作风，今天不用开源的，来尝试自己手撸一个。成品效果参见[展示页面](https://lewinblog.com/labs/220530-css-gallery)

## 0. 总体思路

1. 首先我们要有能力实现一个元素的拖拽
2. 把一个元素提升，做成一组元素
3. 样式美化
4. API设计优化

## 1. 单个组件的拖拽

### 1.1 监听事件

其实HTML也有现成的实现：[draggable](https://developer.mozilla.org/en-US/docs/Web/HTML/Global_attributes/draggable) ，但它的定制化程度很低，尤其是在样式方面，完全不能满足需求，因此不能简单依赖它。

而放弃draggable意味着放弃了整个`Drag`事件家族，因此我们只能依赖`Mouse`（或`Pointer`）事件家族来感知拖拽事件。我们需要四种事件：

- `pointerdown`：开始拖拽
- `pointerup` / `pointercancel`：结束拖拽
- `pointermove`：移动

在注册事件的时候要特别注意，后面三个事件是需要挂载在window上的，这样才能响应全窗口（甚至[全屏幕](https://stackoverflow.com/questions/1685326/responding-to-the-onmousemove-event-outside-of-the-browser-window-in-ie)）的鼠标事件，而不被局限在一个小区域内。

```tsx
const DraggableItem: FC = () => {
  const [dragging, setDragging] = useState(false);

  useEffect(() => {
    if (!dragging) return;

    const handlerMove = () => {};
    const handlerCancel = () => {};

    window.addEventListener('pointermove', handlerMove);
    window.addEventListener('pointerup', handlerCancel);
    window.addEventListener('pointercancel', handlerCancel);
    return () => {
      window.removeEventListener('pointermove', handlerMove);
      window.removeEventListener('pointerup', handlerCancel);
      window.removeEventListener('pointercancel', handlerCancel);
    };
  }, [dragging]);

  return <div onPointerDown={() => setDragging(true)}>我是一些内容</div>;
};
```

### 1.2 坐标与位移

在鼠标拖拽过程中，被拖拽的元素肯定也要在视觉上跟随鼠标一起移动的。

为了计算『移动』这个东西，我们能够想到最熟悉的东西无非就是`transform`这个属性了。同时别忘记了`position`的`relative`和`absolute`。

`transform`需要的是『偏移量』，也是说，我们需要一个初始值和一个当前值。初始值在`pointerdown`的时候固定下来，当前值则随着`pointermove`不断变化，以此得到一个在不断变化的偏移量。

接下来分别用`x`、`y`来表示横向（向右）和纵向（向下）的偏移量，具体的数值用`clientX/Y`来获取：

```tsx
const DraggableItem: FC = () => {
  const [dragging, setDragging] = useState(false);

  const [x0, setX0] = useState(0);
  const [y0, setY0] = useState(0);
  const [x, setX] = useState(0);
  const [y, setY] = useState(0);

  useEffect(() => {
    if (!dragging) return;

    const handlerMove = (e: PointerEvent) => {
      setX(e.clientX);
      setY(e.clientY);
    };
    const handlerCancel = () => {
      setDragging(false);
    };

    window.addEventListener('pointermove', handlerMove);
    window.addEventListener('pointerup', handlerCancel);
    window.addEventListener('pointercancel', handlerCancel);
    return () => {
      window.removeEventListener('pointermove', handlerMove);
      window.removeEventListener('pointerup', handlerCancel);
      window.removeEventListener('pointercancel', handlerCancel);
    };
  }, [dragging]);

  return (
    <div style={{ position: 'relative' }}>
      <div
        onPointerDown={(e) => {
          setDragging(true);
          setX0(e.clientX);
          setY0(e.clientY);
          setX(e.clientX);
          setY(e.clientY);
        }}
        style={{ position: 'absolute', transform: dragging ? `translate3d(${x - x0}px,${y - y0}px,0)` : undefined }}
      >
        我是一些内容
      </div>
    </div>
  );
};
```

这样就做好了一个普通的可拖拽元素组件。

## 2. 一组元素的拖拽

### 2.1 组的框架

即使是一组元素，鼠标也只能同时拖拽一个元素，因此上面所写的绝大部分逻辑都可以保留在列表组件这一级，只需要把`onPointerDown`下放到子元素中去即可

> 啊咧，说起来我好像还真没考虑过移动端环境，在多点触控设备的支持下，如果要同时拖拽多个元素要怎么做……

```tsx
const DraggableList: FC = ({ children }) => {
  const [draggingIndex, setDraggingIndex] = useState(-1);
  // ...
  useEffect(() => {
    if (draggingIndex < 0) return;
    const handlerCancel = () => {
      setDraggingIndex(-1);
    };
    // ...
  }, [draggingIndex]);

  return (
    <div style={{ position: 'relative' }}>
      {children.map((child, index) => (
        <div
          onPointerDown={()=>{ setDraggingIndex(index); /* ... */ }}
          style={{
            position: 'relative',
            transform: draggingIndex === index ? `translate3d(${x - x0}px,${y - y0}px,0)` : undefined,
          }}
        >
          {child}
        </div>
      ))}
    </div>
  );
};
```

上面的代码中，注意`position: 'relative'`这个属性，相比于`absolute`，前者可以在原来的位置撑开属于它的空间，同时又可以相对移动。

### 2.2 新的顺序

当一个元素被拖拽到另一个元素的位置上的时候，我们需要一种交互视觉效果，即让正在被拖拽的元素“挤开”当前位置的元素。

首先第一个问题：如何判断当前拖拽到了某个元素的上方？——直觉告诉我，也许可以使用`mouseenter`之类的事件，然而，由于我们正在拖拽的元素也在跟随鼠标移动，它一直“遮挡”着鼠标指针，因此下方被遮盖的元素无法触发`mouseenter`事件。

因此我的蠢办法是：计算各个元素的尺寸来计算。

可如果以这种方案，想要做得很完美的话，我需要记录所有元素的尺寸，这会让代码变得非常复杂。因此先做一些简化：假设我们只在纵向这一个方向上拖拽，并且上层组件明确指定了所有子元素统一的高度值。

然后是第二个问题：“挤开”的过程，其实就是数组位置的变换。为了实现这个效果，我们需要一个临时的数组顺序。

（代码省略）

### 2.3 保存新的顺序

为了实现这个目标，我们需要设计一个API，大概长这样：`onUpdate: (data: unknown[])=>void`。

为了实现这个API，那么意味着我们需要把`data`这个东西给传进来，既然它来了，那么负责渲染data的React组件也要传进来。

于是我们的API变成了这样：

```tsx
function DraggableList<dataT>(
  props: PropsWithChildren<{
    data: (dataT & { key: React.Key })[];
    Cpn: FC<dataT>;
    height: number;
    onUpdate: (data: dataT[]) => void;
  }>,
): ReactElement | null {
    // ...
    return (
        <div>
            {data2.map(d=> <Cpn {...d}/>)}
        </div>
    )
}
```

### 2.4 滚动问题

我们还需要考虑很多因素。

第一，滚动问题。由于一个列表组件的长度总是有限的，很可能会出现滚动条，因此我们需要上层传入一个滚动发生的元素，称它为`scrollParent`，然后我们从它上面取`scrollTop`来辅助我们的`y`坐标计算以及新的数组排序。

第二，父容器尺寸变化。受到页面本身尺寸或者其他元素的影响，父容器的尺寸也可能是会发生变化的。但实质上依然可以看作是“滚动”问题，我们需要`ResizeObserver`这个东西来通知我们就行了。

至此为止，我们的主要功能就全部实现了，虽然简陋了一些，不过该有的功能都有了。

## 3. 样式与交互优化

这块内容那就可以说是没有上限了，似乎是可以无限优化下去的。

我简单列举几个需要关注的css属性：

- user-select
- transition
- opacity
- z-index
- background-color
- cursor

交互上可能还用到一些比较偏门的js能力，例如：

- Element.clone

（精力有限，暂时不展开讲。）

## 4. API设计优化

这块我希望参考一下成熟的开源组件是如何实践的。

我想关注一下[React DnD](https://react-dnd.github.io/react-dnd/about)这个库，等我体验完了来更新。
