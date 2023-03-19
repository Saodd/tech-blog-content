```yaml lw-blog-meta
title: "重新安装k8s套件"
date: "2023-03-19"
brev: "将 kubeadm, kubelet, kubectl 修复并升级到最新的 v1.26.3 版本"
tags: ["运维"]
keywords: "k8s,Kubernetes,install"
```

## 前言

k8s的整套体系十分的繁杂，想要精通或者至少说熟练使用它，是需要大量的学习和长时间的实践总结才能达到的。就我目前的水平而言，充其量算个“熟悉常规概念、会用基本操作”，虽然对于无论是我们公司还是我个人项目来说，k8s都是一把大得夸张的屠龙刀，但我任然坚持学习使用它，哪怕是皮毛，希望能够保持我的运维技术不掉队太远。

从我2021年初（[《k8s入坑记》](../2021/210206-learn-k8s.md)）正式将其运用到个人博客网站以来，已经过去2年时间了，k8s相关的组件也有三四个版本的迭代了。这几天在配置Drone（[《用Drone配置CI流程》](../2023/230318-drone-CI.md)），配置完了Docker部分之后，还是想要继续深入试试k8s的配置，然后突然发现我开发机上的k8s集群不知道从什么时候开始早就停止运行了。

经过简单尝试未能将我开发机上的k8s集群顺利启动起来，因此我打算彻底地维护一次。操作过程记录成了这篇文章。

## 整体结构

k8s有三大核心组件：

`kubeadm`：用来初始化集群的指令。（只需要在集群中的个别管理机器上安装即可）

`kubelet`：在集群中的每个节点上用来启动 Pod 和容器等。（每个节点机器都要安装！）

`kubectl`：用来与集群通信的命令行工具。（在任何你需要调用API的地方都可以安装，它只是一个简单的命令行工具而已，就像`curl`一样）

## 安装

其实“用来初始化集群的指令”工具一共有三种，分别是：

- kind （单机环境，基于Docker）
- minikube （单机环境，也是官方教程推荐的选择）
- kubeadm （生产集群环境）

kubeadm是最重的，但也是最接近生产的，因此我不计代价选择使用它来作为我的学习步骤。

