```yaml lw-blog-meta
title: Gitlab-CE 设置SMTP邮箱服务
date: "2020-03-13"
brev: 在内网快速搭建了一个Gitlab平台。很多功能都依赖于电子邮箱，所以内建的Gitlab必须要能够发送邮件。选择外部的免费邮箱并在本地配置SMTP是一种比较稳妥的选择。
tags: [DevOps]
```


## 配置

我使用的是Docker官方镜像，配置文件在容器内的`/etc/gitlab/gitlab.rb`位置。使用vim打开编辑，搜索`smtp`和`gitlab_email`等关键字，写入以下参数：

```text
# smtp账户配置
gitlab_rails['smtp_enable'] = true
gitlab_rails['smtp_address'] = "smtp.163.com"
gitlab_rails['smtp_port'] = 465
gitlab_rails['smtp_user_name'] = "lewin_xxx@163.com"
gitlab_rails['smtp_password'] = "your_password"
gitlab_rails['smtp_domain'] = "smtp.163.com"
gitlab_rails['smtp_authentication'] = "login"
gitlab_rails['smtp_enable_starttls_auto'] = true
gitlab_rails['smtp_tls'] = true
```

注意，以上配置适用于163免费邮箱，其他公司邮箱可能配置略有不同（比如端口、验证方式等）。

```text
# 邮件配置
gitlab_rails['gitlab_email_enabled'] = true
gitlab_rails['gitlab_email_from'] = 'lewin_xxx@163.com'
gitlab_rails['gitlab_email_display_name'] = 'XXX-GitlabCE'
gitlab_rails['gitlab_email_reply_to'] = 'lewin_xxx@163.com'
```

修改完毕后保存。然后要重启gitlab（容器）。

## 发送测试邮件

在容器内运行控制台：

```shell-session
gitlab-rails console
......(启动需要一点时间)
irb(main):001:0> Notify.test_email('66666666@qq.com', '邮件标题', '邮件正文').deliver_now
```

## 错误处理

第一可能遇到验证失败。先进入你的邮箱账户，看看smtp(imap)权限有没有开启。对于163邮箱来说，必须要设置授权码才可以使用smtp功能。

第二可能遇到邮件拒收。这个是邮件内容的问题，一般是from字段没有设置好。

```text
Net::SMTPFatalError (553 Mail from must equal authorized user)
```

## 其他方式

其实理论上可以让gitlab直接投递到目标邮箱服务器，不用外部的邮箱代理。比如我能收到一封发自`GitLab <gitlab@192.168.1.92>`的邮件，看起来不错。

我尝试去配置了一下，没有成功。暂时作罢，以后有需要了再研究一下吧。依靠外部邮箱还是更靠谱一些，自建邮箱不排除会被当做垃圾邮件的可能性。

## 参考

官方文档： [SMTP settings](https://docs.gitlab.com/omnibus/settings/smtp.html)

参考文章： [Gitlab之邮箱配置-yellowocng](https://blog.csdn.net/yelllowcong/article/details/79939589)
