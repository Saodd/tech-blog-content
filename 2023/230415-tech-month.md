```yaml lw-blog-meta
title: "技术月刊：2023年3月~4月"
date: "2023-04-15"
brev: "webpack插件数组的条件过滤, 用ts来编写webpack.config, antd升级v5, 如何在React组件中插入一个HTMLDOM, CI运行时安全, 杂谈AI热潮"
tags: ["技术月刊"]
```

## webpack插件数组的条件过滤

`webpack`的配置是支持以函数的形式指定的，也就是说我们可以在函数内部根据当前的运行环境来决定需要应用的配置。

然后自然地就产生了一个需求：仅当在`production`环境下我才需要加入某个`plugin`，而在`development`模式下我要删掉它。

这种条件判断的写法，很容易就能想到`?`运算符，但是麻烦的是，webpack的`plugins`配置数组中并不支持传入`null`等非值，传入的东西必须是符合插件接口定义的。

然后我看到了[Allow empty plugin in plugins list #5699](https://github.com/webpack/webpack/issues/5699)，看见一种我觉得虽然并不难但是感觉眼前一亮的写法：

```js
plugins: [
  null
].filter(Boolean)  // Boolean构造函数当作filter回调函数来用
```

以及一种最经典的写法：

```js
function NothingPlugin() {
    this.apply = function(){};
}

plugins: [
  someBool ? new NothingPlugin() : new TheRealPlugin({ ... })
]
```

两种方式都能解决问题。

## 用ts来编写webpack.config

现在前端工程化能力越来越强，随着项目的膨胀，webpack的配置文件也越写越复杂，为了可靠地维护它，我自然就会想：能否以typescript格式来编写呢？

现在ts已经很成熟了，开源社区里也有了像`ts-node`这样相对成熟可靠的ts运行时可以用来直接运行`.ts`文件，而不需要事先经过`tsc`或者`babel`的编译。因此这件事情理论上已经是可行的了。

所幸，webpack已经提出了对typescript（以及其他js类语言）的支持，参考阅读：

- [Is it possible to write webpack config in typescript? - stackoverflow](https://stackoverflow.com/questions/40075269/is-it-possible-to-write-webpack-config-in-typescript)
- [Configuration Languages - webpack.js.org](https://webpack.js.org/configuration/configuration-languages/#typescript)
- [With TypeScript - webpack-dev-server](https://github.com/webpack/webpack-dev-server#with-typescript)

主要流程：

1、安装依赖

```shell
yarn add ts-node
yarn add -D @types/webpack
# 此外还要确保 webpack-dev-server 是比较新的版本才能支持正确的ts类型 
```

2、改写webpack配置文件，

```ts
import type { Configuration as DevServerConfiguration } from "webpack-dev-server";
import type { Configuration } from "webpack";
import * as path from 'path';
import * as webpack from 'webpack';

const devServer: DevServerConfiguration = {};
const config: Configuration = {
    mode: 'development',
    devServer,
    // ...
};

export default config;
```

并将文件后缀改为`.ts`即可，不需要额外配置，webpack自己会尝试寻找名为`webpack.config.ts`的文件，并根据文件后缀选择相应的解释器（`ts-node`）来运行；或者在运行webpack的时候通过运行参数来指定配置文件。

3、配置`tsconfig.json`

```json
{
  "compilerOptions": {
    "module": "ESNext"
  },
  "ts-node": {
    "compilerOptions": {
      "module": "CommonJS"
    }
  }
}
```

## antd升级v5

[ant-design](https://github.com/ant-design/ant-design)推出V5版本已经有几个月的时间了。

最近终于有时间，看了一下它的[更新文档](https://ant.design/docs/react/migration-v5-cn)，其中最让我感兴趣的是它终于把`moment.js`替换为`day.js`了，这点还是挺吸引我的，因此打算研究一下看看能不能将现在使用 antd v4 的项目升级为 v5 。

更新内容主要包括：

- day.js
- 一些组件API的调整
- CSS引入方式彻底改变

API倒还好，主要影响层面是js代码部分，替换一下的成本倒也还可以接受。最难受的是CSS的兼容性——毕竟antd的兼容标准是"last 2 versions"——而在当前国内的主流环境中，用户的实际情况远远达不到如此高的兼容标准（例如很多用户使用的360浏览器的内核版本就只有80+）。

### 解决where选择器

css改动影响最大的是利用了`:where()`选择器来引入组件样式。为什么需要`:where()`？是因为antd使用了『组合器 + 类选择器』的方式来组织样式类名（复习一下概念：[CSS 选择器 - MDN](https://developer.mozilla.org/zh-CN/docs/Web/CSS/CSS_Selectors)），而这种方式的优先级比较高，会导致我们在希望覆盖antd组件样式的时候，我们自己传入的class优先级更低导致不生效。

举个例子，如下代码中我希望用`styles.MyClass`这个样式类去覆盖antd原本的组件样式：

```tsx
export const MyMenu: React.FC = () => {
  return <Menu className={styles.MyClass} />;
};
```

最后渲染出的来的DOM会是这样的：

```html
<ul class="css-dev-only-do-not-override-1e3x2xa ant-menu-light" role="menu">
</ul>
```

其中生效的样式如下：

```css
.css-dev-only-do-not-override-1e3x2xa.ant-menu-light {
    color: rgba(0, 0, 0, 0.88);
    background: #ffffff;
}

.styles--MyClass {
    /* ...我们的指定样式类被覆盖，不生效 */
}
```

为此，antd（的`cssinjs`）使用了`:where()`选择器来降低优先级，这样我们传入的自定义样式类就可以正常生效了。但是`:where()`选择器的兼容性是 [chrome>88](https://caniuse.com/mdn-css_selectors_where)，不符合我们项目需求。

官方给出的[解决方案](https://ant.design/docs/react/compatible-style-cn)是：

```tsx
import React from 'react';
import { StyleProvider } from '@ant-design/cssinjs';

// `hashPriority` 默认为 `low`，配置为 `high` 后，
// 会移除 `:where` 选择器封装
export default () => (
  <StyleProvider hashPriority="high">
    <MyApp />
  </StyleProvider>
);
```

如果按上面这样的配置，虽然消除了`:where()`选择器，但是取而代之的是组合类选择器，也就是前面所提到的会产生"高优先级问题"的那种方式。

如果一定要在保证浏览器兼容性的基础上再解决优先级问题，我们就必须把我们传入的样式类也同样提高优先级，例如可以同样使用"组合类选择器"。`sass`语法示例如下：

```sass
// 方案一：没有嵌套类，只能从全局类上选择
:global(.app-root) .MyClass {
  /* ... */
}

// 方案二：有嵌套类
.MyContainer {
  .MyClass {
    /* ... */
  }
}
```

但是这样的改造代价太大了，因此最后我不得不决定放弃升级v5版本，继续使用v4版本……

## 如何在React组件中插入一个HTMLDOM

React 的组件是叫做`Component`，组件实例化后的对象叫做`Element`，但是这个Element还只是VDOM虚拟树上的一个数据结构，它还要经过 ReactDOM 的处理（渲染）才能与当前页面上的真实DOM树保持一致。

而在某些特殊场景下，我单独维护了一个真实的`DOM-Element`，然后我需要通过某种方式将其插入渲染到React组件中去。

参考阅读：[How to render a DOM element into a React component?](https://medium.com/@amatewasu/how-to-render-a-dom-element-into-a-react-element-a7ce2aa51976)

如果是原生的HTML字符串，则可以通过`__dangerouslySetInnerHTML`来进行渲染，这个大家应该都很熟悉了。

如果是实例对象（例如`HTMLDivElement`），则需要借助`useEffect`的生命周期来手动挂载、卸载DOM节点了，参考代码：

```typescript jsx
const MyComponent: React.FC = (props) =>{
  const ref = useRef<HTMLDivElement>();

  useEffect(() => {
    const myDomElement = getTheDomElement();
    ref.current.appendChild(myDomElement);
    return () => {
        ref.current.removeChild(myDomElement);
    };
  }, []);

  return <div ref={ref}></div>
}
```

## CI运行时安全

CI 指的是 `continuous integration` ，典型代表是 jenkins, Drone 等自动化组件。

这些组件的本质，是由一些事件触发的自动化运行脚本。那么，既然要“执行脚本”，那就必须要考虑防范“脚本注入攻击”。

公共互联网环境自然不必说，但即使在公司内部的项目，基于“零信任网络”的原则，也依然需要考虑这类安全问题。比如说，假设万一某天某位员工突然决定铤而走险，动用他所有的权限去盗取公司的机密数据，他能盗取到多少？

首先必须承认的是，完全绝对可靠的防御体系是不存在的（——假如公司高层都决定跑路了，谁还能管的住他呢？）同时，越严密的防范体系也必定意味着越高的建设和使用成本。因此，我们在实际工作中只能在安全和便利之间做一个取舍。

ok，接下来我们缩小讨论范围，仅仅讨论在 Drone 这个技术体系下，如何保护 secret 的安全。

如果只是lint检查之类的脚本任务，除了访问仓库本身的代码之外没有其他需求，那自然也用不到secret；需要使用到的时候，往往是涉及到上线部署操作的时候，话句话说，此时已经不仅仅是『CI』了，而是已经进入了『CD』的领域。

参考[官方文档](https://docs.drone.io/secret/)，Drone本身是提供了一定的保护机制的。secret可以储存在Server中，并且在Server平台上只能写入、删除，而不能读取；在runner中运行的时候，可以以环境变量的形式读取出来使用。

因此我们需要关心的部分仅限于Runner部分。

Runner本身也做了一些基本的保护措施：它会检查日志输出，会把与secret完全相等的字符串替换为`********`。但是这种保护太弱了。例如一个最基本的操作，我只需要`echo ${secret:1}`便可以让这个保护机制失效。

如果项目中定义的`.drone.yml`无法锁定，或者其他构建工具（例如`webpack`）的脚本内容无法锁定，可以被团队成员任意编辑的话，那么通过简单的脚本注入攻击，即可获取CI运行时提供的所有环境变量，包括secret。

因此唯一可行的办法，是把构建和部署两个环节的运行环境完全分开，需要使用secret的部署部分的代码由更高权限的人来保管。具体在Drone这个技术栈里，就是业务代码仓库只负责构建，输出docker镜像等产物上传到其他地方，再由专门的部署脚本仓库来负责执行上线步骤。

## 杂谈：AI热潮

最近两个月，随着 `ChatGPT3.5` 以及`ChatGPT4` 的发布，全世界都掀起了一股AI热潮，就连之前早已发布但是遇冷的绘画类AI也鸡犬升天，各行各业的人无论牛鬼蛇神全都打了鸡血一样往里冲，各种应用场景一夜之间纷纷冒了出来，看得我啧啧称奇。

在此我想先发出灵魂拷问：**有多少人真正懂AI？有多少人只是在跟风焦虑？**

我也不懂AI，但是在当前这个风口之下，我觉得这个问题还是值得聊一聊的；我只说说我看见的一些事例以及我的一些看法，可能带有偏见，欢迎交流。

理论上来说，『人』也是经过外界不断地刺激学习而形成的某种智能体。目前AI业界主流的神经网络+深度学习理论也正是受到人脑神经结构启发而产生的，理论上来说假如我们按照培养一个人的完整流程去培养一个AI——从1+1、到乘法口诀表、方程式、几何、微积分、更多……——如此一套培养下来，我们应该是可以培养出等同于人类的智能体的。

虽然我觉得理论方向是对的，但是近几年业界拿出来的东西一直都让我失望。即使是目前最先进、被无数人捧上天的 ChatGPT 系列，在我看来也还远远没有达到『智能』的境界，目前充其量也就是个油腻的骗子——它没有真才实学，它没有核心逻辑，它只会模仿与欺骗。更别说研究了好几年的自动驾驶，就这么一个具体的领域的专用型AI都做不出来，更别说人们期望的泛用型AI了。

在我的视野中，目前我所知的AI有如下几个有价值的应用方向：

第一，作为『强化版搜索引擎』。我们会觉得`ChatGPT`学识渊博，其实是互联网本身就已经知识浩如烟海，掌握正确的使用google等搜索引擎的能力的人早就能够在知识量方面碾压其他人。很多人觉得"ChatGPT比google更好用"，是因为他们原先不具备这种搜索能力，而如今ChatGPT的出现缩小了这种能力门槛。

但是，"使用搜索引擎"可不仅仅是为了找到某个具体问题的答案，我认为更重要的，是找到答案的过程——通过浏览更多结果、观察更多人的观点、别人的失败教训，这个过程中学习到的知识可能远比最初那个具体的问题的答案要重要得多得多。因此我用的是"缩小门槛"这个字眼，而不是"消除门槛"。

第二，作为『草案生成器』。典型场景是设计、绘画创作、编程等场景，我们告诉AI我们想要什么，然后它生成一份（或者多份）有缺陷的画稿、代码等产物，由人类挑选并修复加工，这个过程减少了人力的投入，加快了生产周期。

但是，草案就是草案，对于画稿、单独的前端页面等这类一次性交付的产品还好说，而如果是动画、大型软件工程中的代码这种需要长久迭代和维护的智力结晶，目前的AI能够起到的帮助作用是非常有限的。

目前有一类"创业方向"，是做一个胶水工具去封装大型模型的能力，组合起来提供给用户，以预期实现原本那个模型无法胜任的复杂任务。这种复杂任务与上面所说的动画、大型软件工程等是同样的原理。我认为目前这条路行不通的最大原因在于，目前以ChatGPT为典型代表的生成式AI的行为具有不可控的特性，这种不可控在宏观会表现为"成功率"：少许瑕疵我们人类可以用大脑去填补，或者重试，而连续多次的瑕疵之下，这个成功率会以乘数的方式快速扩大，任务越复杂，成功率越低，低到远远达不到用户预期的水平。

第三，作为『语言生成器』，通俗说就是『陪聊』，这也是ChatGPT的老本行。其中，如果是知识型的陪聊，那我认为仍然属于强化版搜索引擎的范畴；更值得关注的是角色模拟类的陪聊（例如有人用ChatGPT折腾出了"虚拟男友"），这种场景下，ChatGPT所支持的小容量的上下文环境能够发挥巨大的作用，再加上文字本身的信息量有限、人类对于语言的错误容忍度更高，因此在这个细分领域内AI确实可以做得很好。

在不久的未来（甚至从现在开始），AI确实会对一部分行业产生巨大的影响。但是吧，这个"巨大影响"对不同的人的具体的结果可能会是天壤之别的。

假设，假设，绘画AI得到普及，原先绘画行业100万从业者如今只需要50万人+AI就可以完成相同的商业需求，那么对于被淘汰的50万人来说，AI就是"时代的一粒沙落到一个人身上就是一座山"或者是"被资本这个魔鬼吸血吃肉"；而对于留下来的50万人来说，尤其是对于那些已经习惯变化的优秀职场人来说，他们可能只是换了一批生产工具而已，可能还会因为平均生产效率的提高而增加了收入。

说到"增加收入"这个问题，肯定会有很多人跳出来指责说，资本家如何如何压榨，赚的钱肯定分不到普通劳动者手上。说到"被淘汰的那部分人"，可能又有人要说，凭什么被淘汰的不会是你我呢。对于这些问题的解答，已经远远不是一个单纯的技术问题了，涉及到社会、经济学等许多知识，不适合在这里详细展开讨论。（再说，很多人其实并不想听真正理性的解答，他们只想听到自己想听的）

最近我刷到一个视频，是UP主柳行长的[《AI技术能否开启第四次工业革命？我们是不是太看得起它了？》](https://www.bilibili.com/video/BV1cT411r7T4)，我觉得其中的观点深得我心，所以在此推荐给大家。其中核心思想是：AI终究只是个辅助工具，不会凌驾于人类，最终最理想的情况是每个人都有自己专属的AI，自己负责培养，以前比拼的是谁本人更优秀、而以后将会是比拼谁培养出的AI更优秀。

AI将会是人类文明下一个重要的里程碑（之一），没有人可以置身事外，但绝不会如最近这段时间被渲染得那样恐怖，也不会来得想象中的那么快。
