# Discovery Client 使用示例

本目录展示了如何使用 Kubernetes Discovery Client 来发现和查询 API Server 支持的所有资源类型。

## 📚 学习目标

- 理解 Discovery Client 的作用和用途
- 掌握如何发现集群支持的 API 资源
- 了解 Cache Discovery Client 的优势和使用场景

## 🔍 什么是 Discovery Client

Discovery Client 是 client-go 提供的一个专门用于发现 Kubernetes API Server 支持的所有资源类型的客户端。

### 主要用途

1. **动态发现资源**：在运行时发现集群支持的资源类型、版本和操作
2. **API 版本协商**：自动选择服务器首选的 API 版本
3. **资源验证**：检查特定资源类型是否支持某种操作（如 create、list、watch 等）
4. **兼容性检查**：在运行时确保代码兼容不同版本的 K8s 集群

### 两种 Discovery Client

| 类型 | 描述 | 优势 | 适用场景 |
|------|------|------|----------|
| DiscoveryClient | 直接从 API Server 获取资源信息 | 数据实时，无缓存 | 需要最新资源信息 |
| CachedDiscoveryClient | 缓存 API Server 的资源信息 | 减少网络请求，性能更好 | 频繁查询资源信息 |

## 📁 文件说明

- `discovery_client.go` - 使用标准 DiscoveryClient 查询所有资源
- `cached_discovery_client.go` - 使用带缓存的 DiscoveryClient 查询所有资源

## 🚀 运行示例

### 1. 标准 DiscoveryClient

```bash
cd /Users/mervyn/go/src/github/kubernetes-examples/client-go/using-discovery-client
go run discovery_client.go
```

**输出示例**：
```json
{"kind":"Binding","apiVersion":"v1","name":"bindings","verbs":["create"]}
{"kind":"ComponentStatus","apiVersion":"v1","name":"componentstatuses","verbs":["get","list"]}
{"kind":"ConfigMap","apiVersion":"v1","name":"configmaps","verbs":["create","delete","deletecollection","get","list","patch","update","watch"]}
{"kind":"Endpoints","apiVersion":"v1","name":"endpoints","verbs":["create","delete","deletecollection","get","list","patch","update","watch"]}
{"kind":"Event","apiVersion":"v1","name":"events","verbs":["create","delete","deletecollection","get","list","patch","update","watch"]}
{"kind":"LimitRange","apiVersion":"v1","name":"limitranges","verbs":["create","delete","deletecollection","get","list","patch","update","watch"]}
{"kind":"Namespace","apiVersion":"v1","name":"namespaces","verbs":["create","delete","get","list","patch","update","watch"]}
{"kind":"Node","apiVersion":"v1","name":"nodes","verbs":["create","delete","deletecollection","get","list","patch","update","watch"]}
{"kind":"PersistentVolume","apiVersion":"v1","name":"persistentvolumes","verbs":["create","delete","deletecollection","get","list","patch","update","watch"]}
{"kind":"PersistentVolumeClaim","apiVersion":"v1","name":"persistentvolumeclaims","verbs":["create","delete","deletecollection","get","list","patch","update","watch"]}
{"kind":"Pod","apiVersion":"v1","name":"pods","verbs":["create","delete","deletecollection","get","list","patch","update","watch"]}
{"kind":"PodTemplate","apiVersion":"v1","name":"podtemplates","verbs":["create","delete","deletecollection","get","list","patch","update","watch"]}
{"kind":"ReplicationController","apiVersion":"v1","name":"replicationcontrollers","verbs":["create","delete","deletecollection","get","list","patch","update","watch"]}
{"kind":"ResourceQuota","apiVersion":"v1","name":"resourcequotas","verbs":["create","delete","deletecollection","get","list","patch","update","watch"]}
{"kind":"Secret","apiVersion":"v1","name":"secrets","verbs":["create","delete","deletecollection","get","list","patch","update","watch"]}
{"kind":"ServiceAccount","apiVersion":"v1","name":"serviceaccounts","verbs":["create","delete","deletecollection","get","list","patch","update","watch"]}
{"kind":"Service","apiVersion":"v1","name":"services","verbs":["create","delete","get","list","patch","update","watch"]}
{"kind":"Deployment","apiVersion":"apps/v1","name":"deployments","verbs":["create","delete","deletecollection","get","list","patch","update","watch"]}
{"kind":"StatefulSet","apiVersion":"apps/v1","name":"statefulsets","verbs":["create","delete","deletecollection","get","list","patch","update","watch"]}
{"kind":"DaemonSet","apiVersion":"apps/v1","name":"daemonsets","verbs":["create","delete","deletecollection","get","list","patch","update","watch"]}
{"kind":"ReplicaSet","apiVersion":"apps/v1","name":"replicasets","verbs":["create","delete","deletecollection","get","list","patch","update","watch"]}
...
```