根据[安装 kubeadm / 安装 kubeadm、kubelet 和 kubectl](https://kubernetes.io/zh-cn/docs/setup/production-environment/tools/kubeadm/install-kubeadm/#installing-kubeadm-kubelet-and-kubectl)章节中，『基于Debian的发行版』页签下的命令稍微有些坑，我重新记录一下：

1、更新 apt 包索引并安装使用 Kubernetes apt 仓库所需要的包：

```shell
sudo apt-get update
sudo apt-get install -y apt-transport-https ca-certificates curl  # 以前装过的话现在就不用再装了
```

在执行`apt-get update`这一步的时候，如果以前配置过 Kubernetes apt 仓库，那这一步会出现警告：

```text
W: An error occurred during the signature verification. The repository is not updated and the previous index files will be used. GPG error: https://packages.cloud.google.com/apt kubernetes-xenial InRelease: The following signatures couldn't be verified because the public key is not available: NO_PUBKEY B53DC80D13EDEF05
W: Failed to fetch https://apt.kubernetes.io/dists/kubernetes-xenial/InRelease  The following signatures couldn't be verified because the public key is not available: NO_PUBKEY B53DC80D13EDEF05
```

意思就是证书过期了，需要先执行下面的第二步：

2、下载 Google Cloud 公开签名秘钥：

```shell
sudo curl -fsSLo /etc/apt/keyrings/kubernetes-archive-keyring.gpg https://packages.cloud.google.com/apt/doc/apt-key.gpg
```

这一步可能会报错：

```text
curl: (23) Failed writing body (0 != 1210)
```

问题的原因是下载文件的目录是不存在的，解决方案是先创建那个目录再执行curl进行下载：

```shell
mkdir -p /etc/apt/keyrings/
```

3、添加 Kubernetes apt 仓库：

```shell
echo "deb [signed-by=/etc/apt/keyrings/kubernetes-archive-keyring.gpg] https://apt.kubernetes.io/ kubernetes-xenial main" | sudo tee /etc/apt/sources.list.d/kubernetes.list
```

4、更新 apt 包索引，安装 kubelet、kubeadm 和 kubectl，并锁定其版本：

```shell
sudo apt-get update
# 如果是更新，需要先执行 apt-mark unhold kubelet kubeadm kubectl
sudo apt-get install -y kubelet kubeadm kubectl # kubectl如果已经安装则可以省去
sudo apt-mark hold kubelet kubeadm kubectl
```

执行到这一步的时候，其实已经把 kubelet kubeadm kubectl 三个东西全部安装好了，当前最新版本号是`1.26.3-00`

### 关于kubectl

在前面的步骤中已经统一通过`apt`安装好了kubectl。

除此之外如果需要单独安装它的地方，还可以用更简单的方式，参考： [在 Linux 系统中安装并设置 kubectl](https://kubernetes.io/zh-cn/docs/tasks/tools/install-kubectl-linux/) ，简而言之就是用`curl`下载一个二进制文件然后放在系统中可以访问到的地方。

此外值得一提的是，用Docker镜像（[bitnami/kubectl](https://hub.docker.com/r/bitnami/kubectl)）也完全可以替代这个原生的二进制文件。

### 配置cgroup

参考：[《配置 cgroup 驱动》](https://kubernetes.io/zh-cn/docs/tasks/administer-cluster/kubeadm/configure-cgroup-driver/)

其中提到`kubeadm`默认使用`systemd`驱动，因此我们无需自行操作配置。

### 配置容器运行时

容器运行时有三种，分别是：

- containerd （k8s原生支持的运行时）
- CRI-O （默认值，是k8s用于兼容其他运行时的协议标准）
- Docker Engine （用于兼容Docker）

关于三种运行时之间的区别可以参考阅读：[《The differences between Docker, containerd, CRI-O and runc》](https://www.tutorialworks.com/difference-docker-containerd-runc-crio-oci/)

我们选择原生的`containerd`。而这个东西似乎不用专门安装，之前安装过Docker的话应该自动就装好了。

参考：[容器运行时 / containerd](https://kubernetes.io/zh-cn/docs/setup/production-environment/container-runtimes/#containerd) 章节，我们需要对containerd做一些配置。

## 创建集群

为了避免拉取镜像时可能遇到的网速问题，首先配置一下从代理拉取镜像：

```shell
kubeadm config images pull --image-repository=registry.cn-hangzhou.aliyuncs.com/google_containers
```

（其他的方法，例如设置`http_proxy`环境变量、设置`/etc/systemd/system/docker.service.d/http-proxy.conf`等方式都不好用，就用上面的命令最简单有效）

然后执行一条简单的命令正式创建集群：

```shell
kubeadm init
```

### 创建集群报错-1

执行过程中可能会报错：

```text
W0319 06:20:46.097967 1462101 version.go:104] could not fetch a Kubernetes version from the internet: unable to get URL "https://dl.k8s.io/release/stable-1.txt": Get "https://dl.k8s.io/release/stable-1.txt": EOF
W0319 06:20:46.098553 1462101 version.go:105] falling back to the local client version: v1.26.3
[init] Using Kubernetes version: v1.26.3
[preflight] Running pre-flight checks
error execution phase preflight: [preflight] Some fatal errors occurred:
        [ERROR CRI]: container runtime is not running: output: E0319 06:20:46.292752 1462122 remote_runtime.go:948] "Status from runtime service failed" err="rpc error: code = Unimplemented desc = unknown service runtime.v1alpha2.RuntimeService"
time="2023-03-19T06:20:46Z" level=fatal msg="getting status of runtime: rpc error: code = Unimplemented desc = unknown service runtime.v1alpha2.RuntimeService"
, error: exit status 1
[preflight] If you know what you are doing, you can make a check non-fatal with `--ignore-preflight-errors=...`
To see the stack trace of this error execute with --v=5 or higher
```

解决方案其实在[《配置 systemd cgroup 驱动》](https://kubernetes.io/zh-cn/docs/setup/production-environment/container-runtimes/#containerd-systemd)章节中已经提示过了，核心问题是出在文件中的`disabled_plugins = ["cri"]`这一行上面，把`cri`取消禁用即可解决。

### 创建集群报错-2

执行过程中还可能报错，大意是提示kubelet没有在正确运行。然后查看kubelet的日志可以发现：

```text
service connection: CRI v1 runtime API is not implemented for endpoint \"unix:///var/run/containerd/containerd.sock\": rpc error: code = Unimplemented desc = unknown service runtime.v1.RuntimeService"
```

参考[这个帖子](https://serverfault.com/questions/1118051/failed-to-run-kubelet-validate-service-connection-cri-v1-runtime-api-is-not-im)可知是当前`1.26.3`的kubelet要求`1.6+`版本的containerd，而我当前机器上安装的是`1.4.3`版本的containerd 。

然后升级containerd的时候还遇到版本不兼容的问题，参考[这个帖子](https://unix.stackexchange.com/questions/724518/the-following-packages-have-unmet-dependencies-containerd-io)，把docker整个重装一遍最新版本。

安装完成之后再看kubelet，此时它不断提示错误『cni plugin not initialized』，这个状态就可以（先`kubeadm reset`然后再）继续重新运行`kubeadm init`了。

### 配置CNI

此时的k8s集群是啥都没有的，甚至连必需的内部网络通信插件（CNI）都没有装。

官方建议了一批[CNI列表](https://kubernetes.io/zh-cn/docs/concepts/cluster-administration/addons/)，其中比较流行的是`Calico`和`Flannel`，我之前一直选用的是前者所以这次也装它。

但是要注意`Calico`有很多种安装配置，我尝试了几种，唯一每次都能成功的只有[Install Calico networking and network policy for on-premises deployments](https://docs.tigera.io/calico/3.25/getting-started/kubernetes/self-managed-onprem/onpremises) 这个章节中的『Manifest』标签页下的安装方式，即运行：

```shell
curl https://raw.githubusercontent.com/projectcalico/calico/v3.25.0/manifests/calico.yaml -O
kubectl apply -f calico.yaml
```

稍等它运行大概几分钟的时间，然后再检查`k get node`确定当前的机器节点已经变成`Ready`状态了，CNI就安装成功了。

### 去除taint

如果像我一样只在单机运行k8s集群，那么在启动pod之前，还需要把当前的节点（机器）去除掉标记以允许在本机上调度运行pod资源：

```shell
# taint的名字跟之前的版本不同了，而且最后这个减号必须要有
kubectl taint nodes --all node-role.kubernetes.io/control-plane-
```

## 运行

平时维护的时候可能不会一直使用root账户，所以为了方便，可以给当前用户拷贝一份配置。这份配置是给`kubectl`使用，里面含有用户证书信息，用来鉴别用户身份的。

按照刚才`kubeadm init`命令的输出提示，运行下面的命令：

```text
mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config
```

最后随便运行一条`kubectl`的命令来检测运行是否正常：

```shell
kubectl get po
```

这下真的可以正式使用k8s来运行应用了。随便跑个最简单的pod：

```shell
k run --rm -it --image=alpine --image-pull-policy=IfNotPresent mypod -- sh
```

## 附录：apt国内镜像

国内有好几个apt镜像站点，随便搜[一篇帖子](https://www.cnblogs.com/zqifa/p/12910989.html)都可以找到全部的配置。

这里要吐槽，阿里云的镜像是限制了速度的，我这里实测速度只有200KB左右，感觉还不如人家apt官方的源。然后又切换成了清华源，速度一下子提升到几兆每秒，感受差别太大了。

不过阿里云还算良心的了，毕竟还是做了贡献的。像某些抠门的大厂（比如腾讯）甚至从来没听说过他们做过什么开源贡献。
