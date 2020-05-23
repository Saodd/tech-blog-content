---
title: 'JS Promise 队列实现限流'
description: '前端基本操作'
pubDate: '2021-07-22'
updateDate: '2021-07-22'
tags: ['前端']
---

然后我们把原来的`apiCall`这个业务请求重新包装一下，返回一个新的同样泛型的Promise（这样调用方才不会感知到底层的变化）：

```typescript
function throttledApiCall(params: any): Promise<any> {
  return new Promise<any>((resolve, reject) => {
    queue.push(() => {
      // todo: 闭包内封装真实请求逻辑
    });
    // todo: 在任务推入队列之后，决定是否要触发
  });
}
```

其实，就写出上面的代码结构，就已经需要想到后续的逻辑是怎么安排才行。主要逻辑如下：

1. 将任务闭包推入队列。
2. 任务完成后，要触发下一个任务。
3. 如果队列只有1个任务，（即这个任务是第一个）那么将没有前置任务去触发它，所以我们要额外判断并触发一次。

```typescript
function throttledApiCall(params: any): Promise<any> {
  return new Promise<any>((resolve, reject) => {
    queue.push(() => {
      // 1. 将任务闭包推入队列。
      apiCall(params)
        .then(resolve)
        .catch(reject)
        .finally(() => {
          queue.shift();
          // 2. 任务完成后，要触发下一个任务。
          if (queue.length) queue[0]();
        });
    });
    // 如果队列只有1个任务，额外触发一次
    if (queue.length === 1) queue[0]();
  });
}
```

这样就实现了限流逻辑。

可以把上述代码运行一下，任意折腾，例如加入随机异常，例如随机添加队列任务，看看是否能正常运行。

## 进阶版：限制n个请求并发

在上面的实现基础上稍作改造。

首先改变一下思路，这次不把所有任务放入队列了，而是只把等待中的任务放入队列。每当一个任务完成之后，尝试从队列中取出一个任务继续执行。

然后我们需要一个计数器，一个简单的number变量即可，以及一个常量来限制计数器的最大值。

```typescript
let queue: (() => void)[] = [];
let count = 0;
const countLimit = 3;

function throttledApiCall(params: any): Promise<any> {
  return new Promise<any>((resolve, reject) => {
    const task = () => {
      apiCall(params)
        .then(resolve)
        .catch(reject)
        .finally(() => {
          if (queue.length) {
            queue.shift()();
          } else {
            count--;
          }
        });
    };
    if (count < countLimit) {
      count++;
      task();
    } else {
      queue.push(task);
    }
  });
}
```
