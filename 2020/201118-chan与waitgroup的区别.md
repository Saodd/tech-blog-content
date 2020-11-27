```yaml lw-blog-meta
title: 'chan 与 sync.Waitgroup 的区别'
date: "2020-11-18"
brev: "与同事聊起这个问题，觉得有点意思，所以总结一下。"
tags: [Golang]
```

## 由头

最近公司举办了一期编程比赛，内容大概是写一个服务，对另一个服务做高强度的并发调用然后返回结果。

于是就聊到了关于如何在Golang中处理并发调用的等待问题。

这两个数据结构都能用，但是区别是什么呢？我虽然大概知道什么情况下该用什么，但是一时半会还真不能用两句话把这个问题说清楚。

于是索性写成一篇文章。

## 关键区别

`sync.Waitgroup`其实就是一个带锁的`int`。

`chan`其实就是一个带锁的数组。

> 我没有看过它们的源码实现，都是道听途说+自我感觉，以上内容如有错误概不负责：）

那么，理论上，一个int能做的事情，用数组都能做。（至少我可以用len来表示一个int）所以，理论上来说，`chan`是比`Waitgroup`更强大的。

更何况`chan`还能够储存和传递数值的作用。

但并不是越强越好。毕竟操作数组的开销比操作一个int的开销会大很多。另一方面，`Waitgroup`具有语义性，有更强的目的性，会让代码更可读。如果使用`chan`来代替的话，我们可能需要仔细阅读上下文代码才能理解它的真实作用。

所以，还是要看具体情况来选用。

下面，我列举一些我能够想象到的场景，并分别讨论哪种更加合适。

## 可以互换的场景：并发等待

其实`Waitgroup`也就只有 并发等待 这一个功能了。（我不会造词，理解意思就好）

比如一种情况是，我们要写一个简单的爬虫，发起一批并发请求然后把内容保存到硬盘。此时，在我们的程序流程中，我们其实并不关心每个并发任务的返回值，因此用`Waitgroup`就最好。

示例代码：

```go
func main() {
    var urls []string
    wg := &sync.WaitGroup{}
    wg.Add(len(urls))
    for _, url := range urls {
        go DownloadWebsite(wg, url)
    }
    wg.Wait()
}

func DownloadWebsite(wg *sync.WaitGroup, u string) {
    // http.Get(u)
    wg.Done()
}
```

我们用`chan`也完全可以达到相同的效果：

```go
func main() {
    var urls []string
    cb := make(chan struct{})
    for _, url := range urls {
        go DownloadWebsite(cb, url)
    }
    for range urls {
        <-cb
    }
}

func DownloadWebsite(cb chan<- struct{}, u string) {
    // http.Get(u)
    cb <- struct{}{}
}
```

> 如果你对Golang的一些骚操作不太熟悉的话，可以把上面代码中的`struct{}`替换为`int`，这会让你更好理解。  
> 如果你根本看不懂我在说什么，那你可能需要去看一些更基础的教程。

上面两份代码，虽然代码量几乎完全相同，但是我相信不管是新手还是老鸟，一定都会认为前面那份代码具有更好的可读性。（特别是当代码逻辑变得更复杂之后。）

## 场景2：需要返回值

最经典的场景之一，是我们需要返回值的场景。比如我们下载一批接口返回值，然后进行汇总。示例代码：

```go
func main() {
    var urls []string
    cb := make(chan int)
    for _, url := range urls {
        go DownloadWebsite(cb, url)
    }
    var sum int
    for range urls {
        sum += <-cb
    }
}

func DownloadWebsite(cb chan<- int, u string) {
    var data int
    // data =  http.Get(u) && Parse(resp)
    cb <- data
}
```

这种情况我们必须要使用`chan`来返回值。

> 当然，如果一定要杠的话，你也可以写一个带锁的数组来储存结果，然后用Waitgroup来控制并发流程。但没人会觉得这样写有多牛逼。

## 场景3：裂变任务

有时候可能我们的异步任务的数量并不是固定的。

比如一种情况是，我们的爬虫在下载了网页之后，会分析该网页中还包含哪些链接，然后对这些新链接发起新的并发下载任务。

示例代码：

```go
func main() {
    var urls []string
    wg := &sync.WaitGroup{}
    wg.Add(len(urls))
    for _, url := range urls {
        go DownloadWebsite(wg, url)
    }
    wg.Wait()
}

func DownloadWebsite(wg *sync.WaitGroup, u string) {
    resp, _ := http.Get(u)
    links := ParseLinksFromWebsite(resp)
    wg.Add(len(links))
    for _, link := range links {
        go DownloadWebsite(wg, link) // 递归调用
    }
    wg.Done()
}

func ParseLinksFromWebsite(resp *http.Response) (links []string) {
    // 解析网页中的超链接
    return links
}
```

这种情况下，我们会发现根本无法使用`chan`来代替——因为我们事先并不知道我们需要从这个`chan`里面等待多少次"回调"。而`Waitgroup`就可以很轻松地完成这个任务。

## 小结

目前暂时只能想到这几种情况（在并发等待这个话题下）。如果以后想到别的会再来补充。

其实`chan`的用途非常广泛，可以用来做很多很有意思的事情，比如我之后会写一篇关于流量控制的文章。
