# Dynamic Client 使用示例

本目录展示了如何使用 Kubernetes Dynamic Client 动态操作任意类型的 Kubernetes 资源，包括自定义资源（CRD）。

## 📚 学习目标

- 理解 Dynamic Client 的核心概念和优势
- 掌握 Unstructured 类型的使用
- 学会动态操作标准 K8s 资源
- 掌握 CRD 的动态操作

## 🔍 什么是 Dynamic Client

Dynamic Client 是 client-go 提供的一个特殊的客户端，它可以在运行时操作**任何**类型的 Kubernetes 资源，而无需预先生成类型化的代码。

### 核心特点

1. **类型无关**：使用 `unstructured.Unstructured` 处理所有资源
2. **运行时灵活**：通过 GVR（GroupVersionResource）指定资源
3. **通用 CRUD**：统一的 API 操作所有资源类型
4. **CRD 友好**：特别适合操作自定义资源

### Client 对比

| 客户端 | 类型安全 | 灵活性 | 适用场景 |
|--------|---------|--------|----------|
| **ClientSet** | ✅ 强类型 | ❌ 需要预生成代码 | 已知的标准资源 |
| **RESTClient** | ❌ 原始 HTTP | ✅ 完全灵活 | 需要完全控制 HTTP 请求 |
| **DiscoveryClient** | - | ✅ 只读发现 | 发现资源信息 |
| **DynamicClient** | ⚠️ 弱类型 | ✅ 高度灵活 | 未知资源、CRD、通用操作 |

### Unstructured 类型

`Unstructured` 是一个通用的容器，使用 `map[string]interface{}` 存储任意资源数据：

```go
type Unstructured struct {
    // Object 存储完整的资源数据
    Object map[string]interface{}
}
```

## 📁 目录结构

```
using-dynamic-interface/
├── list-pods/              # 列出 Pod 示例
│   └── main.go
├── create-pod/             # 创建 Pod 示例
│   └── main.go
├── get-and-update-crds/    # 获取和更新 CRD 示例
│   ├── main.go
│   ├── pizza_crd.yaml      # Pizza CRD 定义
│   └── margherita.yaml     # Pizza 实例
└── README.md               # 本文档
```

## 🚀 运行示例

### 1. 列出 Pod

```bash
cd /Users/mervyn/go/src/github/kubernetes-examples/client-go/using-dynamic-interface/list-pods
go run main.go
```

**输出示例**：
```
coredns-5d78c9869d-abcde
coredns-5d78c9869d-fghij
etcd-onex-control-plane
kindnet-54qwz
kube-apiserver-onex-control-plane
kube-controller-manager-onex-control-plane
kube-proxy-f5g8j
kube-scheduler-onex-control-plane
```

### 2. 创建 Pod

```bash
cd ../create-pod
go run main.go
```

**输出示例**：
```
Created pod "example-pod".
```

**验证**：
```bash
kubectl get pod example-pod
```

### 3. 操作 CRD（自定义资源）

#### 3.1 应用 CRD

```bash
cd ../get-and-update-crds
kubectl apply -f pizza_crd.yaml
```

**验证 CRD**：
```bash
kubectl get crd | grep pizza
# NAME                    CREATED AT
# pizzas.bella.napoli.it   2025-12-28T16:20:00Z
```

#### 3.2 创建 Pizza 实例

```bash
kubectl apply -f margherita.yaml
```

**查看 Pizza 实例**：
```bash
kubectl get pizzas
# NAME         COST (€)
# margherita   5.00
```

#### 3.3 运行更新程序

```bash
go run main.go
```

**验证更新**：
```bash
kubectl get pizzas
# NAME         COST (€)
# margherita   6.50
```

## 📖 代码详解

### 示例 1：列出 Pod (`list-pods/main.go`)

```go
// 创建 Dynamic Client（第 34-38 行）
dc, err := dynamic.NewForConfig(rc)
if err != nil {
    panic(err.Error())
}

// 定义 GVR - 资源的"身份证"（第 41-44 行）
gvr := schema.GroupVersionResource{
    Version:  "v1",        // API 版本
    Resource: "pods",      // 资源名称（复数形式）
    // Group: ""           // 核心资源 Group 为空
}

// 列出指定命名空间的所有 Pod（第 47-49 行）
res, err := dc.Resource(gvr).
    Namespace(*namespace).
    List(context.TODO(), metav1.ListOptions{})

// 遍历结果并打印 Pod 名称（第 57-59 行）
for _, el := range res.Items {
    fmt.Printf("%v\n", el.GetName())
}
```

**关键点**：
- `dc.Resource(gvr)` - 获取资源接口
- `.Namespace(ns)` - 指定命名空间
- `.List()` - 列出资源
- `Items` - Unstructured 列表

### 示例 2：创建 Pod (`create-pod/main.go`)

