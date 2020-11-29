```yaml lw-blog-meta
title: "FastAPI 源码速读"
date: "2020-10-08"
brev: ""
tags: [Python]
```

## 基本概念

FastAPI 其实和 Gin, Flask 做的事情是一样的——只是一套路由系统罢了。更底层的网络实现是依赖于 Starlette （类比于 Werkzerk），产品用服务引擎是 Uvicorn （类比于 Gunicorn），以及类型系统是依赖于 Pydantic 。

[Starlette](https://www.starlette.io/) 和 [Uvicorn](https://www.uvicorn.org/) 都是轻量级的 [ASGI (Asynchronous Server Gateway Interface)](https://asgi.readthedocs.io/en/latest/)  服务器的实现。

我们按照引擎、路由、类型系统的顺序来快速看一下它们的实现风格。

最基本的启动步骤是，先实例化一个app（我称之为引擎对象/路由对象），然后挂载到网络框架中去run：

```python
app = FastAPI()
uvicorn.run(app, host="0.0.0.0", port=8000)
```

## 1. 启动服务 Uvicorn

```python
# 来自 uvicorn/main.py
def run(app, **kwargs):
    config = Config(app, **kwargs)
    server = Server(config=config)
    # 中间简化
    server.run()
```

`Config`的作用主要就是保存`**kwargs`中的内容，然后做一些转化以及默认值的设置。注意的是它把 app （路由）也放在这一层了。

`Server`则更像是一个运行时的状态。

```python
class Server:
    def run(self, sockets=None):
        self.config.setup_event_loop()
        loop = asyncio.get_event_loop()
        loop.run_until_complete(self.serve(sockets=sockets))
```

在上面的`setup_event_loop()`中，我们会通过字符串来动态加载一个模块，这个字符串的默认值是"uvicorn.loops.auto:auto_loop_setup"。深入追查我们最后可以看到其实就是`asyncio`的基本用法：

```python
def asyncio_setup():
    # 此处简化
    loop = asyncio.new_event_loop()
    asyncio.set_event_loop(loop)
```

由于这是一个基于协程的框架，因此在后续的代码中都是通过`asyncio.get_event_loop()`来获取当前的协程事件循环。

然后在`serve()`方法中可以看到引擎启动的主要流程：

```python
class Server:
    async def serve(self, sockets=None):
        # 简化
        await self.startup(sockets=sockets)
        if self.should_exit:
            return
        await self.main_loop()
        await self.shutdown(sockets=sockets)
```

### 1.1 startup阶段

在startup阶段，会根据config中的配置来决定监听的套接字。默认情况下会绑定一个端口进行监听：

```python
class Server:
    async def startup(self, sockets=None):  # 简化版本
        # 先执行路由中定义的 startup 函数们
        await self.lifespan.startup()
        # 然后绑定一个套接字（端口）
        loop = asyncio.get_event_loop()
        try:
            server = await loop.create_server(
                create_protocol,
                host=config.host,
                port=config.port,
                ssl=config.ssl,
                backlog=config.backlog,
            )
        except OSError as exc:
            logger.error(exc)
            await self.lifespan.shutdown()
            sys.exit(1)
```

TODO：那么这个 `lifespan`是什么呢，我们后面再说。

### 1.2 main_loop阶段

在这个阶段，在这个函数中并不会做什么事情，而仅仅是观察`should_exit`这个退出信号变量而已：

```python
class Server:
    async def main_loop(self):
        counter = 0
        should_exit = await self.on_tick(counter)
        while not should_exit:
            counter += 1
            counter = counter % 864000
            await asyncio.sleep(0.1)
            should_exit = await self.on_tick(counter)

    async def on_tick(self, counter) -> bool:  # 简化
        # Update the default headers, once per second.
        if counter % 10 == 0:
            self.server_state.default_headers = [...]
        # Determine if we should exit.
        if self.should_exit:
            return True
        return False
```

主要思路呢，就是每睡一次0.1秒，检查一次是否准备退出；每睡十次0.1秒，检查一下当前的时间。

注意睡眠是异步睡眠，会抛出控制权给其他协程来处理请求等工作。

关于这里的实现我个人觉得非常非常的别扭，毕竟，在这么大的框架中使用sleep无限循环真的好吗？而且这里并没有直接处理请求。特别是相比于 Golang 的 Gin 框架的做法，这个 main_loop 阶段的实现我觉得非常的丑陋。或许这就是 python协程 的代价吧。

用sleep来计时的话，在高负载情况下，"睡十次"的时间可能会远远地超过1秒，这里很容易埋坑。

### 1.3 shutdown阶段

由于前面的无限循环阻塞，因此进入这个阶段的必要条件是`should_exit`为正值。并且可能附带一个`force_exit`的情况。

shutdown流程有五个步骤：

1. 关闭引擎和端口，停止接受新连接
2. 关闭现有的连接
3. 等待现有链接的响应值全部发出去
4. 等待剩余的异步任务
5. 执行`lifespan`中的shutdown钩子们

```python
class Server:
    async def shutdown(self, sockets=None):
        logger.info("Shutting down")

        # Stop accepting new connections.
        for server in self.servers:
            server.close()
        for sock in sockets or []:
            sock.close()
        for server in self.servers:
            await server.wait_closed()

        # Request shutdown on all existing connections.
        for connection in list(self.server_state.connections):
            connection.shutdown()
        await asyncio.sleep(0.1)

        # Wait for existing connections to finish sending responses.
        if self.server_state.connections and not self.force_exit:
            msg = "Waiting for connections to close. (CTRL+C to force quit)"
            logger.info(msg)
            while self.server_state.connections and not self.force_exit:
                await asyncio.sleep(0.1)

        # Wait for existing tasks to complete.
        if self.server_state.tasks and not self.force_exit:
            msg = "Waiting for background tasks to complete. (CTRL+C to force quit)"
            logger.info(msg)
            while self.server_state.tasks and not self.force_exit:
                await asyncio.sleep(0.1)

        # Send the lifespan shutdown event, and wait for application shutdown.
        if not self.force_exit:
            await self.lifespan.shutdown()
```

## 2. 路由

`APIRouter`就是路由组的实现，可以用`include_router`来进行多级嵌套。

get, post 等方法都是`api_route`方法的快捷方式，通过装饰器调用`add_api_route`方法来把路由函数注册到路由组中，注册后的形态是`APIRoute`类的实例并且保存在`APIRouter.routes`列表中。