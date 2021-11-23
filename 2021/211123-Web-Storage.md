```yaml lw-blog-meta
title: "WEB前端存储技术"
date: "2021-11-23"
brev: "关键字: Cookie Storage IndexedDB"
tags: ["前端"]
```

## 导读

`cookie`我们都很熟悉了，由于HTTP设计上就是无状态的，所以必须依赖它来保持状态，典型的就是用户的登录/认证状态。而作为一项上古时代流传下来的技术，它有很多缺陷，也有很多限制，于是后来有了更安全、容量更大的`sessionStorage`和`localStorage`。前面这些都是简单的Key-Value型存储，然后又有了数据库型存储方案`IndexedDB`。

顺带一提，`WebSQL`这个概念短暂地存在过，很快被废弃了（理由是依赖其他技术栈例如SQLite），它的替代方案就是`IndexedDB`。

## Cookie

> [MDN](https://developer.mozilla.org/en-US/docs/Web/HTTP/Cookies) 。另外，关于跨域相关的Cookie可以参考我另一篇文章 [详解CORS](../2021/210922-Dig-CORS.md) 

`cookie`是服务端储存在客户端的一小段数据。浏览器**可以**把它储存并在后续的请求中附带上。典型例子，cookie用来证明请求是否来自于同一个（登录过的）用户浏览器。

容量方面，Cookie的容量很小，只有4KB？毕竟每个请求都会带上的，不能太大了。

它的主要设计目标有三点：会话管理`Session management`，个性化`Personalization`，用户行为跟踪`Tracking`。

曾经它是唯一的客户端存储方案，但是现在有更好的替代方案。一个典型缺陷是，由于每个请求都会携带所有cookie，会导致请求体积增大，特别是在移动端等网络条件较差的环境中会表现很糟糕
（嗯其实对于5G时代来说这个无所谓了）。

### Cookie用法

服务端在响应头中设置一个或者多个`Set-Cookie`头 就行了。浏览器正常情况下会把（所有）cookie保存下来并且放在后续请求的`Cookie`头中。

你可以设置过期时间。有两种设置方式，一种设置为本次session有效，但是要注意可能有少数浏览器对待"session"的方式不一样，可能会导致一些问题。另一种更常用的是设置到期时间，在一个明确的时间点之前一直有效。

你可以设置访问限制。`Secure`要求客户端必须使用HTTPS，`HttpOnly`则屏蔽了JS的访问。

你可以限制域名和路径：

- `Domain`属性，如果忽略则只能准确匹配当前域名时才可以用，如果设置了则可以在子域名之间共享。
- `Path`属性则限制可以用的url路径，只限制前缀。
- `SameSite`属性，默认值`Lax`只有在符合`Domain`规定的情况以及导航跳转的时候才能使用这个cookie。
- 名称前缀。因为普通的cookie上不会显示来源，所以为了防御子域名的恶意行为而有了这个。

### Cookie安全

首先，最大面积的，要防御来自其他域名的攻击。中小型项目可以考虑单域名，跨域项目则好好研究CORS，无非就是上面列举的那些属性的运用规则罢了。

然后，要防御来自本域名的攻击（XSS注入攻击），以及来自子域名的攻击（这个应该很少吧，用好cookie名称前缀）。

最后，还要记得防用户！一种是用户不知情但是有具有等同用户权力的，例如浏览器插件；一种是用户故意的，例如竞争对手逆向破解代码。

总之，不要相信前端的任何东西（

### 跟踪与隐私

一般是平台类的公司会用。有些公共平台服务，例如Google广告或者百度广告，都是在其他网站上跨域访问过来的。

例如假设， 网站A 接入了 平台P 的广告，那么用户在访问 网站A 的时候，浏览器发起了向 平台P 的请求，请求过程中可以使用 P域名 的cookie；当用户再次访问 接入了平台P的网站B 的时候，再次请求 P域名 的资源，此时P域名的cookie中就可能携带有用户的历史信息（不一定在cookie里，更可能cookie只是一个标识符，数据储存在平台P的后端数据库中）。这就是所谓的跟踪用户行为。

假如我"偷偷"访问了某个见不得人的网站C，而这个网站也接入了P的广告，那么就等同于，平台P 可以知道我曾经访问过这个见不得人的网站C。这就是隐私问题。

目前有不少法律法规对Cookie的使用作出了规定，典型的如GDPR。主要内容有：①告知用户你在使用cookie；②让用户可以选择禁用cookie；③在禁用cookie的时候也能够使用网站的大部分功能。

## Storage

> [MDN](https://developer.mozilla.org/en-US/docs/Web/API/Web_Storage_API/Using_the_Web_Storage_API)

`Storage`以对象的形式直接放在window上下文上，可以像普通对象一样去使用它们。但是要明确的是，它们只能做Key-Value储存，并且Value只能是字符串。

```javascript
localStorage.colorSetting = '#a4509b';
localStorage['colorSetting'] = '#a4509b';
localStorage.setItem('colorSetting', '#a4509b');
```

再看一下lib，可以看到`Storage`的类型定义是`[name: string]: any;`加一些函数方法。

它有两种主要类型：

- `sessionStorage`：生存周期是当前页面的会话周期。（特别强调一下，page可以刷新和恢复，但是另开page的话就是另一个session了）
- `localStorage`：保存在浏览器中（可以认为是永久存储）。

兼容性方面，现代浏览器已经全部支持，是一个很基础的API。

容量方面，大约是 2.5MB-10MB 这个数量级，不同的浏览器实现不同。

值得一提的是，好像localStorage是会触发Event的，虽然我测试并没有成功……

## IndexedDB基本特性

> [MDN](https://developer.mozilla.org/en-US/docs/Web/API/IndexedDB_API/Basic_Terminology)

它允许在浏览器中持久化地保存数据。它有如下特性：

1. **以键值对形式存储**。value可以是复杂的数据结构，key可以是数据结构的某些属性。你可以以属性建立索引（类似Mongo）。Key可以是二进制对象。
2. **支持事务**。提供了API给你操作索引、表、游标等东西，所有的这些都与事务绑定。事务的生命周期有仔细设计，并且只能自动提交而不能手动提交。因为有事务，所以你可以处理多个页面（即多个线程）上的操作冲突。
3. **全异步API**。要传入一个callback，然后通过监听DOM事件的方式得到回调。
4. **请求**概念。所有的读写操作都是一种"请求"，像xhr一样可能会失败哦，有状态值，也可以监听它的事件。
5. **DOM事件**。关注`type`和`target`属性，后者应该是个`IDBRequest`对象。失败的请求的事件会向上冒泡(bubble up)
6. **面向对象**。它不是传统的关系型数据库，没有典型的database和table的概念。
7. **NoSQL**。查询的时候，在索引(`index`)上查询并且产生游标(`cursor`)
8. **遵循同源策略**。也就是说不能跨域共享数据，每个数据库都有被标记它所属的源(origin=协议+域名+端口)。

它有一些限制：

- 跨语言排序（影响不大）
- 数据同步（要你自己实现逻辑）
- 全文检索（没有类似`LIKE`的操作符）

还要注意，数据可能被丢掉：

- 用户要求浏览器清理
- 隐私模式（会话结束后清理）
- 硬盘容量限制
- 数据损坏
- 不兼容特性的引入

### 核心术语

库表相关：

- `Database`：包含多个`store`，至少要有name和version两个属性
- `Store`：类似于table的概念，持久化地保存着许多`Record`（键值对）。它有名字。
- `durable`：在触发`complete`事件的时候，只是系统提交了写入操作，并没有保证已经写入硬盘。
- `index`：索引，是一种特殊的`store`，键是某个索引属性，值是主键（即非聚簇索引）
- `connection`：一个database可以同时有多个连接。
- `transaction`：事务，保证读写操作的原子性。
  - 一个connection可以同时开启多个事务，对写事务来说不允许scope范围重叠，对读事务来说则没有限制。
  - 事务被设计为短命的，浏览器可以选择杀死一个长时间运行的事务，你自己也可以主动放弃。
  - 三种模式`readwrite`, `readonly`和`versionchange`

键值相关：

- `key`：主键，唯一。它的数据类型有很多，string, date, float, blob, array
- `in-line key`：内联键，即普通索引。
- `key generator`：生成key的策略，一般来说你不用管它。
- `key path`：也是生成key的策略，从数据对象中依据规则取出属性作为key
- `out-of-line key`：非聚簇索引
- `value`：被储存的值。可以是js中的任意数据类型，包括object和array

范围相关：

- `cursor`：遍历多个record时用到的一种机制，保存着依赖的索引、范围、当前位置、方向。
- `key range`：
- `scope`：一个事务所影响到的stores的集合（强调一下是stores而不是records）

### 读后感

作为一个原专业后端程序员，我觉得IndexedDB这一套，至少这些术语，看下来显得很简单很粗糙。对于一些概念的解释讲得太入门了，而且似乎是故意不用成熟的后端数据库术语，而用什么 record, inline-key 这些奇怪的术语，这样对于像我一样已经有知识基础的人来说会感觉很累。我是以Mongo为原型来去理解它的，脑内转换之后基本上差不多吧。

强调了事务和scope挺好的，但是仔细想想吧，所谓scope就是一个表级读写锁，简单粗暴，没有太大深入挖掘的价值。

我还觉得它的设计不太靠谱。value能支持多种类型挺好的，可是为什么连key也支持多种类型，不同类型之间如何比较和排序？这样在实际使用的时候很容易引入坑吧。我想，本来IndexedDB定位就是放在前端的一个简化版NoSQL数据库，直接把key简化为字符串其实就很好，为啥要搞幺蛾子？

## IndexedDB使用

> [MDN](https://developer.mozilla.org/en-US/docs/Web/API/IndexedDB_API/Using_IndexedDB)

它的基本使用模式：

1. 打开一个database（获得一个连接对象）
2. 创建并保存数据
3. 开启事务然后做一些操作
4. 监听并等待事件
5. 对查询结果做一些处理

### 1. 打开数据库

```javascript
const request = indexedDB.open('test', 3);  // IDBOpenDBRequest
```

第一个参数是database的名字，第二个参数是版本号，是个自定义的整形数字（如果传入float则会被取整）。如果database不存在则会被创建，并且会有`onupgradeneeded`事件被触发。

可以给request对象定义事件处理函数，典型的有`open`和`error`两个事件。和其他DOM事件类似，可以用onxxx或者addEventListener任意方式去写：

```javascript
request.onerror = function (event) {
    console.log('失败', event);
};
request.addEventListener('success', function (event) {
    console.log('成功', event.target === request);  // true
    const db = request.result;  // IDBDatabase
});
```

### 2. 升级版本

version增大的时候就会触发这个事件，一般用于兼容场景：

```javascript
  const request = indexedDB.open('test', 4);  // +1
  request.onupgradeneeded = function (event) {
    const db = request.result;
    const store = db.createObjectStore('test1');
    console.log(store);  // IDBObjectStore
  };
```

### 3. 生成主键和索引

拿到一个`IDBObjectStore`对象，就可以对store进行操作了，createObjectStore 的时候建立的是主键，后续可以用 createIndex 建立额外索引。

通过下面的例子，可以理解什么叫「键路径 keyPath」：其实就是指定了从数据object的哪个属性作为主键。在指定了keyPath的时候，仅允许将object作为value传入（其他类型无法用这个keyPath取出属性，会抛出异常）。

```javascript
  const request = indexedDB.open('test', 5);
  request.onupgradeneeded = function () {
    const db = request.result;
    // 建表、索引
    const store = db.createObjectStore('test2', { keyPath: '_id' });
    store.createIndex('gender', 'gender', { unique: false });
  };
  request.onsuccess = function () {
    const db = request.result;
    // 开启一个事务
    const tr = db.transaction('test2', 'readwrite');
    // 打开表，写入数据
    const store = tr.objectStore('test2');
    store.add({ _id: 1001 }); // ok, 缺失了gender索引因此在gender表里不会出现
    store.add({ _id: 1002, gender: 'male' }); // ok
    store.add(100);  // 异常
  };
```

另一种自动索引是生成器，比如最典型的自增数字索引：

```javascript
async function main() {
  const request = indexedDB.open('test', 6);
  request.onupgradeneeded = function () {
    const db = request.result;
    db.createObjectStore('test3', { autoIncrement: true });
  };
  request.onsuccess = function () {
    const db = request.result;
    const tr = db.transaction('test3', 'readwrite');
    // 打开表，写入两条数据
    const store = tr.objectStore('test3');
    store.add(100); // key:1 value:100
    store.add(200); // key:2 value:200
  };
}
```

### 4. 数据的增删改查

前面的代码已经展示过了，先要获取一个`db`对象，然后开启一个`transaction`对象，再在里面找到`store`来做操作。

开启事务时要指定权限，默认是只读的。要对表(store)做操作，例如修改索引，则必须借助`version`的变更来实现。

这次我们来试试一个修改操作——即先查询`get`，然后`put`修改；并且将其封装为更易用的Promise形式：

```typescript
function update(db: IDBDatabase, tableName: string, key: any, toSet: any): Promise<Event> {
  return new Promise((resolve, reject) => {
    const store = db.transaction(tableName, 'readwrite').objectStore(tableName);
    const req1 = store.get(key);
    req1.onerror = reject;
    req1.onsuccess = function () {
      const req2 = store.put({ ...req1.result, ...toSet });
      req2.onerror = reject;
      req2.onsuccess = resolve;
    };
  });
}
```

上面代码写起来稍微有些别扭。

- 首先，`IDBDatabase`对象是可以长期持有的，所以我们可以单独包装一个Promise版本的`open`函数。
- `transaction`和`store`都是同步的，可以连续调用；
- `get`和`put`之类的方法，都是request，所以都要通过注册事件处理函数的方式去处理回调。
- 对于设置了keyPath的store，在`put`方法中不允许再传入key，这个事情就很危险，万一上游调用方修改了数据对象中的主键值，就会作为一条新的记录插入。要保护的话那得自己写一些逻辑去判断。

然后看一下游标cursor的用法：

```typescript
  const db = await open('test');
  const store = db.transaction('test2', 'readonly').objectStore('test2');
  const req = store.openCursor();
  req.onsuccess = function () {
    const cursor = req.result;
    if (cursor) {
      console.log(cursor.key, cursor.value);
      cursor.continue();
    } else {
      console.log('no more!');
    }
  };
```

主要关注的点，`cursor`对象本身可以读取key和value，然后通过`.continue()`方法来前进到下一个，直到null。

然后游标也可以指定范围，（这种全语义化的声明方式我之前在Golang一些数据库SDK中见过），例如：

```javascript
// 查询范围为：主键>=1002
const req = store.openCursor(IDBKeyRange.lowerBound(1002));
```

再然后游标还可以指定方向，放在第二个参数上。

### 5. 使用索引

在后端数据库中，正常情况下数据库服务本身会根据查询条件去选择合适的索引。但是在IndexedDB中，你必须手动指定走哪个索引，这就是前面所说的`inline-store`的作用。

```javascript
  const db = await open('test');
  const index = db.transaction('test2', 'readonly').objectStore('test2').index('gender');
  const req = index.get('female2');
  req.onsuccess = function () {
    console.log(req.result);
  };
```

上面的代码中利用了`gender`这个索引，这个索引是在创建store的时候一并创建过的，可以看看前面的代码。如果不用`index`而直接在`store`上执行操作的话，那就是走的主键。

### 当另一个页面试图升级version

不可以！如果一个页面打开了一个db对象，那么其他页面是不允许执行upgrade操作的，执行会触发`blocked`事件。

### 当浏览器被关闭时

别怕！事务可以保护你！

你也有可能通过`abort`事件来得知未被执行完毕的事务。

## 小结

有一说一，IndexedDB的这套查询条件的设计，作为一种数据库组件来说的话，还是非常简陋的。但毕竟是前端临时用一下的东西，有就不错了啊是吧（

语法方面，回调式的用法还是挺蛋疼的，所以在实际使用中可能要像`xhr`依赖`axios`那样，选择一个三方拓展库（例如 [idb](https://www.npmjs.com/package/idb) ?）来增强它的可用性。然后另一方面，别对这个存活于浏览器的数据库的可靠性抱太大的期望，把它当做是一种补充功能去使用。
