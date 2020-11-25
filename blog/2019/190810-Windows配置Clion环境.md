```yaml lw-blog-meta
title: Windows配置Clion环境
date: "2019-08-09"
brev: 本来不想再入C的坑了，可是无奈学操作系统必须要懂C，那么就只好配置一下开发环境了。借鉴以往的经验，决定继续充分利用IDE的强大功能来帮助学习。
tags: [C]
```


## 安装C/C++环境

按照步骤的话首先是应该装好编译器环境的。

虽然C和C++我都学过一些，但是对它们的生态环境还是不太清楚。大概浏览一下，感觉C一般都是跟C++绑定的吧。
主流一般是`Microsoft VC`或者`GCC`吧，前者是Win环境后者是Linux环境。

由于讨厌`VS`那种超级臃肿的体积和不友善的用户界面，我选择`GCC`系列，所以在Windows机器上安装`MinGW`。

这篇文章：[CLion 中 的 MinGW 配置（及中文坑解决）- 刘慰](https://zhuanlan.zhihu.com/p/43680621)，
介绍的方法比较好用。比直接去MinGW的网站下载一个安装器，然后自己选择要装那些库，靠谱多了。

### 下载

我们直接去[MinGW-w64 - for 32 and 64 bit](https://sourceforge.net/projects/mingw-w64/files/Toolchains%20targetting%20Win64/Personal%20Builds/mingw-builds/5.3.0/threads-posix/seh/)，这个网站下载离线版本的压缩包，我这里选择的是`x86_64-5.3.0-release-posix-seh-rt_v4-rev0.7z`。

### 解压

下载回来需要解压。由于我厌倦了什么360压缩什么2345压缩那些烦人的广告，所以选择在Linux环境下解压。
方法很简单，跑一个`Ubuntu容器`就可以了：

```shell-session
docker run --rm -v C:/Users/lewin/mydata:/data -it ubuntu bash
apt update
apt install p7zip-full
7z x x86_64-5.3.0-release-posix-seh-rt_v4-rev0.7z -r
```

好吧，还是挺复杂的，不过我挺喜欢Docker的，折腾一下也无妨。解压后把整个文件夹拷贝到C盘中去，比如`C:\MinGW`

这样就可以了。接下来安装Clion（或者已经安装了就进入界面）。

## Clion设置

一般来说在安装Clion的时候，就会让你选择编译器的相关信息。

如果当时没有选（或者当时还没装编译器），那么也简单，在界面中找到`Settings → Build, Execution, Deployment → Toolchains`，这时Clion会自己搜索本地的编译器。

![Clion](https://saodd.github.io/tech-blog-pic/2019/2019-08-10-Clion.png)

这里可能会出现一些错误警告：

如果你是安装的原生版的`MinGW`的话：

```text
CMake Error: Generator: execution of make failed.
```

之类的，意思是告诉你`cmake`没有正常工作。如果你查看一下日志，会发现其中有一些乱码，是因为`cmake`对中文没有支持然后你系统中某些变量（比如用户名）是中文的话就让它无法正常工作了（其实我认为它是可以工作的，只是Clion在测试cmake的时候输入了中文参数导致异常）。

这种情况最好的办法就是重新安装一下，用我上面提供的链接。

另一种错误提示是：

```text
cmake project is not loaded
```

这是由于你在已有的文件夹上打开项目而不是新建一个项目，没有`CMakeLists.txt`和相关的文件，所以报警。

解决办法就是重新建立一个工程，让Clion自动生成相应的`CMakeLists.txt`文件。

当然，这只是治标不治本，最重要的还是要理解`cmake`的运行机制（为什么Clion要求你必须有cmake配置才能工作？）

这个等我深入了解了再更新吧。

## 取巧方法

如果你只是想要一个**编译器**，你不需要IDE的帮助（比如Notepad流甚至vim流），那么我觉得一个Docker容器就够了。

```shell-session
docker pull gcc
docker run --rm -v C:/host/path:/container/path -w /container/path -it gcc bash

gcc -o helloworld helloworld.c
./helloworld
```

## 小结

感慨！我第一门系统性学习的语言是Python，接着是Golang，两门语言都是在生态上有相对主流的维护社区，所以如果只是基本的应用（比如helloworld），完全可以在分分钟配置好环境，没有任何的坑而且还有大量的帮助文档。

而以前学习C/C++的时候，都是什么`Turbo C`或者`VS`环境，根本不用任何配置，编译器和编辑器就是一体的，所以也没有思考过环境配置的问题。

也许是用C的牛人太多了，谁都可以自己搞个分支，所以弄得现在的环境乱七八糟吧。至少在我这个新人眼里是绝对绝对的混乱了。
