```lw-blog-meta
{"title": "Paramiko实现SFTP操作", "date": "2019-07-13", "tags": ["Python"], "brev": "Paramiko(http://www.paramiko.org/)是python环境下实现SFTP的常用第三方库。支持SSHv2，底层使用C语言拓展，但是编写的时候是完全的python接口。"}
```

## 需求

公司要每天从另一家公司下载数据，对方提供了一个sftp环境供我们下载。

但这次给的是**账号+密码**的形式，而不是密钥文件的形式，所以传统的`shell`+`sftp`组合不好使了。
所以我们这次用python脚本来替代shell。

## 实现

按照以往的经验来考虑，我们要实现一个命令行参数指定日期的功能。类似于这种感觉：

```shell
$ python xxxsftp.py --since 20190710 --today 20190713
```

在此思想指导下，完成主体逻辑部分：

```python
# ------------------------- Functions -------------------------
# 入口，给两个日期参数
def Main(since: datetime, today: datetime):
    username = '***'
    password = '***'
    server = ("***", 22)

    with paramiko.Transport(server) as transport:
        transport.connect(username=username, password=password)
        with paramiko.SFTPClient.from_transport(transport) as sftp:
            while since <= today:
                doGET(since, sftp)
                since += timedelta(1)ya

# 主体逻辑部分
def doGET(today: datetime, sftp: paramiko.SFTPClient):
    remote_path = "/download/GEN/%s/" % today.strftime("%Y%m%d")
    local_path = "/home/***"
    # 有可能当天没有文件夹
    try:
        sftp.chdir(remote_path)
    except Exception as e:
        logger.error("Failed when chdir [%s]: %s" % (remote_path, e))
        return
    # 文件夹存在，就下载所有文件
    for remote_filename in sftp.listdir():
        local_filepath = os.path.join(local_path, today.strftime("%Y%m%d") + "_" + remote_filename)
        try:
            sftp.get(remote_filename, local_filepath)
        except Exception as e:
            logger.error("Failed when get %s: %s" % ([remote_filename, local_filepath], e))
        else:
            logger.info("Success get %s" % ([remote_filename, local_filepath],))

```

> `logger`对象是我之前写的`logging`模块的一个简单类`Logger_Easy_Time`，
> 我这个模块投入了大量心血，并构建了很多测试，相当好用，详情请点击或搜索：
> [Saodd/LewinTools](https://github.com/Saodd/LewinTools/blob/master/lewintools/base/logging.py#L137)

然后实现main部分：

```python
# ------------------------- Main -------------------------
if __name__ == "__main__":
    import argparse

    PARSER = argparse.ArgumentParser(
        description="""none""")
    PARSER.add_argument('--since', type=str, default="", help="Loop 'since' to 'today'. format 20190131.")
    PARSER.add_argument('--today', type=str, default=(datetime.now() - timedelta(1)).strftime('%Y%m%d'),
                        help='Default yesterday. format 20190131.')
    ARGS = PARSER.parse_args()
    today = datetime.strptime(ARGS.today, "%Y%m%d")
    since = datetime.strptime(ARGS.since, "%Y%m%d") if len(ARGS.since) else today
    # run
    Main(since, today)
```

## 小结

`Python`在运维和自动任务方面还是很强大的。虽然在计算性能上会有相当的损失，不过这并不是它专长之处，
扬长避短就好，不要死磕。

打个比方的话，python就像是王牌杀手，什么都会做，但是一群杀手很可能会乱哄哄的；
golang就像是集团军，兄弟们分工协作组成一个更强大的集团。


## 使用密钥登录（20191014更新）

使用密钥证书总比写下明文密码好多了。

我们首先生成ssh密钥，注意，`paramiko`默认支持的是`pem`格式的密钥，因此我们在生成的时候要添加参数
([参考](https://gist.github.com/batok/2352501#gistcomment-2811353))：

```shell-session
$ ssh-keygen -m pem -t rsa -C "test"
```

然后我们把公钥放在服务器上（也就是在服务器上装个门），然后把私钥放在你需要访问文件的地方（带着钥匙）并通过`paramiko`来使用：

```python
def login() -> paramiko.SFTPClient:
    key = paramiko.RSAKey.from_private_key_file("/ap/ApmosReconcile/utils/pk")

    ts = paramiko.Transport(self.server)
    ts.connect(username="lewin", pkey=key)

    sftp = paramiko.SFTPClient.from_transport(ts)
    return sftp
```

示例用法：

```python
@_limit_path
def sftp_get(path: str, force=False) -> str:
    """
    原来：f = open("/some/file")
    现在：f = open(sftp_get("/some/file"))
    函数会帮你下载相应的文件到本地容器里。
    """
    if not os.path.exists(path) or force:
        sftp = login()
        os.makedirs(os.path.dirname(path), exist_ok=True)
        sftp.get(path, path)
    return path
```
