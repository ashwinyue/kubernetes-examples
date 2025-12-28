# 2.2 DeltaFIFO 队列

## 📚 学习目标

- 理解 DeltaFIFO 的核心作用
- 掌握 Delta 类型（Added/Updated/Deleted）
- 了解 FIFO 顺序保证机制

## 🔍 什么是 DeltaFIFO

DeltaFIFO 是一个先进先出（FIFO）队列，专门用于存储和分发资源变更事件。

### 核心特性

1. **Delta 类型**：区分 Added、Updated、Deleted 等事件
2. **FIFO 顺序**：保证事件按添加顺序处理
3. **去重机制**：相同 Key 的重复事件会合并
4. **Pop 处理**：批量 Pop Delta 集合进行处理

## 📖 代码解析

### 示例文件: `../using-deltafifo/main.go`

#### 1. 创建 DeltaFIFO

```go
// 创建一个 DeltaFIFO 对象
fifo := cache.NewDeltaFIFO(cache.MetaNamespaceKeyFunc, nil)
```

**参数说明**：
- `cache.MetaNamespaceKeyFunc`：生成 Key 的函数，格式为 `namespace/name`
- `nil`：Key 函数（不使用自定义）

#### 2. 添加对象（Added 事件）

```go
dep1 := &appsv1.Deployment{
    ObjectMeta: metav1.ObjectMeta{
        Name: "dep1",
        Namespace: metav1.NamespaceDefault,
    },
}
// 1. 将对象添加事件放入 DeltaFIFO 中
fifo.Add(dep1)
```

**效果**：
```
DeltaFIFO 队列：
├── Added: default/dep1
```

#### 3. 更新对象（Updated 事件）

```go
dep1.Name = "dep1-modified"
// 2. 将对象变更事件放入 DeltaFIFO 中
fifo.Update(dep1)
```

**效果**：
```
DeltaFIFO 队列：
├── Added: default/dep1
├── Added: default/dep2
├── Updated: default/dep1
```

#### 4. 删除对象（Deleted 事件）

```go
// 4. 将对象删除事件放入 DeltaFIFO 中
fifo.Delete(dep1)
```

**效果**：
```
DeltaFIFO 队列：
├── Added: default/dep2
├── Updated: default/dep1
└── Deleted: default/dep1
```

#### 5. Pop 处理 Delta

```go
// 5. "不断"从 DeltaFIFO 中 Pop 资源对象
for {
    fifo.Pop(func(obj interface{}, isInInitialList bool) error {
        for _, delta := range obj.(cache.Deltas) {
            deploy := delta.Object.(*appsv1.Deployment)

            // 区分不同事件，执行不同回调
            switch delta.Type {
            case cache.Added:
                fmt.Printf("Added: %s/%s\n", deploy.Namespace, deploy.Name)
            case cache.Updated:
                fmt.Printf("Updated: %s/%s\n", deploy.Namespace, deploy.Name)
            case cache.Deleted:
                fmt.Printf("Deleted: %s/%s\n", deploy.Namespace, deploy.Name)
            }
        }

        return nil
    })
}
```

**Pop 参数**：
- `obj interface{}`：通常是 `Deltas` 切片（多个 Delta 的集合）
- `isInInitialList bool`：是否为初始 List 操作

**Deltas 类型**：
```go
type Delta struct {
    Type   DeltaType   // 事件类型
    Object interface{} // 资源对象
}

type DeltaType string
const (
    Added    DeltaType = "Added"
    Updated  DeltaType = "Updated"
    Deleted  DeltaType = "Deleted"
    Sync     DeltaType = "Sync"
    Replaced DeltaType = "Replaced"
)
```

## 🎯 Delta 类型详解

### 1. Added

```go
fifo.Add(obj)
// 等同于：
// delta := Delta{Type: Added, Object: obj}
```

**触发场景**：
- 新对象被创建
- 对象从不存在变为存在

### 2. Updated

```go
fifo.Update(obj)
// 等同于：
// delta := Delta{Type: Updated, Object: obj}
```

**触发场景**：
- 对象被修改
- 对象状态发生变化

### 3. Deleted

```go
fifo.Delete(obj)
// 等同于：
// delta := Delta{Type: Deleted, Object: obj}
```

**触发场景**：
- 对象被删除
- 对象从存在变为不存在

### 4. Sync

```go
// 在 Resync 时自动添加
```

**触发场景**：
- Reflector 的 Resync 周期到达
- 所有对象都会触发 Sync 事件

### 5. Replaced

```go
// 在 Relist 时批量替换时使用
```

