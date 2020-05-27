```lw-blog-meta
{"title": "GoTour后续练习题之 Web爬虫 exercise-web-crawler.go", "date": "2019-06-15", "tags": ["Golang"], "brev": "Go-Tour的最后一个练习题(https://tour.golang.org/concurrency/10)，感觉稍微有点点难度，在这里把解题思路和参考答案分享一下。"}
```

## 审题

> Exercise: Web Crawler
> In this exercise you'll use Go's concurrency features to parallelize a web crawler.
>
> Modify the Crawl function to fetch URLs in parallel without fetching the same URL twice.
>
> Hint: you can keep a cache of the URLs that have been fetched on a map, 
but maps alone are not safe for concurrent use!
>
> 在这个练习中，我们将会使用 Go 的并发特性来并行化一个 Web 爬虫。  
修改 Crawl 函数来并行地抓取 URL，并且保证不重复。  
提示：你可以用一个 map 来缓存已经获取的 URL，但是要注意 map 本身并不是并发安全的！


练习给出了程序的主体部分，我们要做的是对其进行改进，实现两个`TODO`目标。
```golang
package main

import (
    "fmt"
)

type Fetcher interface {
    // Fetch 返回 URL 的 body 内容，并且将在这个页面上找到的 URL 放到一个 slice 中。
    Fetch(url string) (body string, urls []string, err error)
}

// Crawl 使用 fetcher 从某个 URL 开始递归的爬取页面，直到达到最大深度。
func Crawl(url string, depth int, fetcher Fetcher) {
    // TODO: 并行的抓取 URL。
    // TODO: 不重复抓取页面。
        // 下面并没有实现上面两种情况：
    if depth <= 0 {
        return
    }
    body, urls, err := fetcher.Fetch(url)
    if err != nil {
        fmt.Println(err)
        return
    }
    fmt.Printf("found: %s %q\n", url, body)
    for _, u := range urls {
        Crawl(u, depth-1, fetcher)
    }
    return
}

func main() {
    Crawl("https://golang.org/", 4, fetcher)
}
```
上面的部分是一个爬虫的控制模块，主要是通过递归调用`Crawl`，实现爬取指定的层数。
在函数内部调用了`Fetcher`接口的`Fetch(url)`方法，它封装了网络请求的过程，这里只要结果。

接下来还给了我们一个实现了`Fetcher`接口的`fakeFetcher`(假的下载器)。
主要意思就是不从Web上下载内容了，直接用本地缓存好的数据通过`Fetch(url)`方法返回给你。
```golang
// fakeFetcher 是返回若干结果的 Fetcher。
type fakeFetcher map[string]*fakeResult

type fakeResult struct {
    body string
    urls []string
}

func (f fakeFetcher) Fetch(url string) (string, []string, error) {
    if res, ok := f[url]; ok {
        return res.body, res.urls, nil
    }
    return "", nil, fmt.Errorf("not found: %s", url)
}

// fetcher 是填充后的 fakeFetcher。
var fetcher = fakeFetcher{
    "https://golang.org/": &fakeResult{
        "The Go Programming Language",
        []string{
            "https://golang.org/pkg/",
            "https://golang.org/cmd/",
        },
    },
    "https://golang.org/pkg/": &fakeResult{
        "Packages",
        []string{
            "https://golang.org/",
            "https://golang.org/cmd/",
            "https://golang.org/pkg/fmt/",
            "https://golang.org/pkg/os/",
        },
    },
    "https://golang.org/pkg/fmt/": &fakeResult{
        "Package fmt",
        []string{
            "https://golang.org/",
            "https://golang.org/pkg/",
        },
    },
    "https://golang.org/pkg/os/": &fakeResult{
        "Package os",
        []string{
            "https://golang.org/",
            "https://golang.org/pkg/",
        },
    },
}
```



## 步骤1：实现并发

这里我们可以看到，题目要求我们递归地调用`Crawl()`，那么如何在递归函数内实现并发？  

