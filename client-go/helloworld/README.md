# Client-go Hello World 示例

本示例演示 client-go 的基础用法，包括配置加载、ClientSet 初始化和资源删除操作。

## 前置条件

确保 Kind 集群正在运行：

```bash
kind get clusters
```

## 运行步骤

### 1. 创建测试 Deployment

本示例会删除名为 `demo-deployment` 的 Deployment，因此需要先创建它：

```bash
# 拉取镜像（如果本地没有）
docker pull nginx:alpine

# 加载镜像到 Kind 节点
kind load docker-image nginx:alpine --name onex

# 创建 Deployment
kubectl create deployment demo-deployment --image=nginx:alpine --replicas=3

# 验证 Deployment 已就绪
kubectl get pods -l app=demo-deployment
```

### 2. 运行示例程序

```bash
go run main.go
```

### 3. 验证 Deployment 已删除

```bash
kubectl get deployments
kubectl get pods -l app=demo-deployment
```

## 代码解析

### 1. 配置加载 (第 26 行)

```go
config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
```

- `BuildConfigFromFlags` 用于加载 Kubernetes 配置
- 第一个参数为空字符串，表示使用默认的 kubeconfig
- 第二个参数指定 kubeconfig 文件路径
- 支持 in-cluster 和 kubeconfig 两种配置方式

### 2. ClientSet 初始化 (第 30 行)

```go
clientset, err := kubernetes.NewForConfig(config)
```

- `NewForConfig` 创建 Kubernetes 客户端
- ClientSet 是类型安全的，提供所有 Kubernetes 资源的 API
- 包含所有 API Group 的客户端

### 3. 资源删除 (第 36 行)

```go
clientset.AppsV1().Deployments(apiv1.NamespaceDefault).Delete(
    context.Background(),
    "demo-deployment",
    metav1.DeleteOptions{},
)
```

- `AppsV1()` 获取 Apps API Group 的客户端
- `Deployments()` 获取 Deployment 资源的接口
- `Delete()` 删除指定名称的 Deployment

## 学习要点

1. **client-go 配置加载**
   - in-cluster 配置：使用 ServiceAccount
   - kubeconfig 配置：使用 ~/.kube/config

2. **ClientSet 的使用**
   - 类型安全的 API 调用
   - 按资源类型组织 API
   - 支持所有 Kubernetes 资源

3. **资源操作**
   - CRUD 操作：Create、Get、List、Update、Delete
   - 错误处理机制
   - Context 用于超时和取消

## 清理

```bash
# 删除 Deployment（如果还存在）
kubectl delete deployment demo-deployment

# 或者重新运行示例程序
go run main.go
```
