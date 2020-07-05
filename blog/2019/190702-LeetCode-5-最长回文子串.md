```json lw-blog-meta
{"title":"LeetCode[5]: 最长回文子串","date":"2019-07-02","brev":"","tags":["算法与数据结构"]}
```



## 原题

```text
给定一个字符串 s，找到 s 中最长的回文子串。你可以假设 s 的最大长度为 1000。

示例 1：
输入: "babad"
输出: "bab"
注意: "aba" 也是一个有效答案。

示例 2：
输入: "cbbd"
输出: "bb"

来源：力扣（LeetCode）
链接：https://leetcode-cn.com/problems/longest-palindromic-substring
著作权归领扣网络所有。商业转载请联系官方授权，非商业转载请注明出处。
```

## 末端法——即暴力破解法

最容易想到的就是：

1. 遍历整个字符串，逐个取出字符；
2. 以每个取出的字符位置`i`为**末端**，在它前面就能形成`i+1`个**子字符串**（`s[0:i+1], s[1:i+1]...s[i:i+1]`）；
3. 分别对这些**子字符串**进行回文判断，如果是回文，尝试记录下它是否是最长的回文。

贴上代码：

```golang
func longestPalindrome_end(s string) string {
    sb := []byte(s)
    longest := []byte{}

    for i := range sb {
        for j := 0; j <= i; j++ {
            isPalindrome := true
            for k := j; k <= i-(k-j); k++ {
                if sb[k] != sb[i-(k-j)] {
                    isPalindrome = false
                    break
                }
            }
            if isPalindrome && (i-j+1 > len(longest)) {
                longest = sb[j : i+1]
            }
        }
    }
    return string(longest)
}
```

这种解法非常简单，直观。虽然空间复杂度O(1)，但是时间复杂度爆炸了O(n<sup>3</sup>)。提交成绩是(2108 ms, 2.3 Mb)。

> 其实我给出的代码的空间复杂度是O(n)，但是如果不转化为[]bytes，并且用两个指针代替longest的话，
空间复杂度就是O(1)了。对于后面的中心法同理。

## 中心法

在前面的`末端法`中，是有三层循环的。而如果仔细研究`回文`，我们可以发现它是有`中心对称`的特性的。
如果我们以中心点开始向外查找，就可以省去一层循环了（不用分割`子字符串`）。

```text
对于任意的*奇数回文字符串s，将它的中点记为i，则在定义域内，对任意x满足下列条件：
s[i-x] == s[i+x]

对于任意的*偶数回文字符串s，将它的中心两点记为i-1和i，则在定义域内，对任意x满足下列条件：
s[i-1-x] == s[i+x]

```

实现思路：

1. 遍历整个字符串，逐个取出字符。
2. 以每个取出的字符位置`i`为**中心**，（分奇数/偶数两种情况）向外拓展；
如果满足回文条件，则尝试更新`最长回文子串`；如果不满足条件，那就break。

```golang
func longestPalindrome_mid(s string) string {
    sb := []byte(s)
    lengthTol := len(s)
    if lengthTol == 0 {
        return ""
    }
    longest := []byte{}

    for i := range sb {
        id := i * 2
        stop := id - lengthTol
        // 奇数回文
        for j := i; (j >= 0) && (j > stop); j-- {
            //fmt.Print(string(sb[j]), string(sb[id-j]))
            if sb[j] == sb[id-j] {
                //fmt.Println((id - j*2 + 1), len(longest))
                if (id - j*2 + 1) > len(longest) {
                    longest = sb[j : id-j+1]
                    //fmt.Println("longest 奇数", string(longest))
                }
            } else {
                break
            }
        }
        // 偶数回文
        for j := i; (j > 0) && (j > stop); j-- {
            if sb[j-1] == sb[id-j] {
                if (id - j*2 + 2) > len(longest) {
                    longest = sb[j-1 : id-j+1]
                    //fmt.Println("longest 偶数", string(longest))
                }
            } else {
                break
            }
        }
    }
    return string(longest)
}
```

空间复杂度O(1)，时间复杂度O(n<sup>2</sup>)。提交成绩是(16 ms, 2.4 Mb)。

> 时间复杂度虽然是平方，在日常情况下应该是比较接近线性的，因为一般回文子串不会太大（即第二层循环一般次数不多）。
当然，极端的测试输入状况下，最坏情况是平方级的。

## Manacher算法

