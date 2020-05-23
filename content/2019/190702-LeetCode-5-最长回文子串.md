---
title: ''LeetCode[5]: 最长回文子串''
description: '""'
pubDate: '2019-07-02'
updateDate: '2019-07-02'
tags: [算法与数据结构]
keywords: ''
---

注意！`i`在已知的回文子串中的确是对称的，也必须满足`RL[i]=RL[i']`的条件，但是:

1. `定理1`是在它**定义域**内成立的，如果回文子串太大，超过了定义域，那就无效了。所以不能超出已经探测的区域`maxRight`；
2. 在右侧未触及的区域，可以形成以`i`为对称轴的新的，更大的回文子串。

```text
情况1：超过定义域，定理1无效
                            ┌───────────关于pos对称的已知回文子串 s─────────────┐
s.index:     ........ p1-n ... p1 ... p1+n ........pos......... p2-n ... p2 ... p2+n .................
s.value:     ........  a ..... b ...... a  ..................... a ..... b ..... a ..............
                      └────回文子串s1───┘                       └───回文子串s2───┘

情况2：在右侧形成新的回文子串
               i'     pos      i   maxRight
s.index:  .....↑.......↑.......↑......↑.........
             └s1┘            └s2┘
                    └───────sNew───────┘

```

所以在这一步只能保证`RL[i]>=Min(RL[i'], maxRight-i)`。

但这就已经足够了，已经达成了我们缓存的目的，我们第二轮循环可以从`Min(RL[i'], maxRight-i)`开始递增，而不是从`0`开始递增。

#### i落于右侧未知范围内

那就老老实实的从零开始探测吧。

### Manacher算法代码

```go
func longestPalindrome(s string) string {
    lengthTol := len(s)
    lengthRL := lengthTol*2 + 1
    if lengthTol == 0 {
        return ""
    }
    // 插入虚拟元素使其成为奇数长度
    const null = byte('#')
    sb := make([]byte, lengthRL)
    sb[lengthTol*2] = null
    for i := 0; i < lengthTol*2; i += 2 {
        sb[i], sb[i+1] = null, s[i/2]
    }
    // 计算RL数组
    RL := make([]int, lengthRL)
    posRight, maxRight := 0, 0
    for i := range sb {
        if i < maxRight { //i落在已知范围内，使用对称位置的缓存
            RL[i] = mymin(RL[posRight*2-i], maxRight-i)
        }else {           //i落在未知区域内，从1开始算
            RL[i] = 1
        }
        // 向右探测，注意边界
        for (i-RL[i]>=0) && (i+RL[i]<lengthRL) && (sb[i-RL[i]]==sb[i+RL[i]]){
            RL[i]++
        }
        // 尝试更新maxRight
        if i+RL[i]-1>maxRight{
            posRight, maxRight = i, i+RL[i]-1
        }
    }
    // 找到最大的RL值
    maxR, maxpos := 0, 0
    for i, r := range RL {
        if r > maxR {
            maxR, maxpos = r, i
        }
    }
    // 根据最大的RL值，返回相应的回文字符串
    var result []byte
    for _, b := range sb[(maxpos - maxR)+1 : (maxpos+maxR)] {
        if b != null {
            result = append(result, b)
        }
    }
    return string(result)
}
```

测试用例：

```go
func Main0005() {
    var input string
    var output []string

    input = "babad"
    output = []string{"bab", "aba"}
    check(longestPalindrome(input), output)

    input = "cbbd"
    output = []string{"bb"}
    check(longestPalindrome(input), output)

    input = "a"
    output = []string{"a"}
    check(longestPalindrome(input), output)

    input = "ab"
    output = []string{"a", "b"}
    check(longestPalindrome(input), output)

    input = ""
    output = []string{""}
    check(longestPalindrome(input), output)

    input = "bb"
    output = []string{"bb"}
    check(longestPalindrome(input), output)

    input = "abcda"
    output = []string{"a", "b", "c", "d"}
    check(longestPalindrome(input), output)

    input = "babadada"
    output = []string{"adada"}
    check(longestPalindrome(input), output)
}

func check(s string, sanswer []string) {
    for _, sa := range sanswer {
        if s == sa {
            fmt.Println("Pass")
            return
        }
    }
    fmt.Println("Failed!! ", "Answer:", sanswer, "Yours: ", s)
}
```

复杂度分析：

- 空间复杂度O(n)，因为构建了RL数组，并且将输入的字符串扩容了两倍。
- 时间复杂度O(n)。这个不太好理解，但是只要搞清楚：

  每个位于右侧未知区域的元素只会被访问一次，访问之后`maxRight`就会更新；  
   位于`maxRight`范围内的已知元素不会再被访问了，因为它们都是已知回文子串的一部分，相应的信息以回文半径的形式都存入了RL数组。

```text
提交成绩：

执行用时 : 4 ms, 在所有 Go 提交中击败了93.41%的用户
内存消耗 : 3.2 MB, 在所有 Go 提交中击败了40.07%的用户
```

## 小结 & 展望

1. 首先依然是要把大问题分解为小问题，复杂的问题抽象为通用的问题；
2. 对于本题来说，我隐约觉得这样解释`Manacher算法`还不够抽象，还没有把真正的通用逻辑归纳出来。
   在参考答案中提到了一句“基于后缀树”这么个说法，但是稍微搜索了一下没得到结果。期待以后能真正遇见一次吧。
