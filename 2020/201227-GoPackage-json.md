```yaml lw-blog-meta
title: Go源码：encoding/json标准库
date: "2020-12-27"
brev: 数据结构的序列化与反序列化，是网络通信中的重要基础功能之一。而Json应该可以说是当前最流行的格式之一。
tags: [Golang, 源码]
```

## 前言

本文阅读的Go版本是 1.15.6 windows/amd64

这个包非常依赖reflect包的用法，因此强烈建议你先学习后者。

## 基本使用

```go
func main() {
    j, _ := json.Marshal(&Blog{
        Title:   "First Blog",
        Content: "Hello, world!",
    })
    fmt.Println(string(j))
}
```

其实 json 这个包最主要的函数就是两个： `Marshal` 和 `Unmarshal`，前者负责将结构体转为字符串（字节数组），后者相反。

重点在于，Go是静态类型的语言，那么这个包是如何读取到静态结构体的字段名称等信息呢？

## 序列化： Marshal

这个函数的注释特别的长，我们慢慢地看一下：

`Marshal` 递归地遍历参数v 。如果遇到一个实现了`Marshaler`接口的值并且它是非nil的，就会调用这个值的`MarshalJSON`方法；没有的话就检查`encoding.TextMarshaler`接口的`MarshalText`方法。

如果上述两个接口都没有，则按照以下规则：
1. Boolean类型转化为 JSON 的 boolean类型.
2. 浮点，整形，数字类型都转化为JSON 的 number类型.
3. String 则强制使用UTF-8编码，将无效字节转化为Unicode形式的rune。并且默认使用 HTMLEscape （将 `<` `>` `&` `U+2028` `U+2029` 替换为 `\u003c` `\u003e` `\u0026` `\u2028` `\u2029`），这个行为可以通过 `Encoder` 来自定义。
4. Array, slice 相应地转化为 JSON array，但是 `[]byte`会编为 base64字符串。空切片则转换为 JSON null 。
5. Struct 转为为 JSON object 。每个公开成员（大写开头）都会作为object的一个成员，使用字段名作为键，除非：
    1. 可以通过 字段tag 来指定名称，作为 object的键；
    1. 名称后面，可以用逗号分割来附带一些额外的配置；名称可以留空，以保留默认的键，同时附带配置。
    1. 配置`omitempty`时，则当字段为空值（零值）的时候忽略这个字段。
    1. 如果名称指定为`-`，则总是忽略这个字段。（注意如果想让键名就是`-`，则要写`-,`）
    1. 除了`omitempty`之外还有个`string`选项，它会把相应的值以string的形式转化，这只对数字和布尔类型生效。

```go
// 举一些例子：
type Data struct {
    Field1 string `json:"field1"` // 指定json对象中的键名
    Field2 string `json:"field2,omitempty"`  // 当这个字段是零值时不将其写入json对象中
    Field3 string `json:",omitempty"` // 使用默认的键名，并且当这个字段是零值时不将其写入json对象中
    Field4 string `json:"-"` // 总是忽略这个字段
    Field5 string `json:"-,"`  // 将这个字段在json对象中的键名指定为减号，像这样{"-":"value"}
    Field6 int64  `json:",string"` // 会转化为字符串形式，例如{"Field6":"888"}
}
```

6. 键名必须符合如下规则：只包含Unicode中的字母、数字 和 除了引号、反斜杠、逗号之外的ASCII标点符号。
7. 对于匿名字段：
    1. 如果没有给它指定tag，那么它其中的字段会平铺在父对象中。（译者注：因为Go里没有继承，只有联合，因此一个结构体包含若干个匿名结构体是很常见的，在这种情况下需要平铺，而不是作为子对象）
    1. 如果指定了tag，则视为一个子对象（译者注：显式指定tag才会视为子对象）
    1. 如果是interface类型，一律视为子对象