第一个念头是想到直接加一个`go`嘛：
```golang
    for _, u := range urls {
        go Crawl(u, depth-1, fetcher) // 这里加上go
    }
```
但是这样是不行的，运行结果会发现：只爬取了第一个网页：
```text
found: https://golang.org/ "The Go Programming Language"

Process finished with exit code 0
```

为什么呢？其实也很简单，因为你通过`go Crawl(u, depth-1, fetcher)`创建了一个线程，
却又马上`return`了。要知道Goroutine是有内存回收机制的，如果当前的函数返回了，那么即使是正在运行的goruntine也会被回收。
所以第一个子进程还没来得及开始"下载"，就被杀死了。

那么我们就用`channel`阻塞住这个函数，相当于python中的`jion()`用法：
```golang
func Crawl(url string, depth int, fetcher Fetcher, chParent chan bool) {  // 增加了一个channel参数！
    defer func() {chParent <- true}()     // 返回的时候通知上一级！
    if depth <= 0 {
        return
    }
    body, urls, err := fetcher.Fetch(url)
    if err != nil {
        fmt.Println(err)
        return
    }
    fmt.Printf("found: %s %q\n", url, body)
    chChild := make(chan bool)              // 建立一个channel与子线程通讯！
    for _, u := range urls {
        go Crawl(u, depth-1, fetcher, chChild)   // 使用channel与子线程通讯！
    }
    for range urls{    // 等待所有的子进程结束！
        <- chChild     
    }
    return
}

func main() {
    ch := make(chan bool)        // 因为是递归调用，最顶层也必须要通过channel通信
    go Crawl("https://golang.org/", 4, fetcher, ch)
    <- ch
}
```

另外，我们模拟一下网络IO等待的时间，别让线程跑的太快了：
```goalng
func (f fakeFetcher) Fetch(url string) (string, []string, error) {
    time.Sleep(time.Duration(rand.Intn(1000))*time.Millisecond)  // 增加这一行，sleep一下
    if res, ok := f[url]; ok {
        return res.body, res.urls, nil
    }
    return "", nil, fmt.Errorf("not found: %s", url)
}
```

这样就实现了并发的目标：
```text
found: https://golang.org/ "The Go Programming Language"
found: https://golang.org/pkg/ "Packages"
not found: https://golang.org/cmd/
found: https://golang.org/pkg/os/ "Package os"
found: https://golang.org/pkg/fmt/ "Package fmt"
found: https://golang.org/ "The Go Programming Language"
found: https://golang.org/pkg/ "Packages"
not found: https://golang.org/cmd/
found: https://golang.org/pkg/ "Packages"
found: https://golang.org/ "The Go Programming Language"
found: https://golang.org/pkg/ "Packages"
found: https://golang.org/ "The Go Programming Language"
not found: https://golang.org/cmd/

Process finished with exit code 0
```




## 步骤2：排除重复

这个问题更简单一些，我们可以想到用一个`[]string`去记录已经爬取过的url，但是在实现的时候会发现，我们是用遍历的方式在切片中查找，
这样效率会非常低。学过算法我们知道有一些更高级的基于树的查找算法，我们就姑且认为Go的`map`默认就使用高级的查找算法吧，
所以用`map[string]bool`来储存已经爬取过的url。

```golang
// 建立一个带锁的map
var urlfetched *urlFetched = &urlFetched{urls: map[string]bool{}}

type urlFetched struct {
    urls map[string]bool
    lock sync.Mutex
}

func (self *urlFetched) IsFetched(url string) bool {
    self.lock.Lock()
    defer self.lock.Unlock()
    if b, _ := self.urls[url]; !b {    // Go很方便的是，如果Key不存在，返回的是（value的零值，error），不像python会崩溃
        self.urls[url] = true
        return false
    }
    return true
}


// 在之前的Crawl函数中增加一个if
func Crawl(url string, depth int, fetcher Fetcher, chParent chan bool) {
    ......
    if depth <= 0 {
        return
    }
    if urlfetched.IsFetched(url){ // 增加这个if条件
        return
    }
    ......
}
```

