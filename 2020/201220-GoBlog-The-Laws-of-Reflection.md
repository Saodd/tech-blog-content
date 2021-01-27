```yaml lw-blog-meta
title: '[TheGoBlog] The Laws of Reflection'
date: "2020-12-20"
brev: "发布时间2011-9-6，官方对于反射包的解读。"
tags: ["Golang"]
```

# The Laws of Reflection

[原文地址](https://blog.golang.org/laws-of-reflection)

Rob Pike  
6 September 2011

## 引言

「反射Reflection」是一种让程序能够检查它自己的数据结构的能力。它是「元编程metaprogramming」的一种。它也带来了很多的难点。

在这篇文章中，我们将通过解释Go语言中的反射是如何工作的来阐明它。每个语言的反射模型都不一样（甚至有很多语言根本就不支持反射）。在下文中，“反射”都指代“Go语言中的反射”。

## 类型和接口

我们稍微回顾一下Go语言中的类型系统。

Go是静态类型。每个变量都有一个静态类型，并且只有一个、已知的、在编译时就固定的类型。如果我们声明如下：

```go
type MyInt int

var i int
var j MyInt
```

那么`i`的类型就是`int`，而`j`的类型就是`MyInt`。他们两个有不同的类型，并且虽然他们底层类型一样，他们（在不经过转换的情况下）无法互相赋值。

一个重要的类型分类是接口类型，它代表着一套固定的函数方法。一个接口变量可以储存任意的「确定类型的concrete」值（非接口类型），只要这个确定类型实现了这个接口声明的所有方法。一对经典的例子是`io.Reader`和`io.Writer`：

```go
// Reader is the interface that wraps the basic Read method.
type Reader interface {
    Read(p []byte) (n int, err error)
}

// Writer is the interface that wraps the basic Write method.
type Writer interface {
    Write(p []byte) (n int, err error)
}
```

`Reader`接口变量可以储存任意实现了它的方法的变量：

```go
var r io.Reader
r = os.Stdin
r = bufio.NewReader(r)
r = new(bytes.Buffer)
// and so on
```

这里需要注意的是，无论接口变量`r`储存的是何种确定类型，它的类型都是`io.Reader`这个接口类型（译者注：即你只能使用这个接口所声明的方法。这是一种契约式编程思想。）

一个极端的例子是`interface{}`这个接口，它没有声明任何方法，因此任何确定类型都实现了这个接口，因此这个接口的变量可以储存任意确定类型的值。

有些人说Go的接口是一种动态类型，这是一种误解。（译者注：看完这篇文章你就懂了）

## 接口是如何代表变量的

Russ Cox 写过一篇 [详细的博客](https://research.swtch.com/2009/12/go-data-structures-interfaces.html) 来介绍接口值，我们这里简单总结一下。

一个接口类型的变量，储存着2个值：一个是赋给这个变量的值（一个指针），一个是这个值对应的确定类型的描述符。简单说，就是一个值和一个类型。举例：

```go
var r io.Reader
tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
if err != nil {
    return nil, err
}
r = tty
```

此时变量`r`储存着的`(值, 类型)`是`(tty, *os.File)`。注意，`*os.File`不仅仅实现`Read`一个方法，但是在接口类型的封装下，我们只能访问`Read`这一个方法。但是`值`里面存放着相应类型的所有信息，因此我们可以做这样的反向转化：

```go
var w io.Writer
w = r.(io.Writer)
```

上述代码是一个「类型断言 type assertion」；它断言，`r`里面的“东西”同样实现了接口`io.Writer`，然后因此我们可以将这个“东西”复制给接口变量`w`。这次赋值后，`w`储存着`(tty, *os.File)`，跟`r`里面的东西是一样的。但，在接口的封装下`w`变量只能使用`Write`方法。

同理，我们还可以这样赋值：

```go
var empty interface{}
empty = w
```

这次不需要做类型断言，是因为`interface{}`接口是包含`io.Writer`接口的，后者的变量都是实现了前者的。此时`empty`变量中储存的依然是`(tty, *os.File)`这两个东西。（译者注：可以试着用`print`函数打印一个接口变量，会得到两个内存地址）

这里有个细节，接口变量里面不能存放接口类型，只能存放确定类型。

好，接下来开始反射。

## 反射法则一

从基础来讲，反射只是一种“检查接口变量的值和类型的机制”。在`reflect`包中有两个类型我们需要学习：`Type`和`Value`，以及相应的函数`ValueOf`和`TypeOf`。

我们从`TypeOf`开始：

```go
func main() {
    var x float64 = 3.4
    fmt.Println("type:", reflect.TypeOf(x))
}
```

```text
type: float64
```

（结合上文）你可能会感到奇怪，接口变量在哪里？看起来我们只是给`TypeOf`函数传入了一个确定类型。但它是存在的，它存在于函数参数中：

```go
// TypeOf returns the reflection Type of the value in the interface{}.
func TypeOf(i interface{}) Type
```

`Type`和`Value`都有很多的方法给我们去检查和篡改（底层的数据）。例如下面这几个方法：

```go
var x float64 = 3.4
v := reflect.ValueOf(x)
fmt.Println("type:", v.Type())
fmt.Println("kind is float64:", v.Kind() == reflect.Float64)
fmt.Println("value:", v.Float())
```

输出：

```text
type: float64
kind is float64: true
value: 3.4
```

关于反射包，还有一些细节可能需要提一下。

第一，为了保持API简单，`getter`和`setter`方法都在可以容纳目标值的最大类型上进行操作。例如：int64会用来操作所有的有符号整形。即，一个储存着`int`类型的`Value`对象，调用它的`Int`方法会返回一个`int64`，所以在使用的时候要自己做一下类型转换。

第二，`Kind`描述的是一个对象的底层数据类型，而不是其表层的静态类型。如果一个反射对象包含着一个用户自定义的整形类型，那么`v.Kind`报告的是`reflect.Int`而不是`MyInt`，如果需要静态类型那么应该用`Type`方法。

```go
type MyInt int
var x MyInt = 7
v := reflect.ValueOf(x)
```

## 反射法则二

可以用`Interface`方法来将`reflect.Value`对象转换为一个接口对象：

```go
y := v.Interface().(float64) // y will have type float64.
fmt.Println(y)
```

`fmt.Println`的参数是`interface{}`，它接收接口变量，然后在内部用反射来取得确定类型和其值。但要注意的是，直接传入一个`Value`是不会得到其底层值的：

```go
fmt.Println(v.Interface())  // 打印出的是其原本类型
fmt.Println(v)  // 打印出的是reflect.Value这个类型
```

## 反射法则三

接下来说的这个法则，非常地微妙而且容易搞错。下面展示一段错误代码：

```go
var x float64 = 3.4
v := reflect.ValueOf(x)
v.SetFloat(7.1) // Error: will panic.
```

```text
panic: reflect.Value.SetFloat using unaddressable value
```

问题在于，`v`这个Value是不可设置的。`Settability`是`Value`的一个属性，并且并不是所有的Value都有。

我们用`CanSet`方法可以查看这个属性：

```go
var x float64 = 3.4
v := reflect.ValueOf(x)
fmt.Println("settability of v:", v.CanSet())  // 输出: false
```

那么，什么是「可设置性」？——它取决于Value变量是否拥有那个传入的东西。（译者注：记住，go的一切都是值，函数参数是传值的！！）

在上面的例子中，我们调用`reflect.ValueOf(x)`的时候，我们是给这个函数**传递了一个float64的副本**，而不是`x`变量本身。因此，假如我们允许修改`v`中的数值，但是实际上它又不改变原始的`x`变量中的值，这显然是不对的。

回想一下，我们在函数调用的时候，如果希望函数可以修改参数，我们要如何做？——传入变量的指针即可。那么我们就这么试试：

```go
var x float64 = 3.4
p := reflect.ValueOf(&x) // Note: take the address of x.
fmt.Println("type of p:", p.Type())
fmt.Println("settability of p:", p.CanSet())
```

```text
type of p: *float64
settability of p: false
```

这里依然是不可修改的，但是，我们并不像修改这个指针记录的地址本身，而是想修改指针指向的那片内存中保存着的东西。此时，我们需要`Elem`方法来取得指针指向的东西：

```go
v := p.Elem()
fmt.Println("settability of v:", v.CanSet())  // 输出: true
```

然后我们就可以操作这个东西来改变原始对象了：

```go
var x float64 = 3.4
p := reflect.ValueOf(&x)
p.Elem().SetFloat(7.1)
fmt.Println(x)  // 输出: 7.1
```

## 结构体

对于结构体来说会稍微复杂一点。我们要做的是通过`Field`方法来遍历它的字段：

```go
type T struct {
    A int
    B string
}
t := T{23, "skidoo"}
s := reflect.ValueOf(&t).Elem()
typeOfT := s.Type()
for i := 0; i < s.NumField(); i++ {
    f := s.Field(i)
    fmt.Printf("%d: %s %s = %v\n", i, typeOfT.Field(i).Name, f.Type(), f.Interface())
}
```

```text
0: A int = 23
1: B string = skidoo
```

有一点需要注意的是，由于Go的命名规则，只有大写开头的字段才可以被外部（的package）访问。因此`f.Interface()`不能对小写开头的字段使用，否则panic。可设置性也是同理。

## 总结

- 反射 可以将接口变量转化为反射对象；
- 反射 可以将反射对象转化回接口变量；
- 要修改反射对象，它必须是可设置的。

反射很强大，也很危险，请小心使用。

关于反射其实还有很多内容：chan的发送和接收、分配内存、切片和字典，调用函数……等等。但这篇博客先到此为止。
