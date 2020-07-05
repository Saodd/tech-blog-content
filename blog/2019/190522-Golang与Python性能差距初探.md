```yaml lw-blog-meta
title: Golang与Python性能差距初探
date: "2019-05-22"
brev: 在学习的时候的确能感受到Golang运行时的清爽感，但是到底有多强还没有定量地分析一下。之前看过大佬的帖子，大概的性能差距应该是10-50倍这个数量级吧
tags: [Golang, Python]
```


## Golang程序

今天简单地写了个基本的排序算法，试一试他们之间差距到底有多大。

```golang
func sort_Select(li []int) {
    minindex := 0 // 健壮性先不考虑
    length := len(li)
    var i, ii int
    for i = range li {
        minindex = i
        // 在i之后的剩余部分中寻找最小值对应的位置
        for ii = i + 1; ii < length; ii++ {
            if li[ii] < li[minindex] {
                minindex = ii
            }
        }
        // 遍历完成后把最小值交换到i的位置
        li[i], li[minindex] = li[minindex], li[i]
    }
}
```

吐槽一下，pycharm里的MD居然没有golang语法支持……

## python程序

```python
def Sort_Select(li:list):
    minindex = 0
    length = len(li)
    i, ii = 0,0
    for i in range(length):
        minindex = i
        for ii in range(i+1, length):
            if li[ii] < li[minindex]:
                minindex = ii
        li[i], li[minindex] = li[minindex], li[i]
```

二者几乎是同样的语法结构。都尽力用了缓存。

## 结论

 | 条件   | 成绩 |
 | ---    | ---   |
 | Python 1万  | 2.5秒 |
 | Python 5万  | 84.3秒,22.8M内存 |
 | Golang 10万  | 5.4秒 |
 | Golang 20万  | 22.6秒,2.6M内存 |

根据选择排序算法的平方关系计算，`python`跑20万条需要1348秒，是`golang`的**59.68倍**。这个倍数与我之前看过的帖子里研究的差不多。
emmmm…………

当然，我们使用Python一般就别要求他的性能了，一般用在快速开发，或者运维，定时任务，小工具之类的，真的非常趁手，写的很爽。  
在学习`golang`的过程中也重新复习了很多更接近底层的东西，受益良多。  
要说`golang`目前给我最深刻的印象就是它的多线程了吧，也就是`goruntine`，开线程真的快，而且是真线程
（学了python让我一度认为一个`进程`只能占用一个`CPU`哈哈哈哭了哭了）  

另一方面，也不是说`python`真的就比`golang`慢60倍了，
因为我这次简单的测试只能反映他们的简单循环速度和数组访问速度（以及栈的切换速度），
在实际工程中，我们一般不太可能会写这样的代码，一般都是引用了第三方库。
而第三方库如果是对性能有要求的，一般都会用`C`写（比如Pandas），
所以我认为他们在实际工程中的性能差距并不会差到几十倍的数量级。
