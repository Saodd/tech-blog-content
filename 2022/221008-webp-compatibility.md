```yaml lw-blog-meta
title: "浏览器兼容：webp"
date: "2022-10-09"
brev: "兼容性判断与格式转化"
tags: ["前端","Golang"]
description: "本文介绍了如何判断浏览器对webp的兼容性，并针对性地做兼容方案。"
keywords: "兼容,webp,jpg,转换,picture,浏览器"
```

## 背景

我这个个人网站，有个『Bangumi模块』，其中会用到大量图片资源。

我还有个ipad，是比较老的2018款式，再加上我坚持不更新系统，因此其自带的safari也是有些古老的版本。我偶然发现，我用这个ipad，无法正常访问我个人网站的Bangumi模块上的图片资源。

一开始我以为是safari的`referer`策略没有正确生效，导致图片资源访问被其他站点屏蔽了。但是等我把图片全部切换本站提供后，ipad上依然不能正常显示这些图片。

然后我才意识到，旧版本safari所不兼容的，是`webp`。

## 参考：豆瓣的CDN系统设计

豆瓣电影、豆瓣读书等站点，其中的内容基本上能够覆盖全网能够搜索到的影视资源，这样一个百科类的网站，它对多媒体资源的需求是非常强烈的。

据我观察，我个人认为它的系统设计也是比较人性化的，因此单独拿出来简单讲讲。

首先一个最重要的点，豆瓣它的前端页面，是由后端渲染生成的；后端在渲染模板的时候，就已经会对客户端的情况进行一定的判断，然后将合适的资源内容填入HTML中返回给浏览器。

然后看看它的图片资源URL大概长这样：

```text
https://img1.doubanio.com/view/photo/l/public/p287412xxxx.webp
```

域名。`doubanio.com`应当就是豆瓣专门用来托管文件资源的专用域名了，前面的`img1`应当是集群代号，任意一个资源，都可以从`img1`，`img2`，`img3`等等多个集群中分别读到，我在杭州市解析到了湖州市、宁波市的IP地址。

图片尺寸。每个图片资源，可能会在缩略图、大图等多种场景用到，也就需要多种尺寸。豆瓣通过`/l`这个路径来表明需要的图片尺寸，`/l`意思是大图，此外还有`/s`等选项。

图片格式。最后的后缀`.webp`，这样指定的话就会返回webp格式的图片；如果改成`.jpg`则会给回jpg格式图片。

简而言之，豆瓣的参数都是直接体现在 path 中的，可阅读性非常好；与之相对的，B站、百度百科、萌娘百科等渠道则使用一长串丑陋的 query 来表示。虽然效果是一样的，但我觉得豆瓣的形式更优雅一些。

## webp

[webp](https://developers.google.com/speed/webp) 是一种相对现代化的图片压缩格式。

它的压缩效率比较高，显示效果好，因此替代部分`jpg`成为了主流的web图片格式之一。目前[兼容性](https://caniuse.com/webp)已经可以说非常好了，除了苹果体系可能出于政策上的考虑很晚才兼容之外（我的ipad2018属于此列），其他浏览器都是很早就支持了。

因此对于`webp`的兼容支持，实质上就是对safari浏览器的特殊兼容。

豆瓣是如何处理这个问题的？前面说了，豆瓣页面是后端渲染的，因此`<img>`标签中的src直接就被写入了`.jpg`格式的资源，而不是`.webp`。

那么我的个人网站应该如何兼容？

核心思想：参考豆瓣的方案，我也在后端检查浏览器的UA，并返回对应的内容。

方案一：在**JSON接口**上处理，不同的浏览器给他不同的url 。

方案二：在**图片资源**上处理，同一个url，不同的浏览器给他返回不同的图片格式。

方案三：给多个url，由**浏览器**自己选择需要哪种格式。

几种方式各有优缺点，我选择的是方案二。

## 浏览器中判断支持情况

通过js是有能力判断当前浏览器是否支持`webp`的，参考阅读：[https://stackoverflow.com/questions/5573096/detecting-webp-support](https://stackoverflow.com/questions/5573096/detecting-webp-support)

主要有两种思路：一种是利用`canvas`来导出webp格式的图片，优势是可以同步执行，缺陷是“能否导出”与“能否显示”是并不完全一致的，会导致判断错误。第二种思路是直接加载一个webp格式图片然后观察这个图片是否加载成功，优势是可靠，缺陷是加载图片是个异步的过程，会给程序设计造成一些麻烦。

从浏览器使用图片的角度来说，其实还可以直接通过HTML的层面来实现。

就像`video`标签可以同时支持多种视频格式一样，我们使用神奇的`picture`标签也可以同时支持多种图片格式，由浏览器自行选择最佳的那种格式。参考[MDN文档](https://developer.mozilla.org/en-US/docs/Web/HTML/Element/picture)，基本用法：

```html
<picture>
  <source srcset="xxx.webp"/>
  <img src="xxx.jpg"/>
</picture>
```

`picture`标签的[兼容性](https://caniuse.com/?search=picture)其实也非常好了。而且，就算不兼容，浏览器也会自动降级到其中兜底的`img`标签上去，不会造成额外影响。

## 后端判断支持情况

参考阅读：[Detect if browser supports WebP format? (server side)](https://stackoverflow.com/questions/18164070/detect-if-browser-supports-webp-format-server-side)

一个我们很熟悉但是可能被忽视的东西，是HTTP中的`MIME type`。

在每个HTTP请求过程中，Request头部会携带`Accept`字段，表明当前浏览器支持的特性。这个字段里的内容是由浏览器自动添加的，并且会根据请求来源的不同而添加不同的内容（例如对img标签就会添加所有支持的`image/?`）。（但是如上面的帖子所说，可能有些浏览器明明支持某种格式但却不告诉后端，这种情况直接降级到兼容性最好的jpg即可。）

Response头部则会携带`Content-Type`字段，用于表明当前内容的格式，帮助浏览器选择合适的解析器。可选值例如有`image/webp`, `image/jepg`等。（但是这个值只是个参考，浏览器可以有它自己的判断，例如如果故意给webp标记`Content-Type: image/jpeg`，在我的chrome浏览器上也依然可以正常解析出来的。）

因此，后端要做的事情也就很简单了，只需要判断请求头就够了。Go语言示例代码：

```go
func view1(c *gin.Context) {
	if strings.Contains(c.GetHeader("Accept"), "image/webp") {
		// ...
    }
}
```

## 后端图片格式转化

有两种思路，一种是更普遍的，空间换时间，即预先准备好多种格式、多种尺寸的图片，客户端需要什么就给什么。

另一种思路是时间换空间，只保存一种格式图片，需要其他格式的时候则临时转化。

由于我的使用场景仅仅是兼容极少数浏览器，因此我选择了时间换空间的思路。我只保留了一份webp格式（最好应该储存无损webp格式），需要兼容的时候现场转化为jpg 。

在Go语言的实现中，首先我们需要用`golang.org/x/image/webp`这个库将webp格式地图片内容的`[]byte`转化为`image.Image`，然后使用标准库`image/jpeg`将其转化为jpg格式的`[]byte`并发送给客户端，[参考](https://stackoverflow.com/questions/39577318/encoding-an-image-to-jpeg-in-go)。

HTTP缓存相关内容在[《HTTP缓存控制: Cache-Control》](../2022/221007-http-headers-cache-control.md)详细介绍过，这里不讲了。
