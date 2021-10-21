```yaml lw-blog-meta
title: "用React重构我的网站"
date: "2021-09-12"
brev: "确实比Angular快乐"
tags: ["前端","架构"]
```

## 背景

闲来无……

不对，不能这么说，因为最近确实根本完全不闲。等待挖掘的课题一个接着一个，哪怕是连续工作学习58天没有休息的我，仍然感觉学海无涯看不到尽头。

所以呢，作为一个阶段性成果的汇报，这个周末两天时间，我把我这个网站整个地重构了一遍。

从 `Angular` 重构到 `React` 。

## Step1: 搭框架

最近我确实已经对`Webpack`已经非常得心应手了，所以这次我依然没有使用脚手架（其实在开发过程中跑了一遍脚手架观察学习了一下），所以现在这个项目中的每一个配置细节都是我自己亲自抠的。

不过有一说一，Webpack的内容确实非常庞杂。到目前为止，我已经为各种类型的项目（Nodejs, Web, Chrome-extension, 旧项目改造）折腾了N多内容了，可是这次搭建框架的时候，依然遇到了没有见过的场景。虽然理所应当地一一攻克了，不过客观想一想，其实还是挺可怕的。

主要内容就是引入`CDN`，特别是对于一些UI库（如`antd`）除了js还要引入css，配置过程中要仔细一点。

使用CDN的代价：

- 首先是polyfill会工作不正常，如果选用的库的dist没有做兼容编译的话，那么引入这个库在非兼容环境就会报错；目前我对浏览器兼容这一块没有深入研究，但至少目前随意的配置下，能在我老旧的ipad的safari上面运行，（IE当然挂了不过谁要管它呢，）这就够了。
- 然后是CDN的稳定性的问题，这个可以认为可用性是超过我们自己的前端入口的。对于商业项目，可以有fallback逻辑去处理。盲猜一下大概逻辑，无非就是自己监控`import`的执行情况，超时或者失败的情况下降级到其他源。
- 最后一个小小小问题，CDN是写在html里的，写多了之后，html体积会膨胀。

然后还遇到`webpack-dev-server`的坑，由于我这个网站有两个域名，跨域增加了不少的麻烦。这里吸取的教训是，看文档的时候一定要仔细，第一要看清楚配置项是写在哪一级的，第二要看清楚当前使用的工具版本是否与文档的一致。不过要吐槽，即使再小心，也敌不过文档上的内容跑不通啊……这种事情是最令人暴躁没有之一的了。

跨域还踩到一个`PreFlight`的坑，不影响正常工作，但是增加性能负担。这个以后另找一个话题再说。

## Step2: 最难的Markdown渲染

简而言之，就是用`marked`去解析markdown语法，然后用`Prismjs`去解析代码块。

这个领域就相对小众一些了，而且为了实现一些高级功能（与锚点组件配合），要稍微深入一下它们的高级用法，有那么一点挑战。

挑战成功之后，成就感还是很强的。

为啥选用这两个库？前者应该比较主流吧，后者的领域可能更常用`highlight.js`，不过由于我之前在`Angular`体系内使用的`ngx-markdown`选用的是`Prismjs`，我用的比较熟了，就没换。使用体验也确实很棒，超出我的预期。

## Step3: 路由

用最主流的`react-router-dom`。

遇到的坑一，`BrowserRouter`和`HashRouter`是不能混用的，要实现特殊功能的话需要自己处理，推荐选用前者。之前在公司项目中用的`HashRouter`，是因为Flask后端提供了常规路径路由，所以在前端可以用哈希路由。

遇到的坑二，切换路由不够智能，即使路由匹配结果相同，也会触发组件重载。这一点主要是针对状态管理（我用的`mobx`）会造成一些麻烦，通过适当的语法来解决它。

后来领悟了，其实`Router`就是一个`Provider`嘛，把一些东西放进`Context`里了，所以要按照`Context`的行为方式去对待它。

然后表扬一下webpack和React，懒加载配置起来好方便啊，而且写过这么一次之后，我对所谓的微前端的实现方案已经心里有数了。

## Step4: 体力活

难点都解决了，剩下的就是写组件，写样式，调试……

纯体力活，对于我来说已经没有任何挑战，两天写了1500-2000行代码，写饱了。

## Step5: 移动端适配

主要就是在css里`@media`，然后在js里也要配合做。抓准宽度，一切都好说。

唯一不好就是确实增加了工作量，有一部分组件差不多是写了两遍的，甚至有些antd组件的行为也略显古怪，不能满足需求，还得自己造几个轮子。

## Step6: 后端协议优化

以前我还比较推崇`REST`标准的，后来结合业务思考了很久，觉得还是问题多多。

首先 HTTP CODE 这个东西还是不太好用。一个最典型的场景，非`200`系列的code在`axios`中会默认抛出异常。这个就很啰嗦，要么去拦截一下响应，要么就得每个地方去`try`——后者显然不可能。而且最重要的是，拦截下来的异常到底是系统异常还是业务错误？混在一起就很不清爽。

所以，还是把业务异常代码放进`200`的响应中，让业务层去处理；网络层(`axios`)只负责网络层真正的网络连通问题。这样在实践上会比较顺手。

so，既然要在响应体里加code了，那么顺手做一个标准响应体格式，把业务数据放进一个`data`字段里，也就顺理成章了。这样对于`Golang`和`Typescript`这些有类型语言来说，其实反而是好事。

## Step7: 自信上线

`yarn build`跑一下，构建产物才几十KB。装进`nginx-alpine`里，`docker push`简直就是一瞬间完成的，我甚至为此专门登录腾讯云控制台去确认了一下它真的上传成功，笑死。

然后后端也构建上传，然后重启服务，然后线上验证……

嗯，一次成功，可太自信了。

下班，健身走起！