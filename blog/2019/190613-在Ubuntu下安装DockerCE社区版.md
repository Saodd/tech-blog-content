```lw-blog-meta
{"title": "在Ubuntu下安装DockerCE社区版", "date": "2019-06-13", "tags": ["Docker"], "brev": "CentOS 通过 yum 就可以非常方便的安装了，但是在 Ubuntu 上直接安装好像有点问题，所以我们这里下载二进制文件的方式安装。"}
```

## 参考 

[官方文档](https://docs.docker.com/install/linux/docker-ce/ubuntu/#install-from-a-package).  

## Install from a package

**我们这里采用最人性化的安装方式，即通过package安装。**

1. 访问`https://download.docker.com/linux/ubuntu/dists/`，根据你的Ubuntu版本选择文件夹：  
（版本名称参考：`Xenial 16.04(LTS)`, `Bionic 18.04 (LTS)`, `Cosmic 18.10`）  
我们服务器都是用16.04，所以进入`Xenial/pool/stable/amd64/`文件夹，里面可以看到很多文件。

2. 选择版本。注意一共需要安装3个包，分别是`docker-ce`, `docker-ce-cli`, `containerd.io`, 
需要先装前面两个之后才能装第三个（好像是这个顺序，如果错了会提示你）。

3. 下载文件到本地，把三个文件都下载下来。  
    ```bash
    curl -fsSL https://下载链接 -o 保存文件名
    ```

4. 安装
    ```bash
    sudo dpkg -i 文件1 文件2 文件3
    ```
5. 测试是否正常运行
    ```bash
    sudo docker run hello-world
    ```


## 安装完成后顺便设置权限

要知道`Docker Deamon`是运行在sudo权限下的，我们要对docker或者container进行操作，
都是需要加sudo的，但是普通开发人员要如何使用docker呢？

官方推荐的方法是使用`docker`用户组。
```bash
sudo usermod -aG docker your-user
```

当然，官方同时也给出了这样做的风险:
>Warning:
>
>Adding a user to the “docker” group grants them the ability to run containers which 
can be used to obtain root privileges on the Docker host. Refer to 
[Docker Daemon Attack Surface](https://docs.docker.com/engine/security/security/#docker-daemon-attack-surface)
 for more information.

但是添加用户组之后还是不能使用docker，我们还需要一些刷新的操作，下面的操作来一波吧：
```bash
# 查找 docker 组，确认其是否存在，如果存在会显示组里的用户
cat /etc/group | grep docker 

# 重启服务
sudo service docker restart

# 切换一下用户组（刷新缓存）
newgrp - docker;

# 注销，重新登录
exit
```

搞定。
