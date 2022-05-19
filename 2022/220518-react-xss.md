```yaml lw-blog-meta
title: "XSS小结"
date: "2022-05-18"
brev: "dangerouslySetInnerHTML"
tags: ["安全"]
```

## 小结

其实，所谓XSS（脚本注入攻击），它最核心的问题在于：**把用户数据写入了进程代码区域**。

很多动态编译语言都可能出现这种问题，例如`js`，`java`，以及`SQL`；或者某些支持指针运算从而可能导致内存溢出的语言也可能出现，例如`c++`。

## 在React中

目前我所知，在正确使用React的情况下，唯一的隐患就在于`dangerouslySetInnerHTML`这个属性。

一个典型例子（[来源](https://stackoverflow.com/questions/33644499/what-does-it-mean-when-they-say-react-is-xss-protected) ）：

```tsx
const xss = `
   <img onerror='alert("Hacked!");fetch("/api/user/me").then(console.log)' src='invalid-image' />
`

export const UserContent = () => {
  return (
    <div>
      <div dangerouslySetInnerHTML={{ __html: xss }} />
    </div>
  );
};
```

上面代码的意思是，XSS插入了一个`img`标签，由于它的`src`属性是非法的，因此一定会执行`onerror`中所声明的代码。因此攻击者可以在这里为所欲为。XSS攻击的权限等同于网站所有者所拥有的权限，与CSRF完全不同。

## 在SQL中

防御的关键在于，不要自己去拼接SQL语句，而是要把数据以**参数**形式传递给MySQL。
