```json lw-blog-meta
{"title":"Python-GC垃圾回收机制","date":"2019-07-19","brev":"之前一直认为像Python这样的“高级脚本语言”没必要考虑内存回收机制。不过最近受到启发，在某些情况下利用（或禁用）这个特性，对性能会有不错的提升，所以也必须要了解一下。","tags":["Python"],"path":"blog/2019/190719-Python-GC垃圾回收机制.md"}
```



## 概述

> 参考自[外部1](https://www.jianshu.com/p/1e375fb40506)，
> [外部2](https://juejin.im/post/5b34b117f265da59a50b2fbe)，
> 混合了自己的分析和思考，侵删。

一句话来说：**python采用的是引用计数机制为主，标记-清除和分代收集两种机制为辅的策略**。

## 引用计数GC

其实也好理解，在每次申请内存的时候，在上面加一层壳（类似装饰器,getattr()之类的感觉）：

```C
 typedef struct_object {
 int ob_refcnt;
 struct_typeobject *ob_type;
} PyObject;
```

只要一个变量用于储存数字，加一个指针指向真实内存地址，以实现引用计数的功能。

对于这个`PyObject`，官方文档是这样描述的：

> All object types are extensions of this type. This is a type which contains the information Python needs to treat a pointer to an object as an object. In a normal “release” build, it contains only the object’s reference count and a pointer to the corresponding type object. Nothing is actually declared to be a PyObject, but every pointer to a Python object can be cast to a PyObject*. Access to the members must be done by using the macros Py_REFCNT and Py_TYPE.
>  
> 所有对象类型都是这个的拓展。Python需要它所包含的信息，来将**指向对象的指针**视为**对象**。
> 一般情况下它只包含一个计数器和一个指针。没有任何东西被声明为PyObject，但所有对象都可以转化为PyObject。
> 任何对成员（我理解为是对指向的内容）的访问，都必须经过Py_REFCNT和Py_TYPE这两个宏。

```C
// Py_TYPE(o)
(((PyObject*)(o))->ob_type)
// Py_REFCNT(o)
(((PyObject*)(o))->ob_refcnt)
```

逻辑很简单，当有新的引用（变量，属性，成员等）指向这个对象时，引用+1；
当引用移除（删除变量，函数返回，重新定义等）时，引用-1；
当引用为0时马上被GC干掉。

引用计数的优点：

1. 简单。
2. 实时性；因为立即回收，相当于是GC运行的时间平摊到了整个程序过程中。

缺点也很明显：

1. 中间加了一层壳，所以增加一点点时间/空间的消耗；
2. 维护计数，增加了时间消耗；
3. 无法应对循环引用。

### 循环引用问题

循环引用一般发生在容器对象中。比如：

```python
a , b = [1,], [1,]
a.append(b)
b.append(a)
```
或者看图。我们先创建两个容器对象：

![循环引用1](https://saodd.github.io/tech-blog-pic/2019/2019-07-19-Circular-Ref-1.webp)

然后互相引用：

![循环引用2](https://saodd.github.io/tech-blog-pic/2019/2019-07-19-Circular-Ref-2.webp)

这种情况下，它们的引用计数都是2；我们分别删除它们之后，它们里面还有互相引用，这样它们的计数永远不会下降到0了，也就永远不会被销毁了。

我们来建一个容器，跑个循环引用造成的内存泄露看一下：

```python
class A():
    pass

import gc

gc.disable()

while True:
    c1 = A()
    c2 = A()
    c1.t = c2  # 两行去掉后可以正常运行
    c2.t = c1  # 两行去掉后可以正常运行
    del c1
    del c2
```

```shell-session
PS > docker run --rm -m 100M -v C:/Users/lewin/mycode/APMOS:/scripts/APMOS -it appython:1.02
root@d67209491314:/scripts# python lewin.py
Killed
root@d67209491314:/scripts#
```

可以看到，没有GC的话秒秒钟就内存爆炸，被内核杀掉了。

## 标记-清除GC

在之前`PyObject`加壳（计数器，指针）的基础上，再加两个指针，让所有对象（容器对象）组成链表。

考虑之前的情况，分两种：

```python
a , b = [1,], [1,]
a.append(b)
b.append(a)
del a
del b
```

```python
a , b = [1,], [1,]
a.append(b)
b.append(a)
del a # 只删一个
```

执行`del`命令后，a和b中间是有循环引用的，这时候`标记-清除算法`就上场了。

`标记-清除算法`维护两个链表，一个是**root链表(root object)**，一个是**unreachable链表**。

情况1：

> 先找到其中一端a,开始拆这个a,b的引用环（我们从A出发，因为它有一个对B的引用，则将B的引用计数减1；然后顺着引用达到B，因为B有一个对A的引用，同样将A的引用减1，这样，就完成了循环引用对象间环摘除。），去掉以后发现，a,b循环引用变为了0，所以a,b就被处理到unreachable链表中直接被做掉。

情况2（只删a）：

> b取环后引用计数还为1，但是a取环，就为0了。这个时候a已经进入unreachable链表中，已经被判为死刑了，但是这个时候，root链表中有b。如果a被做掉，那世界上还有什么正义... ，在root链表中的b会被进行引用检测引用了a，如果a被做掉了，那么b就...凉凉，一审完事，二审a无罪，所以被拉到了root链表中。

这样就实现了循环引用的破解。

我们可以看到，这样操作很强大，但是也很麻烦，如果程序中变量多了，运行一次`标记-清除算法`将会非常费劲。
所以引入`GC阈值`机制。

## 标记-清除GC的优化之一：GC阈值

设`A=创建的对象数`，`B=销毁的对象数`，那么`A-B=我们正在使用的对象数+应该销毁而未被销毁的对象数`。
我们程序中使用的对象数量总是有限的，那么我们就可以设定一个`GC阈值`，当`A-B`的数值超过`GC阈值`的时候，
才运行`标记-清除算法`。

## 标记-清除GC的优化之二：分代收集

首先我们了解一下，一个基于经验的假说——**弱代假说**：

> 来看看代垃圾回收算法的核心行为：垃圾回收器会更频繁的处理新对象。一个新的对象即是你的程序刚刚创建的，而一个老的对象则是经过了几个时间周期之后仍然存在的对象。Python会在当一个对象从零代移动到一代，或是从一代移动到二代的过程中提升(promote)这个对象。

> 弱代假说由两个观点构成：首先是年轻的对象通常死得也快，而老对象则很有可能存活更长的时间。

怎么理解呢，我们设想一下，我们在一个循环中会有一些中间变量（年轻变量），它们基本上会在第一次垃圾收集中死去；
而对于一些全局变量、或者长途传递的函数参数，它们往往会经历多次垃圾收集而一直存活。

每熬过一次垃圾收集，对象等级就提升一级(promote)；升到2级就是老油条了，垃圾回收算法倾向于相信这些老油条们正是程序员所需的全局变量。所以每次达到`GC阈值`的时候，先对0级（初代）的新人下手；有机会再去找1级，实在不行了找2级老油条。

这正是我们需要的：临时变量尽快销毁，而对全局变量减少检查次数。

> 分代回收思想将对象分为三代（generation 0,1,2），0代表幼年对象，1代表青年对象，2代表老年对象。根据弱代假说（越年轻的对象越容易死掉，老的对象通常会存活更久。）
新生的对象被放入0代，如果该对象在第0代的一次gc垃圾回收中活了下来，那么它就被放到第1代里面（它就升级了）。如果第1代里面的对象在第1代的一次gc垃圾回收中活了下来，它就被放到第2代里面。gc.set_threshold(threshold0[,threshold1[,threshold2]])设置gc每一代垃圾回收所触发的阈值。从上一次第0代gc后，如果分配对象的个数减去释放对象的个数大于threshold0，那么就会对第0代中的对象进行gc垃圾回收检查。 从上一次第1代gc后，如过第0代被gc垃圾回收的次数大于threshold1，那么就会对第1代中的对象进行gc垃圾回收检查。同样，从上一次第2代gc后，如过第1代被gc垃圾回收的次数大于threshold2，那么就会对第2代中的对象进行gc垃圾回收检查。

## GC特性怎么用

GC很强大，但是很明显也会影响性能。
如果我们的应用对性能有很高的要求，我们可以选择手动控制GC的行为，像这样：

```python
import gc

gc.disable()

i=1
while i>0:
    c1 = A()
    c2 = A()
    c1.t = c2
    c2.t = c1
    del c1
    del c2
    if i==100:
        gc.collect() # 手动执行GC
        i=0
    i+=1
```

或者这样：

```python
import gc
gc.disable()
# ...核心代码
gc.enable()
```

或者简单的提高`GC阈值`：

```python
gc.set_threshold(threshold0, threshold1=None, threshold2=None)
```

当然，如果核心代码部分必须要考虑内存问题（确保不造成内存溢出问题）。

## 小结

学到新知识很开心，不过更令我遗憾的是：

**相比于埋头苦学，在一个更高的平台所带来的视野优势，真的会让你事半功倍。**

Golang的后发优势就是典型吧。所以既然已经落后了，更要付出加倍的努力，才能得到更高层次的认可。
