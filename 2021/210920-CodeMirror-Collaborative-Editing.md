```yaml lw-blog-meta
title: "[译] CodeMirror: 协同编辑"
date: "2021-09-20"
brev: "前端终极大BOSS之一"
tags: ["前端"]
```

## 背景

昨天把Websocket重新捡起来玩了一下，体验不错。那么趁热打铁，继续深入下去研究。

在我的视野看来，接下来有两个方向的研究：一是后端方向，直接读 `gorilla/websocket` 的源码，把协议的各种处理细节都熟悉一遍；二是前端方向，探索更多的应用场景。

嘛，既然立志要当一个合格的前端工程师，那就先把后端的东西放一放，先看前端吧。

## 事前思考

如果不引入新的框架工具，就用手头熟悉的工具，能够实现「协同编辑」吗？

### 实现1：全量同步

`textarea`熟悉对吧，那每次`onChange`的时候，都把**全文**发回服务器，让服务器把全文广播给所有客户端。

嗯，肯定能做，而且简单得一批，我来写大概一个小时能把前后端全搞定。

但是这也太粗糙太没追求了吧，pass。

### 实现2：分行同步

既然全文同步太傻，那每次只同步**修改的行**，就可以了吧？

假如先简化一下，假设网络是稳定且迅速的，不考虑多端用户在同一个瞬间做操作的情况。那么做起来也简单，前后端分别split一下，加不了太多工作量。

可是在现实状态下，冲突问题变得无法忽视：假如行本身发生了变化（例如删除了行），这时候怎么处理？遇到网络波动又如何确认和恢复？

为了应对这种冲突，如果只是`{行数：内容数据}`这样简单的数据结构显然无法满足要求，至少，要增加一个**修改前的状态**这个东西，才能确保不会发生误操作。

一个简单的思路，可以维护一个**顺序号**，服务器每次检查顺序号，对于冲突的顺序号的同步请求予以拒绝，然后客户端得知被拒绝之后，要等其他操作同步完毕之后才能提交自己的操作。

原理上应该差不多就这样了，可是可以想象实现起来会非常复杂，所以先看看业界主流方案。