```go
// 创建 Dynamic Client（第 31-34 行）
client, err := dynamic.NewForConfig(config)
if err != nil {
    panic(err)
}

// 定义 Pod 的 Unstructured 对象（第 42-58 行）
obj := &unstructured.Unstructured{
    Object: map[string]interface{}{
        "apiVersion": "v1",
        "kind":       "Pod",
        "metadata": map[string]interface{}{
            "name": "example-pod",
        },
        "spec": map[string]interface{}{
            "containers": []map[string]interface{}{
                {
                    "name":  "nginx",
                    "image": "nginx:latest",
                },
            },
        },
    },
}

// 创建 Pod（第 59-61 行）
result, err := client.Resource(gvr).
    Namespace(corev1.NamespaceDefault).
    Create(context.TODO(), obj, metav1.CreateOptions{})
```

**关键点**：
- 手动构建 `map[string]interface{}`
- 嵌套结构用 `[]map[string]interface{}`
- `Create()` 返回创建后的 Unstructured 对象

### 示例 3：操作 CRD (`get-and-update-crds/main.go`)

```go
// 定义 CRD 的 GVR（第 35-39 行）
gvr := schema.GroupVersionResource{
    Group:    "bella.napoli.it",  // 自定义 API 组
    Version:  "v1alpha1",          // API 版本
    Resource: "pizzas",            // 资源名称
}

// 获取名为 'margherita' 的 Pizza（第 41-43 行）
res, err := dc.Resource(gvr).
    Namespace(namespace).
    Get(context.TODO(), "margherita", metav1.GetOptions{})

// 获取或创建 status（第 52-56 行）
status, ok := res.Object["status"]
if !ok {
    status = make(map[string]interface{})
}

// 更新 price（第 59 行）
status.(map[string]interface{})["cost"] = 6.50
res.Object["status"] = status

// 更新 CRD（第 63 行）
_, err = dc.Resource(gvr).Namespace(namespace).
    Update(context.TODO(), res, metav1.UpdateOptions{})
```

**关键点**：
- 自定义资源的 Group 通常不是空
- 需要类型断言 `status.(map[string]interface{})`
- 可以修改 `.Object` 中的任何字段

## 🎯 学习要点

### 1. GroupVersionResource（GVR）

GVR 是标识 Kubernetes 资源的"三要素"：

```go
type GroupVersionResource struct {
    Group    string  // API 组，如 "apps", "networking.k8s.io"
    Version  string  // 版本，如 "v1", "v1beta1"
    Resource string  // 资源名称（复数），如 "pods", "deployments"
}
```

**常见 GVR 示例**：

| 资源 | Group | Version | Resource |
|------|-------|---------|----------|
| Pod | `""` | v1 | pods |
| Deployment | apps | v1 | deployments |
| Service | `""` | v1 | services |
| Ingress | networking.k8s.io | v1 | ingresses |
| Custom Resource | 自定义 | v1alpha1 | 自定义 |

### 2. Unstructured 操作

#### 获取字段

```go
// 简单字段
name := unstructuredObj.GetName()
namespace := unstructuredObj.GetNamespace()
apiVersion := unstructuredObj.GetAPIVersion()
kind := unstructuredObj.GetKind()

// 嵌套字段
spec := unstructuredObj.Object["spec"]
containerName := spec.(map[string]interface{})["containers"].([]map[string]interface{})[0]["name"]
```

#### 设置字段

```go
// 设置简单字段
unstructuredObj.SetName("new-name")
unstructuredObj.SetNamespace("default")

// 设置嵌套字段
unstructuredObj.Object["spec"].(map[string]interface{})["replicas"] = 3

// 使用 Unstructured.SetNestedField
unstructured.SetNestedField(unstructuredObj.Object, 3, "spec", "replicas")
```

#### 删除字段

```go
// 删除简单字段
delete(unstructuredObj.Object, "labels")

// 删除嵌套字段
delete(unstructuredObj.Object["spec"].(map[string]interface{}), "replicas")
```

### 3. Dynamic Client API

#### 基础 CRUD

```go
// Create
obj := &unstructured.Unstructured{...}
result, err := dc.Resource(gvr).Namespace(ns).
    Create(context.TODO(), obj, metav1.CreateOptions{})

// Get
obj, err := dc.Resource(gvr).Namespace(ns).
    Get(context.TODO(), name, metav1.GetOptions{})

// List
list, err := dc.Resource(gvr).Namespace(ns).
    List(context.TODO(), metav1.ListOptions{})

// Update
obj, err := dc.Resource(gvr).Namespace(ns).
    Update(context.TODO(), obj, metav1.UpdateOptions{})

// Delete
err := dc.Resource(gvr).Namespace(ns).
    Delete(context.TODO(), name, metav1.DeleteOptions{})
```

#### 高级操作

```go
// Patch
patch := []byte(`{"spec":{"replicas":5}}`)
result, err := dc.Resource(gvr).Namespace(ns).
    Patch(context.TODO(), name, types.MergePatchType, patch, metav1.PatchOptions{})

// Watch
watcher, err := dc.Resource(gvr).Namespace(ns).
    Watch(context.TODO(), metav1.ListOptions{})
for event := range watcher.ResultChan() {
    obj := event.Object.(*unstructured.Unstructured)
    fmt.Printf("Event: %s, Type: %s\n", obj.GetName(), event.Type)
}

// DeleteCollection
err := dc.Resource(gvr).Namespace(ns).
    DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{})
```

