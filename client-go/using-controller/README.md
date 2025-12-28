# 2.3 Controller 模式

## 📚 学习目标

- 理解 Controller 的核心设计
- 掌握 Reflector + DeltaFIFO + WorkQueue 集成
- 了解事件处理流程

## 🔍 什么是 Controller

Controller 是一个核心模式，通过组合 Reflector、DeltaFIFO、WorkQueue 等组件，实现自动化的资源管理逻辑。

### Controller 组成

```
┌──────────────┐
│  Reflector  │ ──► List/Watch API Server
└──────┬─────┘
       │
       │ 变更事件
       ▼
┌──────────────┐
│  DeltaFIFO  │ ──► 事件队列
└──────┬─────┘
       │
       │ Pop Delta
       ▼
┌──────────────┐
│   Indexer    │ ──► 本地缓存（Store + Index）
└──────┬─────┘
       │
       │ Key
       ▼
┌──────────────┐
│  WorkQueue   │ ──► 工作队列
└──────┬─────┘
       │
       │ 并发处理
       ▼
┌──────────────┐
│   Workers    │ ──► 业务逻辑
└──────────────┘
```

## 📖 代码解析

### 示例文件: `main.go`

#### 1. 创建 Pod Watcher

```go
// 创建 Pod watcher
podListWatcher := cache.NewListWatchFromClient(
    clientset.CoreV1().RESTClient(),
    "pods",
    v1.NamespaceDefault,
    fields.Everything(),
)
```

#### 2. 创建 Indexer 和 Informer

```go
// 创建 IndexerInformer，绑定 WorkQueue 到缓存
indexer, informer := cache.NewIndexerInformer(
    podListWatcher,
    &v1.Pod{},
    0,
    cache.ResourceEventHandlerFuncs{
        AddFunc: func(obj interface{}) {
            fmt.Println("New object added:", obj.(*v1.Pod).Name)
        },
        UpdateFunc: func(oldObj, newObj interface{}) {
            fmt.Println("Object updated. Old:", oldObj.(*v1.Pod).Name,
                "New:", newObj.(*v1.Pod).Name)
        },
        DeleteFunc: func(obj interface{}) {
            fmt.Println("Object deleted:", obj.(*v1.Pod).Name)
        },
    },
    cache.Indexers{},
)
```

**ResourceEventHandler**：

```go
type ResourceEventHandler interface {
    OnAdd(obj interface{})
    OnUpdate(oldObj, newObj interface{})
    OnDelete(obj interface{})
}
```

#### 3. 启动 Informer

```go
stopCh := make(chan struct{})
defer close(stopCh)

go informer.Run(stopCh)

// 等待缓存同步
if !cache.WaitForCacheSync(stopCh, informer.HasSynced) {
    runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
    return
}
```

#### 4. 从 Indexer 查询资源

```go
go func() {
    // 不断轮询 Indexer，当有数据时打印
    for {
        if len(indexer.ListKeys()) > 0 {
            obj, _, err := indexer.GetByKey(indexer.ListKeys()[0])
            if err != nil {
                panic(err)
            }
            accessor, _ := meta.Accessor(obj)
            fmt.Printf("Resource name: %s\n", accessor.GetName())
            return
        }
    }
}()
```

## 🎯 核心概念

### 1. Indexer

Indexer 是 Store 的扩展，增加了索引功能：

```go
type Indexer interface {
    Store
    Index(indexName string, obj interface{}) ([]string, error)
    ByIndex(indexName, indexKey string) ([]interface{}, error)
    GetIndexers() Indexers
    AddIndexers(newIndexers Indexers) error
}
```

**优势**：
- 快速按索引查询
- 支持多个索引
- 例如：按 Pod 状态、节点等索引

### 2. Informer

Informer 是 Reflector + Handler 的组合：

```go
type Informer interface {
    Run(stopCh <-chan struct{})
    HasSynced() bool
    AddEventHandler(handler ResourceEventHandler)
}
```

**工作流程**：
1. Reflector List/Watch 资源
2. 事件存储到 DeltaFIFO
3. DeltaFIFO Pop 触发 Handler
4. Handler 更新 Indexer

### 3. ResourceEventHandler

三种事件类型：

```go
AddFunc: func(obj interface{}) {
    // 对象被添加
}

UpdateFunc: func(oldObj, newObj interface{}) {
    // 对象被更新
}

DeleteFunc: func(obj interface{}) {
    // 对象被删除
}
```

## 💡 实战示例

### 示例 1：简单的 Pod 监控

```go
informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
    AddFunc: func(obj interface{}) {
        pod := obj.(*v1.Pod)
        fmt.Printf("Pod Added: %s\n", pod.Name)
    },
    UpdateFunc: func(old, new interface{}) {
        newPod := new.(*v1.Pod)
        oldPod := old.(*v1.Pod)
        
        if newPod.Status.Phase != oldPod.Status.Phase {
            fmt.Printf("Pod %s status changed: %s -> %s\n",
                pod.Name, oldPod.Status.Phase, newPod.Status.Phase)
        }
    },
    DeleteFunc: func(obj interface{}) {
        pod := obj.(*v1.Pod)
        fmt.Printf("Pod Deleted: %s\n", pod.Name)
    },
})
```

### 示例 2：使用索引查询

```go
// 添加索引
indexers := cache.Indexers{
    "byPhase": func(obj interface{}) ([]string, error) {
        pod := obj.(*v1.Pod)
        return []string{string(pod.Status.Phase)}, nil
    },
}

// 创建带索引的 Informer
indexer, informer := cache.NewIndexerInformer(
    lw, &v1.Pod{}, 0, handlers, indexers,
)

// 按索引查询
runningPods, err := indexer.ByIndex("byPhase", string(v1.PodRunning))
if err != nil {
    panic(err)
}
fmt.Printf("Running pods: %d\n", len(runningPods))
```

## ⚠️ 注意事项

1. **缓存同步**：必须等待 `WaitForCacheSync` 完成
2. **并发安全**：Indexer 是线程安全的
3. **事件顺序**：不保证事件顺序
4. **资源版本**：NewObj 可能比触发事件更新的版本更新
5. **内存使用**：Indexer 缓存所有资源

## 🔄 完整流程

```
1. Reflector.List()         ──► 获取所有 Pod
2. Reflector.Watch()        ──► 监听变更
3. DeltaFIFO.Add()         ──► 存储 Added 事件
4. DeltaFIFO.Pop()         ──► 处理事件
5. Indexer.Add()           ──► 更新本地缓存
6. AddFunc()              ──► 触发用户回调
7. 业务逻辑               ──► 处理资源
```

## 📚 相关资源

- [Informer 接口文档](https://pkg.go.dev/k8s.io/client-go/tools/cache#Informer)
- [Indexer 接口文档](https://pkg.go.dev/k8s.io/client-go/tools/cache#Indexer)
- [ResourceEventHandler](https://pkg.go.dev/k8s.io/client-go/tools/cache#ResourceEventHandler)

## 🚀 下一步

继续学习 [WorkQueue](../workqueue/)，了解工作队列和并发处理机制。