这样就实现了排除重复的目标：
```text
found: https://golang.org/ "The Go Programming Language"
found: https://golang.org/pkg/ "Packages"
not found: https://golang.org/cmd/
found: https://golang.org/pkg/os/ "Package os"
found: https://golang.org/pkg/fmt/ "Package fmt"

Process finished with exit code 0

```




## 附加题：控制线程数量

从之前的代码可以发现，在递归调用`Crawl()`的时候，可能会并发出很多很多很多个线程（Go程），
虽然Go程的开销极小，但是一台服务器上起几十万几百万个Go程也没有意义，毕竟网络IO也是有瓶颈的。

所以我们参考python的模式，建立一个Go程的Pool。

那仔细想一想的话，这个实现就比较复杂了：如何控制Go程的数量？

我们先来实现**控制同时下载的Go程数量**。引入一个"令牌"的概念，用channel去实现它。
只有拿到令牌的人才允许参加工作，只要控制了令牌的总数，自然就控制了同时工作的Go程数。
这里先不考虑美观性，我们优先实现功能，所以定义一个全局的Pool：

```golang
var goPool chan bool


func main() {
    goPool = make(chan bool, 100)   // 在main中准备好令牌
    for i:=0; i<100; i++{           // 在main中准备好令牌
        goPool <- true              // 在main中准备好令牌
    }
    ch := make(chan bool)        
    go Crawl("https://golang.org/", 4, fetcher, ch)
    <- ch
}

// 在之前的Crawl函数中增加一个获取令牌的过程（或者直接放到fetcher.Fetch(url)里面去获取令牌会更好）
func Crawl(url string, depth int, fetcher Fetcher, chParent chan bool) {
    ......
    <- goPool  // 获取令牌
    body, urls, err := fetcher.Fetch(url)
    goPool <- true  // 归还令牌
    ......
}

```

完整代码：
```golang
type Fetcher interface {
    // Fetch 返回 URL 的 body 内容，并且将在这个页面上找到的 URL 放到一个 slice 中。
    Fetch(url string) (body string, urls []string, err error)
}

// Crawl 使用 fetcher 从某个 URL 开始递归的爬取页面，直到达到最大深度。
func Crawl(url string, depth int, fetcher Fetcher, chParent chan bool) {
    defer func() { chParent <- true }()
    if depth <= 0 {
        return
    }
    if urlfetched.IsFetched(url) {
        return
    }
    <-goPool
    body, urls, err := fetcher.Fetch(url)
    goPool <- true
    if err != nil {
        fmt.Println(err)
        return
    }
    fmt.Printf("found: %s %q\n", url, body)
    chChild := make(chan bool)
    for _, u := range urls {
        go Crawl(u, depth-1, fetcher, chChild)
    }
    for range urls {
        <-chChild
    }
    return
}

var goPool chan bool

var urlfetched *urlFetched = &urlFetched{urls: map[string]bool{}}

type urlFetched struct {
    urls map[string]bool
    lock sync.Mutex
}

func (self *urlFetched) IsFetched(url string) bool {
    self.lock.Lock()
    defer self.lock.Unlock()
    if b, _ := self.urls[url]; !b {
        self.urls[url] = true
        return false
    }
    return true
}

func Main0013() {
    goPool = make(chan bool, 1)
    for i := 0; i < 1; i++ {
        goPool <- true
    }
    ch := make(chan bool)
    go Crawl("https://golang.org/", 4, fetcher, ch)
    <-ch
}

// fakeFetcher 是返回若干结果的 Fetcher。
type fakeFetcher map[string]*fakeResult

type fakeResult struct {
    body string
    urls []string
}

func (f fakeFetcher) Fetch(url string) (string, []string, error) {
    time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
    if res, ok := f[url]; ok {
        return res.body, res.urls, nil
    }
    return "", nil, fmt.Errorf("not found: %s", url)
}

// fetcher 是填充后的 fakeFetcher。
var fetcher = fakeFetcher{
    "https://golang.org/": &fakeResult{
        "The Go Programming Language",
        []string{
            "https://golang.org/pkg/",
            "https://golang.org/cmd/",
        },
    },
    "https://golang.org/pkg/": &fakeResult{
        "Packages",
        []string{
            "https://golang.org/",
            "https://golang.org/cmd/",
            "https://golang.org/pkg/fmt/",
            "https://golang.org/pkg/os/",
        },
    },
    "https://golang.org/pkg/fmt/": &fakeResult{
        "Package fmt",
        []string{
            "https://golang.org/",
            "https://golang.org/pkg/",
        },
    },
    "https://golang.org/pkg/os/": &fakeResult{
        "Package os",
        []string{
            "https://golang.org/",
            "https://golang.org/pkg/",
        },
    },
}
```
测试一下，可以正常运行。再去`main`函数中把Pool的容量设置为`1`，可以发现的确是"一个一个地"在下载数据。

