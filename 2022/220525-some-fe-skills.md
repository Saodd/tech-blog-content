```yaml lw-blog-meta
title: "技术月刊：2022年5月"
date: "2022-05-25"
brev: "近期在项目中遇到的一些问题与应对方法"
tags: ["技术月刊"]
```

## 1. 重复触发css动画

直接利用 React Hooks 的生命周期来重置css动画。

假如我现在有这样一个动画：

```css
.buttonJump {
  animation: buttonJump 0.5s;
}

@keyframes buttonJump {}
```

那么我准备两个状态，一个状态是清空了状态的，一个状态是有动画的：

```typescript
const state0 = styles.common;
const state1 = classNames(styles.common, styles.buttonJump);
```

然后我就给组件加一个`useffect`，先设置动画，然后等动画执行完毕之后，恢复到无动画的状态：

```tsx
const MyButton: FC = () => {
  const [clsn, setClsn] = useState(state1);

  const text = useMemo<string>(() => {
    // ... 这里的值可能会发生变化
  }, []);
  useEffect(() => {
    setClsn(state1);  // 设置动画
    setTimeout(() => setClsn(state0), 600);  // 清空动画
  }, [text]);

  return (
    <Button className={clsn}>
      {text}
    </Button>
  );
};
```

或者有另一个更简单的方法：直接利用React框架的`key`属性，来直接换上一个全新的DOM 。

在实际使用过程中表现良好。

但不完美：如果用户在动画还未执行完毕的时候再次改变状态，会对动画过程有一点点影响。一定要优化的话可以优化，但是这种极端情况很少出现，可以满足需求就先不改了。

## 2. 倒计时

利用mobx管理状态，`setInterval(xx, 1000)`写几行代码即可最简单地实现。虽然有些误差，但可以满足部分要求不高的场景。

这里会遇到好几个问题：

**问题：客户端本地时间不准**

