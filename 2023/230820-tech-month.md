```yaml lw-blog-meta
title: "技术月刊：2023年8月"
date: "2023-08-20"
brev: "yarn link，抖店app小程序，人才不是培养出来的，很久不见"
tags: ["技术月刊"]
```

# yarn link 尚未发布的包

假如我有至少3个npm包，他们的依赖关系如下：

- 工具包A 依赖 工具包B
- 业务包C 依赖 工具包A

A、B、C 都是尚未在npm（内建npm）上发布的包。

按照`yarn link`的基本流程，操作如下：

```shell
# 在目录A
yarn link

# 在目录B
yarn link

# 在目录C
yarn link xxx-xxx-A
yarn link xxx-xxx-B
yarn add xxx-xxx-A@0.0.0 # 报错
```

在上面最后一行`yarn add`的时候会提示报错：

```text
yarn add v1.22.17
[1/4] Resolving packages...
error An unexpected error occurred: "https://.../@xxx-xxx-A: no such package available".
```

解决方案参考[github](https://github.com/yarnpkg/yarn/issues/1297#issuecomment-620812103)，只需要添加`link:`前缀即可解决

```shell
yarn add xxx-xxx-A@link:0.0.0 # 正常运行
```

## 大坑：依赖解析

上述情景中，被业务方依赖的工具包A（或者B）他们自己的依赖依然会去他们自己目录下的`node_modules`目录中查找，而不是在业务包C目录下的`node_modules`目录中查找。

一个很容易遇到的问题是`React`库加载超过1份的时候会产生报错：

```text
Invalid hook call. Hooks can only be called inside of the body of a function component. This could happen for one of the following reasons:
1. You might have mismatching versions of React and the renderer (such as React DOM)
2. You might be breaking the Rules of Hooks
3. You might have more than one copy of React in the same app
See https://reactjs.org/link/invalid-hook-call for tips about how to debug and fix this problem.
```

解决方案：

一种思路是[文章](https://medium.com/@vcarl/problems-with-npm-link-and-an-alternative-4dbdd3e66811)，简而言之就是先用`npm pack`功能把被依赖的、未发布的工具包A先打包起来，拷贝`tar`文件过去再`npm install`本地文件。但是如果需要频繁修改、调试代码的话，这样操作就会非常麻烦，而且也无法解决多重依赖的问题（C依赖A，A依赖B）。

另一种思路，我的最终解决方案是老老实实告诉`webpack`解析到同一个`node_modules`目录中去，具体来说就是使用`alias`能力来处理，在业务包C的webpack配置如下：

```js
{
    alias: {
        react: path.resolve('./node_modules/react'),
        'react-dom': path.resolve('./node_modules/react-dom')
    }
}
```

# 抖店app小程序开发体验

最近公司的项目组做了一个抖店app小程序（移动端html5应用），在开发过程中还是踩了不少坑的，简单记录一下。

首先是他们提供的脚手架（以及构建工具）叫 [bytedance/mona](https://github.com/bytedance/mona) ，这个框架做的还是很仓促的（才刚刚脱离`alpha`阶段），不少地方细节都做得有些粗糙，因此无奈之下我没延续之前`npm workspace`的方式、而是创建了一个独立的(git)工程目录来管理这个项目的代码，以避免命名空间（尤其是typescript类型定义）之间的相互污染。

于是冒出另一个问题：为了共享一部分原有的基础代码，我对原代码仓库的一部分代码做了抽离处理，作为独立的npm包发布在内建的npm仓库中。这个过程中需要注意的细节还是有一些的，需要对`npm workspace`和`npm publish`的机制有比较充分的理解才能设计好这套方案。

UI组件库选用的是`antd-mobile@v5`，与原项目web端想要升级却失败只能保留在的`antd@v4`相比，改进还是比较明显的。移动端与pc端的UI库的核心思想和主要用法都是相同的，唯一可能需要注意的是`rem`这种相对尺寸的概念，这个在其他端（例如微信小程序）中早已见过，理解就好。

兼容性方面，大部分常规代码都可以正常运行，少部分能力，例如 `XMLHttpRequest`是受到限制的、 `iframe`, `Websocket`等是明文禁止使用的，但是奇怪的是，当代码中尝试使用的时候并不会抛出异常而是静默地不反馈任何执行结果，这会导致一些依赖了这些功能的库（例如`sentry`等）运行行为变得诡异，需要深入研究源码才能找出问题所在并加以解决，这个过程是挺考验技术水平的。

# 闲谈：字节跳动：人才不是培养出来的

标题言论出处： https://www.bilibili.com/video/BV1Ta4y1c7en/?spm_id_from=333.880.my_history.page.click

我觉得这句话，可以算对，也可以算不对；主要是引发了我的一些共鸣，因此想要分享两句。

如果是站在业务高速发展的公司的角度来看，这种略带噱头的言论其实某种程度上也确实反映了现实——毕竟，培养人才真的太难了，特别是中国这么多人口，一个人不行，马上换一个会高效得多。

就我个人经验来看，一个人是否能够“成材”，确实跟他自己的个人特质有很强的关联，如果人不行，就算投入再多的资源、给再好的机会、甚至就是等待足够的耐心，都不会一定保证能够成功的。再加上，人都很善于伪装，面试时的表现和入职之后的实际工作能力或者未来的成长潜力可能会相差很大。

一个人是否具有培养潜力，是很难从短时间内的面试环节来准确判断出的，不要说我，就是比我经验更丰富的前辈也不敢妄言。有的人可能因为背景不够亮眼/不够匹配、性格内向软弱、短期的发挥失常等原因可能在面试时表现一般，但是入职后可以通过合理的引导来扬长避短，在工作能力上快速提升；有的人可能面试时很积极向上，但是到了实际工作中会发现很多细节与之前的印象有较大差距。

因此把有限的资源投入给那些更有希望的人，是一种理性的选择，因此这也就意味着有一部分人会被（相对地）放弃，因此在简单总结的时候确实可以表述为：“人才不是培养的，人才是天生的”。

但是具体到个例上来说，经过“培养”而成长起来的人才，会比外面快速招聘而来的人有一些特别的好处。公司业务不可能永远一路狂奔，卷王也有卷不动（或者卷无可卷）的一天，人员频繁流动导致管理风格变化、劣币驱逐良币更是糟糕的情况。在回归常态的过程中，一个过于“精英化”的团队很可能出现一些管理上的问题（基于我自己这个卷王的心理分析得出的结论）。我认为，在理想的、常态化的团队中，“培养”是个充分必要的工作环节，不管这个人是否能够变成“人才”，只要他能变得“比以前更好”哪怕一点点，只要人不是太离谱，培养就是有意义的。

我没有接受过正规的教育学训练，但是我的家族中有不少亲戚都是当老师的，当我尝试用教书育人的理念去解释这个问题，会发现这件事本就应该是正道的光：学生成绩不好不能怪生源质量差吧，重要的还是学生自己跟自己比，这才是教育的本质、是教师的工作意义所在。当然，职场到底能不能与校园相提并论这可能见仁见智，不过至少我个人还是真诚地希望这个世界上多一些善意的。

# 闲谈：我很久没有写博客了

从5月到8月，三个月的时间没有更新任何一篇文章。

有几个方面的原因吧，主要是：

1. 我自己没那么卷了，业余时间大多用去放松娱乐、研究厨艺、以及装修房子等家庭琐事上了。
2. 随着公司项目的快速发展，作为“全村的希望”的项目的前端负责人，我的目标开始向业务导向/结果导向倾斜。
3. 作为技术兜底角色，工作中处理的问题更加小众化、细节化、多样化，而这些工作经验如果总结成博客文章的话会显得很琐碎（写出来可能有用，但用得上的人很可能找不到我的文章）。
4. 该学的都学差不多了，很多技术难点对我来说已经成为日常，很难会有想要记录的欲望。

我能够捕捉到自己的心态变化。我也会不时地对自己发出灵魂拷问：“你是否已经放下了技术的追求？是否已经成为了无趣无用的所谓管理者？如今的工作状态能否承受时间的考验，五年十年之后是否还能像今天这样意气风发？”——所幸，目前我仍然可以自信地对这些问题给与乐观的答复，甚至比以前更加乐观和坚定。

我以前开过玩笑说：“如果生活能像编程这么容易就好了”，技术这个东西呢，说简单也不简单，因为确实可能90%的人在这件事上就被难倒了；说难也不算难，因为它就像我们学生时代读书考试一样，有相对明确和客观的评判标准以及学习路线，在如今开源技术日新月异的背景下，被后来者加速超车是经常都可能发生的事情。而真正能够成让中年技术人更进一步，形成护城河的，是那些年轻人或者AI无法替代的东西——例如处理各种复杂问题积累的经验、最佳实践、工作流程、甚至政治操作水平等等。

注意，我不是说技术不重要，而是说，扎实的技术实力是基础，再往上步入金字塔顶端（或者说金字塔的中上部）则需要一些技术之外的东西相辅相成。