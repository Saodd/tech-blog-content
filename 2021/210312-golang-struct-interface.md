```yaml lw-blog-meta
title: "Go语法：struct与interface"
date: "2021-03-12"
brev: "好害怕哦"
tags: ["Golang"]
```

## 引子

事情的起源，是我[在论坛上顺手回答了一个Golang相关的问题](https://v2ex.com/t/760802) ，我把问题重新梳理一下，是这样的：

```go
type A struct {
}

func (a *A) Work(){
} 

type Worker interface {
	Work()
}
func main() {
	var itf Worker = A{}  // 编译失败：不能赋值
	itf.Work()
}
```

看起来`A`是实现了接口，但是并不能赋值给接口变量`itf`。

不想解释太多，所以我给出的答案是：「写成`&A{}`就可以了。因为`A`并没有实现接口，`*A`才实现了。」

但实际上，这里面还有亿点点细节。

## 结构体方法 与 指针

> 2021-03-24更正：结构体和指针方法是可以互相调用的，它们都会被编译器隐式转换，这是一个特性。  
> 那么重新考虑前面一节的问题，为什么在接口变量中不能互相转换？  
> 我认为这是「契约式编程」的一种体现。如果接口声明的是指针方法，那么我们就应该认为，调用方法可能会改变变量对象本身；这时如果传入一个只实现了结构体方法的变量，我们预期调用它的方法不会导致它本身的改变，这就与接口所声明的不一样了。

我们知道，在定义结构体方法的时候，可以有两种写法：`func (a A) Work()` 和 `func (a *A) Work()` ，前者以一个结构体本身作为`接受体`，后者以一个指针作为`接受体`。

我们看三段代码：

```go
type A struct {
}
func (a *A) Work(){  // 指针！
}
func main() {
	A{}.Work()  // 编译失败，A没有这个方法。（IDE会提示你将其转换为本地变量）
}
```

```go
type A struct {
}
func (a *A) Work(){  // 指针！
}
func main() {
    a := A{}
    a.Work() // 正确！
}
```

```go
type A struct {
}
func (a A) Work(){  // 结构体！！
}
func main() {
	(&A{}).Work()  // 正确
}
```

第一种情况，不能对`A{}`调用指针方法，我想原因应该是编译器将它作为常量来处理了，毕竟在生成了它之后，再也不能访问到它的本体了（因为结构体的传递都伴随着浅拷贝）。

第二、三段代码中，结构体受体和指针受体方法是可以通用的。如果我们查看它的汇编代码，编译器会自动生成一个指针的方法。

```shell
# 查看汇编命令
go tool compile -N -l -S main.go
go tool objdump main.o
```

## 不能取址的类型

1. 字符串中的字节

```go
var a string = "123"
println(&a[1])  // 编译失败

var b = []byte("123")
println(&b[1])  // 正确
println(&b[100])  // 编译正确，虽然会panic
```

2. map 对象中的元素 [参考#11865](https://github.com/golang/go/issues/11865#issuecomment-124801193) [参考#3117](https://github.com/golang/go/issues/3117#issuecomment-428867324) (因为取出来的结构体是副本啊)

```go
var c = map[int]int{1: 666}
println(&c[1])  // 编译失败
```

3. 常量
4. 包级别的函数等

```go
func MyFunc()  {	
}
func main() {
	println(&MyFunc)  // 编译失败
	
	f := func() {}
	println(&f)  // 正确

	const d = 123
    println(&d)  // 编译失败
}
```

## 结构体作为接受体的代价

从上面看来，似乎只要把所有的方法都用结构体本身作为接受体，那就能保证最大程度的兼容咯，`*A`也能用、`A`也能用，皆大欢喜？

从语法来说，这么做的确没问题。但是要记住，使用结构体本身，会导致大量的拷贝操作，当结构体本身体积比较大时，会造成性能负担。

做个小实验：

```go
type A struct {
	Data int
	Data2 int
	Data3 int
	Data4 int
}
//go:noinline
func (a A) Work(){
	panic("故意的")
}

type Worker interface {
	Work()
}

func main() {
	var itf Worker = &A{Data: 666}
	itf.Work()
}
```

> 注意 `go:noinline`这个注解，因为函数太简单，会被编译器优化（内联），因此需要这个注解来强行禁止内联。这样才能看到传入这个函数的参数列表。

```text
panic: 故意的

goroutine 1 [running]:
main.A.Work(0x29a, 0x0, 0x0, 0x0)  // A的四个字段平铺在这里
	C:/xxx/main.go:68 +0x45
main.main()
	C:/xxx/main.go:77 +0x52
```

看到了吗，如果使用`func (a A)`这样用`A`作为接受体的话，整个结构体会平铺在参数上。 而如果使用`func (a *A)`的话，则只会传入一个指针。

当然，指针寻址也有消耗，但是这个损耗相对来说忽略不计；同时多用指针可能导致更多的变量放在堆上，给GC增加压力。但是，既然都用go了，我们一般不会抠门到这个程度；追求极致性能请用C去。

## 接口变量

在之前 [关于反射包的文章](../2020/201220-GoBlog-The-Laws-of-Reflection.md) 中介绍过，`接口变量`实际上是包含两个字段的结构体，分别是`底层类型`和`底层指针`。

重要的话说两遍，接口变量中保存的原始对象，是一个指针！

如果我们给类型`A`实现了某个接口，然后把`A`传给一个接口变量（而不是用`*A`），会发生什么？

看看下面的代码，想想它应该打印出什么内容：

```go
type A struct {
	Data int
}
func (a A) Work(){
	println(a.Data)
}

type Worker interface {
	Work()
}

func main() {
	var a = A{Data:233}
	var itf Worker = a

	a.Data = 666
	a.Work()
	itf.Work()
}
```

```text
# 答案
666
233
```

继续探索一下，用内置的`print()`函数打印出它们两个变量的地址，会发现它们并不相等：

```go
println(&a)  // 输出 0xc00011df68
println(itf)    // 输出 (0xdcb540,0xe83748)
```

作为对照，我们把`a`改为指针，则发现`a`和`itf`是同一个东西：

```go
func main() {
	var a = &A{Data:233}
	var itf Worker = a

	a.Data = 666
	a.Work()    // 输出 666
	itf.Work()  // 输出 666

	println(a)    // 输出 0xc0000a2930
	println(itf)  // 输出 (0xfab320,0xc0000a2930)
}
```

解释也很简单：

- 传入结构体这个行为，等同于给函数传参时使用结构体，会导致结构体的浅拷贝。
- 因为接口变量储存的只是指针，因此传入一个结构体，它会隐式地取址。
- 综上所述，接口变量内保存的是副本的指针。

这个特性非常容易导致bug，所以当要传值给接口变量的时候，请尽量使用指针。（某些情况可能也会故意不这样干，但请确认你知道你在做啥）

## nil的坑

本节参考自： [Go 的一些"坑" - wudaijun](https://wudaijun.com/2018/08/go-is-not-good/)

同样的接口变量，带类型的和不带类型的，表现是完全不同的。如果把一个`(*A)(nil)`传给一个接口变量，那么这个接口变量中是保存着`*A`这个类型和一个`nil`的指针的，因此它不等于nil。而直接令一个接口变量为`nil`，那它此时的类型是空、指针也是空，所以等于nil。

看代码：

```go
func main() {
	var a *A = nil
	var itf Worker = a
	println(itf, itf == nil) // (0xb9b2c0,0x0) false

	itf = nil
	println(itf, itf == nil) // (0x0,0x0) true
}
```

对于结构体方法，使用nil时也有坑要注意。给指针方法传入空指针，没问题，只要不在方法里使用这个空指针就行。但是给结构体方法传入空指针，会崩溃，因为对空指针解引用是非法的。（简而言之就是要注意隐式的解引用）

看代码：

```go
type A struct{}
func (a *A) Work() {}
func (a A) Stop()  {}

func main() {
	var a *A = nil
	a.Work()  // 正常运行！（IDE会提示建议）
	a.Stop()  // panic！（IDE会提示警告）
}
```

## 小结

总而言之，（大多数情况下）闭着眼睛用指针就对了。

目前想到的可以不用指针的场景：

1. 刻意保护它本身不被更改。
2. 结构体的组合（内嵌）
3. 性能优化场合（逃逸分析，减少GC压力）

其实这里面的坑还是很多，之前自信满满地总结了一番，没想到很快就发现有错误。所以上面写的这些内容建议你也不要轻易相信，还是更多以自己实践和查阅权威资料为准，欢迎指正交流。

想要更进一步探究，我觉得我该开始学习汇编了 :)
