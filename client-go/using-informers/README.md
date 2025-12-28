# 2.5 Informer 机制

## 📚 学习目标

- 掌握 SharedInformerFactory 的使用
- 了解三种类型的 Informer
- 熟悉事件处理器和 Lister 缓存

## 🔍 什么是 Informer

Informer 是 client-go 的高级抽象，组合了 Reflector、DeltaFIFO、Indexer 等组件，提供了简洁的事件驱动接口。

## 📖 三种类型的 Informer

### 2.5.1 Typed Informer

**文件**: `informer-typed-simple/`

**特点**：
- ✅ 类型安全，强类型 API
- ✅ 编译时类型检查
- ✅ IDE 自动补全支持

**使用场景**：
- 处理标准 Kubernetes 资源
- 需要类型安全的代码
- 已知资源结构

**示例代码**：

```go
// 创建 SharedInformerFactory
factory := informers.NewSharedInformerFactory(clientset, 5*time.Second)

// 获取 Typed Informer
cmInformer := factory.Core().V1().ConfigMaps().Informer()

// 注册事件处理器（类型安全）
cmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
    AddFunc: func(obj interface{}) {
        cm := obj.(*corev1.ConfigMap)  // ✅ 类型断言
        fmt.Printf("ConfigMap ADDED: %s/%s\n", cm.Namespace, cm.Name)
    },
    UpdateFunc: func(old, new interface{}) {
        newCM := new.(*corev1.ConfigMap)
        fmt.Printf("ConfigMap UPDATED: %s/%s\n", newCM.Namespace, newCM.Name)
    },
    DeleteFunc: func(obj interface{}) {
        cm := obj.(*corev1.ConfigMap)
        fmt.Printf("ConfigMap DELETED: %s/%s\n", cm.Namespace, cm.Name)
    },
})

// 启动
factory.Start(ctx.Done())
cache.WaitForCacheSync(ctx.Done(), cmInformer.HasSynced)
```

### 2.5.2 Generic Informer

**文件**: `informer-generic-simple/`

**特点**：
- ✅ 通用 Informer，不依赖具体类型
- ✅ 通过 GVR 指定资源
- ⚠️ 需要类型断言

**使用场景**：
- 处理已知 API 组的资源
- 需要统一接口
- 减少代码重复

**示例代码**：

```go
// 创建 SharedInformerFactory
factory := informers.NewSharedInformerFactory(clientset, 5*time.Second)

// 通过 GVR 获取 Generic Informer
gvr := schema.GroupVersionResource{
    Group:    "",
    Version:  "v1",
    Resource: "configmaps",
}
cmInformer, _ := factory.ForResource(gvr)

// 注册事件处理器（需要类型断言）
cmInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
    AddFunc: func(obj interface{}) {
        cm := obj.(*corev1.ConfigMap)  // ⚠️ 手动类型断言
        fmt.Printf("ConfigMap ADDED: %s/%s\n", cm.Namespace, cm.Name)
    },
    UpdateFunc: func(old, new interface{}) {
        cm := new.(*corev1.ConfigMap)
        fmt.Printf("ConfigMap UPDATED: %s/%s\n", cm.Namespace, cm.Name)
    },
    DeleteFunc: func(obj interface{}) {
        cm := obj.(*corev1.ConfigMap)
        fmt.Printf("ConfigMap DELETED: %s/%s\n", cm.Namespace, cm.Name)
    },
})

// 启动
factory.Start(ctx.Done())
cache.WaitForCacheSync(ctx.Done(), cmInformer.Informer().HasSynced)
```

### 2.5.3 Dynamic Informer

**文件**: `informer-dynamic-simple/`

**特点**：
- ✅ 完全动态，无需预生成代码
- ✅ 适用于 CRD 和未知资源
- ✅ 使用 Unstructured 类型

**使用场景**：
- 操作自定义资源（CRD）
- 处理未知类型的资源
- 需要最大灵活性

**示例代码**：

```go
// 创建 Dynamic Client
client, _ := dynamic.NewForConfig(config)

// 创建 Dynamic SharedInformerFactory
factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(
    client,
    5*time.Second,
    namespace,
    func(*metav1.ListOptions) {},
)

// 通过 GVR 获取 Dynamic Informer
gvr := schema.GroupVersionResource{
    Group:    "bella.napoli.it",
    Version:  "v1alpha1",
    Resource: "pizzas",
}
dynamicInformer := factory.ForResource(gvr)

// 注册事件处理器（使用 Unstructured）
dynamicInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
    AddFunc: func(obj interface{}) {
        pizza := obj.(*unstructured.Unstructured)
        name := pizza.GetName()
        fmt.Printf("Pizza ADDED: %s\n", name)
    },
    UpdateFunc: func(old, new interface{}) {
        newPizza := new.(*unstructured.Unstructured)
        fmt.Printf("Pizza UPDATED: %s\n", newPizza.GetName())
    },
    DeleteFunc: func(obj interface{}) {
        pizza := obj.(*unstructured.Unstructured)
        fmt.Printf("Pizza DELETED: %s\n", pizza.GetName())
    },
})

// 启动
factory.Start(ctx.Done())
factory.WaitForCacheSync(ctx.Done())
```

## 🎯 Informer 核心组件

### 1. SharedInformerFactory

