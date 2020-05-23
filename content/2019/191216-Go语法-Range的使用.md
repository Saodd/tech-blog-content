---
title: 'Go语法：Range的使用'
description: 'Go的语法虽然简洁，但是有些地方也容易掉坑，在我看来range就是其中之一。我们仔细梳理一下。'
pubDate: '2019-12-16'
updateDate: '2019-12-16'
tags: [Golang]
keywords: ''
---

## 字符串

字符串也是相似的。不过，字符串虽然可以遍历为字符，但是不能修改字符元素。

## 通道 channel

这个其实有点偏题了，因为`chan`与前面的几种数据结构是完全不同的。（不过，chan的底层也是一个用于缓冲的循环数组，因此从这个角度来说也是可以与切片它们相提并论的。）

我们在[A Tour of Go](https://tour.golang.org/concurrency/4)中领略了`range chan`的用法，我们复习一下：

```go
func fibonacci(n int, c chan int) {
    x, y := 0, 1
    for i := 0; i < n; i++ {
        c <- x
        x, y = y, x+y
    }
    close(c)
}

func main() {
    c := make(chan int, 10)
    go fibonacci(cap(c), c)
    for i := range c {
        fmt.Println(i)
    }
}
```

那我们试一下替换这个chan变量，答案跟预期的一样，**不影响**range的返回值：

```go
func main() {
    c := make(chan int, 10)
    go fibonacci(cap(c), c)
    for i := range c {
        c = nil
        fmt.Print(i, " ")
    }
}
// 输出：0 1 1 2 3 5 8 13 21 34
```

## 小结

其实总的看下来，还是非常简单的！只要记住Golang是**传值的**，并且把range看成一个函数，所有问题都迎刃而解了。