### 2. Cached DiscoveryClient

```bash
go run cached_discovery_client.go
```

**输出示例**：
```json
{"kind":"Pod","apiVersion":"v1","name":"pods","verbs":["create","delete","deletecollection","get","list","patch","update","watch"]}
{"kind":"Deployment","apiVersion":"apps/v1","name":"deployments","verbs":["create","delete","deletecollection","get","list","patch","update","watch"]}
...
```

**缓存位置**：
```
~/.cache/discovery/  # 发现信息缓存
~/.cache/http/       # HTTP 响应缓存
```

## 📖 代码解析

### 1. DiscoveryClient 使用 (`discovery_client.go`)

```go
// 配置加载（第 13-21 行）
configLoader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
    clientcmd.NewDefaultClientConfigLoadingRules(),
    &clientcmd.ConfigOverrides{},
)

rc, err := configLoader.ClientConfig()
if err != nil {
    panic(err)
}

// 创建 DiscoveryClient（第 25 行）
dc, err := discovery.NewDiscoveryClientForConfig(rc)
if err != nil {
    panic(err)
}

// 获取服务器首选资源的列表（第 35 行）
lists, err := dc.ServerPreferredResources()
if err != nil {
    errs = append(errs, err)
}
```

**关键 API**：
- `ServerPreferredResources()` - 获取所有资源的首选版本
- `ServerGroups()` - 获取所有 API 组
- `ServerResourcesForGroupVersion()` - 获取特定 GroupVersion 的资源

### 2. CachedDiscoveryClient 使用 (`cached_discovery_client.go`)

```go
// 创建带缓存的 DiscoveryClient（第 28-33 行）
dc, err := disk.NewCachedDiscoveryClientForConfig(
    rc,                    // REST 配置
    filepath.Join(homedir.HomeDir(), ".cache/discovery"), // 发现信息缓存目录
    filepath.Join(homedir.HomeDir(), ".cache/http"),      // HTTP 缓存目录
    time.Minute*60,        // 缓存有效期（60 分钟）
)
```

**关键参数**：
- `rc` - REST 客户端配置
- `discoveryCacheDir` - 发现信息缓存目录
- `httpCacheDir` - HTTP 响应缓存目录
- `cacheTTL` - 缓存过期时间

### 3. 遍历资源信息

```go
// 定义信息结构（第 41-46 行）
type info struct {
    Kind       string   `json:"kind"`        // 资源类型
    APIVersion string   `json:"apiVersion"`  // API 版本
    Name       string   `json:"name"`        // 资源名称（复数形式）
    Verbs      []string `json:"verbs"`       // 支持的操作
}

// 遍历所有资源（第 49-70 行）
for _, list := range lists {
    if len(list.APIResources) == 0 {
        continue
    }
    
    for _, el := range list.APIResources {
        if len(el.Verbs) == 0 {
            continue
        }
        
        tmp := info{el.Kind, list.GroupVersion, el.Name, el.Verbs}
        res, err := json.Marshal(&tmp)
        if err != nil {
            errs = append(errs, err)
            continue
        }
        fmt.Printf("%s\n", res)
    }
}
```

## 🎯 学习要点

### 1. DiscoveryClient 核心方法

