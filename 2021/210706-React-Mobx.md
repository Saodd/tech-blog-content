```yaml lw-blog-meta
title: "React+Mobx体验小结"
date: "2021-07-06"
brev: "从此做一个不想写后端的前端程序员！"
tags: ["前端"]
```

## 背景

没啥好说的，前端谁不会写？不能不会。

作为一个先学更多`Angular`的后端出身的全干程序员，我之前一直觉得React的数据管理，有点痛。

后来用了`Hooks`，诶，有那么点意思，但是离Angular还差点东西。

然后这次接触了`Mobx`，嗯，虽然写法有点不够清爽，但这个就是我想要的东西了。

这里不做完整的用法介绍，反正我自己是根据Angular和`knockout.js`的经验，半猜半学地就把Mobx给用了。

这里只记两个自己琢磨的比较有趣的逻辑实现。

## 版本

```json
{
  "mobx": "^6.0.4",
  "mobx-react": "^7.0.5",
  "mobx-react-lite": "^3.1.6",
  "react": "^16.9.0",
  "react-dom": "^16.9.0"
}
```

## Mobx基本用法

它的官网也介绍道，Mobx是独立的，但是基本上大家会将它跟React一起用。在写法上：

Mobx是一个独立的对象，生命周期可以独立于React组件。

```typescript
export class ActStore {
    constructor() {
        makeObservable(this);  // 这行必须要有
    }

    @observable
    name: string = '';
    @action
    setName = (name: string) => {
        this.name = name
    }
}
```

一般通过`useContext`来传递Mobx对象。

```typescript
// 这里传入的 new ActStore() 只是充当一个默认值，实际上并不会用到
export const actStoreContext = React.createContext<ActStore>(new ActStore())
```

```typescript
function ParentComponent() {
    const actStore = React.useMemo(() => new ActStore(), [])
    return (
        <actStoreContext.Provider value = {actStore} >
            <ChildComponent / >
            </actStoreContext.Provider>
    )
}

// 注：这里没有observer装饰，组件不会随数据变化。
function _ChildComponent() {
    const actStore = React.useContext(actStoreContext)
    return <p>{actStore.name} < /p>
}
```

要将React组件装进`obeserver`，这样数据的变化才会更新到组件上。

```typescript
import {observer} from "mobx-react-lite";

const ChildComponent = observer(_ChildComponent)
```

## 场景一：防抖

业务需求是，有一个表单字段是选择时间，而不同的时间，会对应不同的计费额度。

```typescript
export class ActStore {
    // 需求大概是这样
    @observable
    time: dayjs.Dayjs;
    @observable
    count: number;
}
```

当时间变化时，计费额度要随之变化并且展示。

同时考虑到web页面的生存时间较短，可以顺手做一个防抖+缓存优化。

```typescript
export class ActStore {
    @observable
    time: dayjs.Dayjs;
    @observable
    _countCache: Map<string, number>

    @computed
    get count(): number {
        // 因为time是个obeservable，所以会自动触发更新
        const month = this.time.format('YYYYMM')

        // 如果没有数据，就向后端请求
        if (!this._countCache.has(month)) {
            // 设置一个值，这样下次就不会进入到if条件里。（这里应该用action）
            this._countCache.set(month, 0)

            // 也就是说后端请求只会执行一次。回调更新map之后，这个computed的值会自动更新。
            api.getCount({month}).then(count => this._countCache.set(month, count))
        }

        // 立即返回数据
        return this._countCache.get(month) || 0
    }
}
```

## 需求二：节流

业务需求是，用户在表单上的任何操作都要保存下来，页面重载后也要能够恢复。

首先，最理想的情况，每次更新数据之后都保存一次。

```typescript
export class ActStore {
    @observable
    name: string = '';
    @action
    setName = (name: string) => {
        this.name = name
        this.save()
    }
}
```

但这样很容易造成性能问题。如果只是用户的手动操作还好，而如果是一些批量操作，那可能在短时间内触发多次保存，那就很麻烦。

我们做一个节流，用户操作2秒之后才保存。

这里用到「锁」的理念，而且考虑到js是基单线程运行（且是基于协作式的事件循环），我们用最简单的乐观锁就行。

```typescript
async function saveStore(store: ActStore) {
    // 先生成一个UUID，这里简化直接用随机数
    const me = Math.random()
    
    // 把自己占住锁，然后等待2秒
    store.setCacheLock(me)
    await new Promise(resolve => setTimeout(resolve, 2000))
    
    // 2秒后恢复，检查是不是自己依然持有锁
    if (store.cacheLock === me) {
        api.doSomething()
    }
}
```

如果只是这样还不行，因为如果有连续操作的话，那就一直无法执行到了。

接下来我们再增加一个保底机制，保证至少10秒执行一次。

```typescript
async function saveStore(store: ActStore) {
    const me = Math.random()
    store.setCacheLock(me)
    await new Promise(resolve => setTimeout(resolve, 2000))
    
    // 加一个或条件，超过10秒也立即执行
    if (store.cacheLock === me || store.isTooOld()) {
        api.doSomething()
    }
}
```

这个`isTooOld`用术语来说，应当是一个TAS的操作（`Test and Set`），同上，因为js是单线程balabala，简单粗暴地搞就行了。

因为涉及到数据更新，所以包装成action。（其实因为要修改的数据并不需要observable，这个操作不用action装饰也应该没问题，但是装饰一下更符合习惯。）

```typescript
export class ActStore {
    @action
    isTooOld = (): boolean => {
        const now = Date.now()
        if (now - this._lastCachedTime > 10000) {  // 10秒
            this._lastCachedTime = now
            return true
        }
        return false
    }
}
```

## 小结

受够了Python的GIL，写了很多Golang的并发，如今尝试用js单线程特性来实现无锁编程思想，别有一番风味。

关于Mobx，与其说它是什么样的库，不如说它代表了一种怎样的思想。它补齐了React状态管理的短板。

如今在我看来，只要自己再借鉴一些Angular在工程文件组织上的思想，以及全文Typescript，这样搭配下来的React，我觉得体验是比Angular本身更好的。

要知道，之前我可是Angular的狂热粉啊。

写前端，真TMD快乐！

## 取舍

这几天遇到一个比较典型的场景：之前写了一个组件时偷懒了，把几个可以拆分的组件写在一个function里。虽然也不大，也就一百多行，但是现在有需要的时候发现还是可以进行拆分的。

拆分时发现，如果使用Mobx这种模式，将所有状态独立于组件之外了，那么拆分组件的工作将变得异常容易，只需要一个`useContext`就能把东西拿出来，太方便了。

作为对比，如果用Hooks来管理的状态，就会发现在父子之间必须要很啰嗦地去传递这些状态量。如果是一个比较复杂的业务组件，例如有几十个状态量，那么这样的逐级传递会变得几乎不可操作。

在这种复杂的状态管理场景下，Mobx胜出。

但Mobx也并不是完美的。也正是因为在组件中使用了`useContext`，这意味着所有的组件都与这个Mobx对象绑定了，没它就不能用。并且由于Mobx几乎不支持继承，所以拓展性极差，基本写成什么样就是什么样，后续只能不断地添加使其臃肿。

再次作为对比，如果用Hooks来管理的状态，更像是一种「契约式编程」，无论父组件是什么，只要能提供类型正确的props，那么这个组件可以在任何地方进行复用。

在简单组件的场景下，Hooks胜出。
