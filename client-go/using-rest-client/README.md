# RESTClient 使用

本目录演示 RESTClient 的使用，RESTClient 是 client-go 中最底层的客户端，提供对 Kubernetes API 的直接 HTTP 访问。

## 📋 与 ClientSet 的区别

| 特性 | ClientSet | RESTClient |
|------|----------|------------|
| 类型安全 | ✅ 高度类型安全 | ❌ 需手动处理序列化 |
| 抽象级别 | 高层抽象 | 底层 HTTP 调用 |
| 使用场景 | 常规开发 | 需要精细控制、调试、自定义 API |
| API 覆盖 | 受限 | 可以访问任何 API 端点 |
| 学习曲线 | 低 | 高 |

## 🚀 运行示例

### 1. 创建 Deployment

```bash
cd client-go/using-rest-client
go run creating_deployment.go
```

**输出示例**：
```
deployment.apps/nginx created
```

**验证**：
```bash
kubectl get deployments
kubectl describe deployment nginx
```

### 2. 更新 Deployment 镜像

```bash
go run updating_deployment_image.go
```

**输出示例**：
```
before patching: deployment.apps/nginx image is nginx:1.21.6
after  patching: deployment.apps/nginx image is nginx:1.14.2
```

**验证**：
```bash
kubectl get deployment nginx -o yaml | grep image
```

### 3. 删除 Deployment

```bash
go run deleting_deployment.go
```

**输出示例**：
```
deployment.apps "nginx" deleted
```

**验证**：
```bash
kubectl get deployments
```

### 4. 列出 Pod

```bash
go run listing_pods.go
```

**输出示例**：
```
NAME       STATUS    AGE
coredns-xxx Running    2m
coredns-yyy Running    2m
```

**验证**：
```bash
kubectl get pods -n kube-system
```

## 📚 代码解析

### 1. RESTClient 初始化（所有示例共有）

```go
// 加载 kubeconfig 配置
configLoader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
    clientcmd.NewDefaultClientConfigLoadingRules(),
    &clientcmd.ConfigOverrides{},
)

// 获取当前上下文的 namespace
namespace, _, err := configLoader.Namespace()
if err != nil {
    panic(err)
}

// 获取配置对象
cfg, err := configLoader.ClientConfig()
if err != nil {
    panic(err)
}

// 设置 API 路径和版本
cfg.APIPath = "apis"              // 基础 API 路径
cfg.GroupVersion = &appsv1.SchemeGroupVersion  // Apps Group, v1 版本
cfg.NegotiatedSerializer = scheme.Codecs.WithoutConversion()  // 序列化器

// 创建 RESTClient
rc, err := rest.RESTClientFor(cfg)
if err != nil {
    panic(err.Error())
}
```

**学习要点**：
- `APIPath` - 设置基础 API 路径（`/apis` 或 `/api`）
- `GroupVersion` - 指定 API Group 和版本
- `NegotiatedSerializer` - 序列化和反序列化
- `RESTClientFor()` - 创建 RESTClient 实例

### 2. 创建 Deployment (creating_deployment.go)

```go
// 定义 Deployment 对象
deployment := &appsv1.Deployment{
    ObjectMeta: metav1.ObjectMeta{
        Name: "nginx",
    },
    Spec: appsv1.DeploymentSpec{
        Replicas: i32Ptr(1),
        Selector: &metav1.LabelSelector{
            MatchLabels: map[string]string{
                "app": "nginx",
            },
        },
        Template: corev1.PodTemplateSpec{
            ObjectMeta: metav1.ObjectMeta{
                Labels: map[string]string{
                    "app": "nginx",
                },
            },
            Spec: corev1.PodSpec{
                Containers: []corev1.Container{
                    {
                        Name:  "nginx",
                        Image: "nginx:1.21.6",
                    },
                },
            },
        },
    },
}

// 序列化为 JSON
body, err := json.Marshal(deployment)
if err != nil {
    panic(err.Error())
}

// 创建 Deployment
res := &appsv1.Deployment{}
err = rc.Post().
    Namespace(namespace).
    Resource("deployments").
    Body(body).
    Do(context.TODO()).
    Into(res)
```

**学习要点**：
- `json.Marshal()` - 手动序列化对象
- `rc.Post()` - POST 请求创建资源
- `.Namespace()` - 设置 namespace
- `.Resource()` - 设置资源类型
- `.Body()` - 设置请求体
- `.Do()` - 执行请求
- `.Into()` - 将响应反序列化到对象

