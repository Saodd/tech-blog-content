```yaml lw-blog-meta
title: GoTour后续练习题之 掷骰子游戏 Pig
date: "2019-06-19"
brev: '在Gotour的后续页面中，我们进入的是《Codewalk: Go中的一等函数（First Class Functions in Go）》。简单查看代码之后，发现这个游戏纯粹是电脑左右跟右手玩，我们人类玩家只是看个结果而已。凭什么呀！我也想玩！'
tags: [Golang]
```


## 原代码

```go
// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
    "fmt"
    "math/rand"
)

/*
const (
    win            = 100 // The winning score in a game of Pig
    gamesPerSeries = 10  // The number of games per series to simulate
)
*/

const (
    win            = 100 // 在一场Pig游戏中获胜的分数
    gamesPerSeries = 10  // 每次连续模拟游戏的数量
)

// A score includes scores accumulated in previous turns for each player,
// as well as the points scored by the current player in this turn.

// 总分 score 包括每个玩家前几轮的得分以及本轮中当前玩家的得分。
type score struct {
    player, opponent, thisTurn int
}

// An action transitions stochastically to a resulting score.

// action 将一个动作随机转换为一个分数。
type action func(current score) (result score, turnIsOver bool)

// roll returns the (result, turnIsOver) outcome of simulating a die roll.
// If the roll value is 1, then thisTurn score is abandoned, and the players'
// roles swap.  Otherwise, the roll value is added to thisTurn.

// roll 返回模拟一次掷骰产生的 (result, turnIsOver)。
// 若掷骰的结果 outcome 为1，那么这一轮 thisTurn 的分数就会被抛弃，
// 然后玩家的角色互换。否则，此次掷骰的值就会计入这一轮 thisTurn 中。
func roll(s score) (score, bool) {
    outcome := rand.Intn(6) + 1 // A random int in [1, 6]
    if outcome == 1 {
        return score{s.opponent, s.player, 0}, true
    }
    return score{s.player, s.opponent, outcome + s.thisTurn}, false
}

// stay returns the (result, turnIsOver) outcome of staying.
// thisTurn score is added to the player's score, and the players' roles swap.

// stay 返回停留时产生的结果 (result, turnIsOver)。
// 这一轮 thisTurn 的分数会计入该玩家的总分中，然后玩家的角色互换。
func stay(s score) (score, bool) {
    return score{s.opponent, s.player + s.thisTurn, 0}, true
}

// A strategy chooses an action for any given score.

// strategy 为任何给定的分数 score 返回一个动作 action
type strategy func(score) action

// stayAtK returns a strategy that rolls until thisTurn is at least k, then stays.

// strategy 返回一个策略，该策略继续掷骰直到这一轮 thisTurn 至少为 k，然后停留。
func stayAtK(k int) strategy {
    return func(s score) action {
        if s.thisTurn >= k {
            return stay
        }
        return roll
    }
}

// play simulates a Pig game and returns the winner (0 or 1).

// play 模拟一场Pig游戏并返回赢家（0或1）。
func play(strategy0, strategy1 strategy) int {
    strategies := []strategy{strategy0, strategy1}
    var s score
    var turnIsOver bool
    // 随机决定谁先玩
    currentPlayer := rand.Intn(2) // Randomly decide who plays first
    for s.player+s.thisTurn < win {
        action := strategies[currentPlayer](s)
        s, turnIsOver = action(s)
        if turnIsOver {
            currentPlayer = (currentPlayer + 1) % 2
        }
    }
    return currentPlayer
}

// roundRobin simulates a series of games between every pair of strategies.

// roundRobin 模拟每一对策略 strategies 之间的一系列游戏。
func roundRobin(strategies []strategy) ([]int, int) {
    wins := make([]int, len(strategies))
    for i := 0; i < len(strategies); i++ {
        for j := i + 1; j < len(strategies); j++ {
            for k := 0; k < gamesPerSeries; k++ {
                winner := play(strategies[i], strategies[j])
                if winner == 0 {
                    wins[i]++
                } else {
                    wins[j]++
                }
            }
        }
    }
    // 不能自己一个人玩
    gamesPerStrategy := gamesPerSeries * (len(strategies) - 1) // no self play
    return wins, gamesPerStrategy
}

// ratioString takes a list of integer values and returns a string that lists
// each value and its percentage of the sum of all values.
// e.g., ratios(1, 2, 3) = "1/6 (16.7%), 2/6 (33.3%), 3/6 (50.0%)"

// ratioString 接受一个整数值的列表并返回一个字符串，
// 它列出了每一个值以及它对于所有值之和的百分比。
// 例如，ratios(1, 2, 3) = "1/6 (16.7%), 2/6 (33.3%), 3/6 (50.0%)"
func ratioString(vals ...int) string {
    total := 0
    for _, val := range vals {
        total += val
    }
    s := ""
    for _, val := range vals {
        if s != "" {
            s += ", "
        }
        pct := 100 * float64(val) / float64(total)
        s += fmt.Sprintf("%d/%d (%0.1f%%)", val, total, pct)
    }
    return s
}

func main() {
    strategies := make([]strategy, win)
    for k := range strategies {
        strategies[k] = stayAtK(k + 1)
    }
    wins, games := roundRobin(strategies)

    for k := range strategies {
        fmt.Printf("Wins, losses staying at k =% 4d: %s\n",
            k+1, ratioString(wins[k], games-wins[k]))
    }
}

```

