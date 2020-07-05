```yaml lw-blog-meta
title: Go语法：Range的使用
date: "2019-12-16"
brev: Go的语法虽然简洁，但是有些地方也容易掉坑，在我看来range就是其中之一。我们仔细梳理一下。
tags: [Golang]
```


## 语法定义

> `range` iterates over elements in a variety of data structures.

`range`可以遍历多种数据结构体的元素。包括『数组`array`』、『切片`slice`』、『字典`map`』、『字符串`string`』等，还有一个比较特殊的『通道`chan`』。我们一个一个看。

## 数组与切片

这个是最基础的了吧。常用形式是`for i := range s {}`，此时只遍历索引；还有`for i, n := range s {}`同时获取索引和元素值；如果只要元素值，用`for _, n := range s {}`。

我个人经验是，把`range`看成一个函数，可以破解一切问题：

```go
func Range(s iterable) (index int, value int) {}
```

其实整个Golang都是传值的，只要记住这一点，一切都好说。更多关于数组与切片的底层特性，请参考[Go Slices: usage and internals - The Go Blog](https://blog.golang.org/go-slices-usage-and-internals)。

### 一号坑：元素值的传递

比如元素值是整形之类的值，或者某结构体本身，那么修改返回值**不会**影响原值：

```go
func main() {
    var slice = []myInt{ {1}, {2}, {3}, {4} }
    for i, n := range slice {
        n.int = n.int +i
    }
    for _, n := range slice {
        fmt.Print(n, " ")
    }
}
// 输出：{1} {2} {3} {4} 
```

如果元素是指针，那么对指针的修改当然**会**影响原结构体：

```go
func main() {
    var slice = []*myInt{ {1}, {2}, {3}, {4} }
    for i, n := range slice {
        n.int = n.int +i
    }
    for _, n := range slice {
        fmt.Print(n, " ")
    }
}
// 输出：&{1} &{3} &{5} &{7} 
```

### 二号坑：原数组/切片元素改变的情况

我们还是根据传值的概念来理解。

1. 给`range`传递一个切片。因为切片的值是（底层数组指针，长度，容量），传递的是这三个值而不是切片本身；但是由于这三个值就能完全代表一个切片，因此从表现上来说，切片的传递就可以看做是引用传递。修改切片内的元素，**会**影响到range返回的值：

```go
func main() {
    var slice = []int{1, 2, 3, 4}
    for i, n := range slice {
        if i == 0{
            slice[i+1]=100
        }
        fmt.Print(n, " ")
    }
    fmt.Println(slice)
}
// 输出：1 100 3 4 [1 100 3 4]
```

2. 给`range`传递一个数组。传递数组会发生什么？——整个数组的拷贝！那么答案也就很明显了，修改数组内的元素，**不会**影响range返回的值：

```go
func main() {
    var array = [4]int{1, 2, 3, 4}
    for i, n := range array {
        if i == 0{
            array[i+1]=100
        }
        fmt.Print(n, " ")
    }
    fmt.Println(array)
}
// 输出：1 2 3 4 [1 100 3 4]
```

3. 给`range`传递一个数组指针。效果当然是不同的啦。修改元素**会**影响到range返回值。

```go
func main() {
    var array = [4]int{1, 2, 3, 4}
    for i, n := range &array {
        if i == 0{
            array[i+1]=100
        }
        fmt.Print(n, " ")
    }
    fmt.Println(array)
}
// 输出：1 100 3 4 [1 100 3 4]
```

### 三号坑：原数组/切片长度改变的情况

1. 切片改变。**不影响**原来range返回的值。为什么呢？回忆一下切片的传递（底层数组指针，长度，容量），我们给`range`指定的那个切片已经充分地指向了内存中的一块区域，当我们修改切片变量时，不会影响range内的指向。

```go
func main() {
    var slice = []int{1, 2, 3, 4}
    for i, n := range slice {
        if i == 0 {
            slice = []int{100, 200}
            // 或者： slice = slice[:2] 同样不影响range
        }
        fmt.Print(n, " ")
    }
    fmt.Println(slice)
}
// 输出：1 2 3 4 [100 200]
```

2. 数组改变。当然也**不影响**啦！

```go
func main() {
    var array = [4]int{1, 2, 3, 4}
    for i, n := range array {
        if i == 0 {
            array = [4]int{100, 200, 300, 400}
        }
        fmt.Print(n, " ")
    }
    fmt.Println(array)
}
// 输出：1 2 3 4 [100 200 300 400]
```

3. 数组指针改变。当然也也**不影响**啦！为什么呢？因为原来的指针地址已经传入range函数了，因此外部我们给变量重新赋值一个地址，也不会影响range内部的指针。（修改指针变量，与修改指针指向的值，是不同的）

```go
func main() {
    var array = &[4]int{1, 2, 3, 4}
    for i, n := range array {
        if i == 0 {
            array = &[4]int{100, 200, 300, 400}
        }
        fmt.Print(n, " ")
    }
    fmt.Println(array)
}
// 输出：1 2 3 4 &[100 200 300 400]
```

## 字典

字典的表现与切片类似。我理解为字典也是类似切片的一种引用对象形式。

1. 替换元素，**会影响**：

```go
func main() {
    var mp = map[int]int{1:1, 2:4, 3:6}
    for k, v := range mp {
        if k == 1 {
            mp[2]= 200
            mp[3]= 300
        }
        fmt.Print(v, " ")
    }
    fmt.Println(mp)
}
// 可能的输出：1 200 300 map[1:1 2:200 3:300]
// 可能的输出：6 1 200 map[1:1 2:200 3:300]
// 因为遍历字典的顺序是不定的，因此只能说是可能的输出
```

2. 修改变量，**不影响**：

```go
func main() {
    var mp = map[int]int{1:1, 2:4, 3:6}
    for k, v := range mp {
        if k == 1 {
            mp = map[int]int{1:100, 2:200, 3:300}
        }
        fmt.Print(v, " ")
    }
    fmt.Println(mp)
}
// 可能的输出：1 4 6 map[1:100 2:200 3:300]
```

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