```go
factory := informers.NewSharedInformerFactory(
    clientset,           // Kubernetes ClientSet
    5*time.Second,         // Resync 周期
)

// 过滤特定命名空间
filteredFactory := informers.NewFilteredSharedInformerFactory(
    clientset,
    5*time.Second,
    namespace,           // 只监听特定命名空间
    func(listOptions *metav1.ListOptions) {
        listOptions.LabelSelector = labels.SelectorFromSet(labels.Set{"app": "myapp"})
    },
)
```

**优势**：
- 共享缓存，减少资源消耗
- 统一管理所有 Informer
- 自动启动和停止

### 2. Lister

Lister 提供从本地缓存查询资源的能力：

```go
// 获取 Typed Lister
configMapLister := factory.Core().V1().ConfigMaps().Lister()

// 查询所有 ConfigMap
configs, err := configMapLister.List(labels.Everything())

// 查询特定 ConfigMap
cm, err := configMapLister.ConfigMaps(namespace).Get("my-config")

// 按命名空间过滤
configsInNs, err := configMapLister.ConfigMaps(namespace).List(labels.Everything())
```

**优势**：
- 从本地缓存读取，速度快
- 不访问 API Server
- 线程安全

### 3. ResourceEventHandler

```go
type ResourceEventHandler interface {
    OnAdd(obj interface{})
    OnUpdate(oldObj, newObj interface{})
    OnDelete(obj interface{})
}

type ResourceEventHandlerFuncs struct {
    AddFunc    func(obj interface{})
    UpdateFunc func(oldObj, newObj interface{})
    DeleteFunc func(obj interface{})
}
```

## 💡 使用模式

### 模式 1：单一资源监控

```go
factory := informers.NewSharedInformerFactory(clientset, 30*time.Second)
podInformer := factory.Core().V1().Pods().Informer()

podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
    AddFunc: func(obj interface{}) {
        pod := obj.(*corev1.Pod)
        fmt.Printf("Pod %s added\n", pod.Name)
    },
})

factory.Start(stopCh)
```

### 模式 2：多资源监控

```go
factory := informers.NewSharedInformerFactory(clientset, 30*time.Second)

// 监控 Pod
podInformer := factory.Core().V1().Pods().Informer()
podInformer.AddEventHandler(podHandler)

// 监控 Deployment
deployInformer := factory.Apps().V1().Deployments().Informer()
deployInformer.AddEventHandler(deployHandler)

factory.Start(stopCh)
```

### 模式 3：条件过滤

```go
filteredFactory := informers.NewFilteredSharedInformerFactory(
    clientset,
    30*time.Second,
    "default",
    func(options *metav1.ListOptions) {
        options.LabelSelector = labels.SelectorFromSet(labels.Set{"app": "myapp"})
    },
)

podInformer := filteredFactory.Core().V1().Pods().Informer()
```

### 模式 4：事件去重

```go
type DeletionTrackingHandler struct {
    queue workqueue.RateLimitingInterface
    cache cache.Store
}

func (h *DeletionTrackingHandler) OnAdd(obj interface{}) {
    key, _ := cache.MetaNamespaceKeyFunc(obj)
    h.cache.Add(obj)
    h.queue.Add(key)
}

func (h *DeletionTrackingHandler) OnUpdate(old, new interface{}) {
    oldKey, _ := cache.MetaNamespaceKeyFunc(old)
    newKey, _ := cache.MetaNamespaceKeyFunc(new)
    
    if oldKey != newKey {
        // Key 变化，重新处理
        h.queue.Add(newKey)
    }
}

func (h *DeletionTrackingHandler) OnDelete(obj interface{}) {
    key, _ := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
    h.cache.Delete(obj)
    h.queue.Add(key)
}
```

## ⚠️ 注意事项

1. **缓存同步**：必须等待 `WaitForCacheSync`
2. **事件顺序**：不保证事件顺序
3. **资源版本**：NewObj 可能比触发事件的版本更新
4. **内存使用**：所有资源都会缓存到内存
5. **Resync 影响**：会触发所有资源的 Update 事件
6. **线程安全**：Lister 是线程安全的

## 🔄 完整工作流程

```
1. Factory.Start()
       │
       ▼
2. Informer.Run()
       │
       ▼
3. Reflector.List()     ──► 初始同步
4. Reflector.Watch()    ──► 持续监听
       │
       ▼
5. DeltaFIFO
       │
       ▼
6. Indexer.Update()   ──► 更新缓存
       │
       ▼
7. ResourceEventHandler  ──► 触发回调
       │
       ▼
8. 业务逻辑          ──► 处理资源
       │
       ▼
9. Lister.Get()       ──► 从缓存查询
```

## 📚 相关资源

- [SharedInformerFactory 文档](https://pkg.go.dev/k8s.io/client-go/informers#SharedInformerFactory)
- [DynamicSharedInformerFactory](https://pkg.go.dev/k8s.io/client-go/dynamic/dynamicinformer#NewDynamicSharedInformerFactory)
- [Lister 接口](https://pkg.go.dev/k8s.io/client-go/listers#Lister)

## 🚀 阶段 2 总结

完成本阶段学习后，你将掌握：

✅ Reflector 的 Watch/List 机制
✅ DeltaFIFO 的事件队列
✅ Controller 模式和架构
✅ WorkQueue 的并发处理
✅ 三种 Informer 的使用
✅ 事件驱动编程模式

## 🎓 下一步

继续学习 [阶段 3: CRD 与 Operator](../../LEARNING_PATH.md#阶段-3-crd-与-operator)
