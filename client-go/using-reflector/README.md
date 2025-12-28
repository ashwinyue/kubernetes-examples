# 2.1 Reflector 机制

## 📚 学习目标

- 理解 Reflector 的核心作用
- 掌握 Watch/List API 的使用
- 了解资源同步到本地 Store 的机制

## 🔍 什么是 Reflector

Reflector 是 client-go 中负责从 Kubernetes API Server 监听和同步资源的核心组件。

### 核心职责

1. **初始同步**：通过 List API 获取所有资源
2. **持续监听**：通过 Watch API 监听资源变更
3. **事件存储**：将资源变更存储到本地 Store
4. **定期 Resync**：定期重新同步所有资源

### 工作流程

```
┌─────────────┐
│   API      │
│  Server   │
└─────┬─────┘
      │
      │ 1. List All Resources
      ▼
┌─────────────┐
│  Reflector  │
└─────┬─────┘
      │
      │ 2. Watch for Changes
      │    (Add/Update/Delete)
      ▼
┌─────────────┐
│   Store     │
│   (Cache)   │
└─────────────┘
```

## 📖 代码解析

### 示例文件: `../using-reflector/main.go`

#### 1. 创建 ListWatch

```go
// 创建 ListWatch，用于监视 Pod 资源对象的变更事件
lw := cache.NewListWatchFromClient(
    clientset.CoreV1().RESTClient(),
    "pods",
    metav1.NamespaceAll,      // 监听所有命名空间
    fields.Everything(),       // 不过滤任何字段
)
```

**关键点**：
- `NewListWatchFromClient()` 创建 ListWatch 对象
- 第一个参数：REST Client
- 第二个参数：资源名称（pods）
- 第三个参数：命名空间（NamespaceAll = 所有命名空间）
- 第四个参数：字段选择器

#### 2. 创建 Store

```go
// 创建本地存储，用于缓存 Pod 对象
store := cache.NewStore(cache.MetaNamespaceKeyFunc)
```

**MetaNamespaceKeyFunc**：
- 生成 Key 的格式：`<namespace>/<name>`
- 例如：`kube-system/etcd-onex-control-plane`
- 用于在 Store 中唯一标识资源

#### 3. 创建 Reflector

```go
// 创建 Reflector，用于从 API Server 获取 Pod 资源并缓存到本地
reflector := cache.NewReflector(lw, &corev1.Pod{}, store, 10*time.Second)
```

**参数说明**：
- `lw`：ListWatch 对象
- `&corev1.Pod{}`：期望的类型（用于类型检查）
- `store`：本地存储
- `10*time.Second`：Resync 周期（10秒重新同步一次）

#### 4. 启动 Reflector

```go
// 启动 Reflector，开始监听 API Server 上 Pod 资源的变更事件
stopCh := make(chan struct{})
go reflector.Run(stopCh)
```

#### 5. 等待缓存同步

```go
var wg sync.WaitGroup
wg.Add(1)

// 测试：打印本地缓存中，缓存的一条 Key
go func() {
    defer wg.Done()
    for {
        if len(store.ListKeys()) > 0 {
            fmt.Printf("Local store cached a key: %q\n", store.ListKeys()[0])
            return
        }
    }
}()

wg.Wait()
```

## 🎯 核心概念

### 1. ListWatch

ListWatch 封装了 List 和 Watch 两个操作：

```go
type ListWatch struct {
    ListFunc  ListFunc   // List 所有资源
    WatchFunc WatchFunc  // Watch 资源变更
}
```

**执行流程**：
1. 首次运行：调用 `ListFunc` 获取所有资源
2. 后续运行：调用 `WatchFunc` 监听变更事件

### 2. Resync 机制

Resync 是 Reflector 定期重新同步所有资源的机制：

```go
reflector := cache.NewReflector(lw, &corev1.Pod{}, store, 10*time.Second)
                                                            ↑
                                                    Resync 周期
```

