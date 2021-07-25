```yaml lw-blog-meta
title: "JS Promise 队列实现限流"
date: "2021-07-22"
brev: "前端基本操作"
tags: ["前端"]
```

## 背景

「用队列」只是实现方式之一。也是比较简单直观的一种。

但是思路稍微有点绕，让我想了好一会儿，所以简单记录一下。代码可以参考 [github](https://github.com/Saodd/learn-webpack/commit/8d30f025bf9f76ffada40fa29875a53c8be8d30c)

## 需求

我们可能有一个后端接口调用，这里用一个200ms延迟的Promise来假装一下：

```typescript
async function apiCall(params: any): Promise<any> {
    console.log("HTTP 请求……")
    await new Promise(r => setTimeout(r, 500))
    return {data: "123"}
}
```

有时候在一个页面上一瞬间要大量访问这个接口。

一个典型场景是列表页，列表中每个元素都要独立地调用这个接口，而且为了代码复用，我们不希望在这些元素的上层来手动地实现限流，而是直接在接口定义处就直接实现，使用处只管简单无脑地使用就好了。

写一个循环来模拟短时间内的大量接口调用：

```typescript
for (let i = 0; i < 20; i++) {
    apiCall({}).then(console.log)
}
```

此时直接运行，是没有任何限流效果的，所有请求都并发执行了。

## 简单版：每次仅限1个请求执行

为了让上一个请求A执行完毕之后能顺利触发下一个请求B，可能有如下思路：

思路一：A中保存B的信息。这显然不行，因为A创建的时候，B还不知道在哪里呢。

思路二：B中保存A的信息。变形一下，其实只要全局保存最后一个请求的信息，或者做成一个链表的形式。

对于思路二，因为`await`一个已经resolve的Promise时，可以立即返回，因此无论A和B之间间隔多久都可以实现，这个思路是可行的。但是在这里不展开讲。

思路三：保存一个全局的先进先出队列，在JS中只要普通的数组就可以实现。

由于此时我们限制并发数量只有1，因此每次完成的请求都必然是队列头部第一个任务，只需要无脑`.shift()`即可实现。

首先定义一个全局队列。这里要注意，因为`Promise`在实例化的时候就立即执行了，所以要把 new 的方法装在一个闭包里，把这个闭包作为任务放入队列

```typescript
let queue: (() => void)[] = []
```

然后我们把原来的`apiCall`这个业务请求重新包装一下，返回一个新的同样泛型的Promise（这样调用方才不会感知到底层的变化）：

```typescript
function throttledApiCall(params: any): Promise<any> {
    return new Promise<any>((resolve, reject) => {
        queue.push(() => {
            // todo: 闭包内封装真实请求逻辑
        })
        // todo: 在任务推入队列之后，决定是否要触发
    })
}
```

其实，就写出上面的代码结构，就已经需要想到后续的逻辑是怎么安排才行。主要逻辑如下：

1. 将任务闭包推入队列。
2. 任务完成后，要触发下一个任务。
3. 如果队列只有1个任务，（即这个任务是第一个）那么将没有前置任务去触发它，所以我们要额外判断并触发一次。

```typescript
function throttledApiCall(params: any): Promise<any> {
    return new Promise<any>((resolve, reject) => {
        queue.push(() => {  // 1. 将任务闭包推入队列。
            apiCall(params)
                .then(resolve)
                .catch(reject)
                .finally(() => {
                    queue.shift()
                    // 2. 任务完成后，要触发下一个任务。
                    if (queue.length) queue[0]()
                })
        })
        // 如果队列只有1个任务，额外触发一次
        if (queue.length === 1) queue[0]()
    })
}
```

这样就实现了限流逻辑。

可以把上述代码运行一下，任意折腾，例如加入随机异常，例如随机添加队列任务，看看是否能正常运行。

## 进阶版：限制n个请求并发

在上面的实现基础上稍作改造。

首先改变一下思路，这次不把所有任务放入队列了，而是只把等待中的任务放入队列。每当一个任务完成之后，尝试从队列中取出一个任务继续执行。

然后我们需要一个计数器，一个简单的number变量即可，以及一个常量来限制计数器的最大值。

```typescript
let queue: (() => void)[] = []
let count = 0
const countLimit = 3

function throttledApiCall(params: any): Promise<any> {
    return new Promise<any>((resolve, reject) => {
        const task = () => {
            apiCall(params)
                .then(resolve)
                .catch(reject)
                .finally(() => {
                    if (queue.length) {
                        queue.shift()()
                    } else {
                        count--
                    }
                })
        }
        if (count < countLimit) {
            count++
            task()
        } else {
            queue.push(task)
        }
    })
}
```
