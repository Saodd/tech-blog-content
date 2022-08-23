```yaml lw-blog-meta
title: "用 dayjs 替换 antd 中的 moment"
date: "2022-08-23"
brev: "真的折腾"
tags: ["前端"]
```

## 背景

前端同学应该都很熟悉`moment.js`和`day.js`这两个库了吧。一般项目中的主流都是用明显更轻量的后者去替代前者。

然而在`antd`~~这个业界毒瘤~~中，日期/时间相关的组件依赖的依然是`moment.js`。

这几天我在优化webpack的构建，因此也顺便尝试做一下用`day.js`去替换。

## 官方方案

首先`antd`他自己是有推出[方案](https://ant.design/docs/react/replace-moment-cn)的。

对上面这个文章，在我的印象中似乎以前是没见过的，因此猜测可能是最近才公布的。话里话外的意思，似乎是antd还暂时没有在后续版本中默认替换掉moment的计划，所以要我们继续用魔法来改造。

归纳一下上述文章中的方案。

首先是方案一，用antd内置的一个工具来构建几个新的组件，用这些新的组件就可以指定依赖dayjs了。

```tsx
import * as React from 'react';
import { Dayjs } from 'dayjs';
import dayjsGenerateConfig from 'rc-picker/es/generate/dayjs';
import generatePicker from 'antd/es/date-picker/generatePicker';
import { PickerTimeProps } from 'antd/es/date-picker/generatePicker';
import generateCalendar from 'antd/es/calendar/generateCalendar';

/**
 * -------------------- DatePicker --------------------
 */
export const DatePicker = generatePicker<Dayjs>(dayjsGenerateConfig);

/**
 * -------------------- TimePicker --------------------
 */
export const TimePicker = React.forwardRef<unknown, TimePickerProps>((props, ref) => {
  // @ts-ignore
  return <DatePicker {...props} picker="time" mode={undefined} ref={ref} />;
});
type TimePickerProps = Omit<PickerTimeProps<Dayjs>, 'picker'>;
TimePicker.displayName = 'TimePicker';

/**
 * -------------------- Calendar --------------------
 */
export const Calendar = generateCalendar<Dayjs>(dayjsGenerateConfig);

```

用上述代码简单改造了一下，没遇到什么大坑。

然后是方案二，用`antd-dayjs-webpack-plugin`这个webpack插件来替我们做这个事情。对这个方案我没有深入研究，但是感觉他的兼容性难免会有问题——他能改antd中的依赖，可是对于我们项目代码的依赖肯定是解决不了的。

## external的矛盾

由于`antd`本身就是一个体积庞大的组件库，因此在我们的项目中已经为它配置了`external`，由独立的`script`标签来引入，而不经过webpack的处理。

这就造成一个很大的问题：预先构建好的`antd.min.js`文件中依赖的依然是moment，而经过webpack处理的业务代码却希望它依赖dayjs，这显然是不合理的。

说到这里我们看一下`antd.min.js`，简单了解一下它是如何引入依赖的：

```js
// antd.js v4.21.5
(function webpackUniversalModuleDefinition(root, factory) {
    root["antd"] = factory(root["moment"], root["React"], root["ReactDOM"]);
})(window, function(__WEBPACK_EXTERNAL_MODULE_moment__, __WEBPACK_EXTERNAL_MODULE_react__, __WEBPACK_EXTERNAL_MODULE_react_dom__) {})
```

由上面的代码可以看出，这个js文件会尝试从`window`上分别读取已经挂载了的`moment`, `React`, `ReactDOM`三个属性（模块对象）。其中，后面两个是必须的，否则无法渲染出UI界面来（在调试CDN引入顺序的时候会遇到这个问题，调整顺序即可）；而moment这个东西呢，如果我们业务代码中没有使用`DatePicker`, `TimePicker`, `Calendar`这三个时间相关的组件的话，缺了它也是可以正常运行的，即此时它被以`undefined`的值传入，不调用就不会报错。

我还尝试魔改，将`root["moment"]`改为`root["dayjs"]`，但是失败了，因为dayjs与moment的api并不完全相同，不能做到直接无感替换。

因此，我面临取舍：坚持使用external，那就要把moment也引入external；坚持去除moment，那就要把antd引入打包。

## 各方案构建体积对比

> 注：下面所说的文件体积都是文件在服务器上的原始体积，没有经过gzip等压缩处理。

### 方案一：保留moment

继续保留当前antd以及我们业务代码对moment的使用，通过external来引入。

这其实也是最简单的方案，或者说对目前项目代码兼容性最好的方案，改动最小。

所需的文件：

- `moment.min.js`(含`zh-cn`): 59 KB
- `antd.min.js`: 923 KB
- 构建出的`vendors.js`: 314 KB

### 方案二：自定义组件

依照官方的方案一，声明几个自定义组件，用`dayjs`全面替换掉`moment`。

在这种模式下，由于我们需要引入`rc-picker`和`generatePicker`等组件，这部分是不能被webpack的external处理的，我们可以借助打包分析工具来观察到相关几个额外包的引入。因此这是一种“混合”方案，其实看起来不太靠谱，而且由于部分代码重复使用了，总体积是最大的。

所需的文件：

- `antd.min.js`: 923 KB
- 构建出的`vendors.js`: 647 KB

### 方案三：借助插件构建整个antd

依照官方的方案二，用插件来替换antd的组件，我们只需要手动将业务代码中的moment改为dayjs依赖即可。

其实插件只是让我们少写了几行代码，实际的依赖一个不少。不过由于tree-shaking机制的存在，打包后的总体积是最小的。

所需的文件：

- 构建出的`vendors.js`: 1094 KB

### 小结：方案选择

如果选择方案一，可以得到最小的构建产物体积。虽然对比方案三来说总体积更大一些；但是由于可以依赖cdn的缓存机制，二者的实际流量表现可能很难估计，只能从统计结果来得出结论。

除了体积之外，再看看其他因素：

- 方案一构建输入少，开发/构建时的硬件压力会明显低一个档次。方案一+1
- 方案一依赖了更零碎的js，可能导致用户首屏加载时间延长。方案三+1
- 方案一的配置项更多。方案三+1

其实冷静一下，上面几个方案的核心区别在于external范围的抉择，这并不是我最关心的问题。思考一下我的最初目的——为了减少构建体积而放弃moment改用dayjs 。如果仅仅是这个目的的话，那么dayjs给我带来的好处是弥补不了它给我带来的麻烦的。

因此，最后结合项目实际情况，我的决定是继续保留`moment`。

你会做何选择呢？
