```yaml lw-blog-meta
title: "gevent入坑体验"
date: "2021-01-07"
brev: 原来这个协程框架这么方便啊。
tags: [Python]
```

## 前言

最近写了点 Node ，感觉它的异步流程写得还是挺舒服的。虽然与 Golang 比起来，在异步流程方面还是稍有一些些小麻烦，但总体体验良好。

然后再切换回 Python ，我想到那一套 eventloop 的东西就感觉有点脑壳疼，甚至想过是不是放弃性能，直接用多线程来实现呢？

还好同事给我指了一条明路： gevent . 

## 关于协程

在 Golang 中我们知道有 `goroutine` 这个东西。而在传统C系语言中，协程的单词是 `coroutine`。

这个单词应该作何解释？——`co`词头代表「协作」的意思，`routine`的原意是「日程，例行程序」，两者合在一起，就翻译成了「协程」。

那么为什么 Golang 的协程要独树一帜给自己起一个新名字呢？——因为它的原理的确稍有不同。

传统的`coroutine`一般是基于事件循环的异步IO机制，而`goroutine`是基于用户态的线程调度。这里不展开讲，可以参考 [官方博客](https://golang.org/doc/faq#goroutines). 

与在Node中稍有不同的是，在Python中（以及其他后端语言中）使用协程，我们需要主动地把主线程卡在事件循环上，否则main跑完了就直接退出了。

## 为什么不用标准的 async/await

关于这一块不详细讲，详情可以参考廖雪峰的教程。

Python3.4开始引入了`asyncio`标准库，然后在Python3.5引入了`async/await`关键字。

嗯，理想是挺美好的。

但是我们有大量的依赖库，比如数据库客户端，都是同步版本的实现。

因此，在不改变这些基础库的情况下，一个合理的选择就是 gevent 了。

> 还有一个问题，asyncio这套语法把loop这种底层的东西暴露出来了，而且很多接口（至少在我看来）挺反直觉的，所以我对它一直都是排斥态度。

## gevent

它好在哪里呢？

只需要一个`monkey.patch_all()`，然后几乎不需要改原来的代码，就可以无痛地从同步改造成异步。

我们先看一段最简单的代码，展示它的基本用法：

```python
import gevent

def work(name):
    gevent.sleep(1)
    print(name, 1)

if __name__ == '__main__':
    g = gevent.spawn(work, "Jack")
    g.join()
```

上述代码看起来就是同步的写法，而且代码风格与 Node.js 非常相似，这让我很满意。

接下来展示一下魔法，看一下它是如何与同步版本的第三方库结合，这里我们选用redis：

```python
import gevent
from gevent import monkey
import redis

monkey.patch_all()  # 猴子补丁

def work(name):
    """从Redis队列中取一个值，5秒超时。完全的同步语法。"""
    client = redis.Redis(host="10.0.6.239")
    value = client.brpop("list1", 5)
    print(name, value)

if __name__ == '__main__':
    """启动并等待4个协程并发"""
    gevent.joinall([gevent.spawn(work, i) for i in range(4)])

```

有一个术语叫做`monkey patch`，中文一般直译为「猴子补丁」。原理不细讲了，总之就是利用import的机制。

这里通过猴子补丁的方式，把原来Python中内置的阻塞IO调用 改为了 异步版本的。所以我们不需要改变原先同步版本的redis库的用法。

## 进阶：协程变量共享与检测重启

在协程模式下，与我们所认知的Node.js的原理相同，都是单线程运行的，因此可以放心的在不同的协程之间共享、读、写、变量，而且不需要加锁。

一个典型的IO密集型工作场景是，我需要保持很多个（一定数量地）协程作为Worker，不断地进行IO操作。

这里有个小问题是异常处理，当部分协程Worker遭遇异常退出时，如果我们用的是`joinall()`方法，那就会导致剩下的Worker数量比我们预期的少，导致执行效率降低。所以需要一套检测并且重启的方法。

> 当然，直接在协程根部加一个大try也可以保证协程不崩溃。这里我主要讲解一下变量共享。

```python
bucket = 4  # 名额限制，最大4个worker

def work_wrapper():
    global bucket
    try:
        work()
    except:
        pass
    finally:
        bucket += 1  # worker下班时交还名额

if __name__ == '__main__':
    """启动并等待4个协程并发"""
    while True:
        if bucket > 0:
            bucket -= 1  # 创建worker时领取一个名额
            gevent.spawn(work_wrapper)
        else:
            gevent.sleep(1)
```

这里用到一个「令牌桶」的概念（当然灵感来源于Golang）。主线程不断地检查是否还有剩余的令牌，如果有，则新建一个worker；如果没有，则异步沉睡1秒。

这里的令牌桶没有加锁，因为gevent的协程（即传统的coroutine事件循环模式）不会被抢断，所以变量写入是安全的。

## 小结

其实这篇文章没有什么新鲜东西。只不过是因为gevent确实给了我惊喜，所以写点简单用法，宣传一下。