简单看一下，从`main()`进入，然后创建了100个策略`[]strategy`，
然后设置了一个裁判员`roundRobin(strategies)`，
让这100个策略捉对厮杀`play(strategy0, strategy1)`，
每个策略的战绩记录在`var wins []int`中，然后打印出来看。

每次战斗`play(strategy0, strategy1)`中，先随机一个人开始`currentPlayer := rand.Intn(2)`，
然后他根据当前的情况`action := strategies[currentPlayer](s)`选择行动(`stay()`或者`roll()`)，
然后执行行动`s, turnIsOver = action(s)`，并根据行动的结果继续下一轮循环。

从结果来看，大概7~36这个范围内的策略胜率较高。





## 增加人类玩家

我们人类玩家也应当是策略的一种，所以必须满足类型条件`type strategy func(score) action`，
在此条件上实现与控制台输入的互动（我把这也理解为一种接口）。

```go
func playerHuman() strategy {
    return func(s score) action {
        fmt.Printf("你的分数：%d，对手的分数：%d，本轮得分：%d     ", s.player, s.opponent, s.thisTurn)
        var choice string
        for {
            fmt.Println("输入0停止并得到本轮得分，输入1继续掷骰子：")
            _, e := fmt.Scanln(&choice)
            if e != nil {
                println("Error!", e)
            }
            switch choice {
            case "0":
                return stay
            case "1":
                return roll
            default:
                println("输入错误，请从新输入")
            }
        }
    }
}
```

人类玩家定义了，那么也要重新定义一下裁判。现在的裁判不需要循环监控990场比赛了，
只需要看《人类 VS AI》这一场比赛就可以了。
我这里偷懒，没有留下自定义AI策略的功能（其实也就是输入一个int），所以很简单：

```go
func roundHumanVSComputer() {
    if winner := playHumanVSComputer(playerHuman(), stayAtK(1)); winner == 0 {
        println("you WIN !!!!")
    } else {
        println("you lose ...")
    }
}
```

另外，为了更好的体验，我们在比赛中间提示一下“交换选手”；
否则的话AI玩家运行速度很快，马上又让你进行下一轮的感觉会很奇怪：

```go
func playHumanVSComputer(strategy0, strategy1 strategy) int {
    strategies := []strategy{strategy0, strategy1}
    var s score
    var turnIsOver bool
    currentPlayer := rand.Intn(2) 
    for s.player+s.thisTurn < win {
        action := strategies[currentPlayer](s)
        s, turnIsOver = action(s)
        if turnIsOver {
            fmt.Println("交换对手！")   // 注意！只增加了这一行，其他的不变
            currentPlayer = (currentPlayer + 1) % 2
        }
    }
    return currentPlayer
}
```

然后重新定义一下`main()`，这里用了大写因为我把真正的`main()`放在另一个package里了。

```go
func Main0014() {
    roundHumanVSComputer()
}
```
```go
package main

import "learnTour"

func main()  {
    learnTour.Main0014()
}
```



## 试玩！

这里定义的AI是`stayAtK(1)`：