**触发场景**：
- 重新 List 时发现大量变化
- 批量替换本地缓存

## 🔄 去重机制

DeltaFIFO 会合并相同 Key 的重复事件：

```go
// 示例 1：多个 Add
fifo.Add(dep1)
fifo.Add(dep1)
fifo.Add(dep1)

// 结果：只有一个 Added 事件
// Deltas: [{Type: Added, Object: dep1}]

// 示例 2：Add -> Update -> Add
fifo.Add(dep1)
fifo.Update(dep1)
fifo.Add(dep1)

// 结果：最新的 Add 事件
// Deltas: [{Type: Added, Object: dep1}]
```

**去重规则**：
1. 相同 Key 的事件会合并
2. 最新的事件会覆盖旧的
3. 保证每个 Key 在队列中只有一个 Delta

## 💡 使用场景

### 场景 1：事件分发器

```go
type EventHandler struct {
    queue cache.DeltaFIFO
}

func (h *EventHandler) OnAdd(obj interface{}) {
    h.queue.Add(obj)
}

func (h *EventHandler) OnUpdate(oldObj, newObj interface{}) {
    h.queue.Update(newObj)
}

func (h *EventHandler) OnDelete(obj interface{}) {
    h.queue.Delete(obj)
}

// 消费队列
for {
    h.queue.Pop(func(obj interface{}, isInInitialList bool) error {
        deltas := obj.(cache.Deltas)
        for _, delta := range deltas {
            switch delta.Type {
            case cache.Added:
                // 处理 Added
            case cache.Updated:
                // 处理 Updated
            case cache.Deleted:
                // 处理 Deleted
            }
        }
        return nil
    })
}
```

### 场景 2：批量处理

```go
// 收集多个 Delta 后批量处理
for {
    deltas := make(cache.Deltas, 0, 100)
    
    // Pop 最多 100 个 Delta
    for i := 0; i < 100; i++ {
        _, err := fifo.Pop(func(obj interface{}, isInInitialList bool) error {
            batch := obj.(cache.Deltas)
            deltas = append(deltas, batch...)
            return nil
        })
        if err != nil {
            break
        }
    }
    
    // 批量处理
    processDeltas(deltas)
}
```

### 场景 3：事件过滤

```go
fifo.Pop(func(obj interface{}, isInInitialList bool) error {
    for _, delta := range obj.(cache.Deltas) {
        deploy := delta.Object.(*appsv1.Deployment)
        
        // 只处理特定 Namespace
        if deploy.Namespace != "default" {
            continue
        }
        
        // 只处理特定事件
        if delta.Type != cache.Updated {
            continue
        }
        
        // 处理符合条件的 Delta
        handleUpdate(deploy)
    }
    return nil
})
```

## ⚠️ 注意事项

1. **死锁风险**：Pop 是阻塞操作，队列空时会等待
2. **Pop 后必须处理**：Pop 后 Delta 会被从队列移除
3. **并发安全**：DeltaFIFO 是线程安全的
4. **内存使用**：队列中存储所有未处理的 Delta
5. **返回错误**：Pop 回调返回非 nil 错误会终止 Pop

## 🔧 关键方法

```go
// 添加对象
fifo.Add(obj interface{})

// 更新对象
fifo.Update(obj interface{})

// 删除对象
fifo.Delete(obj interface{})

// 添加 Sync 事件
fifo.Sync(obj interface{})

// 添加 Replaced 事件
fifo.Replace(list []interface{}, resourceVersion string)

// Pop Delta
fifo.Pop(process PopProcessFunc) (interface{}, error)

// 检查队列是否为空
fifo.HasSynced() bool

// 获取所有 Key
fifo.ListKeys() []string

// 获取队列长度
fifo.Len() int
```

## 🔄 与其他组件的关系

```
Reflector
    │ 监听到变更
    │
    ▼
DeltaFIFO (存储事件)
    │ Added/Updated/Deleted
    │
    ▼
Pop 回调 (处理事件)
    │
    ▼
Store/Indexer (更新缓存)
```

## 📚 相关资源

- [DeltaFIFO 源码](https://github.com/kubernetes/client-go/blob/master/tools/cache/delta_fifo.go)
- [Delta 类型定义](https://github.com/kubernetes/client-go/blob/master/tools/cache/delta_fifo.go#L41)
- [FIFO 接口](https://pkg.go.dev/k8s.io/client-go/tools/cache#FIFO)

## 🚀 下一步

继续学习 [2.3 Controller 模式](./2.3-Controller.md)，了解如何组合多个组件实现完整控制器。