```go
// 情况一：没有tag的匿名字段
func main() {
    j, _ := json.Marshal(&Blog{
        Content: Content{Text: "Hello!"},
    })
    fmt.Println(string(j)) // 得到{"Text":"Hello!"}而不是{"Content":{"Text":"Hello!"}}
}

type Blog struct {
    Content
}
type Content struct {
    Text string
}
```

```go
// 情况三：interface类型
func main() {
    j, _ := json.Marshal(&Blog{
        Any: Content{Text: "Hello!"},
    })
    fmt.Println(string(j))  // 得到{"Any":{"Text":"Hello!"}}而不是{"Text":"Hello!"}
}

type Blog struct {
    Any
}
type Any interface {}
```

8. 键有冲突时，优先使用有tag指定的字段。
9. 键名必须是string或者int，或者实现了`encoding.TextMarshaler`接口。
10. 指针会被转换为其所指向的值。空指针转化为null。
11. channel, complex, 函数 不能被序列化，会返回错误。
12. 循环引用会返回错误。

## Marshal 源码

```go
func Marshal(v interface{}) ([]byte, error) {
    e := newEncodeState()

    err := e.marshal(v, encOpts{escapeHTML: true})
    if err != nil {
        return nil, err
    }
    buf := append([]byte(nil), e.Bytes()...)

    encodeStatePool.Put(e)

    return buf, nil
}
```

首先是取得一个编码上下文对象`e`，它里面用了`sync.Pool`来提升性能。

```go
type encodeState struct {
    bytes.Buffer // accumulated output
    scratch      [64]byte
    // 下面两个字段用来防止循环引用
    ptrLevel uint
    ptrSeen  map[interface{}]struct{}
}
```

然后在这个上下文上调用`marshal`方法，其中用opt显式地指定了HTML转义。

```go
func (e *encodeState) marshal(v interface{}, opts encOpts) (err error) {
    defer func() {
        if r := recover(); r != nil {
            if je, ok := r.(jsonError); ok {
                err = je.error
            } else {
                panic(r)
            }
        }
    }()
    e.reflectValue(reflect.ValueOf(v), opts)
    return nil
}
```

这里有个小语法值得学习，是关于这个`jsonError`，它以一种类似继承的方式来包装error值。

```go
type jsonError struct{ error }
```

可是，`error`是一个接口啊，它放进一个结构体中时，它到底是什么东西？(TODO: 这个问题还真有点意思，值得另外找时间研究一下。)

