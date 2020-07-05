```json lw-blog-meta
{"title":"基于Redis的Markdown渲染缓存","date":"2019-07-18","brev":"为了适应国内的形势，我得把博客在我的个人网站(www.lewinblog.com)上也实现一份。","tags":["Python"],"path":"blog/2019/190718-基于Redis的Markdown渲染缓存.md"}
```



## 传统方法

直接使用python三方库`markdown`实现`.md`到`html`格式的渲染：

```python
from markdown import markdown

def post_detail(request, post_id):
    post = get_object_or_404(Post, id=post_id)
    post.body = markdown(post.body)
    return render(request, 'detail.html', context={'post': post})
```

然后引入`Pygments`实现代码高亮：

```shell

```

## 反思

但是想一想，不对吧，难道我每次访问都要来临时渲染一遍吗？
读取markdown内容可以由数据库和ORM缓存，但是这个渲染过程是没有的呀。

这也效率太低了~ 虽然我们是个小网站，但是我们也不能浪费性能呀。

所以我想到：**参考`Jekyll`的静态模式，手动（或定时）对所有页面进行完整的构建，并保存渲染结果至`Redis`。**

进一步考虑，所有应用基于容器：

1. 我们使用2个持久容器，一个实现uWSGI+Django，一个实现Redis，这两个是之前已经存在着的；
2. 在渲染页面的时候会用到临时python容器，存入Redis后就可以销毁了。

基于这种架构我们就可以实现一个高性能的基于Redis的类静态博客页面了，同时也不影响`Django`的各种动态功能。

## 自定义一个渲染网页的脚本

这个功能大概思考一下应该不难，而且出于练手的考虑，我们就来手动撸这个轮子。

首先这个脚本是运行于python容器的，但是必须要放在Django工程目录下（因为要共享一些配置，而且单独开一个git仓库的话也不利于管理）。我们整体建立一个app：

```shell
PS > python manage.py startapp appBlogpage
```

然后在app目录下建立一个python脚本，我们命名为`make_dm.py`。

### 搜索所有markdown文件

因为这个脚本会独立运行，没有Django的path环境，所以我们要手动明确路径：

```python
# ------------------------- Project Environment -------------------------
def _find_root(n):
    if n > 0: return os.path.dirname(_find_root(n - 1))
    return os.path.abspath(__file__)


_path_project = _find_root(2)
if _path_project not in sys.path: sys.path.insert(0, _path_project)

from BlogDj import secrets

if secrets.DEBUG:
    path_post = os.path.join(_find_root(3), "Saodd.github.io/_posts")
else:
    path_post = "/_posts/"
if not os.path.isdir(path_post): raise Exception("_posts文件不存在，无法运行，退出！")
```

然后遍历目录，找出所有`.MD`文件：

```python
# ------------------------- Functions -------------------------
def list_md_in_dir(path: str) -> List[str]:
    """给定一个目录字符串，返回一个列表，包含所有以.MD结尾的文件的绝对路径。"""
    mds = []
    for file in os.listdir(path):
        if os.path.isdir(os.path.join(path, file)):
            mds += [os.path.join(path, x) for x in list_md_in_dir(os.path.join(path, file))]
        if file.endswith(".MD"):
            mds.append(os.path.join(path, file))
    return mds
```

我们做一个简单的单元测试，如果有问题，改。pass，进入下一步。

### 初步渲染markdown

在简单地了解了一下`python-markdown`库之后，我发现一个很致命的问题：

markdown的语法标准并没有100%地达成共识，至少，`python-markdown`的开发者坚持了一些自己的意见。
但是虽然他们很坚定，却并不努力
——他们的开发水平与`github`专业团队所维护的api实在相差太远了，
一个渲染工具没有很好的容错率的话，那这个工具就没什么价值了。

所以我选择`api.github.com/markdown`。我们简单试验一下是否好用：

```python
def make_md(path: str):
    for file in list_md_in_dir(path)[:1]:
        with open(file, encoding='utf-8') as f:
            md = f.read()
    api_url = "https://api.github.com/markdown"
    data = {"text":md, "mode":"markdown"}
    try:
        response = requests.post(api_url, json=data, headers={"Content-Type": "text/plain"})
        html = '<div class="card-body markdown-body">%s</div>' %response.text
        return html
    except Exception as e:
        print("Post failed: %s" % e)
        return ""
```

然后把这个函数嵌入到我们网页中的一个试验页面中去，再导入[`github-markdown-css`包](https://github.com/sindresorhus/github-markdown-css)。

这样我们在浏览器中可以看到与github相同风味的markdown页面了。

### 进一步渲染

因为我的markdown文件，之前是为了适配`Jekyll`的要求，在每个文件的首部都有一段配置信息，像这样：

```text
---
layout: post
title:  基于Django模型的Markdown渲染缓存
date:   2019-07-18
tags: Python
---
```

但是如果我不使用`Jekyll`的话，这一段信息就是累赘了。所以我们对它进行一些处理：

```python
def read_head(md: str, filename: str) -> dict:
    # init blog info ----------------------------------------------
    post = {}
    post["title"] = os.path.basename(filename)[11:-3]
    try:
        post["date"] = datetime.strptime(os.path.basename(filename)[:10], "%Y-%m-%d")
    except:
        post["date"] = datetime(year=2050, month=12, day=31)
    post["tags"] = []
    # read head ----------------------------------------------
    if md.startswith("---\n"):
        mdlines = md.split("\n")
        for i in [1, 2, 3, 4, 5, 6, 7, 8, 9]:
            line = mdlines[i]
            if line.startswith("---"):
                break
            elif line.startswith("title:"):
                post["title"] = line.replace("title:", "").strip(' "')
            elif line.startswith("date:"):
                s = line.replace("date:", "").strip()
                try:
                    post["date"] = datetime.strptime(s, "%Y-%m-%d")
                except:
                    pass
            elif line.startswith("tags:"):
                post["tags"] = line.replace("tags:", "").strip().split()
        post["md"] = "\n".join(mdlines[i + 1:])
    return post
```

这样就把配置信息pop出来了。

### 使用api生成并存入Redis

调用github-api很简单，我们只需要一个`request`库就可以了：

```python
def somefunc():
    # use api -------------------------------------------------
    api_url = "https://api.github.com/markdown"
    data = {"text": post["md"], "mode": "markdown"}
    try:
        response = requests.post(api_url, json=data, headers={"Content-Type": "text/plain"})
        post["html"] = '<div class="card-body markdown-body">%s</div>' % response.text
    except Exception as e:
        logger.error("Post failed: %s" % e)
        continue
```

存入数据库这一步遇到一点小小的意外，那就是发现Redis对于中文的储存是encode过的。

```python
def make_md(path: str):
    key = "BlogpageHtml_%s" % post["title"]
    rd.set(key, value=post["html"], ex=864000)
    posts.append([post["date"].strftime("%Y-%m-%d"), post["title"], post["tags"]])
    logger.info("Set <%s>." % key)
```

> 其实要用try/except，我这里省略了

## 在Django应用中实现

### 详情页

从Redis取出的数据要decode：

```python
def xxx(request):
    # ...
    html = rd.get("BlogpageHtml_使用Docker部署MongoDB").decode()
    return render(request, "xxx", {"xx":html})
```

这样就可以看到成功了！

### 列表页

在模板中设置一下就好了！


## 小结

Redis可能不是最好的数据库解决方案。考虑到数据的持久性和拓展性，对于博客这种典型的Web应用，
我认为Mongo还是更好用。