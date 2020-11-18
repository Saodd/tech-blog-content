```yaml lw-blog-meta
title: Nginx升级Https过程
date: "2019-07-13"
brev: 今天开始折腾证书的配置问题吧。不过还好，在tx云上很方便的就申请了证书下来，直接配置到服务器就可以了。
tags: [中间件]
```


## 下载证书

看一下下载来的证书，有一批文件：

```tree
|-- Apache
|   |-- 1_root_bundle.crt
|   |-- 2_www.lewinblog.com.crt
|   `-- 3_www.lewinblog.com.key
|-- IIS
|   |-- keystorePass.txt
|   `-- www.lewinblog.com.pfx
|-- Nginx
|   |-- 1_www.lewinblog.com_bundle.crt
|   `-- 2_www.lewinblog.com.key
|-- Tomcat
|   |-- keystorePass.txt
|   `-- www.lewinblog.com.jks
`-- www.lewinblog.com.csr
```

可以看到，证书密钥是给每种代理服务器都相应地生成好了。
我们这里是用的`Nginx`，所以只需要取出Nginx文件夹里的东西就好了。

```tree
.../nginx/
|-- crt
|   |-- 1_www.lewinblog.com_bundle.crt
|   `-- 2_www.lewinblog.com.key
|-- html
|   |-- favicon.ico
|   `-- index.html
`-- nginx.conf
```

## 配置Nginx

按照[官方](https://cloud.tencent.com/document/product/400/35244)的指引，
我们对`nginx.conf`文件做以下修改：

```conf
server {
         listen 443 ssl; # 注意，这里是新版的写法，腾讯云提供的是旧的
         server_name www.lewinblog.com; 
         ssl_certificate /etc/nginx/crt/1_www.lewinblog.com_bundle.crt;#证书文件位置
         ssl_certificate_key /etc/nginx/crt/2_www.lewinblog.com.key;#私钥文件位置
         ssl_session_timeout 5m;
         ssl_protocols TLSv1 TLSv1.1 TLSv1.2; #请按照这个协议配置
         ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:HIGH:!aNULL:!MD5:!RC4:!DHE;#请按照这个套件配置
         ssl_prefer_server_ciphers on;

        # 这里转发所有的"/dj/..."
        location /dj {
            include  uwsgi_params;
                     uwsgi_pass  unix:///docker/BlogDj/uwsgi.sock;
        }

server {
        listen       80;
        server_name  www.lewinblog.com;
        rewrite ^(.*)$ https://$host$1 permanent; #把http的域名请求转成https
    } 
```

但是由于我们使用的是`Docker`容器运行`Nginx`，所以在挂载配置文件的时候一定要小心！
（`build`方式也有类似的问题）

因为`Nginx`容器默认从`/etc/nginx/`目录下读取配置文件，我们很容易不小心写成：

```shell-session
$ docker run -v /host/path/nginx/:/etc/nginx/ -dit nginx
```

这样很危险！因为`Nginx`容器在`/etc/nginx/`还有很多其他的文件：

```shell-session
root@e846a699993f:/# ls /etc/nginx/
conf.d  fastcgi_params  koi-utf  koi-win  mime.types  modules  nginx.conf  scgi_params   wsgi_params  win-utf
```

我们看一下**主要配置**文件`nginx.conf`的内容：

```conf
root@bd82a743e1ae:/# cat /etc/nginx/nginx.conf

user  nginx;
worker_processes  1;

error_log  /var/log/nginx/error.log warn;
pid        /var/run/nginx.pid;


events {
    worker_connections  1024;
}


http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;

    log_format  main  '$remote_addr - $remote_user [$time_local] "$request" '
                      '$status $body_bytes_sent "$http_referer" '
                      '"$http_user_agent" "$http_x_forwarded_for"';

    access_log  /var/log/nginx/access.log  main;

    sendfile        on;
    #tcp_nopush     on;

    keepalive_timeout  65;

    #gzip  on;

    include /etc/nginx/conf.d/*.conf;  ## 注意这一行！
}
```

我们有两种选择：

1. 把`server`配置作为子文件，放进`/etc/nginx/conf.d/`目录下；
2. （推荐）直接覆盖`nginx.conf`本体，但是必须包含上述的内容。

选择一种方法就好了。写好配置文件后，重启容器：

```shell-session
$ docker restart nginx
```

没有意外的话，这个时候已经成功升级到https了。不需要修改代理服务器后面的Web服务器（比如`uWSGI`）。

你可以访问[我的网站](http://www.lewinblog.com)看一下。

## 可能遇到的坑

### 忘记开端口

我的网页已经从http跳转到https了，但是却没有任何响应了，为什么！

通过查看nginx日志我发现，原来是忘记开启`443端口`了。因为https访问的不再是`80端口`了。

我们把容器的启动命令完整地修改一下：

```shell-session
$ docker run --name nginx -v /docker/:/docker/   \
-v /home/lewin/github/BlogDj/collectstatic/:/static/dj/   \
-v /home/lewin/github/BlogDj/settings/nginx/nginx.conf:/etc/nginx/nginx.conf   \
-v /home/lewin/github/BlogDj/settings/nginx/html/:/etc/nginx/html/   \
-v /home/lewin/github/BlogDj/settings/nginx/crt/:/etc/nginx/crt/     \
--net=lewin -p 80:80 -p 443:443   \
-dit nginx
```

这个命令真的是特别长了……而且感觉以后还会更长……不过也没有办法了，保存在一个文件里随着git走吧。
这个命令本身改的也不多。平时只要`restart`就可以了，不用完整的命令。

### 忘记改websocket地址

随着http升级为https，我们之前设置的`websocket`也要从ws升级为wss，自己根据情况修改一下html就好了。

## 小结

折腾了一回证书问题，在排除bug的过程中对`Nginx`的了解又多了一分。