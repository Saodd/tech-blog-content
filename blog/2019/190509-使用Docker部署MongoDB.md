```json lw-blog-meta
{"Title":"使用Docker部署MongoDB","Date":"2019-05-09","Brev":"用过Mysql，了解了Postgres，但归根结底二者都是关系型数据库。所以今天选择MongoDB作为Nosql的一次尝试。","Tags":["Docker","DB"]}
```



## 官方文档

首先是从官方途径[mongo - Docker hub](https://hub.docker.com/_/mongo "mongo - Docker hub")获取信息了。  

## What is MongoDB?

做一下简单的翻译：

> MongoDB is a free and open-source cross-platform document-oriented database program. Classified as a NoSQL database program, MongoDB uses JSON-like documents with schemata. MongoDB is developed by MongoDB Inc., and is published under a combination of the Server Side Public License and the Apache License.  

免费，开源，跨平台，面向文档。被归类为NoSQL数据库，使用类似JSON的格式（BSON）。


## How to use this image - 1

最简单的形式就是直接实例化一个容器。

```shell-session
$ docker run --name some-mongo -d mongo:tag
```

其中：

- `some-mongo` 是你给这个实例起的名字，这个名字可以用来与其他容器进行交互。
- `-d` 是作为服务启动，即挂在后台。
- `mongo:tag` *mongo*是这个`repository`的名字，*tag*是版本号，合在一起就定义了这个`image`。例如*mongo:latest*, *mongo:4.0.9*  


## How to use this image - 2

使用Docker stack来组合使用。先新建一个配置文件：

```yaml
# 在任意位置新建一个 stack.yml
# Use root/example as user/password credentials
version: '3.1'

services:

  mongo:
    image: mongo
    restart: always
    environment:
      MONGO_INITDB_ROOT_USERNAME: root
      MONGO_INITDB_ROOT_PASSWORD: example

  mongo-express:
    image: mongo-express
    restart: always
    ports:
      - 8081:8081
    environment:
      ME_CONFIG_MONGODB_ADMINUSERNAME: root
      ME_CONFIG_MONGODB_ADMINPASSWORD: example

```

这里配置了两个container。一个叫*mongo*，一个叫做*mongo-express*（在web上管理mongo的项目）；并且初始化了默认用户`root`和密码`example`。  

运行docker命令启动这项`stack`并命名为*mongodb*

```shell-session
$ docker stack deploy -c stack.yml mongodb
Ignoring unsupported options: restart
Creating network mongodb_default
Creating service mongodb_mongo-express
Creating service mongodb_mongo
```

查看已经创建的`stack`：

```shell-session
$ docker stack ls
NAME                SERVICES            ORCHESTRATOR
mongodb             2                   Swarm

$ docker service ls
ID                  NAME                    MODE                REPLICAS            IMAGE                  PORTS
nudil678wgt9        mongodb_mongo           replicated          1/1                 mongo:latest           *:27017->27017/tcp
gpcfkym0rter        mongodb_mongo-express   replicated          1/1                 mongo-express:latest   *:8081->8081/tcp

$ docker container ls
CONTAINER ID        IMAGE                  COMMAND                  CREATED             STATUS              PORTS               NAMES
d2cec79f67e2        mongo-express:latest   "tini -- /docker-ent…"   18 minutes ago      Up 18 minutes       8081/tcp            mongodb_mongo-express.1.utv18p6if1pqdjs6ab93gbldl
029472c4445c        mongo:latest           "docker-entrypoint.s…"   18 minutes ago      Up 18 minutes       27017/tcp           mongodb_mongo.1.pdjv597fnm2l8hkvp7q26amng
```

然后就可以在浏览器[本地8081端口](http://127.0.0.1:8081/) 访问Mongo-Express了。
此时应该可以看到3个`database`，分别叫*admin*, *config*和*local*。  
但是现在整个mongo中只有一个用户，就是我们在配置文件中设置的*root*用户，一般来说我们肯定要创建一个普通用户，这一步需要在命令行中执行。  






## 在容器中建表插入

首先切换到正在运行的容器中执行命令(根据之前查看到的serviceid)。

```bash
$ docker exec -it 029472c4445c bash
root@029472c4445c:/#
```

然后登录mongo

```bash
root@029472c4445c:/# mongo

MongoDB shell version v4.0.9
connecting to: mongodb://127.0.0.1:27017/?gssapiServiceName=mongodb
Implicit session: session { "id" : UUID("9ada810f-2c04-446d-be07-d886dff690e2") }
MongoDB server version: 4.0.9
Welcome to the MongoDB shell.
For interactive help, type "help".
For more comprehensive documentation, see
        http://docs.mongodb.org/
Questions? Try the support group
        http://groups.google.com/group/mongodb-user
>
```

此时我们是看不到任何`database`的，因为mongo保留的三个数据库是隐藏的。

```bash
> show dbs
>
```

我们可以试一试能不能建库建表并插入：

```bash
> use db001
switched to db db001
> db.tb0001.insert({msg:"hahahahahaha!"})
WriteCommandError({
        "ok" : 0,
        "errmsg" : "command insert requires authentication",
        "code" : 13,
        "codeName" : "Unauthorized"
})
> show dbs
>
```

显示"Unauthorized"意思就是你还没有登录，也没有权限对数据库进行任何操作。新建的表*db001*也当然没有生效。

那么就登录吧。

```bash
> use admin
switched to db admin
> db.auth("root","example")
1
>
```

登录之后返回1，就是认证成功了。我们再来执行一下之前的`insert`操作。

```bash
> show dbs  # 以root身份此时可以看到隐藏的数据库了
admin   0.000GB
config  0.000GB
local   0.000GB
> use db001
switched to db db001
> db.tb0001.insert({msg:"hahahahahaha!"})
WriteResult({ "nInserted" : 1 })
>
```

这样就插入成功了。我们可以在`Mongo-Express`端口上查询到我们刚才的操作。



## 在容器中新增用户

保持刚才登录的*root*超级用户身份。我们新建一个用户名叫*lewin*，密码就是*pwd*，拥有对所有数据库的读写权限。

```bash
> use admin
switched to db admin
> db.createUser({
... user:"lewin",
... pwd:"pwd",
... roles:["readWrite"]
... })
Successfully added user: { "user" : "lewin", "roles" : [ "readWrite" ] }
>
```

修改用户权限：

```bash
> db.updateUser(
... "lewin",
... {pwd:"pwd",
... roles:["readWriteAnyDatabase"]
... })
> db.getUser("lewin")
{
        "_id" : "admin.lewin",
        "userId" : UUID("772396a0-0fbe-4b3c-b3cc-8749b5f37736"),
        "user" : "lewin",
        "db" : "admin",
        "roles" : [
                {
                        "role" : "readWriteAnyDatabase",
                        "db" : "admin"
                }
        ],
        "mechanisms" : [
                "SCRAM-SHA-1",
                "SCRAM-SHA-256"
        ]
}
>
```

试一试权限：

```bash
> use admin
switched to db admin
> db.auth("lewin","pwd")
1
> use db001
switched to db db001
> db.tb0001.insert({msg:"Im back!"})
WriteResult({ "nInserted" : 1 })
>
```

注意，对于*readWriteAnyDatabase*权限的用户，必须在*admin*数据库登录之后，再切换到其他数据库进行操作，才有权限。
直接在其他数据库是无法登录的。

```bash
> use db001
switched to db db001
> db.auth("lewin","pwd")
Error: Authentication failed.
0
>
```

## 小结

到此为止，`Mongo` + `Mongo-Express`的服务堆栈`stack`就构建完成了。
接下来可以使用在容器环境下开心的学习Mongo运用了。