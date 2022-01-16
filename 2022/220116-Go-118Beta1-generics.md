```yaml lw-blog-meta
title: "Go1.18 Beta1 泛型尝鲜"
date: "2022-01-16"
brev: "盼星星盼月亮"
tags: ["Golang"]
```

## 背景

早在很多年前Golang就有了泛型提案。但是总体来说发展很慢，而且提案的设计稿还有过大改。

我的理解是，Golang是一门介于底层语言和高层语言之间的一门专注于现代Web开发的语言，核心团队确实比较谨慎务实，在语法特性的选择上显得保守、而且更倾向于优化运行性能。

磨磨蹭蹭这么久，不过总算还是按照预期的时间计划——2022年，1.18版本——推出了泛型。

截至目前的Beta1版本仅提供了最基础的泛型实现。一方面是好奇/赶潮流，另一方面也算是响应社区/核心团队的号召，（这里吐槽一下，中文社区内关于这个版本的研究文章目前几乎没有，）所以今天我来亲自体验一下这个版本的泛型实现。

## Go1.18 Beta1

原文地址:  [Go 1.18 Beta 1 is available, with generics](https://go.dev/blog/go1.18beta1)

Go1.18 的正式版京不会在最近几个月内推出，这次先推出了Beta1.

这个版本是第一个支持『使用类型参数的泛型代码』的版本。泛型是Go语言自从Go1以来最为显著的变化，也是对语言使用上影响最大的一次变化。巨大的变化可能带来严重的BUG，所以我们非常谨慎。些更深入的使用场景，例如递归泛型，将被推迟到后续的Beta版本中推出。我们希望精力充沛的尝鲜者积极地在你们需要的场景下使用这些特性。同时我们推出了一份新的 Tutorial 以及 playground上对泛型的支持。

除了泛型以外，Go1.18Beta1还添加了对『fuzzing-based tests（模糊测试）』的支持 [参考](Fuzzing is Beta Ready) ；还增加了『Go workspace mode』是  go mod 的强化版，用于一个目录下存放多个Golang项目的场景；以及其他一些冷门细节和优化。

## 泛型

> 关于泛型，作为一个Typescript用户，泛型对我来说早就是家常便饭了。如果对这块不熟悉的同学，建议你去玩一玩Typescript，最容易学到精髓。

参考文章:  [Tutorial: Getting started with generics](https://go.dev/doc/tutorial/generics) 这篇文章讲的过于入门了，仅供娱乐。

最核心的用法就是用中括号`[T any]`像这样声明一个泛型类型`T`，这个T要满足`any`的限制条件，也就是`interface`，这里`any`是一个语法糖 等价于`interface{}`。

基本例子：

```go
type Number interface {
    int | int64 | float64
}

func Sum[T Number](numbers []T) T {
	var sum T
	for _, number := range numbers {
		sum += number
	}
	return sum
}

func main() {
    fmt.Println(Sum([]int{1, 2, 3}))
}
```

- 先声明了一个接口`Number`，它可以是`int/int64/float64`三种类型之一，或者同时兼容它们三种的派生类型（注意理解接口的意思，可以用约束(constraint)这个词语来理解，一个未被采纳的关键字）。
- 然后写了一个泛型函数`Sum`，它可以接受一个`[]Number`作为入参，然后返回一个`Number`。`Number`这个接口本身被保存为『类型参数』`T`。
- 在调用`Sum`时，无需显式指定`T=int`，因为编译器可以从后面的`[]int{}`中推断出`T=int`。

好，基本用法就是这么简单。接下来我们解决一个实际问题，也是用Golang做业务开发可能会遇到的很典型的问题——结构体数组排序。

在以前，标准库`sort`中提供了对某些基本类型的排序函数，而如果是开发者自己定义的结构体类型，那要么从中取出基本类型（例如数字类型的ID字段）然后用标准库排序，要么自己撸一个排序算法。怎么写都算不上优雅。（在业务中的处理，我会让架构中的其他组件代劳，例如让数据库排序/让前端排序）

我们选择最简单的排序算法，冒泡排序，花一分钟手撸一个：

```go
func BubbleSort[T any](elems []T, compareFunc func(a, b T) int) {
	if len(elems) < 2 {
		return
	}
	for i := len(elems); i > 0; i-- {
		for ii := 1; ii < i; ii++ {
			if compareFunc(elems[ii-1], elems[ii]) > 0 {
				elems[ii-1], elems[ii] = elems[ii], elems[ii-1]
			}
		}
	}
}
```

这里有点函数式编程的意思，传入的参数是 一个任意类型的数组 + 一个这个类型的比较函数，如果写过js的同学应该就会非常非常熟悉这个套路（我也确实是从js中取来的灵感）。

接下来我们虚构一个结构体和它配套的比较函数，来试用一下：

```go
type MyStruct struct {
	Value int
}

func main() {
	elems := []MyStruct{ // 为了方便print这里就不用指针了
		{Value: 2}, {Value: 1}, {Value: 3}, {Value: 0},
	}
	fmt.Println("排序前：", elems)
	BubbleSort(elems, func(a, b MyStruct) int {
		return a.Value - b.Value
	})
	fmt.Println("排序后：", elems)
}
```

```text
排序前： [{2} {1} {3} {0}]
排序后： [{0} {1} {2} {3}]
```

这样，借助函数式编程风格，我们用现有的泛型能力实现了一个泛型排序算法。业务上要用的话可以将算法内部实现改成快排，但是函数签名是不用改的。

## 缺陷

目前还是有一些问题的，首先是IDE的支持不足，目前在类型参数的时候，IDE是完全没有任何提示的；另外IDE也无法检测泛型语法的正确性，要实际编译了才知道错在哪里。（这个交给时间吧，过段时间自然就能支持了）

然后还有一个可能更严重的，是语言设计的问题。

我在上面的示例代码中使用的是函数式编程，不是因为我喜欢，而是因为只能这样写。其实我心里更理想的是面向对象的方法，像这样：

```go
type MyStruct struct {
	Value int
}

func (this *MyStruct) Compare(another *MyStruct) int {
	return this.Value - another.Value
}
```

不过这样写的话是无法带入`Sum`函数的类型参数里的。不知道是因为我姿势不对呢，还是因为Go的泛型就是故意这样设计的呢？（毕竟Go是没有面向对象的）

## 总结

其实说起Go的泛型，有很多网友都抱持悲观情绪，主要理由就是Golang最招人喜欢的优点就是简洁，而加入泛型之后整个语言可能会变得复杂至少一些。我个人虽然是总体乐观的，但也不能说完全没有担忧。

不过就目前的这个基础泛型能力来看，它对语言的影响应当还是比较小的。接下来几个月继续跟踪后续发展吧。