关于网络时间校准这个话题，感觉还是比较有趣的，可以参考阅读：[知乎](https://www.zhihu.com/question/21045190) 。其核心原理在于，每个客户端电子设备会向时间服务器去请求时间，请求前后还会考虑请求延迟，来计算一个相对准确的时间，广域误差10~500ms，局域网误差1ms，大概就是我们平常的ping的速度。

如果我们选择不信任客户端本地时间，那么我们需要在客户端程序(js)中自己实现校时逻辑。服务端做一个简单的API，就返回当前服务端的UTC时间戳。校时的过程也应当考虑请求延迟，经过一些计算来尽量消除延迟带来的误差。

但是要知道，js的运行时在系统中应该算是相对较低优先级的，因此可能会受客户端当前系统负载的影响而产生额外误差。不过大多数情况我们可以忽略这个影响，毕竟它也没有解决方案啊。

**问题：js Timer Interval 的误差**

如果用`setTimeout`的话，我们可以在每次回调的时候重新计算下一次的时间偏移量，以此保证误差不会累加。

`setInterval`的触发时间相对准确，但是也会导致误差固定。

**问题：JS运行时负载的影响**

根据JS事件循环原理、以及JS本身的单线程模型可知，Timer它们触发的事件不一定能够及时得到执行。

有的文章说可以借助 `Web Worker` 来保证精确计时，可能有一定改善。但是要知道：多线程同样会受系统调度的影响，并不是绝对安全的；此外，Worker的通信事件本身也是有延迟的，而且终究还是要回归到主线程内部来处理，逃不掉的。

**问题：React等框架的影响**

以React为例，它有fiber的数据结构，它为了性能优化做了一些调度上的策略，也就是说，我们通过框架执行的状态更新未必能及时反应到DOM上。此外v18以后还有并发模式，时间精度可能变得更加难以预测。

如果要更细粒度的倒计时，考虑越过框架直接更新底层DOM吧。

小结一下，虽然问题很多，但总的误差控制在1秒之内，或者在网络、设备条件优秀的情况下做到100ms之内，这件事应该还是可以预期的。在这个误差范围内，用户应该也无法抱怨什么了吧。

## 3. 不阻挡点击事件

```css
.parent {
    pointer-events: none;
}

.parent > * {
    pointer-events: auto;
}
```

这个方案有个小问题，不仅阻止了鼠标点击事件，同时也把鼠标滚轮事件禁用了。因此要注意选好作用对象范围。

## 4. shadow DOM

允许将一个DOM树封装起来，减少与其他节点的交互。（比较典型的场景是避免css命名污染）

参考：[Using shadow DOM](https://developer.mozilla.org/en-US/docs/Web/Web_Components/Using_shadow_DOM) | [神奇的Shadow DOM](https://jelly.jd.com/article/6006b1045b6c6a01506c87ac)

具体应用，我是看到B站顶部Banner嵌入的小游戏是通过这种方式嵌入的，有一定的道理。

其他应用场景的话，我认为这项技术比较适合微前端、插件注入等多源应用。

后来在自己带的项目上试用，体验良好，确实达到了预期效果。

## 5. createPortal

[官方文档](https://reactjs.org/docs/portals.html) 说：当需要在超出根节点以外的地方渲染子节点时，`Portal`是最好的选择。

一般用法应该是用于 Modal Drawer 等组件，他们通常直接append在body上，而不是在`<App/>`里。

而在我的项目中，它还有另一种实际意义：当我的React组件需要挂载在一个不稳定的DOM节点上时，（注入其他应用的节点上，可能会被其他应用删掉的节点），只有`createPortal`可以满足需求。而如果只是普通的`creatElement`，当DOM被删除时，React的VDOM同样不再正常工作了。

所以核心意思就是，`Portal`渲染的节点不会对底层DOM产生强烈依赖，可以很容易地"复活"。

## 6. 代码混淆

之前在对字节调动某个业务的前端页面进行破解的时候接触到的。他们的反作弊token系统代码就是经过高度混淆的，代码极度混乱，变量和值全部都经过序列化处理，几乎无法破解。

后来了解了一下这方面的技术，估计他们应该是使用的`obfuscator`这个库并且加上了少量的自定义逻辑。

后来我把它使用到了业务项目中，使用上挺方便的，只要装一个[webpack插件](https://www.npmjs.com/package/webpack-obfuscator)就行了（推荐用plugin而别用loader），参数的配置可以参考：[官方文档](https://obfuscator.io/)

但是我要提醒，如果使用了`controlFlowFlattening: true`这个参数，可能会不兼容一些诡异的js写法，尽管那些语法是符合语法规范的，但是可能是混淆处理代码不够完美吧，处理之后会产生bug。

## 7. Sentry配置

emm，对于`Sentry`这个东西呢，可以说见仁见智吧。

我一直觉得：它太重了。

### Sentry代码内配置

我在后端是一直使用着它的，不过我是自己定制了处理逻辑，而不是用的它官方的包。

而在前端，我抗拒了好久，最后还是觉得，“哎、别想太多了”，最后就用它最基础的配置。

其实如果只想开箱即用的话，在前端真的超级简单，只要简单地调用一下它的`init()`函数，它会帮你处理好一切。

### Sentry运维配置（失败）

稍微有点麻烦的是`sourcemap`的处理，因为异常上报上来的是经过webpack简化的、不适合人类阅读的代码，如果需要快速定位业务位置，还是需要结合`sourcemap`来一起处理。

为此我又再次犯难，虽然说“前端没有秘密”，但是考虑到混淆与破解的难度的话，其实秘密还是有那么一些的。而如果上传了`sourcemap`，那么就真的是把内裤都脱光光了。虽然我的前端项目并没有什么影响世界和平的秘密，可我依然希望保持一定的隐蔽性。

> 此时我有些后悔，前阵子续费腾讯云服务器的时候，2c2g实例超级便宜，我当时怎么就没多买一个来专门运行自建Sentry呢……

最后还是再度躺平，装上webpack插件，直接把sourcemap传上去吧，没啥大不了的……

结果让我大吃一惊，事情远远比我想象得复杂：

1. 首先装一个`@sentry/webpack-plugin`插件，没问题。
2. 然后要在构建时配置：`devtool: source-map`，这个有点诡异了；不过加一句`find . -name "*.map" -type f -delete`可以解决。
3. 需要登录。在项目中创建一个`.sentryclirc`文件，至少需要写入`defaults.org` `defaults.project` `auth.token` 三个选项 [参考](https://docs.sentry.io/product/cli/configuration/)；或者写入webpack文件中，[参考](https://docs.sentry.io/platforms/javascript/sourcemaps/generating/)
4. 构建，很慢，上传卡住了几分钟。
5. 部署

完事了，简单测试一下。人为制造一个异常，去Sentry上一看，好家伙：

- 异常代码位置给我定位到`vendor.js`里去了，而我明明是写在业务代码里的，也去dist确认过了
- 就`vendor.js`这个文件，它提示"Remote file took too long to load"

好家伙，出了异常你现在分析代码还要临时去我服务器上下载js文件？请问你插件上传sourcemap都传了些啥？

目前看来问题还是很大的，远不足以在实际项目中开箱即用。先回滚了，以后有精神再折腾吧。

## 8. 懒加载js资源

说起懒加载，很容易想到React已经为我们封装好的糖：`React.lazy()`，可以直接以模块为单位进行懒加载。但这种方式仅适用于我们的业务应用代码。

然后还能想到ES6的`import()`语法，可以动态加载资源（[参考](https://blog.bitsrc.io/5-techniques-for-bundle-splitting-and-lazy-loading-in-react-b471004335f5) ）。但这种方式仅适用于经过项目打包的文件，而不适用于外部资源；即，如果你想写`import('https://...')`的话，需要配置`target: es2020`，这个对于项目的兼容度来说是不可接受的。

我现在的需求场景是，对于某些使用频率较低而又体积庞大的库（例如echarts, xlsx等），我希望以一种更加灵活可控的方式来进行懒加载，以提升页面综合性能。

核心解决方案是：通过js控制添加`script`标签来引入外部js资源。

### 示例：懒加载marked

示例用的库的名字叫做`marked.js`。

既然我要手动添加`script`标签，那么我必须让调用方等待，直到js资源加载完毕后再进行回调，因此我需要一个自定义`Promise`：

```typescript
const pr = new Promise((resolve) => {
  const s = document.createElement('script');
  s.async = true;
  s.src = 'https://cdn.lewinblog.com/marked@3.0.3/marked.min.js';
  s.onload = resolve;
  document.head.appendChild(s);
});
```

有了上面这段核心代码，接下来我再做一些封装，以支持多个库，这里我使用一个对象来管理一个`Map`：

```typescript
import { marked } from 'marked';

class LazyLoader {
  loadMap: Map<string, Promise<any>> = new Map();

  get marked(): Promise<typeof marked> {
    if (!this.loadMap.has('marked')) {
      const pr = xxx // 省略...
      this.loadMap.set('marked', pr.then(() => require('marked')));
    }
    return this.loadMap.get('marked');
  }
}

export const loader = new LazyLoader();
```

上面的代码要特别注意：我在`Promise`完成之后，将其返回值改为了`require('marked')`，这个是最关键的。

不用`import`语句的原因是，（经过webpack的处理后）它将在初始化阶段进行查询，而在初始化阶段没有做过懒加载会报undefined错误；而`require()`函数的效果是在调用时进行查询的，在`onload`事件触发之后，是可以在window对象上找到这个依赖包的。

当然，使用了require的同时不要忘记了要把这个库写进`external`配置项中。

我第一行写的`import { marked } from 'marked';`这个语句，仅仅是为typescript服务的，我希望后续对于`marked`这个库的使用都能有正确的类型提示；后续编译时会被external处理掉，不会引入。

最后看一下在调用方是如何使用的：

```tsx
export function App(): JSX.Element {
  const [html, setHtml] = useState<string>('');
  const handleClick = useCallback(() => {
    loader.marked.then((mod) => setHtml(mod('...markdown字符串')));  // 懒加载后使用这个库
  }, []);

  return (
    <div id="app">
      <button onClick={handleClick}>加载</button>
      <div dangerouslySetInnerHTML={{ __html: html }} />
    </div>
  );
}
```

## 9. CDN异常回源

需求场景：依赖CDN也会有挂掉的时候，那么在这种极端情况下，需要一种机制来让用户回源到我们自有server上，来提升系统可用性。

其核心原理：监控`script`标签的加载情况，当遇到`onerror`事件后，再添加另一个源即可。

我们把前一小结的代码稍微改写一下：

```typescript
const url1 = 'https://cdn.lewinblog.com/marked@3.0.3/not_exists';
const url2 = 'https://cdn.lewinblog.com/marked@3.0.3/marked.min.js';

const pr = new Promise((resolve) => {
  const s1 = addScript(url1, resolve);  // 资源不存在
  s1.onerror = () => addScript(url2, resolve);  // 回源地址
});

function addScript(url: string, resolve): HTMLScriptElement {
  const s = document.createElement('script');
  s.async = true;
  s.src = url;
  s.onload = resolve;
  document.head.appendChild(s);
  return s;
}
```

但是在业务场景中，考虑到用户体验，我们可能不能等到error事件抛出来才做对策，更可能的需求是一个超时机制，超过一定时间就认为是失败。

超时好办，用setTimeout很简单。但是现在这种直接添加一个script的实现，可能在超时之后它又活过来了，导致两个script重复加载，造成难以预知的后果。因此我们可以考虑将script内容下载在运行时变量里，等下载完毕后，注入一个script元素中去（类似XSS）。

如果需要加载多个脚本，特别是对加载顺序有要求的时候，情况会变得比较麻烦。原先我们借助script标签的`async`属性的特性使其天然保证了顺序，而现在我们需要一段逻辑代码来判断各个资源的加载情况并主动控制执行顺序，可以想象，这份逻辑代码如果要手动维护会变得非常麻烦。因此我们需要借助一些工具，根据构建时的实际情况来动态生成回源逻辑代码。参考：[前端资源加载失败优化](http://www.alloyteam.com/2021/01/15358/) 这篇文章中展示了一个webpack插件，如果有必要的话可以参考使用。
