```yaml lw-blog-meta
title: 'LeetCode[6]: Z字形变换'
date: "2019-07-03"
brev: 挺有意思的一个题，大概是用作加密？仔细研究一下还是偏向于数学问题，即如何把坐标对应起来。
tags: [算法与数据结构]
```


## 原题

```text
将一个给定字符串根据给定的行数，以从上往下、从左到右进行 Z 字形排列。

比如输入字符串为 "LEETCODEISHIRING" 行数为 3 时，排列如下：

    L   C   I   R
    E T O E S I I G
    E   D   H   N
之后，你的输出需要从左往右逐行读取，产生出一个新的字符串，比如："LCIRETOESIIGEDHN"。

请你实现这个将字符串进行指定行数变换的函数：

string convert(string s, int numRows);

示例 1:
    输入: s = "LEETCODEISHIRING", numRows = 3
    输出: "LCIRETOESIIGEDHN"

示例 2:
    输入: s = "LEETCODEISHIRING", numRows = 4
    输出: "LDREOEIIECIHNTSG"
    解释:
        L     D     R
        E   O E   I I
        E C   I H   N
        T     S     G

来源：力扣（LeetCode）
链接：https://leetcode-cn.com/problems/zigzag-conversion
著作权归领扣网络所有。商业转载请联系官方授权，非商业转载请注明出处。

```

## 直接法——按形状构建

最直接能想到的办法就是建立一个二维数组，以模拟二维坐标的方式把这个Z形状画出来。

```text
sb := [][]bytes

       direction: ↓ ↑ ↑ ↓ ↑ ↑ ↓
sb[0]:            L     D     R
sb[1]:            E   O E   I I
......            E C   I H   N
sb[numRows-1]:    T     S     G
```

外层循环是对输入的`(s string)`进行遍历，内层分别进行向`↓`和向`↑`的构建。

但是要注意s分布的方向问题，`↓`与`↑`是交替的。
我这里不把方向作为变量来设置，而是作为两个独立的循环块来区分：

```go
func convertZigZag_brute(s string, numRows int) string {
    result := make([][]byte, numRows)
    lengthS := len(s)
    for i := 0; i < lengthS; {
        // 方向 ↓
        for pos := 0; pos < numRows && i < lengthS; pos++ {
            result[pos] = append(result[pos], s[i])
            i++
        }
        // 方向 ↑ ，注意不包含顶行与底行
        for neg := numRows - 2; neg > 0 && i < lengthS; neg-- {
            result[neg] = append(result[neg], s[i])
            i++
        }
    }
    // 把二维字符数组转换为string返回
    var resultS string
    for _, s := range result {
        resultS += string(s)
    }
    return resultS
}
```

整体的时间复杂度应该是O(n)级的，因为`(s string)`中每个元素只访问了一次。
空间复杂度也是O(n)，虽然构建了二维字符数组（切片）`(result [][]byte)`，但没有分配内存，中间都是用的`append`操作。

提交成绩是（12 ms, 6.5 MB），排名记得好像是（50%，40%）的水平。

## 计算坐标法

其实前面的算法已经比较简单明晰，而且线性级的时间复杂度也没有太大的优化空间了，能改进的只有线性前面的系数而已了。

唯一的问题就是二维字符数组太难受了，在反复`append`的过程中应该带来了很大的损耗。
所以接下来我们只用一维数组直接记录结果。那首先需要的就是推导公式了。


记`N=numRows`，`r`为当前执行的行数，我们来计算Z形状内的字符映射在result字符串中的位置`i`的表格：

### 首先确定端点的`i`

1. 从`0`开始，向↓画一竖的长度（包含起点，不含终点）是`N-1`，所以第一个底点的坐标是`N-1`;
2. 从`N-1`开始，向↑画斜线的长度（包含起点，不含终点）也是`N-1`，所以第二个顶点的坐标是`2*(N-1)`;
3. 从`2*(N-1)`开始，向↓画一竖的长度（包含起点，不含终点）也是`N-1`，所以第二个底点的坐标是`3*(N-1)`;

![Point1](../../tech-blog-pic/2019/2019-07-03-Point1.png)

### 推导其他点的`i`

我们从任意一个**顶点**出发，把连续的Z字形看成是`人`字形的组合，很容易得出：

1. 左腿的坐标是`顶点-r`;
2. 右腿的坐标是`顶点+r`;

![Point2](../../tech-blog-pic/2019/2019-07-03-Point2.png)

进一步，把所有的点都抽象化，我们把底点算成是右腿里的：

![Point3](../../tech-blog-pic/2019/2019-07-03-Point3.png)

### 考虑边界问题

考虑一下整体的程序结构。

1. 最外层肯定是对`numRows`的遍历循环，因为输出的时候就是按照行的顺序。
2. 内层对`人`字形进行循环。  
     - 那一共会有多少个`人`字形？其实不重要，我们最重要的是判断是否超出了字符串的边界`i<len(s)`。  
     - 不过我们也必须要限定一个数字，否则会进入无限循环。我们设置`c < (len(s)/(2*numRows-2) + 2)`。
3. 在第0层，即`r=0`时，会出现顶点重复的情况`顶点-r == 顶点+r`，我们单独排除它。
4. 在底层，我们只保留右腿的底点，放弃左腿的（这样可以少循环一次）

### 实现

```go
func convertZigZag(s string, numRows int) string {
    lengthS := len(s)
    if numRows == 1 { 
        return s
    }
    var result []byte

    for n := 0; n < numRows; n++ {
        for c := 0; c < (lengthS/(2*numRows-2) +2); c++ {
            ileft := c*2*(numRows-1) - n
            iright := ileft + 2*n
            if ileft >= lengthS {
                break
            }
            if ileft == iright {
                result = append(result, s[ileft])
            } else {
                if ileft >= 0 && (n != numRows-1) {
                    result = append(result, s[ileft])
                }
                if iright < lengthS {
                    result = append(result, s[iright])
                }
            }
        }
    }
    return string(result)
}
```

提交结果

```text
执行用时 : 8 ms, 在所有 Go 提交中击败了91.58%的用户
内存消耗 : 4.2 MB, 在所有 Go 提交中击败了84.62%的用户
```