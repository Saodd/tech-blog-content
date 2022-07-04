```yaml lw-blog-meta
title: "后端监控：Prometheus + Grafana"
date: "2022-07-01"
brev: "以 MySQL 和 web服务 的常见指标为例，介绍监控大屏的搭建方法"
tags: ["中间件"]
```

## 架构综述

`Prometheus`这样描述自己：它是一个监控系统和服务，它定时向目标采集数据，按预设规则处理数据，然后展示结果、触发警报等。

简而言之，它由三个主要模块组成：采集器、储存、查询服务。

![架构结构图](https://cdn.rawgit.com/prometheus/prometheus/e761f0d/documentation/images/architecture.svg)

与它类似功能的另一个选择是`Zabbix`，这个东西似乎在Java的世界中更常用？一眼看过去有一股陈旧的味道，那种味道一般来自于apache、Java、php等这类上古技术环境。我选择`Prometheus`而不是`Zabbix`的原因很简单，前者是Go语言实现并且是CNCF的典型成员，而后者是个php的实现，显然两者没得比，是吧。

Prometheus的采集器，需要强调一下，它是主动“拉取”，而不是被动接收。如果采集目标同样也是主动推送的，那么需要的是`pushgateway`来做一个中转。

由于采集目标可以是任意东西，例如MySQL、statsd、HTTP、甚至单片机都可以纳入监控，因此需要一个适配器来将各种数据源转化为Prometheus的指标。这个适配器在这里术语叫做`exporter`。当然，你自己通过代码来实现格式转化也没问题。

Prometheus中有了数据，我们接下来应该还需要一款可视化工具，毕竟我们看监控更喜欢看图表，而不是密密麻麻的数字对吧。常用的可视化工具是`Grafana`。

所以总体架构是：若干个`exporter` + `Prometheus` + `Grafana`。

## 启动 Prometheus

直接[docker](https://hub.docker.com/r/prom/prometheus)启动。值得一提的是，镜像只有85MB，本身就已经是alpine了。不仅镜像体积小，运行占用内存也非常小。

```shell
docker pull prom/prometheus
docker run --name prometheus --restart always -p 9090:9090 -v /path/to/config:/etc/prometheus -dit prom/prometheus
```

> 如果是线上环境，记得还要给数据卷做持久化，这个镜像的数据目录应该是`/prometheus`路径。

此时访问9090端口，可以发现已经有一个web界面了，它是Prometheus自带的管理界面，可以做一些查询和配置。[官方文档](https://prometheus.io/docs/prometheus/latest/getting_started/#getting-started)

## 启动 Grafana

参考：[官方文档](https://grafana.com/grafana/download?edition=enterprise&platform=docker)

```shell
docker pull grafana/grafana-enterprise
docker run --name=grafana -p 3000:3000 --restart always -d grafana/grafana-enterprise
```

此时访问3000端口，初始用户名和密码都是`admin`。

首先需要配置数据源，点击进入`Data Sources`选项，首选就是Prometheus，把前面启动的服务地址配置进去即可。

> 也可以配置MySQL作为数据源，但是『把MySQL作为数据源』仅仅是把存在MySQL里的数据取出来显示成图形罢了，并不是『监控MySQL的状态』，后者我们在后续章节讨论。  
> 值得一提的是，Prometheus内置了一个时序数据库([TSDB](https://prometheus.io/docs/prometheus/latest/storage/))，并且对外暴露了查询接口([PromQL](https://prometheus.io/docs/prometheus/latest/querying/basics/)，从外部看完全可以看作就是一个类似MySQL的数据库，所以他们会并列在一起供选择配置。

接下来创建面板，面板的作用是把数据源转化为可视化图形界面，进入`New dashboard / Edit Panel`选项，随便选一个参数，例如`go_goroutines`（注：这个是Prometheus本体进程的Go程数量），就可以看到一个折线图展示在上方了。

## 自建 metrics 服务

参考：[文章](https://dev.to/kishanbsh/capturing-custom-last-request-time-metrics-using-prometheus-in-gin-36d6)

简单看一下怎么暴露自定义指标(`metrics`)，这里用`gin`框架来搭建HTTP，抄一段代码：

```go
func main() {
    r := gin.Default()
    r.GET("/metrics", r.GET("/metrics", gin.WrapH(promhttp.Handler())))
    r.Run("0.0.0.0:8080")
}
```

这样我们就在8080端口上运行了一个web服务，其中`/metrics`接口会提供Prometheus所需的测量数据，数据内容是一些默认指标，主要是go进程的一些数据。它是纯文本格式，我们可以在浏览器中手动访问看看其原始结构，我这里就不贴了。

花两分钟简单浏览一下源码，很快就能知道，接口中所包含的数据，都是注册在`promhttp.Handler()`所使用的一个全局默认对象上的；再看两眼，也就知道怎么自定义指标了。

首先我们创建一个`Gauge`对象，它可以被翻译为『指标』。它必须有一个`Name`作为其标识符，然后还可以有`Help`这种注释性的文本内容。它的值必须是数字类型——确切的说是`float64`类型。

然后我们创建一个`Registry`，它可以被翻译为『注册表』或者意译为『指标仓库』，再把前面的`Gauge`对象注册进去。

```go
func main() {
	// 创建Gauge，其默认值就是0
	var counter = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "custom_counter",
		Help: "这是一个自定义指标，其数值是一个计数器的值。",
	})

	// 创建Registry，并注册Gauge
    var register = prometheus.NewRegistry()
    register.Register(counter)
	
	// 创建web路由
    r := gin.Default()
	r.GET("/add", func(c *gin.Context) {
		counter.Inc() // 每次访问此接口，Gauge的值递增1
	})
	r.GET("/metrics", gin.WrapH(promhttp.HandlerFor(register, promhttp.HandlerOpts{})))
	r.Run("0.0.0.0:8080")
}
```

试着访问`:8080/metrics`接口，会得到以下内容，非常清晰明了：

```text
# HELP custom_counter 这是一个自定义指标，其数值是一个计数器的值。
# TYPE custom_counter gauge
custom_counter 0
```

接下来我们要通知`Prometheus`，让它来拉取这个新的web服务的接口中的指标。在它的yaml配置文件中添加下列内容：

```yaml
scrape_configs:
  - job_name: 'gin_metrics'  # 名字随便取
    scrape_interval: 5s
    static_configs:
      - targets: ['127.0.0.1:8080']  # 刚才代码启动的gin服务的ip和地址号
```

重启Prometheus之后，我们就可以在它的管理界面（9090端口）上搜索到新的名叫`custom_counter`的指标了。

最后我们要通知`Grafana`。在它的web界面（3000端口）里选择一个Dashboard，然后添加一个Panel，在Query中添加一个`Metrics`，其名字就是刚才的`custom_counter`这个名字，然后就能得到这个指标的图表了。

至此，我们已经创建了一个自定义的指标数值，并且成功地将其可视化了。

## 监控 MySQL

MySQL的运行指标需要通过一个exporter来转化，已经有现成的官方工具了：[mysqld_exporter](https://github.com/prometheus/mysqld_exporter)

先在MySQL中创建一个监控专用账号，只赋予部分权限：

```sql
GRANT PROCESS, REPLICATION CLIENT, SELECT ON *.* TO prometheus;
```

然后启动exporter（注意命令中的括号和斜杠不能省略）：

```shell
docker pull prom/mysqld-exporter
docker run --name mysqld-exporter --restart always -p 9104:9104 -e DATA_SOURCE_NAME="prometheus:prometheus@(10.0.6.239:3307)/" -d prom/mysqld-exporter
```

可以手动试试访问`:9104/metrics`接口来确认指标能够正常获取。

然后配置Prometheus让它去拉取数据：

````yaml
scrape_configs:
  - job_name: 'mysql_3307'
    scrape_interval: 30s
    static_configs:
      - targets: ['10.0.6.239:9104']
````

然后通知Grafana生成图表。我这里偷懒，随手搜了一个[模板](https://grafana.com/grafana/dashboards/7362)，直接导入到Grafana中即可，简单看了下它的指标还是比较全的。

![MySQL监控预览图](https://grafana.com/api/dashboards/7362/images/4700/image)

## 监控 statsd

一个很常见的需求场景：后端微服务场景下，需要统计各个服务的响应时间（各档位）、QPS等数据。

传统（至今）的解决方案是使用`statsd`协议，通过`udp`发送到一个专用的数据收集服务上去。

> `statsd`既可以指一种文本协议格式，也可以指能够处理这种协议的服务。

> 为什么要用udp来传输呢？一方面这种统计数据并不算重要，偶尔丢失也可以接受，另一方面它的开销更小，对于海量的QPS统计数据来说，肯定是要求性能越高越好的。

### statsd现有的轮子

现有的轮子：[statsd-exporter](https://registry.hub.docker.com/r/prom/statsd-exporter) 。它的镜像中不仅仅包括了`exporter`的能力，它本身也内置了`statsd`的能力，也就是说不需要另外启动一个专门的`statsd`服务了（也因此它需要暴露多个端口）。

```shell
docker run --name statsd-exporter -p 9102:9102 -p 9125:9125 -p 9125:9125/udp \
      -v $PWD/statsd_mapping.yml:/tmp/statsd_mapping.yml \
      -it prom/statsd-exporter --statsd.mapping-config=/tmp/statsd_mapping.yml
```

然后我们准备一个golang的web服务，这里在gin的框架体系下，选用[gin-statsd](github.com/amalfra/gin-statsd)这个库，示例代码：

```go
package main

import (
	statsdMiddleware "github.com/amalfra/gin-statsd/v2/middleware"
	"github.com/gin-gonic/gin"
	"math/rand"
	"time"
)

func main() {
	r := gin.Default()
	r.Use(statsdMiddleware.New(statsdMiddleware.Options{Host: "10.0.6.239", Port: 9125})) // statsd-exporter服务的ip和端口号
	r.GET("/test", func(c *gin.Context) {
		time.Sleep(time.Duration(rand.Int63n(1000)) * time.Millisecond)
		c.String(200, "ok")
	})
	r.Run("0.0.0.0:8080")
}
```

如果我们捕获这个中间件发出的udp消息，可以观察到它的消息是这样的（这就是`statsd`协议格式）：

```text
status_code.200:1|c
response_time:565.1610999999999|ms
```

许多条这样的数据发到`statsd`服务上，服务会把数据进行聚合。然后通过`:9102/metrics`接口可以观察一下指标，会看到其中有这几行（聚合之后、转化为`PromQL`协议格式的指标数据）：

```text
# HELP response_time Metric autogenerated by statsd_exporter.
# TYPE response_time summary
response_time{quantile="0.5"} 0.5646194
response_time{quantile="0.9"} 0.8249407
response_time{quantile="0.99"} 0.8249407
response_time_sum 1.8010377
response_time_count 3

# HELP status_code_200 Metric autogenerated by statsd_exporter.
# TYPE status_code_200 counter
status_code_200 3
```

前面几行的意思是，对于响应时间，50%请求挡位的值是0.56秒，90%请求的响应时间是0.82秒，99%请求的响应时间是0.82秒。

默认的统计周期是最近10分钟，因此我们需要对`statsd-exporter`进行一些设置。

```yaml
mappings:
  - match: "response_time"
    observer_type: summary
    name: "response_time_avg"
    summary_options:
      quantiles:
        - quantile: 0.99
          error: 0.001
        - quantile: 0.95
          error: 0.01
        - quantile: 0.9
          error: 0.05
        - quantile: 0.5
          error: 0.005
        - quantile: 0
      buckets: [ 0, 0.5, 0.9, 0.95, 0.99 ]
      max_age: 30s
      age_buckets: 3
      buf_cap: 10000
```

由于折腾了好一会都没找到详细的文档，所以这里不再展开了。总之呢，在这种默认的配置下，可以达到最基本的统计需求。

### 自己造一个轮子

在上面对`statsd-exporter`的配置过程中，我发现相关资料特别少，搞得我进展非常不顺利。

因此我想：另一种思路是自己实现。

其实思路挺简单的，无非就是客户端（被监控的进程）从udp发，服务端从udp读，再结合前面章节所说的《自建metric服务》搭起一个服务等着Prometheus来拉取即可。

稍微麻烦的地方在于，服务端收到数据之后还需要做一些算法上的处理，例如分挡位计算、例如按时间滚动等。

- 『分档位计算』的解决方法是：用最小堆，有`O(logN)`的单次插入时间（复用数组内存空间）、以及`O(1)`的总体分位统计时间；如果这样性能还不能满足，那么在海量数据的场景下可以考虑牺牲精确度，例如以1毫秒或者10毫秒分为1组、每组只计一个总数，那么单次插入时间可以降低到`O(1)`，这已经是单机极限了，还不够的话就只能“分库分表”了。
- 『按时间滚动』的解决方法，比较简单，就是做2组或者更多组的数据结构，轮流交替使用即可。

以后若有时间了，我看看自己造这个轮子吧。（但这个事情我感觉不合理，这么普通的需求居然没有成熟的轮子。也许是我搜索的姿势不对？）

## 总结

本文介绍了`Prometheus`+`Grafana`监控系统的搭建，以及典型的数据库（MySQL）和web服务指标的采集的方法。

掌握了这些，大概可以说服务器监控技术已经入门了。