### 4. Dynamic Client vs ClientSet

#### 使用 Dynamic Client 的场景

```go
// ✅ 动态操作未知资源
gvr := schema.GroupVersionResource{
    Group:    "custom.example.com",
    Version:  "v1",
    Resource: "myresources",
}
obj, _ := dc.Resource(gvr).Get(...)

// ✅ 通用 CRUD 框架
func operateOnResource(gvr schema.GroupVersionResource, name string) {
    obj, _ := dc.Resource(gvr).Get(...)
    // 通用处理逻辑
}
```

#### 使用 ClientSet 的场景

```go
// ✅ 类型安全，编译时检查
deployment, err := clientset.AppsV1().Deployments(ns).
    Get(context.TODO(), name, metav1.GetOptions{})
fmt.Println(deployment.Spec.Replicas) // ✅ 有类型提示

// ❌ Dynamic Client 需要类型断言
replicas, ok := obj.Object["spec"].(map[string]interface{})["replicas"].(int)
if !ok {
    // 错误处理
}
```

### 5. 性能考虑

**Dynamic Client 性能开销**：
1. JSON 序列化/反序列化
2. 类型断言
3. 没有编译时优化

**优化建议**：
- 频繁操作已知资源 → 使用 ClientSet
- 一次性操作或 CRD → 使用 Dynamic Client
- 缓存常用的 GVR

### 6. 错误处理

```go
import "k8s.io/apimachinery/pkg/api/errors"

// 检查 NotFound
obj, err := dc.Resource(gvr).Get(...)
if errors.IsNotFound(err) {
    fmt.Println("资源不存在")
}

// 检查 AlreadyExists
_, err := dc.Resource(gvr).Create(...)
if errors.IsAlreadyExists(err) {
    fmt.Println("资源已存在")
}

// 检查 Conflict
_, err := dc.Resource(gvr).Update(...)
if errors.IsConflict(err) {
    fmt.Println("资源版本冲突，需要重试")
}
```

## 🛠️ 实用代码模式

### 模式 1：通用资源操作器

```go
func updateLabel(dc dynamic.Interface, gvr schema.GroupVersionResource, namespace, name, key, value string) error {
    obj, err := dc.Resource(gvr).Namespace(namespace).
        Get(context.TODO(), name, metav1.GetOptions{})
    if err != nil {
        return err
    }

    labels, ok := obj.GetLabels()
    if !ok {
        labels = make(map[string]string)
    }
    labels[key] = value
    obj.SetLabels(labels)

    _, err = dc.Resource(gvr).Namespace(namespace).
        Update(context.TODO(), obj, metav1.UpdateOptions{})
    return err
}
```

### 模式 2：批量操作资源

```go
func scaleAllDeployments(dc dynamic.Interface, namespace string, replicas int) error {
    gvr := schema.GroupVersionResource{
        Group:    "apps",
        Version:  "v1",
        Resource: "deployments",
    }

    list, err := dc.Resource(gvr).Namespace(namespace).
        List(context.TODO(), metav1.ListOptions{})
    if err != nil {
        return err
    }

    for _, item := range list.Items {
        unstructured.SetNestedField(item.Object, replicas, "spec", "replicas")
        _, err := dc.Resource(gvr).Namespace(namespace).
            Update(context.TODO(), &item, metav1.UpdateOptions{})
        if err != nil {
            return err
        }
    }
    return nil
}
```

### 模式 3：动态 Watch 资源

```go
func watchResources(dc dynamic.Interface, gvr schema.GroupVersionResource, namespace string) {
    watcher, err := dc.Resource(gvr).Namespace(namespace).
        Watch(context.TODO(), metav1.ListOptions{})
    if err != nil {
        panic(err)
    }

    for event := range watcher.ResultChan() {
        obj := event.Object.(*unstructured.Unstructured)
        switch event.Type {
        case watch.Added:
            fmt.Printf("Added: %s\n", obj.GetName())
        case watch.Modified:
            fmt.Printf("Modified: %s\n", obj.GetName())
        case watch.Deleted:
            fmt.Printf("Deleted: %s\n", obj.GetName())
        }
    }
}
```

## ⚠️ 注意事项

1. **类型安全**：Dynamic Client 缺少编译时类型检查，需要仔细处理类型断言
2. **性能**：相比 ClientSet 有额外的序列化开销
3. **错误处理**：务必检查类型断言的结果
4. **GVR 正确性**：确保 Group、Version、Resource 都正确
5. **字段修改**：直接修改 `.Object` 后必须调用 `Update()`

## 📚 相关资源

- [Dynamic Client 文档](https://github.com/kubernetes/client-go/tree/master/dynamic)
- [Unstructured 类型文档](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured)
- [CRD 开发指南](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/)

## 🚀 下一步

继续学习 [Informer 与 Controller](../using-informers/)，掌握 Kubernetes 控制器的核心机制！