在参考答案中给出了一个名词`Manacher`，但是并没有展开讲解，我去搜索了很多资料，归纳一下。

首先我们考虑`中心法`中存在的问题：

1. 奇数/偶数分别求解。（是不是很眼熟？之前的`两个有序数组求中位数`也是类似的问题）
2. 对每个位置都独立求解，向两侧延申，会造成重复访问。（考虑到回文的对称性，是可以应用缓存来减少访问的。）

`Manacher算法`其实就是对以上两点进行了改善，我们看：

### 解决奇偶性的问题

与上一篇博文所述同理，只需要在每个元素中间以及数组首末端，插入一个**自定义虚拟元素**即可。我们这里使用`byte('#')`进行插入。
这样新的数组就一定是奇数长度了。

```text
aba  ———>  #a#b#a#
abba ———>  #a#b#b#a#

```

```golang
{
    const null = byte('#')
    sb := make([]byte, lengthRL)
    sb[lengthTol*2] = null
    for i := 0; i < lengthTol*2; i += 2 {
        sb[i], sb[i+1] = null, s[i/2]
    }
}
```

### Manacher的基础证明

我们首先要知道这样做的原理。先定义一些名词：

1. `回文半径`：回文串中最左或最右位置的字符与其对称轴的距离。
2. `Manacher`定义了一个回文半径数组`RL`，用`RL[i]`表示以第`i`个字符为对称轴的回文串的回文半径。
3. `RL[i]-1`恰好是以`i`为对称轴的回文的**长度**。

这里给出一个示例：

```text
char:    # a # b # a #
 RL :    1 2 1 4 1 2 1
RL-1:    0 1 0 3 0 1 0   -> 最大值3，即最大回文长度是3

char:    # a # b # b # a #
 RL :    1 2 1 2 5 2 1 2 1
RL-1:    0 1 0 1 4 1 0 1 0  -> 最大值4，即最大回文长度是4
```

> 证明：为什么`RL[i]-1`恰好是回文子串的长度？参考上面的例子来分析：  
>  
> 先是奇数的情况，即对称轴落在真实元素上（参考上面的`aba`）：  
> 最大回文半径`RL`，意味着（在虚拟数组上的）最大回文长度是`2*RL-1`（半径乘以2减去1），
> 而其中会包含`Rl/2*2`个虚拟加的空元素`#`，所以只剩下`1*RL-1`个真实元素了。
>  
> 对于偶数的情况，即对称轴落在虚拟元素上（参考上面的`abba`）：  
> 同理（在虚拟数组上的）最大回文长度是`2*RL-1`，
> 单侧的虚拟元素数量是`(RL-1)/2`（因为RL一定是奇数），两侧总共有`RL-1`个虚拟元素，再加上中心的1个虚拟元素，
> 所以也只剩下`1*RL-1`个真实元素。

好的，现在问题就转化成了，如何求得回文半径数组`RL`。

我们再回忆一下回文的`中心对称性`，不难得出以下结论：

```text
对于任意回文字符串s，记它的对称轴位置为pos；
在pos的任意一侧如果存在以p1为对称轴的回文子串s1，则必定在对称的位置p2处存在一个相同的回文子串s2，即：
    (p1+p2)/2 = pos
           s1 = s2
           
s.index:     ........ p1-n ... p1 ... p1+n ........pos......... p2-n ... p2 ... p2+n .................
s.value:     ........  a ..... b ...... a  ..................... a ..... b ..... a ..............
                      └────回文子串s1───┘                       └───回文子串s2───┘
```

进一步可以得到：

```text
定理1：
对于任意回文字符串s，记它的对称轴位置为pos；在定义域内，它两侧的回文半径子数组也是中心对称的，即：
    RL[pos-x] = RL[pos+x]
```

### Manacher的实现

假设我们**从左至右**地推算RL数组，遍历下标记为`i`，

#### i落于已知的回文子串中

那么根据`定理1`，如果`i`存在于任何已知的回文子串中，那么我们就可以访问它对称位置的回文半径。
我们使用`maxRight`来记录**已知的回文子串**所达到的最右位置，用以判断`i`是否存在于任何已知的回文子串；
用`pos`来表示`maxRight`所对应的对称轴位置。如图：

![i落于已知的回文子串中](/static/blog/2019-07-02-Manacher-left.png)

```text
               i'     pos      i   maxRight
s.index:  .....↑.......↑.......↑......↑.........

由定理1可得：RL[i]=RL[i']
```

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

```golang
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

```golang
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