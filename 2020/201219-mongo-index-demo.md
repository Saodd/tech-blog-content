```yaml lw-blog-meta
title: 'Mongo联合索引性能试验'
date: "2020-12-19"
brev: "一次小试验。"
tags: ["DB"]
```

## 测试用数据

测试用的数据结构有17个字段。

本次的需求是，用2个字段`uid`,`ts`作为筛选条件，做`count`操作。

在局域网内的一个测试数据库中，模拟插入了50万条数据。其中符合`uid`条件的有30w，同时符合`uid`和`ts`条件的有25w.

## 1. 一个单索引

首先是默认情况，只在`uid`上建立了索引。

性能分析命令是`explain`。不过与`find`之类的操作不同的是，`count`它并不是一个cursor，所以在语法上要把`explain`放在前面。

```mongodb
db.SomeCollection.explain('executionStats').count({'uid':0,'ts':{'$gte':ISODate('2020-12-16T00:00:00.000Z'),'$lt':ISODate('2020-12-17T00:00:00.000Z')}})
```

执行结果：

```text
"executionTimeMillis" : 1220,
"totalKeysExamined" : 300001,
"totalDocsExamined" : 300001,
```

上述结果表明，mongoDB先在`uid`索引上进行检索，找出了30w条记录；然后由于另一个字段没有索引，所以不得不把这些记录全部取出来扫描一编。总时间1220ms，是个非常慢的查询了。

## 2. 一个联合索引

然后尝试，给这个查询条件涉及到的2个字段加一个联合索引。在golang中语法如下：

```go
var col = mg.Database("SomeDatabase").Collection("SomeCollection")
indexName, err := col.Indexes().CreateOne(context.Background(), mongo.IndexModel{Keys: bson.M{"iid":1,"ts":1}})
```

值得一提的是，在当前的情况下（总数据50w条，硬件4核），建立这条索引需要约20秒的时间。

然后进行查询。执行结果如下：

```text
"executionTimeMillis" : 95,
"totalKeysExamined" : 150002,
"totalDocsExamined" : 0,
```

上述结果表明，本次查询仅仅只在索引上进行遍历，并没有去取Documents数据。执行效率也高了很多，本次查询耗时95ms，整整下降了一个数量级。（我想这个时间就是单纯的在一颗二叉树上进行遍历所需的时间，毕竟是`count`是需要遍历所有节点的。）

## 3. 两个单索引？

再尝试一种情况，即分别只在`uid`和`ts`两个字段上**分别**建立索引。执行结果如下：

```text
"executionStats" : {
        "executionSuccess" : true,
        "nReturned" : 150001,
        "executionTimeMillis" : 861,
        "totalKeysExamined" : 250001,
        "totalDocsExamined" : 250001,
        "executionStages" : {
                "stage" : "FETCH",
                "filter" : {
                        "uid" : {
                                "$eq" : 0
                        }
                },
                "nReturned" : 150001,
                "executionTimeMillisEstimate" : 60,
                "works" : 250002,
                "advanced" : 150001,
                "needTime" : 100000,
                "needYield" : 0,
                "saveState" : 1954,
                "restoreState" : 1954,
                "isEOF" : 1,
                "docsExamined" : 250001,
                "alreadyHasObj" : 0,
                "inputStage" : {
                        "stage" : "IXSCAN",
                        "nReturned" : 250001,
                        "executionTimeMillisEstimate" : 26,
                        "works" : 250002,
                        "advanced" : 250001,
                        "needTime" : 0,
                        "needYield" : 0,
                        "saveState" : 1954,
                        "restoreState" : 1954,
                        "isEOF" : 1,
                        "keyPattern" : {
                                "ts" : 1
                        },
                        "indexName" : "ts_1",
                        "isMultiKey" : false,
                        "multiKeyPaths" : {
                                "ts" : [ ]
                        },
                        "isUnique" : false,
                        "isSparse" : false,
                        "isPartial" : false,
                        "indexVersion" : 2,
                        "direction" : "forward",
                        "indexBounds" : {
                                "ts" : [
                                        "[new Date(1608076800000), new Date(1608163200000))"
                                ]
                        },
                        "keysExamined" : 250001,
                        "seeks" : 1,
                        "dupsTested" : 0,
                        "dupsDropped" : 0
                }
        }
},
```

上述结果说明，先用`ts`索引查询出了25w条记录，然后把这25w条记录全部取出来再逐条检查`uid`字段，最后得到结果。总时间花费是861ms。比`uid`索引好一些，是因为本次只逐条检查了25w条记录，少了5w。

也就是说，两个单个的索引，对于联合条件查询，并没有任何作用。（以前我以为会有点作用，看来是理解的不对）