```text
交换对手！
你的分数：0，对手的分数：4，本轮得分：0     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：0，对手的分数：4，本轮得分：6     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：0，对手的分数：4，本轮得分：12     输入0停止并得到本轮得分，输入1继续掷骰子：
0
交换对手！
交换对手！
你的分数：12，对手的分数：6，本轮得分：0     输入0停止并得到本轮得分，输入1继续掷骰子：
1
交换对手！
交换对手！
你的分数：12，对手的分数：8，本轮得分：0     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：12，对手的分数：8，本轮得分：3     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：12，对手的分数：8，本轮得分：8     输入0停止并得到本轮得分，输入1继续掷骰子：
0
交换对手！
交换对手！
你的分数：20，对手的分数：8，本轮得分：0     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：20，对手的分数：8，本轮得分：3     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：20，对手的分数：8，本轮得分：5     输入0停止并得到本轮得分，输入1继续掷骰子：
1
交换对手！
交换对手！
你的分数：20，对手的分数：14，本轮得分：0     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：20，对手的分数：14，本轮得分：5     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：20，对手的分数：14，本轮得分：8     输入0停止并得到本轮得分，输入1继续掷骰子：
0
交换对手！
交换对手！
你的分数：28，对手的分数：18，本轮得分：0     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：28，对手的分数：18，本轮得分：6     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：28，对手的分数：18，本轮得分：12     输入0停止并得到本轮得分，输入1继续掷骰子：
0
交换对手！
交换对手！
你的分数：40，对手的分数：21，本轮得分：0     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：40，对手的分数：21，本轮得分：6     输入0停止并得到本轮得分，输入1继续掷骰子：
1
交换对手！
交换对手！
你的分数：40，对手的分数：24，本轮得分：0     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：40，对手的分数：24，本轮得分：5     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：40，对手的分数：24，本轮得分：9     输入0停止并得到本轮得分，输入1继续掷骰子：
0
交换对手！
交换对手！
你的分数：49，对手的分数：26，本轮得分：0     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：49，对手的分数：26，本轮得分：2     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：49，对手的分数：26，本轮得分：7     输入0停止并得到本轮得分，输入1继续掷骰子：
0
交换对手！
交换对手！
你的分数：56，对手的分数：26，本轮得分：0     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：56，对手的分数：26，本轮得分：4     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：56，对手的分数：26，本轮得分：6     输入0停止并得到本轮得分，输入1继续掷骰子：
1
交换对手！
交换对手！
你的分数：56，对手的分数：32，本轮得分：0     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：56，对手的分数：32，本轮得分：6     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：56，对手的分数：32，本轮得分：10     输入0停止并得到本轮得分，输入1继续掷骰子：
01
输入0停止并得到本轮得分，输入1继续掷骰子：
输入错误，请从新输入
0
交换对手！
交换对手！
你的分数：66，对手的分数：35，本轮得分：0     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：66，对手的分数：35，本轮得分：4     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：66，对手的分数：35，本轮得分：6     输入0停止并得到本轮得分，输入1继续掷骰子：
0
交换对手！
交换对手！
你的分数：72，对手的分数：39，本轮得分：0     输入0停止并得到本轮得分，输入1继续掷骰子：
1
交换对手！
交换对手！
你的分数：72，对手的分数：41，本轮得分：0     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：72，对手的分数：41，本轮得分：5     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：72，对手的分数：41，本轮得分：10     输入0停止并得到本轮得分，输入1继续掷骰子：
0
交换对手！
交换对手！
你的分数：82，对手的分数：45，本轮得分：0     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：82，对手的分数：45，本轮得分：2     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：82，对手的分数：45，本轮得分：6     输入0停止并得到本轮得分，输入1继续掷骰子：
0
交换对手！
交换对手！
你的分数：88，对手的分数：50，本轮得分：0     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：88，对手的分数：50，本轮得分：5     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：88，对手的分数：50，本轮得分：7     输入0停止并得到本轮得分，输入1继续掷骰子：
1
你的分数：88，对手的分数：50，本轮得分：11     输入0停止并得到本轮得分，输入1继续掷骰子：
0
交换对手！
交换对手！
你的分数：99，对手的分数：54，本轮得分：0     输入0停止并得到本轮得分，输入1继续掷骰子：
1
you WIN !!!!

```

感觉还可以~   0_<