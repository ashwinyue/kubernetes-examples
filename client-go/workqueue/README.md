# 2.4 WorkQueue 深度使用

## 📚 学习目标

- 理解 WorkQueue 的核心机制
- 掌握 RateLimitingQueue 限流
- 了解 Add/Get/Done 完整流程
- 掌握错误重试机制

## 🔍 什么是 WorkQueue

WorkQueue 是一个工作队列，用于在 Controller 中处理资源 Key，保证：
- **公平性**：按添加顺序处理
- **去重**：同一 Key 不会被重复处理
- **限流**：错误时自动限流
- **并发安全**：支持多 Worker 并发处理

## 📖 代码解析

### 示例文件: `main.go`

#### 1. 创建 RateLimitingQueue

```go
// 创建限流队列
queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
defer queue.ShutDown()
```

**DefaultControllerRateLimiter**：
- 基础延迟：5ms
- 最大延迟：1000ms
- 指数退避：连续错误时延迟倍增

#### 2. 创建 Dynamic Informer

```go
factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(
    client,
    5*time.Second,  // Resync 周期
    namespace,        // 监听指定命名空间
    func(*metav1.ListOptions) {},
)
dynamicInformer := factory.ForResource(ConfigMapResource)
```

#### 3. 注册事件处理器

```go
dynamicInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
    AddFunc: func(obj interface{}) {
        key, err := cache.MetaNamespaceKeyFunc(obj)
        if err == nil {
            fmt.Printf("New event: ADD %s\n", key)
            queue.Add(key)  // 将 Key 添加到队列
        }
    },
    UpdateFunc: func(old, new interface{}) {
        key, err := cache.MetaNamespaceKeyFunc(new)
        if err == nil {
            fmt.Printf("New event: UPDATE %s\n", key)
            queue.Add(key)
        }
    },
    DeleteFunc: func(obj interface{}) {
        key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
        if err == nil {
            fmt.Printf("New event: DELETE %s\n", key)
            queue.Add(key)
        }
    },
})
```

#### 4. 启动 Workers

```go
// 创建 3 个并发 Worker
for i := 0; i < 3; i++ {
    go func(n int) {
        for {
            select {
            case <-ctx.Done():
                fmt.Printf("Controller's done! Worker %d exiting...\n", n)
                return
            default:
            }

            // 从队列获取任务
            key, quit := queue.Get()
            if quit {
                fmt.Printf("Work queue has been shut down! Worker %d exiting...\n", n)
                return
            }

            fmt.Printf("Worker %d is about to start process new item %s.\n", n, key)

            // 处理任务
            func() {
                defer queue.Done(key)  // 标记任务完成

                // 业务逻辑
                obj, err := dynamicInformer.Lister().Get(key.(string))
                if err != nil {
                    fmt.Printf("Worker %d got error %v\n", n, err)
                    return
                }

                // Worker 1 故意失败，测试重试
                if n == 1 {
                    err = fmt.Errorf("worker %d is a chronic failure", n)
                }

                // 处理错误
                if err == nil {
                    // 成功处理，从队列中移除
                    fmt.Printf("Worker %d reconciled successfully.\n", n)
                    queue.Forget(key)
                    return
                }

                // 重试次数限制
                if queue.NumRequeues(key) >= 5 {
                    fmt.Printf("Worker %d gave up after 5 retries.\n", n)
                    queue.Forget(key)
                    return
                }

                // 重新入队，稍后重试
                fmt.Printf("Worker %d failed, will retry.\n", n)
                queue.AddRateLimited(key)
            }()
        }
    }(i)
}
```

## 🎯 核心概念

### 1. 队列操作

```go
// 添加 Key 到队列
queue.Add(key)

// 添加 Key（带限流）
queue.AddRateLimited(key)

// 获取 Key（阻塞直到有任务）
key, quit := queue.Get()

// 标记处理完成
queue.Done(key)

// 从队列中移除
queue.Forget(key)
```

### 2. 限流机制

**默认限流器**：
```
初始延迟：5ms
最大延迟：1000ms
退避算法：
  - 第 1 次失败：10ms
  - 第 2 次失败：20ms
  - 第 3 次失败：40ms
  - ...
  - 达到最大值：1000ms
```

**自定义限流器**：