这里建议你去读一下我翻译的官方关于反射包的博客文章：[The Laws of Reflection](https://lewinblog.com/blog/page/2020/201220-GoBlog-The-Laws-of-Reflection.md) 然后就会知道，这里的接口就是一个接口变量。对，当我们把一个任意类型装进一个接口中时，其实就生成了一个「接口变量」，这个变量里装的是 原始变量的引用+原始变量的类型。不过，正常情况下我们是不需要深究到这个程度的，只需要把它当成普通的`error`来使用就可以了。（小心nil）

### e.reflectValue

```go
func (e *encodeState) reflectValue(v reflect.Value, opts encOpts) {
    valueEncoder(v)(e, v, opts)
}

func valueEncoder(v reflect.Value) encoderFunc {
    if !v.IsValid() {
        return invalidValueEncoder
    }
    return typeEncoder(v.Type())
}
```

在这里，先根据类型来选择一个处理的函数(`encoderFunc`)，然后调用这个函数去处理。两个细节：

1. 为什么要先判断`IsValid`？——因为在`v`不合法的情况下调用`Type`会造成painc
2. 处理的结果去哪里了？——丢在`e`变量中先储存着了。

然后我们找一个处理函数来看一下：

```go
func invalidValueEncoder(e *encodeState, v reflect.Value, _ encOpts) {
    e.WriteString("null")
}
```

对于“无效值”的处理很简单，直接写一个null就行了。写到哪里去了呢？回顾一下`e`的类型，内嵌了一个`bytes.Buffer`，因此这个`WriteString`方法其实就是`bytes.Buffer.WriteString`：

```go
func (b *Buffer) WriteString(s string) (n int, err error) {
    b.lastRead = opInvalid
    m, ok := b.tryGrowByReslice(len(s))
    if !ok {
        m = b.grow(len(s))
    }
    return copy(b.buf[m:], s), nil
}
```

看到这里好像还没有看到任何的递归调用，那json包是如何处理多级结构体的呢？我们再看一个处理函数：

```go
func typeEncoder(t reflect.Type) encoderFunc {
    // 1. 由于类型可能很复杂，导致生成一个闭包函数代价很大，因此用缓存保护起来
    if fi, ok := encoderCache.Load(t); ok {
        return fi.(encoderFunc)
    }
    
    var (
        wg sync.WaitGroup
        f  encoderFunc
    )
    wg.Add(1)
    // 2. 把新的函数先注册到缓存中。用一个Waitgroup来保护生成它的期间。
    fi, loaded := encoderCache.LoadOrStore(t, encoderFunc(func(e *encodeState, v reflect.Value, opts encOpts) {
        wg.Wait()
        f(e, v, opts)
    }))
    if loaded {
        return fi.(encoderFunc)
    }

    // 3. 构造出这个闭包（即对应某个类型的处理函数）
    f = newTypeEncoder(t, true)
    wg.Done()
    // 4. 构造完成后，在缓存中重新注册一次，去掉wg
    encoderCache.Store(t, f)
    return f
}
```

注意看上面的注释，都是我自己写的。这一段代码展现的对于并发数据保护的思路，令我耳目一新。

其中，`encoderCache`是一个`sync.Map`，这保证了json包是并发安全的。它里面保存的都是处理函数，每个函数针对一个特定类型（`reflect.Value.Type()`返回的表层的静态类型，而非底层真实类型）

> 思考题1：如果用`Value.Kind()`而不是`Value.Type()`，会发生什么？有必要吗？

> 思考题2：在上述代码的第1步到第2步之间，是否会出现并发竞争问题？它是如何保证安全的？

### newTypeEncoder

好，终于进入了关键环节：针对类型的处理函数是如何构造出来的？

```go
func newTypeEncoder(t reflect.Type, allowAddr bool) encoderFunc {
    // 1. 先检查有没有实现特定的接口，如果有的话就直接使用
    //   - 这里分别检查 是指针/非指针 的情况
    //   - 这里分别检查 Marshaler 和 TextMarshaler 这两个接口
    if t.Kind() != reflect.Ptr && allowAddr && reflect.PtrTo(t).Implements(marshalerType) {
        return newCondAddrEncoder(addrMarshalerEncoder, newTypeEncoder(t, false))
    }
    if t.Implements(marshalerType) {
        return marshalerEncoder
    }
    if t.Kind() != reflect.Ptr && allowAddr && reflect.PtrTo(t).Implements(textMarshalerType) {
        return newCondAddrEncoder(addrTextMarshalerEncoder, newTypeEncoder(t, false))
    }
    if t.Implements(textMarshalerType) {
        return textMarshalerEncoder
    }

    // 2. 如果没有用户自定义的序列化函数，则检查是否是一些预置的数据类型
    //   - 注意哦，这里用的是Kind()方法哦
    switch t.Kind() {
    case reflect.Bool:
        return boolEncoder
    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
        return intEncoder
    case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
        return uintEncoder
    case reflect.Float32:
        return float32Encoder
    case reflect.Float64:
        return float64Encoder
    case reflect.String:
        return stringEncoder
    case reflect.Interface:
        return interfaceEncoder
    case reflect.Struct:
        return newStructEncoder(t)
    case reflect.Map:
        return newMapEncoder(t)
    case reflect.Slice:
        return newSliceEncoder(t)
    case reflect.Array:
        return newArrayEncoder(t)
    case reflect.Ptr:
        return newPtrEncoder(t)
    // 3. 不是的话就返回一个会返回错误的处理函数
    default:
        return unsupportedTypeEncoder
    }
}
```

### intEncoder

我们先看一个最简单的预置处理函数：

```go
func intEncoder(e *encodeState, v reflect.Value, opts encOpts) {
    b := strconv.AppendInt(e.scratch[:0], v.Int(), 10)
    if opts.quoted {
        e.WriteByte('"')
    }
    e.Write(b)
    if opts.quoted {
        e.WriteByte('"')
    }
}
```

其中，`e.scratch`是一个`[64]byte`，它的作用正如其名，就是一个小小的临时草稿纸。整个处理函数的流程是，先写一个引号，然后写十进制格式的数字，然后再写一个引号。

### newMapEncoder

然后再看一个稍微复杂一点点的预置处理函数：

```go
func newMapEncoder(t reflect.Type) encoderFunc {
    // 1. 检查 键 的类型
    switch t.Key().Kind() {
    case reflect.String,
        reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
        reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
    default:
        if !t.Key().Implements(textMarshalerType) {
            return unsupportedTypeEncoder
        }
    }
    // 2. 查询 值 的类型对应的处理函数
    me := mapEncoder{typeEncoder(t.Elem())}
    return me.encode
}
```

上面的代码中，它首先检查一个map的键的类型，只支持字符串和数字，这个应当是由JSON协议所规定的。然后再根据map的值的类型查出相应的处理函数，包装起来。

继续看map的处理函数，它把Elem的类型包装了一下，返回了encode方法，这个函数其实也就是做了一些关键字符的补充工作。

```go
func (me mapEncoder) encode(e *encodeState, v reflect.Value, opts encOpts) {
    if v.IsNil() {
        e.WriteString("null")
        return
    }
    e.WriteByte('{')

    // Extract and sort the keys.
    keys := v.MapKeys()
    sv := make([]reflectWithString, len(keys))
    for i, v := range keys {
        sv[i].v = v
        if err := sv[i].resolve(); err != nil {
            e.error(fmt.Errorf("json: encoding error for type %q: %q", v.Type().String(), err.Error()))
        }
    }
    sort.Slice(sv, func(i, j int) bool { return sv[i].s < sv[j].s })

    for i, kv := range sv {
        if i > 0 {
            e.WriteByte(',')
        }
        e.string(kv.s, opts.escapeHTML)
        e.WriteByte(':')
        me.elemEnc(e, v.MapIndex(kv.v), opts)
    }
    e.WriteByte('}')
}
```

### newStructEncoder

然后我们再看struct的处理函数：

```go
func newStructEncoder(t reflect.Type) encoderFunc {
    se := structEncoder{fields: cachedTypeFields(t)}
    return se.encode
}

func cachedTypeFields(t reflect.Type) structFields {
    if f, ok := fieldCache.Load(t); ok {
        return f.(structFields)
    }
    f, _ := fieldCache.LoadOrStore(t, typeFields(t))
    return f.(structFields)
}
```

主要思路是，把这个结构体的所有字段都拿出来分析一遍，然后做个缓存。

值得一提的是，在分析过程中，通过`Type.Tag.Get("json")`来把字段对应的tag取出来。

好了，到此为止，已经完全可以想象到其他数据类型的序列化处理方式了，不再一一细说。

接下来看看更复杂的：数据的反序列化。

## 反序列化： UnMarshal

基本用法：

```go
func main() {
    data := []byte(`{"name":"name123","Content":"content123"}`)

    var blog Blog
    json.Unmarshal(data, &blog)  // 注意要传入指针，而且是非空指针
    fmt.Println(blog)
}
```

`Unmarshal`这个函数依然是有一大串的注释：

- 传入空指针或者非指针的话，会返回错误；
- 它的过程与Marshal相反；它会分配map, slice, 指针 等，按照以下规律：
    + 首先检查 JSON null，如果是则把指针设为nil；如果JSON有数据，则将其填入指针所指向的数据内存中；如果指针为空，则会new一个。
    + 先检查`Unmarshaler`（包括JSON null），然后如果JSON字段是字符串，则检查`TextUnmarshaler`。
- 反序列化过程中，先检查JSON的键。如果struct中没有对应的字段，则默认情况下会忽略。
- 数据转换关系：
    + JSON boolean -> bool
    + JSON numbers -> float64
    + JSON string -> string
    + JSON array -> []interface{}
    + JSON objects -> map[string]interface{}
    + JSON null -> nil
- array转化为切片：先将切片长度设置为0，然后逐个append进去；
- array转化为数组：多出的会被抛弃，不足的会被设为零值；
- object转为map：如果map是nil则会创建一个，如果有旧的map则用旧的map；键的类型必须是string, integer或者实现了`json.Unmarshaler`或`encoding.TextUnmarshaler`；
- 如果一个JSON的值与目标类型不匹配，或者number超出了范围，则会跳过这个值，并继续尽可能地完成剩下的部分。如果后续没有更严重的错误，则会返回`UnmarshalTypeError`来描述遇到的第一个不匹配的类型。注意，如果出现类型不匹配的情况，那么不保证后续字段都会正常工作。
- 当解析字符串的值的时候，无效的 utf-8 或者 utf-16 字符不会被视为一种错误；这些无效字符会被替换为 替换字符 U+FFFD

> 关于Unicode特殊字符，请参考 [Wiki](https://en.wikipedia.org/wiki/Specials_(Unicode_block))

然后看一下它的代码：

```go
func Unmarshal(data []byte, v interface{}) error {
    // 0. 一个上下文对象
    var d decodeState
    // 1. 检查JSON是否有语法错误；避免待会反序列化到一半的时候才发现。
    err := checkValid(data, &d.scan)
    if err != nil {
        return err
    }
    // 2. 执行反序列化
    d.init(data)
    return d.unmarshal(v)
}
```

### checkValid

```go
func checkValid(data []byte, scan *scanner) error {
    scan.reset()
    for _, c := range data {
        scan.bytes++
        if scan.step(scan, c) == scanError {
            return scan.err
        }
    }
    if scan.eof() == scanError {
        return scan.err
    }
    return nil
}
```

这个对JSON语法进行校验的函数，看起来有点原始啊……居然是逐个字节处理的……

简单说，就是在给定的[]byte上进行遍历，遍历完了检查是否可以结束了，如果没问题那就没问题。

继续看，`scan.step`是一个函数，它是在`scan.reset()`这个步骤时传入的一个函数，它长这样：

```go
func stateBeginValue(s *scanner, c byte) int {
    if isSpace(c) {
        return scanSkipSpace
    }
    switch c {
    case '{':
        s.step = stateBeginStringOrEmpty
        return s.pushParseState(c, parseObjectKey, scanBeginObject)
    case '[':
        s.step = stateBeginValueOrEmpty
        return s.pushParseState(c, parseArrayValue, scanBeginArray)
    case '"':
        s.step = stateInString
        return scanBeginLiteral
    case '-':
        s.step = stateNeg
        return scanBeginLiteral
    case '0': // beginning of 0.123
        s.step = state0
        return scanBeginLiteral
    case 't': // beginning of true
        s.step = stateT
        return scanBeginLiteral
    case 'f': // beginning of false
        s.step = stateF
        return scanBeginLiteral
    case 'n': // beginning of null
        s.step = stateN
        return scanBeginLiteral
    }
    if '1' <= c && c <= '9' { // beginning of 1234.5
        s.step = state1
        return scanBeginLiteral
    }
    return s.error(c, "looking for beginning of value")
}
```

哦~~这里有点意思。虽然是逐个字符进行检查，但是每当分析到一个字符时，都会进行相应的判断，然后替换掉`scan.step`这个函数。（这就是为什么需要`scan.reset()`。

挑一个情况来看一下，比如发现第一个字符是`{`时：

```go
func stateBeginStringOrEmpty(s *scanner, c byte) int {
    if isSpace(c) {
        return scanSkipSpace
    }
    if c == '}' {
        n := len(s.parseState)
        s.parseState[n-1] = parseObjectValue
        return stateEndValue(s, c)
    }
    return stateBeginString(s, c)
}
```

那么，在`{`后面的字符，则进行这样的判断：如果是空格则跳过；如果是`}`则在`parseState`里记录一次Object对象的结束；否则视为进入一个子对象，重新用`stateBeginString`这个初始函数来进行下一步的处理。

这个`parseState`是一个`[]int`，它是一个栈，每进入到一级子对象中时就会入栈一个。栈中的值有三个常数：

```go
const (
    parseObjectKey   = iota // parsing object key (before colon)
    parseObjectValue        // parsing object value (after colon)
    parseArrayValue         // parsing array value
)
```

在进入一个子对象时，会push一个常数进去：

```go
func (s *scanner) pushParseState(c byte, newParseState int, successState int) int {
    s.parseState = append(s.parseState, newParseState)
    if len(s.parseState) <= maxNestingDepth {
        return successState
    }
    return s.error(c, "exceeded max depth")
}
```

然后在退出一个子对象时（即上面说的检查到`}`字符的时候）则会把最后一个值弹出。

越说越糊涂，让我们找一个实际的例子来看一下。

```go
func main() {
    var blog Blog
    json.Unmarshal([]byte(`{"n":"12"}`), &blog)
}

type Blog struct {
    Name string `json:"n"`
}
```

1. `{`
    + step替换为`stateBeginStringOrEmpty`
    + parseState中推入了一个`parseObjectKey`
2. `"`
    + step替换为`stateInString`
3. `n`
    + （继续）
4. `"`
    + step替换为`stateEndValue`
5. `:`
    + `parseState[0]`替换为`parseObjectValue`
    + step替换为`stateBeginValue`
6. `"` `1` `2` `"`
    + （跟key是相同的一个循环）
    + step替换为`stateEndValue`
7. `}`
    + parseState取出一个(`.popParseState()`)，然后发现它空了，因此标记`s.endTop = true`，此时可以`.eof()`

我们知道它的大概原理就可以了，不深究细节。

### unmarshal

通过校验后，我们确保拿到一个合法的JSON字符串。接下来就考虑如何把它转化为Go的数据结构。

首先起手依然是祖传技艺reflect：

```go
func (d *decodeState) unmarshal(v interface{}) error {
    rv := reflect.ValueOf(v)
    if rv.Kind() != reflect.Ptr || rv.IsNil() {
        return &InvalidUnmarshalError{reflect.TypeOf(v)}
    }

    d.scan.reset()
    d.scanWhile(scanSkipSpace)
    // We decode rv not rv.Elem because the Unmarshaler interface
    // test must be applied at the top level of the value.
    err := d.value(rv)
    if err != nil {
        return d.addErrorContext(err)
    }
    return d.savedError
}
```

最关键的部分是这个`value()`方法：

```go
func (d *decodeState) value(v reflect.Value) error {
    switch d.opcode {
    default:
        panic(phasePanicMsg)

    case scanBeginArray:
        if v.IsValid() {
            if err := d.array(v); err != nil {
                return err
            }
        } else {
            d.skip()
        }
        d.scanNext()

    case scanBeginObject:
        if v.IsValid() {
            if err := d.object(v); err != nil {
                return err
            }
        } else {
            d.skip()
        }
        d.scanNext()

    case scanBeginLiteral:
        // All bytes inside literal return scanContinue op code.
        start := d.readIndex()
        d.rescanLiteral()

        if v.IsValid() {
            if err := d.literalStore(d.data[start:d.readIndex()], v, false); err != nil {
                return err
            }
        }
    }
    return nil
}
```

最典型的case是`scanBeginObject`，调用`.object()`方法：

```go
func (d *decodeState) object(v reflect.Value) error {
    // Check for unmarshaler.
    u, ut, pv := indirect(v, false)
    // ......
}
```

上面这个方法实在是太可怕了。它第一步是递归遍历整个结构体，给指针变量全部初始化一遍；如果遇到了`json.Unmarshaler`或`encoding.TextUnmarshaler`则会返回，分别对应`u`和`ut`；在我们的例子中，并没有自定义反序列化方法，因此它们跳过。

然后检查是map还是struct：

```go
func (d *decodeState) object(v reflect.Value) error {
    // ......
    switch v.Kind() {
    case reflect.Map:
        // ...
    case reflect.Struct:
        fields = cachedTypeFields(t)
    default:
        d.saveError(&UnmarshalTypeError{Value: "object", Type: t, Offset: int64(d.off)})
        d.skip()
        return nil
    }
    // ......
}
```

然后是一个巨大的循环，一个一个地将JSON字符串中的键和值取出来：（以下代码仅展示逻辑主干）

```go
func (d *decodeState) object(v reflect.Value) error {
    // ......
    for {
        // 1. 扫描一个键
        d.scanWhile(scanSkipSpace)
        item := d.data[start:d.readIndex()]

        if v.Kind() == reflect.Map {
            // ...
        } else {
            // 2. 找到这个键对应的结构体字段
            var f *field = &fields.list[i]
            d.errorContext.FieldStack = append(d.errorContext.FieldStack, f.name)
        }

        // 3. 把值给取出来
        if err := d.value(subv); err != nil {
            return err
        }
    // ......
}
```

在`.value()`函数中将值取出来：（以下代码仅展示逻辑主干）

```go
func (d *decodeState) value(v reflect.Value) error {
    switch d.opcode {
    case scanBeginLiteral:
        // 扫描这个值所在bytes的区间
        start := d.readIndex()
        d.rescanLiteral()

        if v.IsValid() {
            // 把值装进给定的Value里
            if err := d.literalStore(d.data[start:d.readIndex()], v, false); err != nil {
                return err
            }
        }
    }
    return nil
}
```

然后接下来的`literalStore`又是一个非常恐怖的方法……在上面的例子中，我们反序列化的是一个string字段，因此会执行到这里：（以下代码仅展示逻辑主干）

```go
func (d *decodeState) literalStore(item []byte, v reflect.Value, fromQuoted bool) error {
    // 1. 初始化指针
    u, ut, pv := indirect(v, isNull)
    // ... 
    // 2. 去除引号，此时 s 变量中储存的就是即将存入字段中的值了
    s, ok := unquoteBytes(item)
    switch v.Kind() {
    case reflect.String:
        // 3. 存入值
        v.SetString(string(s))
    }
    // ...
}
```

至此，我们分别取得了键和值，并将值存入了指定的数据结构变量中。

虽然我这里展示的只是一个最简单的例子，但是相信其基本结构已经展示清楚了。剩下的分支判断条件里，都是相似的逻辑，不再深究了。

## 小结

早就听说（并且自己也猜到）`json`包会是很重的一个包。它既在性能上重（大量使用反射），也在源码实现上重（`encode.go`, `decode.go`分别都有超过一千行代码），的确是非常恐怖。

但是，学习它的目的也不是为了自己撸一套序列化/反序列化的实现，主要目的还是在于了解其大致的内部运行规律。从结果来看，只要掌握了反射包的用法，就算要完整学习json包的实现也不会是不可能的事情。

还有一个目的，是为接下来学习 proto buf 做铺垫。