但是要注意的是，这个令牌机制仅仅只是限制了"**同时下载的Go程数**"，而不能限制"**同时运行的Go程数**"。





## 附加题2：Worker机制

回想一下python的实现，一般我们是创建一个`Thread.Pool()`，然后通过`Queue`来推送url，每个线程从`Queue`中获取任务（url），
然后再把新的任务推回`Queue`中；当`Queue`为空时退出线程。

但是要注意，python的`Queue`是无限长度的，但是Go中的channel是有容量限制的。

所以我们在Go程序中：

1. 设置一个任务切片`[]task`来存放任务，其中任务类型`task{depth int, url string}`是我们自定义的数据类型。
2. 建立一群Go程不断地读取任务列表，如果没有任务就睡一秒；如果有任务就拿着**令牌**去下载。
3. 如果所有**令牌**都在库房中，任务切片也是空的，那就认为爬虫任务结束了，结束程序。

我们实现一下：

```golang
func Main0013_2() {
    c := NewCrawler(1, 4, "https://golang.org/", fetcher)   // 在这里设置Go程数量
    c.Run()
}

type task struct {
    depth int
    url   string
}

func NewCrawler(numGo int, depth int, startURL string, fetcher Fetcher) *Crawler {
    ch := make(chan bool, numGo)
    for i := 0; i < numGo; i++ {
        ch <- true
    }
    return &Crawler{passport: ch, tasks: []task{ {depth, startURL} }, fetcher: fetcher}
}

// Worker机制的Crawler -----------------------------------------
type Crawler struct {
    passport chan bool
    tasks    []task
    fetcher  Fetcher
    lock     sync.Mutex
}

func (self *Crawler) Run() {
    numGo := cap(self.passport)

    for i := 0; i < numGo; i++ {
        go self.work()
    }
    time.Sleep(time.Second)
    for len(self.passport) != numGo {
        time.Sleep(time.Second)
    }
}

func (self *Crawler) getTask() (t task) {
    self.lock.Lock()
    defer self.lock.Unlock()
    if len(self.tasks) != 0 {
        t = self.tasks[0]
        self.tasks = self.tasks[1:]
    } else {
        t = task{}
    }

    return t
}
func (self *Crawler) putTasks(ts []task) {
    self.lock.Lock()
    defer self.lock.Unlock()
    self.tasks = append(self.tasks, ts...)
}

func (self *Crawler) work() {
    var t task
    for ; ; {
        <-self.passport
        t = self.getTask()
        if t.depth <= 0 {      // 没有任务，交还令牌，睡一秒
            self.passport <- true
            time.Sleep(time.Second)
            continue
        }
        if urlfetched.IsFetched(t.url) {   // 重复的任务放弃，马上领取下一个任务
            self.passport <- true
            continue
        }
        body, urls, err := fetcher.Fetch(t.url)   
        if err != nil {      // 下载任务失败，马上领取下一个任务
            fmt.Println(err)
            self.passport <- true
            continue
        }
        fmt.Printf("found: %s %q\n", t.url, body)
        if d := t.depth - 1; d > 0 {    // 如果发现新的任务，并且没有超过最大深度，那就向任务队列添加
            ts := make([]task, len(urls))
            for i, u := range urls {
                ts[i] = task{d, u}
            }
            self.putTasks(ts)
        }
        self.passport <- true    // 任务完成，交还令牌
    }
}

```