**Resync 的作用**：
- 定期重新 List 所有资源
- 确保本地缓存与 API Server 一致
- 触发所有资源的 Update 事件
- 修复可能的缓存不一致

**注意事项**：
- Resync 会触发所有资源的 Update 事件
- 即使资源没有实际变化
- 增加网络和 CPU 开销
- 生产环境建议设置较长的周期（如 10 分钟）

### 3. Store

Store 是一个线程安全的本地缓存：

```go
type Store interface {
    Add(obj interface{}) error
    Update(obj interface{}) error
    Delete(obj interface{}) error
    List() []interface{}
    ListKeys() []string
    Get(obj interface{}) (item interface{}, exists bool, err error)
    GetByKey(key string) (item interface{}, exists bool, err error)
}
```

**常用操作**：

```go
// 添加对象
store.Add(pod)

// 更新对象
store.Update(pod)

// 删除对象
store.Delete(pod)

// 列出所有对象
pods := store.List()

// 列出所有 Key
keys := store.ListKeys()

// 根据 Key 获取对象
obj, exists, err := store.GetByKey("default/example-pod")
```

## 💡 使用场景

### 场景 1：监控 Pod 变化

```go
reflector := cache.NewReflector(lw, &corev1.Pod{}, store, 30*time.Second)
go reflector.Run(stopCh)

// 持续监控 store 的变化
for {
    pods := store.List()
    fmt.Printf("当前有 %d 个 Pod\n", len(pods))
    time.Sleep(5 * time.Second)
}
```

### 场景 2：获取特定资源

```go
// 等待特定 Pod 出现在缓存中
for {
    obj, exists, _ := store.GetByKey("default/example-pod")
    if exists {
        pod := obj.(*corev1.Pod)
        fmt.Printf("找到 Pod: %s\n", pod.Name)
        break
    }
    time.Sleep(1 * time.Second)
}
```

### 场景 3：多资源监控

```go
// 为不同的资源创建不同的 Reflector
podLW := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "pods", metav1.NamespaceAll, fields.Everything())
podStore := cache.NewStore(cache.MetaNamespaceKeyFunc)
podReflector := cache.NewReflector(podLW, &corev1.Pod{}, podStore, 10*time.Second)

svcLW := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "services", metav1.NamespaceAll, fields.Everything())
svcStore := cache.NewStore(cache.MetaNamespaceKeyFunc)
svcReflector := cache.NewReflector(svcLW, &corev1.Service{}, svcStore, 10*time.Second)

// 同时启动多个 Reflector
go podReflector.Run(stopCh)
go svcReflector.Run(stopCh)
```

## ⚠️ 注意事项

1. **Store 是线程安全的**：可以安全地并发访问
2. **Reflector 阻塞**：Run() 方法会一直运行，直到收到 stopCh
3. **内存使用**：Store 缓存所有资源，注意内存消耗
4. **Resync 开销**：频繁 Resync 会增加网络和 CPU 开销
5. **错误处理**：Reflector 会自动处理网络错误和重连

## 🔄 与其他组件的关系

```
Reflector (监听 API)
    │
    │ List/Watch
    ▼
Store (本地缓存)
    │
    │ Indexer (扩展的 Store)
    ▼
DeltaFIFO (事件队列)
    │
    ▼
WorkQueue (工作队列)
    │
    ▼
Controller (业务逻辑)
```

## 📚 相关资源

- [Store 接口文档](https://pkg.go.dev/k8s.io/client-go/tools/cache#Store)
- [Reflector 源码](https://github.com/kubernetes/client-go/blob/master/tools/cache/reflector.go)
- [ListWatch 文档](https://pkg.go.dev/k8s.io/client-go/tools/cache#ListWatch)

## 🚀 下一步

继续学习 [2.2 DeltaFIFO 队列](./2.2-DeltaFIFO.md)，了解事件队列机制。
