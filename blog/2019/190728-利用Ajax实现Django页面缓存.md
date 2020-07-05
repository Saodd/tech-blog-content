```json lw-blog-meta
{"title":"利用Ajax实现Django页面缓存","date":"2019-07-28","brev":"我们知道Django中自带了缓存中间件，只需要一个装饰器就可以把视图缓存起来。但是这个缓存是基于URL的，会把用户状态也缓存下来，这个是不允许的。所以我将静态部分缓存，通过ajax动态请求缓存内容。","tags":["Web"]}
```



## 拆分原有视图

原来的view函数会进行一次完整的操作：

```python
def blogpage(request: WSGIRequest, title: str):
    # 数据库操作......
    return render(request, "***/blogPage.html", {"data": data})
```

因为在模板中有对`session`的访问并判断用户状态，所以直接缓存是肯定不行的。

由于这个视图是博客网页，博客文章内容是长时间不会改变的，所以我们把这个页面的动态/静态部分分开，
静态部分（博客文章内容）是可以缓存的。

> 当然，如果对于缓存部分有访问控制的话，会稍微麻烦一些。不过是麻烦在前端的判断，后端逻辑验证权限很简单。

我们把原来的view函数拆成两个：

```python
def blogpages(request: WSGIRequest, title: str):
    return render(request, "***/blogPage.html", {"title": title})


@cache_page(86400) # 缓存一天
def cache_blogpages(request: WSGIRequest, title: str):
    # 数据库操作......
    if not mgr:
        return HttpResponseNotFound("不存在这篇文章哦")
    else:
        # 数据库取出的内容
        d = {"html":get_template("***/cache_blogPage.html").render({"**": mgr}, request),} 
        return JsonResponse(d)

```

然后记得启用Django的缓存中间件，因为我们的网页并不大，所以直接缓存在本地内存中。
修改`settings.py`文件：

```python
CACHES = {
    'default': {
        'BACKEND': 'django.core.cache.backends.locmem.LocMemCache',
        'LOCATION': 'unique-snowflake',
    }
}

```

## 拆分原有Html模板

然后把原来的html模板也要拆成两个。

这里我的思路是对于主要内容（静态的），我们放在一个空的块中：

```html
<div id="main" href="xxxxxxx/xxxx/xx">
</div>
```

然后在页面加载的时候，调用一次ajax，请求其中的内容：

```js
<script>
    $(document).ready(function () {
        myElemAjaxGet("#main");
    });
</script>
```

`js`模块分为两个函数，一个是根据指令，修改`#main`的`href`属性；
另一个是读取`#main`的`href`属性，请求到内容后写回`#main`中。

```js
function myElemAjaxGet(elemId) {
    var url = $(elemId).attr("href");
    $.ajax({
        type: "get",
        cache: true,
        url: url,
        success: function (data, textStatus) {
            if ("html" in data) {
                $(elemId).html(data["html"]);
            }
        },
        error: function (ajaxContext) {
            $(elemId).html(ajaxContext.status + ": " + ajaxContext.responseText);
        },
    });
}

function myElemSetMainHref(url) {
    var elem = $("#main");
    elem.attr("href", url);
    myElemAjaxGet("#main");
}
```

这样，这个页面就能**基本运作**起来了。访问一个博客文章页面，其中的博客文章内容块都是异步加载的。

同理，我们对**博客列表页**、**博客标签列表页**都进行相同的拆分操作。

## 调整模板url使得全部动态请求

再进一步，由于我的这三种页面：**博客文章页**、**博客列表页**、**博客标签列表页**
都是基于同一个模板的，它们之间相互跳转，都只需要改变中间的`#main`块就可以了。

所以我们调整一下模板链接的url，使其不直接跳转到新的url，而是通过我们之前写的
js函数来进行ajax请求：

```html
<div class="card-header my-header"
        onclick="myElemSetMainHref('{ % url 'xapp:xview' blog.title % }')">
    <b>{ { blog.title } }</b>
</div>
```

接下来是体力劳动，把所有的链接都改成js函数调用，注意细心就好，多调试几次。

到此为止，在功能上已经实现了**缓存页面中的静态部分**这个目标了。
我们可以通过浏览器network日志，或者服务器的日志来观察改良后的效果。

## 优化浏览器状态栈

切换到`Ajax`进行动态请求，好处很多，但是坏处也有：由于没有进行完整的页面跳转，浏览器的后退/前进功能不好用了。

这时候就要用到一些前端的知识，主要是`history.pushState()`和`popstate`两个东西。

我们知道，浏览器后退的时候，其实就是调用了`history.popState()`方法，
这时候我们可以对这个方法进行监听：

```js
window.addEventListener("popstate", function (event) {
    // 实现逻辑
});
```

相应地，我们每次Ajax请求的时候，都通过`history.pushState()`保存一下当前的状态：

```js
history.pushState({"elemId": "#main", "ajax_url": url}, "")
// 注：第一个页面要使用replaceState
```

然后在后退的时候，再次调用ajax方法，取回上一步url对应的内容（这里用了本地缓存，体验非常ok）：

```js
window.addEventListener("popstate", function (event) {
    var elem = $(event.state.elemId);
    elem.attr("href", event.state.ajax_url);
    myElemAjaxGet(event.state.elemId);
});
```

到此为止，客户体验就跟真实跳转没有区别了。可以前进/后退，地址栏显示的url每次都会变。

最终效果请访问我的[个人网站](https://www.lewinblog.com/).

一定要搞清的是这个state栈的意义，别像我一样搞出莫名其妙的问题来（泪奔

## 小结

1. 感谢前端大佬[BigLiao](https://github.com/BigLiao)陪我解决问题，哈哈哈哈
2. 缓存的引入，可以让我们实现一些骚操作。
   
   比如我要实现一个**标签计数**的功能，第一个想法：遍历整个数据库计数；第二个想法：渲染markdown的时候计数，存到另一个表里。
   
   正常情况下我们会选第二种方案，因为对整个数据库遍历一遍也太惨了，但缺点是要维护另一个表。
   但是如果我们有缓存的话，就可以考虑第一种方案了。

3. 同时也要注意缓存的有效期与更新内容之间的矛盾。我这里同时使用了服务器缓存和客户端缓存，暂时觉得不会有很大问题，等实际运行一段时间看看效果吧。
4. 以上展示代码并不完全，在我的实际项目中还有一些边界检查和其他的优化。