以下内容翻译自 [Example: Collaborative Editing](https://codemirror.net/6/examples/collab/)

## Example: Collaborative Editing

实时协同编辑 是一种允许多个用户在不同的机器上同时编辑同一个文档。内容的修改会被传播到所有的参与者那里去，并且实时显示最新结果。

（页面上有一个模拟工具）

这种技术的最大难题，就是如何处理冲突操作——因为网络传输并不是瞬间完成的，可能会出现多个人同时做了同一个操作的情况，那么该如何协调？

`CodeMirror`使用一些基于「操作转换原理 [operational transformation](https://en.wikipedia.org/wiki/Operational_transformation) 」的工具，以及一个用于指定所有操作的确切顺序的「认证中心（服务器）」，来解决这个协作问题。

这篇文章主要介绍如何实现，如果想了解更多理论知识可以参考 [这个](https://marijnhaverbeke.nl/blog/collaborative-editing-cm.html) 

（当然你也可以用其他协作算法来实现）

### 原则

我们的库叫 [@codemirror/collab](https://codemirror.net/6/docs/ref/#collab) 它的主要原理是：

- 有一个中央服务器「Authority」来维护所有的历史操作。
- 编辑者「Peers」要记录它们自己追踪到哪一个历史版本，以及自己有哪些本地操作还未被确认。
- 编辑者「Peers」要有能力接收并处理来自中央的操作数据，当收到操作之后……
  + 如果有一些是编辑者自己本地的未确认操作，那就从未确认列表中移除这些操作。
  + 其他的操作要反映在当前的编辑器状态上。
  + 如果还有未确认的操作剩下（即产生了操作冲突），那么要使用「操作转换operational transformation」，来把来自远端的操作转换为可以应用在当前状态的操作，（以及反之同理将本地未确认操作转换为可以重新提交到远端的操作，）（即类似于`Git Rebase`）
  + 本地追踪的版本号向前移动
- 只要当前还有未确认的本地修改，编辑者就应当尝试把它提交到中央服务器，（和当前追踪的版本号一起）
  + 如果版本号与服务器的匹配，那么服务器就接受这些操作，并记录在服务器的历史记录中。
  + 如果不匹配，那就拒绝；之后peers必须等待接收那些冲突的操作，合并之后再重新提交。

很多复杂的逻辑处理都包装在这个库里了！但是你必须自己写后端（其实可以很简单了）以及它们之间的通信（这可能稍微有点难度）

### 中央服务器 Authority

在这个例子中，中央服务器是一个`web worker` 。这模拟了异步的通信过程，以及数据的序列化，同时又可以让所有的东西都简单地在浏览器里运行（用作展示Demo）。在现实世界中，一般应当采用 HTTP 或者 websockets 。

服务端只保存两个状态：一个历史操作数组，和一个当前文档内容。 （后者是给新加入的peers用的）

```typescript
import {ChangeSet, Text} from "@codemirror/state"
import {Update} from "@codemirror/collab"

// The updates received so far (updates.length gives the current version)
let updates: Update[] = []
// The current document
let doc = Text.of(["Start document"])
```

服务端需要实现三个接口：（译者注，原内容是基于长连接的，而我的目标是使用ws所以省略了一些）

```typescript
// pullUpdates 拉取当前新的操作
if (data.type == "pullUpdates") {
  if (data.version < updates.length) resp(updates.slice(data.version))
}
```

```typescript
// pushUpdates 推送新的操作
if (data.type == "pushUpdates") {
  if (data.version != updates.length) {
    resp(false)
  } else {
    for (let update of data.updates) {
      // Convert the JSON representation to an actual ChangeSet instance
      let changes = ChangeSet.fromJSON(update.changes)
      updates.push({changes, clientID: update.clientID})
      doc = changes.apply(doc)
    }
    resp(true)
}
```

```typescript
// getDocument 给新人传一份完整内容
if (data.type == "getDocument") {
  resp({version: updates.length, doc: doc.toString()})
}
```

### 编辑者 Peer

也是对应地实现三个接口调用：（这里的`connection`是封装了模拟网络请求）

```typescript
function pushUpdates(connection: Connection, version: number, fullUpdates: readonly Update[]): Promise<boolean> {
  // Strip off transaction data
  let updates = fullUpdates.map(u => ({
    clientID: u.clientID,
    changes: u.changes.toJSON()
  }))
  return connection.request({type: "pushUpdates", version, updates})
}
```

```typescript
function pullUpdates(connection: Connection, version: number): Promise<readonly Update[]> {
  return connection.request({type: "pullUpdates", version})
    .then(updates => updates.map(u => ({
      changes: ChangeSet.fromJSON(u.changes),
      clientID: u.clientID
    })))
}
```

```typescript
function getDocument(connection: Connection): Promise<{version: number, doc: Text}> {
  return connection.request({type: "getDocument"}).then(data => ({
    version: data.version,
    doc: Text.of(data.doc.split("\n"))
  }))
}
```

为了管理与服务器之间的通讯，我们用一个「view plugin」，把所有的异步代码塞进去。这个插件会持续地尝试拉取新操作，然后把新操作应用到本地编辑器上（通过`receiveUpdates`函数）。

而当本地编辑器产生操作的时候，这个插件会尝试把操作推送到服务器上。它简单地保证自己同时只有一个push请求在执行，并且设置一个简易的超时重试机制。

```typescript
function peerExtension(startVersion: number, connection: Connection) {
  let plugin = ViewPlugin.fromClass(class {
    private pushing = false
    private done = false

    constructor(private view: EditorView) { this.pull() }

    update(update: ViewUpdate) {
      if (update.docChanged) this.push()
    }

    async push() {
      let updates = sendableUpdates(this.view.state)
      if (this.pushing || !updates.length) return
      this.pushing = true
      let version = getSyncedVersion(this.view.state)
      await pushUpdates(connection, version, updates)
      this.pushing = false
      // Regardless of whether the push failed or new updates came in
      // while it was running, try again if there's updates remaining
      if (sendableUpdates(this.view.state).length)
        setTimeout(() => this.push(), 100)
    }

    async pull() {
      while (!this.done) {
        let version = getSyncedVersion(this.view.state)
        let updates = await pullUpdates(connection, version)
        this.view.dispatch(receiveUpdates(this.view.state, updates))
      }
    }

    destroy() { this.done = true }
  })
  return [collab({startVersion}), plugin]
}
```

于是你的顶层代码就可以很清爽了：

```typescript
async function createPeer(connection: Connection) {
  let {version, doc} = await getDocument(connection)
  let state = EditorState.create({
    doc,
    extensions: [basicSetup, peerExtension(version, connection)]
  })
  return new EditorView({state})
}
```

完整完代码请看 [github](https://github.com/codemirror/website/blob/master/site/examples/collab/collab.ts)

### 确认本地操作

上面的代码逻辑中，忽略了push操作的返回值，仅仅依靠pull的返回结果。

如果需要的话，也可以根据push的返回值来做一些事情。这样可以节省一点点网络流量，或者你有什么其他的需求。

### 丢掉旧的历史操作

这个实现会无限累加历史记录。

这可能在某些场景是ok的，但是正常情况下你会需要丢掉旧的记录。

有一个函数`ChangeSet.compose`可以帮你压缩一下。（当然也可以自己想办法）

### 共享其他副作用

一般情况下，协同编辑只需要共享文本内容就行了。

但是如果你需要的话，也可以共享其他状态，例如选中区域：（其实理论上只要是本地编辑器有在追踪的状态都可以实现共享）

```typescript
import {StateEffect} from "@codemirror/state"

const markRegion = StateEffect.define<{from: number, to: number}>({
  map({from, to}, changes) {
    from = changes.mapPos(from, 1)
    to = changes.mapPos(to, -1)
    return from < to ? {from, to} : undefined
  }
})
```

（不展开细节了）

这个库并没有为effects封装序列化操作，所以为了与JSON一起工作你需要自己撸代码。

最后强调一下，从理论上说，由于「操作转换」算法的局限性，共享的effect可能并不能保证完全正常工作。虽然对于有些场景（例如共享光标位置）来说转换错了也不大要紧，不过对于其他你觉得重要的场景，可能还是要你自己想别的办法来解决。

## 结语

emm，此刻我内心有些烦躁。

怎么说呢，首先感觉`CodeMirror`这个库就有点陈旧的味道，虽然很多web应用都在用它来做编辑器，但我好像没觉得很酷。

其次`@codemirror/collab`这个协作库呢，感觉充满了黑魔法，至少我研究了一下午都还没有觉得很有信心能顺利地实现。翻译的这篇文章呢，原理讲得还行，但是操作层面上我觉得没有很好的参考价值。

以及有一个重要的缺陷：服务端也要求js运行时才能使用这个库。当然，第一就算我要开一套node.js服务也就是个把小时的事情，第二我也能想到服务端不用js用其他语言加一些取巧的办法也可以实现。

但，我总觉得兴致缺缺，可能是因为这个库包装得太过上层了吧，让我觉得没有太多学习的价值。

所以姑且还是放弃，自己打自己的脸，先用最简单的全文同步的办法来实现吧。