```go
// 线性限流
limiter := workqueue.NewItemExponentialFailureRateLimiter(10*time.Millisecond, 100*time.Millisecond, 2.0)

// 固定延迟
limiter := workqueue.NewMaxOfRateLimiter(
    workqueue.NewItemFastSlowRateLimiter(10*time.Millisecond, 100*time.Millisecond),
    workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 1000*time.Millisecond, 5),
)
```

### 3. 去重机制

**相同 Key 的处理**：
```go
// 第一次添加
queue.Add("default/pod-1")  // 入队

// 第二次添加（处理中）
queue.Add("default/pod-1")  // 忽略，已在处理

// 第三次添加（处理后）
queue.Add("default/pod-1")  // 重新入队
```

**保证**：
- 同一 Key 不会被并发处理
- 多次 Add 会被合并
- 处理完成后再次 Add 会重新入队

### 4. 错误重试

```go
// 获取重试次数
numRequeues := queue.NumRequeues(key)

// 限制重试次数
if numRequeues >= 5 {
    queue.Forget(key)  // 放弃处理
    return
}

// 失败后重试（带限流）
queue.AddRateLimited(key)
```

## 💡 使用模式

### 模式 1：标准 Controller Worker

```go
func (c *Controller) Run(workers int, stopCh <-chan struct{}) {
    for i := 0; i < workers; i++ {
        go wait.Until(func() {
            for c.processNextWorkItem() {
            }
        }, time.Second, stopCh)
    }
}

func (c *Controller) processNextWorkItem() bool {
    key, quit := c.queue.Get()
    if quit {
        return false
    }
    defer c.queue.Done(key)

    err := c.syncHandler(key)
    if err != nil {
        c.queue.AddRateLimited(key)
        return true
    }

    c.queue.Forget(key)
    return true
}
```

### 模式 2：批量处理

```go
func (c *Controller) processBatch(batchSize int) {
    batch := make([]interface{}, 0, batchSize)

    for i := 0; i < batchSize; i++ {
        key, quit := c.queue.Get()
        if quit {
            break
        }
        batch = append(batch, key)
    }

    // 批量处理
    c.processItems(batch)

    // 标记所有完成
    for _, key := range batch {
        c.queue.Done(key)
    }
}
```

### 模式 3：优先级队列

```go
// 创建优先级队列
queue := workqueue.New()

// 添加优先级标记
queue.Add(&item{
    key: "high-priority-pod",
    priority: 10,
})

queue.Add(&item{
    key: "low-priority-pod",
    priority: 1,
})
```

## ⚠️ 注意事项

1. **必须调用 Done()**：每次 Get() 后必须调用 Done()
2. **忘记调用**：会导致内存泄漏
3. **限流影响**：频繁重试会导致延迟累积
4. **并发控制**：通过 Worker 数量控制并发度
5. **优雅关闭**：必须调用 ShutDown()

## 📊 队列状态查询

```go
// 获取队列长度
length := queue.Len()

// 检查是否关闭
shuttingDown := queue.ShuttingDown()

// 检查 Key 是否在队列中
has := queue.Has(key)

// 检查 Key 处理状态
_, exists, _ := queue.Get()
```

## 🔧 最佳实践

### 1. 错误处理

```go
// 临时错误：重试
if isTransientError(err) {
    queue.AddRateLimited(key)
    return
}

// 永久错误：放弃
if isPermanentError(err) {
    queue.Forget(key)
    return
}
```

### 2. 指数退避

```go
// 避免惊群效应
// 初始：10ms
// 递增：指数
// 上限：1000ms
limiter := workqueue.NewItemExponentialFailureRateLimiter(
    10*time.Millisecond,   // 基础延迟
    1000*time.Millisecond,  // 最大延迟
    2.0,               // 指数因子
)
```

### 3. 监控指标

```go
type QueueMetrics struct {
    Adds       int64
    Latency    time.Duration
    Retries    int64
    Errors     int64
}

// 定期收集
metrics := &QueueMetrics{}
metrics.Adds = queue.NumRequeues(key)
```

## 📚 相关资源

- [WorkQueue 文档](https://pkg.go.dev/k8s.io/client-go/util/workqueue)
- [RateLimitingInterface](https://pkg.go.dev/k8s.io/client-go/util/workqueue#RateLimitingInterface)
- [最佳实践](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/controllers.md)

## 🚀 下一步

继续学习 [Informer](../using-informers/)，了解三种类型的 Informer 使用方法。
