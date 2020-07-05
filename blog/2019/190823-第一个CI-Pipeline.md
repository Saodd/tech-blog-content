```yaml lw-blog-meta
title: 第一个CI-Pipeline
date: "2019-08-23"
brev: 使用测试机上已经build好的image，来实现一个真正的测试项目。
tags: [DevOps]
```


## 准备Image

`runners.docker`的运行原理实际上就是实例化一个容器，然后在容器内部运行项目代码。

而我们的Python项目肯定会有大量的第三方库依赖的，不可能运行在一个标准的`python:latest`镜像容器中，因此我们肯定要为项目构建一个有足够依赖环境的镜像。

当然一般说来，我们应当有一个`Docker-registry`搭建的私有镜像服务器，然后无论是测试机还是线上机都应该从这个私有仓库中pull下镜像来用。

但是我们公司目前还在探索阶段（只有我一个人在探路），所以先试一下在测试机上单独build一个镜像然后使用的方法。

我们写一下`Dockerfile`，这里为了提高国内下载速度，使用了清华镜像；然后通过文件指定我们所有的依赖库。

```dockerfile
FROM python:3.7.4

COPY . /home/docker/

RUN pip install -i https://pypi.tuna.tsinghua.edu.cn/simple -r /home/docker/pip0001.txt --no-cache-dir && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

WORKDIR /scripts/

CMD bash
```

然后构建（注意命令最后的点）：

```shell
docker build -f Dockerfile0001 -t appython:0001 .
```

这样我们就拥有了一个tag为`appython:0001`的镜像了。

## 配置runner

GitLab-runner有点坑，因为照着官方文档做下去的话，很难注意到他的pull政策。

因为runner默认是调用`docker pull`命令的，所以本地就算已经构建好了镜像，也会被忽略（你可以手动运行一下`docker pull 镜像名`，会发现docker会尝试去docker-hub拉取而不会用本地的）

因此我们要对runner进行设置。根据[官方文档](https://docs.gitlab.com/runner/configuration/advanced-configuration.html#the-runnersdocker-section)的说明，
我们要设置一个`pull_policy`参数：

> pull_policy : Specify the image pull policy: never, if-not-present or always (default);

我们用`docker exec`从原来的容器中复制出默认配置文件，以免造成意外后果。加入`pull_policy`后是这样的：

```toml
concurrent = 1
check_interval = 0

[session_server]
  session_timeout = 1800

[[runners]]
  name = "sz-242server"
  url = "http://gitlab.apcapital.local/"
  token = "马赛克"
  executor = "docker"
  [runners.custom_build_dir]
  [runners.docker]
    tls_verify = false
    image = "python:3.7.4"
    privileged = false
    disable_entrypoint_overwrite = false
    oom_kill_disable = false
    disable_cache = false
    volumes = ["/cache"]
    shm_size = 0
    pull_policy = "if-not-present"
  [runners.cache]
    [runners.cache.s3]
    [runners.cache.gcs]
  [runners.custom]
    run_exec = ""
```

然后我们要关掉这个容器，重新启动一个并挂载这个配置文件进去：

```shell
docker run -d --name gitlab-runner --restart always \
-v /var/run/docker.sock:/var/run/docker.sock \
-v /home/users/lewin/docker/config.toml:/etc/gitlab-runner/config.toml  \
-it gitlab/gitlab-runner:latest 
```

原来在GitLab上的配对信息会被保留（因为写在配置文件中了），不需要重新去网页中设置。

## 安排测试脚本

我们写一段单元测试代码，保存为`_test/SEHK_Short_Report.py`：

```python
import os
import sys
import unittest
import logging

from Broker_APMOS.OtherTools import SEHK_Short_Report

class MyTestCase(unittest.TestCase):
    def test__main__got_none(self):
        dfs = SEHK_Short_Report.main("2000-01-01")
        self.assertEqual(len(dfs["ARF"]), 0)
        self.assertEqual(len(dfs["APC"]), 0)

    def test__main__given_date(self):
        dfs = SEHK_Short_Report.main("2019-08-24")
        self.assertEqual(len(dfs["ARF"]), 1)
        self.assertEqual(len(dfs["APC"]), 1)

if __name__ == '__main__':
    unittest.main()
```

然后把这个脚本写入`.gitlab-ci.yml`文件中。（TIPS：我们可以在文件中定义系统环境变量，这样我们就可以很容易地在测试环境与生产环境配置中切换了）

```yaml
image: appython:0001

variables:
  AP_ENV: "test"
  AP_LOG: "warning"

SEHK_Short_Report:
  script:
    - python _test/SEHK_Short_Report.py
```

## 推送代码观察结果

安排好以上设置后，我们什么都不需要做，只需要写代码-add-commit-push，推送上去之后GitLab会自动生成Pipeline任务，帮我们完成测试任务：

![settings](/static/blog/2019-08-23-first-CI.png)

这样我们就可以看到测试结果了，并且可以看到详细的输出，帮助定位bug。

如果分支功能开发完成了，此时我们可以发起`pull request`，我们可以清晰地看到我们的commit是经过测试的：

![settings](/static/blog/2019-08-23-branch.png)

接受merge请求，成功合并到原始分支中！

![settings](/static/blog/2019-08-23-merge.png)
