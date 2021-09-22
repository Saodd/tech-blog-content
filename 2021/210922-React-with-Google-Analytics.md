```yaml lw-blog-meta
title: "React 配置 Google Analytics"
date: "2021-09-22"
brev: "stackoverflow上居然都没有一个漂亮的解决方案，我不能忍"
tags: ["前端"]
```

## 背景

在建站之初，我的这个个人网站就已经接入了 `Google Analytics` 。虽然一直没怎么认真去研究更多的运营套路，不过偶尔上去看一看PV数，还是有一些成就感的。

最早使用`Django`的时候非常容易配置，直接写在模板代码里就行了，因为每次路由都是页面的跳转；后来用了`Angular`，问题也不大，因为当时直接找到了一个封装好的轮子，拿起来直接用就行了。

这次重构为`React`，很多东西都是要靠自己动手，也还好我前端技能已经足够扎实，这次就让我好好研究一下。

## 参考阅读

- 关于如何将GA与Router搭配使用： [How to set up Google Analytics for React-Router?](https://stackoverflow.com/a/47385875/12159549)
- 关于如何写一个防抖函数 [Typescript debounce function not calling function passed as parameter](https://stackoverflow.com/questions/59104425/typescript-debounce-function-not-calling-function-passed-as-parameter)
- [GA官方文档](https://developers.google.com/analytics/devguides/collection/gtagjs)

## GA工作原理

在GA控制台上，首先需要创建一个app账户，然后会得到一个ID，以及一段代码：

```javascript
<script async src="https://www.googletagmanager.com/gtag/js?id=GA_MEASUREMENT_ID"></script>
<script>
  window.dataLayer = window.dataLayer || [];
  function gtag(){window.dataLayer.push(arguments);}
  gtag('js', new Date());

  gtag('config', 'GA_MEASUREMENT_ID');
</script>
```

这段代码是要求直接原样放入html模板中的。

我们仔细看看，咦，它在做啥？

——先访问了一个全局变量`dataLayer`，然后声明了一个全局函数`gtag`，这个函数的作用就是向数组里推送数据……

咋一看好像很奇怪，但是只要结合前面的`<script async ... >`标签一起看，一切都豁然开朗：

原来，在页面加载的时候，`ga`这个库还没有加载好，可是这个时候又要记录 events ，所以谷歌工程师的解决方案就很简单：先存进一个数组里呗！

然后顺便我们可以再想象一下当`ga`加载成功的时候会做些什么？肯定要把`dataLayer`里已经存入的events取出来执行掉，然后再用自己带来的真正的`gtag`去替换这个全局的`gtag`函数。这个应该没问题吧？

了解到这一层之后，就可以放心地去调用`gtag`函数了。（不要像有些回答中说的那样直接调用`ga`函数）

## 路由事件 PageView

当页面加载的时候，或者准确地说，在执行`config`动作的时候，`gtag`会将事件上报。

因此对于传统的后端渲染应用（`Django`之类）来说，每次路由时`ga`都要重新加载，然后重新执行`config`进行上报。

但是对于SPA应用来说，路由都被js控制了，页面不会重载，也就不会重复地执行`config`操作。所以我们需要也在js运行时内去触发ga事件。

```typescript
import { createBrowserHistory } from 'history';
import { History, Location } from 'history';

const history = createBrowserHistory();

export function initGA(history: History): void {
    history.listen((location: Location): void => {
        gtag('event', 'page_view', {
            page_title: document.title,
            page_location: window.location.href,
            page_path: location.pathname,
        });
    });
}
```

那么，事件都有哪些种类呢，都需要哪些参数呢？

不好意思，我是个懒人，我选择`yarn add @types/gtag.js`，赞美Typescript！

## 防抖 Debounce

如果只是像上面那样写，如果你用了嵌套路由，那么很快就会出现问题，例如：

用户点击`/blog`路径，触发一次事件；然后路由自动匹配了子路由`/blog/Timeline/1`，又触发了一次事件。

在上面这个例子中，前面那个父级路由的事件其实是没有意义的——毕竟用户没有在这个页面上停留。

所以一个很合理的解决方案，就是使用防抖。

防抖函数的基本形态应该像这样：

```typescript
function debounce<Params extends any[]>(func: (...args: Params) => any, timeout: number): (...args: Params) => void {
    let timer: NodeJS.Timeout;
    return (...args: Params) => {
        clearTimeout(timer);
        timer = setTimeout(() => {
            func(...args);
        }, timeout);
    };
}
```

返回一个闭包，闭包中含有一个计时器，每次重复调用都会刷新计时器（并且取消上一次还未执行的动作）。

emm，但是这个实现有个问题，似乎对重载函数的支持不太好，所以我给他变形一下，每次调用时不是传入函数的参数，而是传入一个包含函数调用的闭包：

```typescript
export function debounce(timeout: number): (func: () => void) => void {
  let timer: NodeJS.Timeout;
  return (func: () => void) => {
    clearTimeout(timer);
    timer = setTimeout(func, timeout);
  };
}
```

用这个东西把前面的`gtag`函数给包装一下，时间呢，就拍脑袋定个100ms吧。

这可能导致一些现象，例如在模块代码懒加载的时候，用户可能真的在父级路由停留了一小段时间，然后就当作一个PageView事件上报了。但其实这个我觉得是合理的，相当于认为只要用户在某个路径上停留了100毫秒，就认为是一个PageView事件。

## 初始化问题

不要忘记前面还多执行了一个`config`动作哦，这个是还不在防抖的控制之下的。

所以需要干两件事情：让`config`不发送PV事件；页面加载时我们自己调用一次防抖后的PV事件：

```typescript
export function initGA(history: History): void {
  gtag('js', new Date());
  gtag('config', 'xx-xxxxx-xx', { send_page_view: false });
  sendPageView(history.location);
  history.listen(sendPageView);
}
```

## 其他优化细节

例如，一个典型场景是希望在开发过程中不要发送任何GA事件。

这个事情可以利用Webpack的功能去实现。例如设一个`Debug`布尔值，例如给开发/生产环境使用不同的html内容，等等。
