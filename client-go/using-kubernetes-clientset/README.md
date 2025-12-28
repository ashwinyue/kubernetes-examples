# ClientSet 基础操作

本目录包含 ClientSet 的 CRUD 操作示例，演示如何使用 client-go 管理 Kubernetes 资源。

## 📋 示例列表

1. **creating_deployment.go** - 创建 Deployment
2. **updating_deployment_image.go** - 更新 Deployment 镜像
3. **deleting_deployment.go** - 删除 Deployment
4. **listing_pods.go** - 列出 Pod

## 🚀 运行示例

### 1. 创建 Deployment

```bash
cd client-go/using-kubernetes-clientset
go run creating_deployment.go --namespace=default
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
go run updating_deployment_image.go --namespace=default
```

**输出示例**：
```
before patching: deployment.apps/nginx image is nginx:1.21.6
after  patching: deployment.apps/nginx image is nginx:1.20.2
```

**验证**：
```bash
kubectl get deployments nginx -o yaml | grep image
```

### 3. 删除 Deployment

```bash
go run deleting_deployment.go --namespace=default
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
go run listing_pods.go --namespace=default
```

**输出示例**：
```
NAME       STATUS    AGE
coredns-xxx Running    2m
coredns-yyy Running    2m
```

**验证**：
```bash
kubectl get pods
```

## 📚 代码解析

### 1. 配置加载（所有示例共有）

```go
// 获取环境变量 KUBECONFIG 或使用默认路径
defaultKubeconfig := os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
if len(defaultKubeconfig) == 0 {
    defaultKubeconfig = clientcmd.RecommendedHomeFile
}

// 解析命令行参数
kubeconfig := flag.String(clientcmd.RecommendedConfigPathFlag,
    defaultKubeconfig, "absolute path to the kubeconfig file")

// 构建配置
config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
if err != nil {
    panic(err)
}

// 创建 Clientset
cs, err := kubernetes.NewForConfig(config)
if err != nil {
    panic(err)
}
```

**学习要点**：
- `RecommendedConfigPathEnvVar` - KUBECONFIG 环境变量
- `RecommendedHomeFile` - 默认路径 `~/.kube/config`
- `BuildConfigFromFlags()` - 加载配置的核心方法
- `kubernetes.NewForConfig()` - 创建类型安全的客户端

### 2. 创建 Deployment (creating_deployment.go)

```go
// 辅助函数：创建 int32 指针
i32Ptr := func(i int32) *int32 { return &i }

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

// 创建 Deployment
res, err := cs.AppsV1().Deployments(*namespace).
    Create(context.TODO(), deployment, metav1.CreateOptions{})
```

**学习要点**：
- Deployment 结构体的完整定义
- Replicas 使用指针类型
- Selector 用于匹配 Pod
- Template 定义 Pod 模板
- `AppsV1()` 获取 Apps API Group

### 3. 更新 Deployment 镜像 (updating_deployment_image.go)

```go
// 获取当前 Deployment
res, err := cs.AppsV1().Deployments(*namespace).
    Get(context.TODO(), "nginx", metav1.GetOptions{})

// 打印当前镜像
fmt.Printf("before patching: deployment.apps/%s image is %s\n",
    res.Name, res.Spec.Template.Spec.Containers[0].Image)

// 创建 JSON Patch
patch := []byte(`{"spec":{"template":{"spec":{"containers":[{"name":"nginx","image":"nginx:1.20.2"}]}}}`)

// 应用 Patch
res, err = cs.AppsV1().Deployments(*namespace).
    Patch(context.TODO(), "nginx", types.StrategicMergePatchType, patch, metav1.PatchOptions{})

// 打印更新后的镜像
fmt.Printf("after  patching: deployment.apps/%s image is %s\n",
    res.Name, res.Spec.Template.Spec.Containers[0].Image)
```

**学习要点**：
- `Get()` - 获取单个资源
- Strategic Merge Patch - Kubernetes 推荐的 Patch 方式
- `types.StrategicMergePatchType` - Patch 类型
- JSON 格式的 Patch 数据

### 4. 删除 Deployment (deleting_deployment.go)

```go
// 删除 Deployment
err = cs.AppsV1().Deployments(*namespace).
    Delete(context.TODO(), "nginx", metav1.DeleteOptions{})

// 处理 NotFound 错误
if err != nil {
    if errors.IsNotFound(err) {
        return  // 资源不存在，忽略错误
    }
    panic(err.Error())
}

fmt.Println("deployment.apps \"nginx\" deleted")
```

**学习要点**：
- `Delete()` - 删除资源
- `errors.IsNotFound()` - 判断资源是否存在
- 优雅的错误处理

### 5. 列出 Pod (listing_pods.go)

```go
// 列出 Pod
res, err := cs.CoreV1().Pods(*namespace).List(context.TODO(), metav1.ListOptions{})
if err != nil {
    panic(err.Error())
}

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
- `List()` - 列出资源
- `tabwriter` - 格式化表格输出
- `res.Items` - 访问列表项
- `time.Since()` - 计算资源年龄

## 🎯 学习要点总结

### ClientSet 核心概念

1. **类型安全**
   - 所有 API 都是强类型的
   - 编译时检查错误
   - IDE 自动补全

2. **API Group 组织**
   - `AppsV1()` - Apps API (Deployment, StatefulSet, DaemonSet)
   - `CoreV1()` - Core API (Pod, Service, ConfigMap)
   - `BatchV1()` - Batch API (Job, CronJob)
   - 其他 API Group

3. **资源操作**
   - `Create()` - 创建资源
   - `Get()` - 获取单个资源
   - `List()` - 列出资源
   - `Update()` - 完整更新
   - `Patch()` - 部分更新
   - `Delete()` - 删除资源

4. **错误处理**
   - 使用 `errors.IsNotFound()` 判断资源不存在
   - 使用 `errors.IsAlreadyExists()` 判断已存在
   - 区分 API 错误和网络错误

### 实践建议

1. **先读后写**
   - 先用 `Get()` 获取资源
   - 修改后用 `Update()` 提交

2. **使用 Patch 更新**
   - Partial update 性能更好
   - 减少冲突
   - 使用 Strategic Merge Patch

3. **List 使用选项**
   - `LabelSelector` - 按标签筛选
   - `FieldSelector` - 按字段筛选
   - `Limit` - 限制返回数量

## 🔧 常见问题

### Q: 为什么要用指针类型？

A: Kubernetes API 使用指针区分"未设置"和"零值"。例如：
- `Replicas *int32` - 可选，nil 表示未设置
- `Replicas int32` - 必须有值，0 也是有效值

### Q: Update vs Patch 有什么区别？

A:
- `Update()` - 完整替换资源，需要提供完整对象
- `Patch()` - 部分更新，只提供需要修改的字段

**推荐**：优先使用 `Patch()`，性能更好且冲突更少。

### Q: 如何处理并发更新冲突？

A:
```go
// 使用乐观锁
res, err := client.Get(name, metav1.GetOptions{})
if err != nil {
    return err
}

// 修改资源
res.Spec.Replicas = newReplicas

// 使用 ResourceVersion 确保一致性
res.ResourceVersion = oldResourceVersion

_, err = client.Update(res)
if errors.IsConflict(err) {
    // 冲突，重试
    return retry()
}
```

## 📖 下一步

完成本阶段后，继续学习：

- [阶段 1.4: RESTClient 使用](../using-rest-client/)
- [阶段 1.5: Discovery Client](../using-discovery-client/)
- [阶段 1.6: Dynamic Client](../using-dynamic-interface/)

回到 [主 README](../../LEARNING_PATH.md)
