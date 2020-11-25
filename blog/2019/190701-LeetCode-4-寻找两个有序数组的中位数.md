```yaml lw-blog-meta
title: 'LeetCode[4]: 寻找两个有序数组的中位数'
date: "2019-07-01"
brev: ""
tags: [算法与数据结构]
```


## 原题

```text
给定两个大小为 m 和 n 的有序数组 nums1 和 nums2。

请你找出这两个有序数组的中位数，并且要求算法的时间复杂度为 O(log(m + n))。

你可以假设 nums1 和 nums2 不会同时为空。

示例 1:

nums1 = [1, 3]
nums2 = [2]

则中位数是 2.0

示例 2:

nums1 = [1, 2]
nums2 = [3, 4]

则中位数是 (2 + 3)/2 = 2.5

来源：力扣（LeetCode）
链接：https://leetcode-cn.com/problems/median-of-two-sorted-arrays
著作权归领扣网络所有。商业转载请联系官方授权，非商业转载请注明出处。

```

## 初步思路

其实如果不限定时间复杂度为`O(log(m + n))`的话，这题是非常容易的。

1. 我们类似于`二叉查找树`的思想，构建一棵树；
2. 遍历两个数组，逐个压入树中；
3. 始终保持根节点两侧的`子树`的`节点数`相等（或者相差1），即每次压入新数的时候，（与根节点对比后）
如果左边少就压入左边，如果右边少就压入右边。
4. 遍历完成后，如果是奇数的话（左右`子树`节点数相等）那根节点就是中位数，如果是偶数的话那就从节点数更多的`子树`中取出根节点求平均即可。

这个解法非常简单，而且非常直观。但是时间复杂度是O(m+n)，不符合题目要求。

仔细想一想的话，的确是做了一些无用功。因为两个数组本身已经是有序的了，把有序数组压入树中其实没有意义。

## 进一步思考

题目所要求的`O(log(m + n))`其实非常眼熟，再加上`有序`这个关键词，我们很容易可以想到`二分法`之类的思路。

但是要真的实现出来，会发现有很多坑：

1. 程序的逻辑必须要考虑数组长度的奇偶性，甚至对于数组`nums1`，`nums2` 以及二者长度之和`len(nums1)+len(nums2)`都要分别考虑，
这样的话情况就变得非常复杂；
2. 必须要处理边界值。我们很容易可以想到“在一个数组中取`n`个元素，另一个数组就必须是`(len(nums1)+len(nums2))/2-n`个元素”，
但是在实际中，这样计算数组下标很容易越界

## 参考答案