**RESTful API 路径**：
```
POST /apis/apps/v1/namespaces/{namespace}/deployments
```

### 3. 更新 Deployment 镜像 (updating_deployment_image.go)

```go
// 先 GET 获取当前 Deployment
res := &appsv1.Deployment{}
err = rc.Get().
    Namespace(namespace).
    Resource("deployments").
    Name("nginx").
    Do(context.TODO()).
    Into(res)

// 打印当前镜像
fmt.Printf("before patching: deployment.apps/%s image is %s\n",
    res.Name, res.Spec.Template.Spec.Containers[0].Image)

// 创建 JSON Patch
patch := []byte(`{"spec":{"template":{"spec":{"containers":[{"name":"nginx","image":"nginx:1.14.2"}]}}}`)

// 应用 Patch
err = rc.Patch(types.StrategicMergePatchType).
    Namespace(namespace).
    Resource("deployments").
    Name("nginx").
    Body(patch).
    Do(context.TODO()).
    Into(res)

// 打印更新后的镜像
fmt.Printf("after  patching: deployment.apps/%s image is %s\n",
    res.Name, res.Spec.Template.Spec.Containers[0].Image)
```

**学习要点**：
- `rc.Get()` - GET 请求获取资源
- `.Name()` - 设置资源名称
- `rc.Patch()` - PATCH 请求更新资源
- `types.StrategicMergePatchType` - Patch 类型

**RESTful API 路径**：
```
GET    /apis/apps/v1/namespaces/{namespace}/deployments/{name}
PATCH  /apis/apps/v1/namespaces/{namespace}/deployments/{name}
```

### 4. 删除 Deployment (deleting_deployment.go)

```go
// 删除 Deployment
res := &metav1.Status{}
err = rc.Delete().
    Namespace(namespace).
    Resource("deployments").
    Name("nginx").
    Do(context.TODO()).
    Into(res)

// 处理 NotFound 错误
if err != nil {
    if errors.IsNotFound(err) {
        fmt.Printf("%s\n", err.Error())
        return
    }
    panic(err.Error())
}

fmt.Printf("deployment.apps \"nginx\" delete: %s\n", res.Status)
```

**学习要点**：
- `rc.Delete()` - DELETE 请求删除资源
- 错误处理
- `metav1.Status{}` - 保存删除结果状态

**RESTful API 路径**：
```
DELETE /apis/apps/v1/namespaces/{namespace}/deployments/{name}
```

### 5. 列出 Pod (listing_pods.go)

```go
// 设置 API 路径为 `/api` (legacy 资源)
cfg.APIPath = "api"
cfg.GroupVersion = &corev1.SchemeGroupVersion

// 列出 Pod
res := &corev1.PodList{}
err = rc.Get().
    Namespace(metav1.NamespaceSystem).
    Resource("pods").
    Do(context.TODO()).
    Into(res)

// 格式化输出
w := new(tabwriter.Writer)
w.Init(os.Stdout, 5, 0, 3, ' ', 0)

// 打印表头
fmt.Fprintln(w, strings.Join([]string{"NAME", "STATUS", "AGE"}, "\t"))

// 打印每行数据
for _, p := range res.Items {
    age := time.Since(p.CreationTimestamp.Time).Round(time.Second)
    fmt.Fprintln(w, strings.Join([]string{
        p.Name,
        string(p.Status.Phase),
        fmt.Sprintf("%dm", int(age.Minutes())),
    }, "\t"))
}

w.Flush()
```

**学习要点**：
- `APIPath = "api"` - 用于 legacy 核心资源
- `Namespace(metav1.NamespaceSystem)` - 指定 namespace
- `corev1.PodList{}` - 接收列表响应
- `res.Items` - 访问列表项

**RESTful API 路径**：
```
GET /api/v1/namespaces/{namespace}/pods
```

## 🎯 学习要点总结

### RESTClient 核心概念

1. **底层 HTTP 客户端**
   - 直接调用 Kubernetes API
   - 完全控制请求和响应
   - 需要手动处理序列化

2. **Fluent Interface（流式接口）**
   ```go
   rc.Post().Namespace(ns).Resource("rsc").Body(body).Do(ctx).Into(res)
   rc.Get().Namespace(ns).Resource("rsc").Name("name").Do(ctx).Into(res)
   rc.Patch(ptype).Namespace(ns).Resource("rsc").Name("name").Body(patch).Do(ctx).Into(res)
   rc.Delete().Namespace(ns).Resource("rsc").Name("name").Do(ctx).Into(res)
   ```

3. **API 路径构建**
   - `/api/v1` - 核心资源（Pod, Service）
   - `/apis/{group}/{version}` - 扩展资源（Deployment, CRD）

4. **序列化和反序列化**
   - 使用 `json.Marshal()` 序列化请求体
   - 使用 `scheme.Codecs` 处理响应
   - 使用 `.Into()` 自动反序列化

### RESTClient vs ClientSet

**使用 RESTClient 的场景**：
1. 需要精细控制 HTTP 请求
2. 调试 API 调用
3. 访问未在 ClientSet 中的 API 端点
4. 学习 Kubernetes API 底层原理
5. 性能优化（减少序列化开销）

**使用 ClientSet 的场景**：
1. 常规开发
2. 类型安全更重要
3. 代码可读性要求高
4. 快速开发

### RESTful API 映射

| 资源 | RESTClient 方法 | HTTP 方法 | 路径 |
|------|---------------|-----------|------|
| 创建 | `Post()` | POST | `/apis/{group}/{version}/namespaces/{ns}/deployments` |
| 获取单个 | `Get()` + `Name()` | GET | `/apis/{group}/{version}/namespaces/{ns}/deployments/{name}` |
| 获取列表 | `Get()` | GET | `/apis/{group}/{version}/namespaces/{ns}/deployments` |
| 更新 | `Patch()` | PATCH | `/apis/{group}/{version}/namespaces/{ns}/deployments/{name}` |
| 删除 | `Delete()` | DELETE | `/apis/{group}/{version}/namespaces/{ns}/deployments/{name}` |

## 🔧 高级用法

### 1. 自定义查询参数

```go
err = rc.Get().
    Namespace(namespace).
    Resource("pods").
    VersionedParams(&metav1.ListOptions{
        LabelSelector: "app=nginx",
        Limit: 10,
    }).
    Do(context.TODO()).
    Into(res)
```

### 2. 设置自定义 Headers

```go
err = rc.Get().
    Namespace(namespace).
    Resource("pods").
    SetHeader("User-Agent", "my-client/1.0").
    Do(context.TODO()).
    Into(res)
```

### 3. 处理原始响应

```go
result := rc.Get().
    Namespace(namespace).
    Resource("pods").
    Do(context.TODO())

// 获取原始响应
body, err := result.Raw()
if err != nil {
    panic(err)
}

// 手动解析
fmt.Println(string(body))
```

### 4. 错误处理

```go
result := rc.Post().
    Namespace(namespace).
    Resource("deployments").
    Body(body).
    Do(context.TODO())

// 获取错误
err := result.Error()
if err != nil {
    // 判断错误类型
    if errors.IsNotFound(err) {
        // 处理 NotFound
    } else if errors.IsConflict(err) {
        // 处理 Conflict
    } else {
        panic(err)
    }
}
```

## 🚧 常见问题

### Q: 什么时候使用 RESTClient？

A:
- ✅ 需要精细控制 HTTP 请求时
- ✅ 调试 API 调用时
- ✅ 访问自定义或未文档化的 API 端点
- ✅ 性能敏感场景（减少序列化开销）

❌ 不推荐用于：
- 常规开发（使用 ClientSet）
- 需要类型安全（使用 ClientSet）

### Q: 如何选择 APIPath？

A:
- `api` - 用于 legacy 核心资源（Pod, Service, Node, Namespace）
- `apis` - 用于扩展资源（Deployment, StatefulSet, CRD）

示例：
```go
// Pod (legacy resource)
cfg.APIPath = "api"

// Deployment (extension resource)
cfg.APIPath = "apis"
```

### Q: 如何调试 RESTClient 调用？

A:
```go
// 启用详细日志
cfg.ContentConfig = rest.ContentConfig{
    ContentType:          "application/json",
    GroupVersion:         &corev1.SchemeGroupVersion,
    NegotiatedSerializer: scheme.Codecs,
}

// 打印请求和响应
fmt.Printf("Request URL: %s\n", rc.Get().URL().String())
```

### Q: RESTClient 性能如何？

A:
RESTClient 比 ClientSet 更快，因为：
1. 减少了类型转换开销
2. 直接序列化/反序列化
3. 更少的中间层

但代码复杂度更高，需要在性能和可维护性之间权衡。

## 📖 下一步

完成本阶段后，继续学习：

- [阶段 1.5: Discovery Client](../using-discovery-client/)
- [阶段 1.6: Dynamic Client](../using-dynamic-interface/)

回到 [主 README](../../LEARNING_PATH.md)