| 方法 | 说明 | 返回值 |
|------|------|--------|
| `ServerPreferredResources()` | 获取所有资源的首选版本 | `[]*metav1.APIResourceList` |
| `ServerGroups()` | 获取所有 API 组 | `*metav1.APIGroupList` |
| `ServerResourcesForGroupVersion(gv string)` | 获取特定 GroupVersion 的资源 | `*metav1.APIResourceList` |
| `ServerVersion()` | 获取服务器版本信息 | `*version.Info` |

### 2. APIResource 结构

```go
type APIResource struct {
    Name         string   // 资源名称（如 pods、deployments）
    SingularName string   // 单数名称（如 pod、deployment）
    Namespaced   bool     // 是否为命名空间级别资源
    Group        string   // 所属 API 组
    Version      string   // API 版本
    Kind         string   // 资源类型（如 Pod、Deployment）
    Verbs        []string // 支持的操作（get、list、watch、create 等）
    ShortNames   []string // 简写（如 po、deploy）
    Categories   []string // 分类（如 all）
}
```

### 3. 缓存策略

**为什么使用缓存**：
- 减少对 API Server 的请求
- 提高程序性能
- 降低网络开销
- 支持离线开发

**缓存更新机制**：
- 首次请求时从 API Server 获取并缓存
- 后续请求优先从缓存读取
- 超过 TTL 后重新获取
- 支持手动刷新缓存

### 4. 常见使用场景

#### 场景 1：检查资源是否支持特定操作

```go
dc := discovery.NewDiscoveryClientForConfig(config)
resources, _ := dc.ServerPreferredResources()

for _, r := range resources {
    if r.GroupVersion == "apps/v1" {
        for _, res := range r.APIResources {
            if res.Name == "deployments" {
                // 检查是否支持 scale 操作
                for _, verb := range res.Verbs {
                    if verb == "scale" {
                        fmt.Println("Deployments support scale operation")
                    }
                }
            }
        }
    }
}
```

#### 场景 2：获取资源简称

```go
resources, _ := dc.ServerPreferredResources()
for _, r := range resources {
    for _, res := range r.APIResources {
        if len(res.ShortNames) > 0 {
            fmt.Printf("%s -> %v\n", res.Name, res.ShortNames)
            // 输出：pods -> [po]
            // 输出：deployments -> [deploy]
        }
    }
}
```

#### 场景 3：动态构建 Dynamic Client

```go
dc := discovery.NewDiscoveryClientForConfig(config)
gvrs, _ := dc.ServerPreferredResources()

// 根据发现的 GVR 创建 Dynamic Client
gvr := schema.GroupVersionResource{
    Group:    "apps",
    Version:  "v1",
    Resource: "deployments",
}
dynamicClient, _ := dynamic.NewForConfig(config)
```

## 🔧 进阶用法

### 1. 过滤特定资源

```go
// 只获取命名空间级别的资源
for _, list := range lists {
    for _, res := range list.APIResources {
        if res.Namespaced {
            fmt.Printf("%s is namespaced\n", res.Name)
        }
    }
}
```

### 2. 按 API 版本过滤

```go
// 只获取 v1 版本的核心资源
for _, list := range lists {
    if list.GroupVersion == "v1" {
        fmt.Printf("v1 resources: %v\n", list.APIResources)
    }
}
```

### 3. 监控资源变化

```go
// 定期检查资源是否变化
for {
    time.Sleep(5 * time.Minute)
    lists, _ := dc.ServerPreferredResources()
    // 对比新旧资源列表
}
```

## ⚠️ 注意事项

1. **网络开销**：每次查询都会与 API Server 通信，建议使用缓存
2. **错误处理**：`ServerPreferredResources()` 可能返回部分错误，需要聚合处理
3. **缓存一致性**：CachedDiscoveryClient 可能有过期数据，必要时手动刷新
4. **权限要求**：需要 `system:discovery` 角色权限

## 📚 相关资源

- [Kubernetes API 概述](https://kubernetes.io/docs/concepts/overview/kubernetes-api/)
- [Discovery Client 文档](https://github.com/kubernetes/client-go/tree/master/discovery)
- [API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)

## 🚀 下一步

继续学习 [Dynamic Client](../using-dynamic-interface/)，了解如何动态操作任何类型的 K8s 资源！