简要分析一下其中一个[推荐解答](https://leetcode-cn.com/problems/median-of-two-sorted-arrays/solution/4-xun-zhao-liang-ge-you-xu-shu-zu-de-zhong-wei-shu/)
的思路:

### 1. 把任意**奇偶长的数组**转换为**偶数长的数组**

如果能实现这个，那就解决了之前的问题1。

在这里是用了一个`容量翻倍法`（我自己起的名字）。即在原来的数组中每个元素的前面，
插入一个虚拟的空值（可以理解为#，null，nil等任意概念，因为不会用到这个值）。

这样对应原来数组的任意元素`old[m]`，都映射到新数组的`new[2m+1]`元素；

反之，对于任意新数组的元素`new[m]`，都能通过`old[m/2]`来获取真实值（因为0.5会被抛弃）

![容量翻倍法示意图](https://saodd.github.io/tech-blog-pic/2019/2019-07-01-Algo-DoubleList.png)

核心原理就是：
```text
对于任意奇数或者偶数，乘以2之后一定是偶数。
```


### 2. **割（Cut）**的概念

其实上面的`容量翻倍法`是必须要跟这个概念结合使用的。

我们知道，对任意数组求中位数的话，

```text
>>> 中位数的人类算法：

如果是奇数，那就找中间那个数；如果是偶数长度，那就中间两个数的平均值。
```

我们必须把这个算法抽象化，才能有利于我们的实现：

```text
>>> 中位数的人类算法（通用版）：

如果是奇数，那就把中间那个数变成两个数（劈开），求这个两个相同值的平均值（它本身）；如果是偶数长度，那就中间两个数的平均值。

例如：
对于[1,2,3]我们已知中位数是2；
我们把中间的2劈成两个，[1,2,2,3]，中位数依然是2.
```

再结合前面的`容量翻倍法`，继续抽象化这个算法：

```text
>>> 中位数的人类算法（终极版）：

把数组每个元素前面虚拟地填充一个空值，使得新的数组长度一定为偶数；
把新数组的割成两个长度相等的子数组，把左边最大值与右边最小值求平均即得结果。

例如：
对于old = [1,2,3]我们已知中位数是2；
我们把它填充成new = [#,1,#,2,#,3]，中间两个数（new[2]和new[3]）的平均数依然是2（对应于old[2/2]和old[3/2]）
```

### 3. 复习一下`二分法`的核心思路

```go
    var p int
    for lo <= hi {

        // do something ......

        if someCondition {
            hi = p
        } else {
            lo = p+1
        }
    }
```

### 4. 本题的解题思路

1. 我们先把(`nums1`, `nums2`)虚拟地扩容一倍，记作(`new1`，`new2`)（仅仅在概念上，而不真实操作）。
2. **目标**是把(`new1`, `new2`)分别分成两半，小的放在左边，大的放在右边；
用二分法循环，只要max(左边)<=min(右边)并且len(左边)==len(右边)就满足了条件，跳出循环。（注意，他们都是偶数长度的数组）

    ![条件示意图](https://saodd.github.io/tech-blog-pic/2019/2019-07-01-Algo-Conditions.png)

3. 从`new1`任意取一个下标`c1`(0 < c1 < 2*len(nums1)切开，放在左边，
那么必须从`new2`用下标`c2`切开(c2 = len(nums1)+len(nums2)-c1)，也放在左边。
（这样满足了len(左边)==len(右边)==(len(全部)/2)的条件）
4. 检查是否满足另一个条件（max(左边)<=min(右边)），如果满足就跳出循环。
5. 二分法循环步骤3-4

## 实现

### 1. 编写测试用例

首先要坚持`测试驱动开发`的理念，先从测试用例开始：（因为测试用例不多，所以没用`testing`框架）

```go
func Main0004() {
    var k [2][]int
    var v float64

    k, v = [2][]int{[]int{1, 3}, []int{2}}, 2
    fmt.Printf("Input: '%v', Answer: %v, Yours: %v \n", k, v, findMedianSortedArrays(k[0], k[1]))
    k, v = [2][]int{[]int{1, 2}, []int{3, 4}}, 2.5
    fmt.Printf("Input: '%v', Answer: %v, Yours: %v \n", k, v, findMedianSortedArrays(k[0], k[1]))
    k, v = [2][]int{[]int{1, 2, 3, 4}, []int{0, 5, 7, 9}}, 3.5
    fmt.Printf("Input: '%v', Answer: %v, Yours: %v \n", k, v, findMedianSortedArrays(k[0], k[1]))
    k, v = [2][]int{[]int{1, 2}, []int{1, 1}}, 1
    fmt.Printf("Input: '%v', Answer: %v, Yours: %v \n", k, v, findMedianSortedArrays(k[0], k[1]))
}

func findMedianSortedArrays(nums1 []int, nums2 []int) float64 {
}
```

### 2. 初步实现

```go
func findMedianSortedArrays(nums1 []int, nums2 []int) float64 {
    len1, len2 := len(nums1), len(nums2)
    if len1 > len2 {  // 确保在更小的数组上进行二分法
        return findMedianSortedArrays(nums2, nums1)
    }
    
    lo, hi := 0, len1*2
    var Lmax1, Rmin1, Lmax2, Rmin2, c1, c2 int
    for lo <= hi {
        c1 = (lo + hi) / 2
        c2 = len1 + len2 - c1
        
        Lmax1 = nums1[(c1-1)/2]        
        Rmin1 = nums1[c1/2]        
        Lmax2 = nums2[(c2-1)/2]        
        Rmin2 = nums2[c2/2]        

        if (Rmin2 >= Lmax1) && (Rmin1 >= Lmax2) {
            break
        }
        if Lmax1 > Rmin2 {
            hi = c1
        } else {
            lo = c1 + 1
        }
    }
    return float64(mymax(Lmax1, Lmax2)+mymin(Rmin1, Rmin2)) / 2
}

func mymax(x, y int) int {
    if x > y {
        return x
    }
    return y
}
func mymin(x, y int) int {
    if x < y {
        return x
    }
    return y
}
```

以上就实现了之前的解题思路，但是还没有考虑下标边界值的问题，所以我们加一些判定条件，
并且引入常数`intsets.MaxInt`和`intsets.MinInt`用于在超出边界时的返回值（参与比较）。

### 3. 完整实现

```go
func findMedianSortedArrays(nums1 []int, nums2 []int) float64 {
    len1, len2 := len(nums1), len(nums2)
    if len1 > len2 {
        return findMedianSortedArrays(nums2, nums1)
    }
    
    lo, hi := 0, len1*2
    var Lmax1, Rmin1, Lmax2, Rmin2, c1, c2 int
    for lo <= hi {
        c1 = (lo + hi) / 2
        c2 = len1 + len2 - c1

        if c1 <= 0 {        // ↓↓↓↓增加了这些条件
            Lmax1 = intsets.MinInt
        } else {
            Lmax1 = nums1[(c1-1)/2]
        }
        if c1 >= len1*2 {
            Rmin1 = intsets.MaxInt
        } else {
            Rmin1 = nums1[c1/2]
        }
        if c2 <= 0 {
            Lmax2 = intsets.MinInt
        } else {
            Lmax2 = nums2[(c2-1)/2]
        }
        if c2 >= len2*2 {
            Rmin2 = intsets.MaxInt
        } else {
            Rmin2 = nums2[c2/2]
        }                  // ↑↑↑↑增加了这些条件 

        if (Rmin2 >= Lmax1) && (Rmin1 >= Lmax2) {
            break
        }
        if Lmax1 > Rmin2 {
            hi = c1
        } else {
            lo = c1 + 1
        }
    }
    return float64(mymax(Lmax1, Lmax2)+mymin(Rmin1, Rmin2)) / 2
}
```

输出：

```text
Input: '[[1 3] [2]]', Answer: 2, Yours: 2 
Input: '[[1 2] [3 4]]', Answer: 2.5, Yours: 2.5 
Input: '[[1 2 3 4] [0 5 7 9]]', Answer: 3.5, Yours: 3.5 
Input: '[[1 2] [1 1]]', Answer: 1, Yours: 1 
```

## 小结&收获

遇到难题不要死磕，要学会把`大问题`分解为关键的`小问题`；

然后把复杂的`小问题`抽象出来，提炼出复杂情况下的`通用算法`。

当难题都简化之后，自然就实现了量变到质变，从不可能到可能的实现。