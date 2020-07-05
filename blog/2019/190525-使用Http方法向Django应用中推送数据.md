```json lw-blog-meta
{"title":"使用Http的POST方法向Django应用中推送数据","date":"2019-05-25","brev":"使用POST方法推送数据，利用Django强大的Admin系统对数据进行查看和管理。","tags":["Python"]}
```



## 项目需求

最近一直在构思如何使用`docker`将公司的项目容器化，
其中最难的地方就是在开发过程中逐步将原有的项目与实体服务器进行**解耦**。
但是数据总会需要一个可靠储存的地方，所以即使将大部分代码都装入容器，
仍然会有一些代码必须依赖本地环境（数据文件）。  

所以如何在不同容器之间进行通讯，是个值得思考的问题。  

之前有考虑过使用[`socket`](https://github.com/Saodd/LewinTools/blob/master/lewintools/pro/socket.py)直接进行通讯，
即在每个容器都监听一个端口，
对传来的指令进行分析，然后按预设的方法去调用其他函数，并通过`socket`原路传回结果。  
这个方法我觉得是可行的，但是最大的问题还是在于用户获取到结果以后，如何以适当的形式展现它？  

还考虑过一个方法就是使用`Mongo`+`Mongo-express`进行日志的储存和展现：  
![展示3:Mongo-express列表页](/static/blog/2019-05-25-Mongo-List.png)  
但是问题也是有的，日志全部挤在一行了：  
![展示4:Mongo-express详情页](/static/blog/2019-05-25-Mongo-Detail.png)  

所以我想到了`Django`的Admin界面，它具有简单，大方，通过网页就可以直接操作数据库，
并且将数据库转化为对象，易于开发等等优点。实在是居家旅行之必备。


## Django接收端

首先需要一个`Model`用于保存数据，然后需要一个`View`用于响应POST请求。  
以我的项目为例，我先做了一个储存日志的功能：  
![展示1:Admin列表页](/static/blog/2019-05-25-Admin1.png)  
我可以在详情页中查看详细的日志数据：  
![展示1:Admin详情页](/static/blog/2019-05-25-Admin2.png)

`Model`代码如下：

```python
class Log(models.Model):
    Subject = models.CharField(max_length=50, help_text="Program name, used for classification.")
    Datetime = models.DateTimeField()
    Status = models.BooleanField()
    Msg = models.CharField(max_length=100, help_text="Speak in short.")
    Log = models.TextField(blank=True)
    c_time = models.DateTimeField(auto_now_add=True)

    def __str__(self):
        return "[{}][{}][{}]: {}".format(self.Datetime.strftime("%Y%m%d %H:%M:%S"),
                                       "ok " if self.Status else "err", self.Subject, self.Msg)

    class Meta:
        ordering = ["-c_time"]
```

`View`代码如下：
```python
from django.views.decorators.csrf import csrf_exempt
from django.utils.decorators import method_decorator

@method_decorator(csrf_exempt,name='dispatch')
class Log(generic.View):
    @csrf_exempt
    def post(self, request: WSGIRequest):
        json_data = request.POST.get("data")
        try:
            data = json.loads(json_data)
        except:
            return HttpResponseBadRequest("Cant loads json.")
        if not data:
            return HttpResponseBadRequest("No data in json.")
        try:
            log = models.Log(**data)
            log.save()
        except Exception as e:
            logger.print_exception()
            return HttpResponseBadRequest("%s" % e)
        return HttpResponse("200 ok")
```
特别注意：
1. `csrf_exempt`是用来免除`CSRF`验证的（是Django默认的防御机制，还记不记得在表单中我们总是要加一行`csrf_token`?），
但是他只能用于函数；再配合`method_decorator`就可以用于类/对象方法了。但是要注意安全问题，不要轻易向外暴露这个Django项目。
2. 我们使用`json`来传递数据，发送的时候用`json.dumps()`打包，接收的时候用`json.loads()`解包。
3. 建立模型对象的时候要`somemodel(**data)`展开字典，不能像使用POST那样直接传入`somemodel(request.POST)`。



## 发送端

发送端主要是使用`urllib`库来进行Http操作。代码如下：
```python
import urllib.parse
import urllib.request
class ApdjApiPost:
    api_url = "http://{}:{}/api/".format(running.Project_host, running.Project_apdj_port)

    def log(self, sbj: str, dt: datetime, status: bool, msg: str, log: str):
        url = self.api_url + "log/"
        self.logger.debug("Opening %s ..." % url)
        if len(log) > 30000:
            log = log[:15000] + "...\n" * 6 + log[-15000:]
        data = {"Subject": sbj,
                "Datetime": dt.strftime("%Y-%m-%d %H:%M:%S"),
                "Status": status,
                "Msg": msg,
                "Log": log, }
        try:
            data_encode = urllib.parse.urlencode({"data": json.dumps(data)}).encode()
            response = urllib.request.urlopen(url=url, data=data_encode, timeout=60)
            self.logger.info("Post done. Response: %s" % response.read().decode())
        except Exception as e:
            self.logger.error("Post failed: %s" % e)
            # self.logger.print_exception()
```
这样就可以运行了。  
有很多代码是可以重用的，自己考虑吧~



## 小结
1. 注意控制日志大小。其实把日志存在Postgres里我觉得还是挺奢侈的……最好的方式应当是使用models.FileField()字段，
以文件的形式保存。我这里只是想监控一下自动任务运行的状态，所以决定用精简日志（分级输出）的办法就好了。
2. 对于这个日志模型来说，可以加一些识别条件，这样就可以实现：`这边初始化日志 -> 主动请求那边执行 -> 
那边执行完毕后上传日志 -> 保存日志` 这样的流程了，这样的话如果定时任务没有正确运行的话，是可以看到空白的记录的，
有利于我们追踪。再进一步就可以替代`Airflow`了。
3. 我接触过`Airflow`，感觉不太好，感觉他太笨重了，`pip`的时候装了一大堆东西，看着还挺害怕的，
天知道出了问题要如何去维护？如果要深入钻研的话，学习成本恐怕挺大的，而且对于目前公司的项目来说，是真的杀鸡用牛刀了哈哈。
4. `Django`的确非常成熟，隐藏着很多宝藏，每当我登上一个台阶的时候都会发现我需要的功能他都准备好了，赞一个。



## 后记：新的想法

其实使用`Django`的模型系统接收日志，也是存在一些问题的。
使用`urllib`通过http来推送，还需要加码解码啥的，挺麻烦；
我觉得还是直接使用`pyMongo`来推送比较省事，对格式要求极低。  
其实核心诉求就是：①容易上传；②容易查看。  
所以用`pyMongo`加一个`Django页面`我觉得就非常ok了。
最近几天应该会实现，实现完了再分享吧。