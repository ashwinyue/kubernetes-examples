# Pod Operator 示例

完整的 Operator 开发示例，展示：
- CRD 定义和代码生成
- Reconcile 循环实现
- Status 更新和 Event 记录
- Finalizer 资源清理
- OwnerReference 级联删除

## 项目结构

```
pod-operator/
├── api/
│   └── v1/
│       ├── groupversion_info.go
│       ├── podmanager_types.go
│       └── zz_generated.deepcopy.go
├── controllers/
│   └── podmanager_controller.go
├── main.go
├── go.mod
├── go.sum
├── config/
│   ├── crd/
│   │   └── bases/
│   │       └── apps.mycompany.com_podmanagers.yaml
│   ├── rbac/
│   │   └── podmanager_controller_role.yaml
│   ├── manager/
│   │   └── manager.yaml
│   └── samples/
│       └── apps_v1_podmanager.yaml
└── Makefile
```

## 快速开始

### 1. 本地运行

```bash
cd pod-operator

# 安装依赖
go mod download

# 安装 CRD
kubectl apply -f config/crd/bases/apps.mycompany.com_podmanagers.yaml

# 运行 Controller
go run main.go
```

### 2. 部署到集群

```bash
# 构建镜像
docker build -t pod-operator:latest .

# 部署
kubectl apply -f config/manager/manager.yaml
kubectl apply -f config/rbac/podmanager_controller_role.yaml
```

### 3. 测试

```bash
# 创建 PodManager
kubectl apply -f config/samples/apps_v1_podmanager.yaml

# 查看状态
kubectl get podmanager
kubectl describe podmanager my-pod-manager

# 查看创建的 Pod
kubectl get pods -l app=my-app

# 删除 PodManager
kubectl delete podmanager my-pod-manager

# 验证 Pod 被清理
kubectl get pods -l app=my-app
```

## 学习要点

### 1. CRD 定义

**文件**: `api/v1/podmanager_types.go`

```go
// PodManagerSpec 定义 PodManager 的期望状态
type PodManagerSpec struct {
    // +kubebuilder:validation:Minimum=1
    // +kubebuilder:default=3
    Replicas int32 `json:"replicas"`

    Image string `json:"image"`
}

// PodManagerStatus 定义 PodManager 的观察状态
type PodManagerStatus struct {
    ReadyReplicas int32 `json:"readyReplicas"`
    CurrentReplicas int32 `json:"currentReplicas"`
    Conditions []PodCondition `json:"conditions,omitempty"`
}
```

**要点**:
- `// +kubebuilder:validation:*` 用于 OpenAPI 验证
- `// +kubebuilder:default=` 设置默认值
- Spec 和 Status 分离

### 2. Reconcile 循环

**文件**: `controllers/podmanager_controller.go`

```go
func (r *PodManagerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := log.FromContext(ctx)

    // 1. 获取 PodManager
    podManager := &appsv1.PodManager{}
    if err := r.Get(ctx, req.NamespacedName, podManager); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // 2. 处理 Finalizer（删除时）
    if !podManager.DeletionTimestamp.IsZero() {
        return r.handleFinalizer(ctx, podManager)
    }

    // 3. 添加 Finalizer
    if !containsString(podManager.Finalizers, finalizerName) {
        podManager.Finalizers = append(podManager.Finalizers, finalizerName)
        if err := r.Update(ctx, podManager); err != nil {
            return ctrl.Result{}, err
        }
        return ctrl.Result{Requeue: true}, nil
    }

    // 4. 列出关联的 Pod
    podList := &corev1.PodList{}
    if err := r.List(ctx, podList,
        client.InNamespace(req.Namespace),
        client.MatchingLabels(ownerLabels(podManager)),
    ); err != nil {
        return ctrl.Result{}, err
    }

    // 5. 调整 Pod 数量
    if int32(len(podList.Items)) < podManager.Spec.Replicas {
        // 创建 Pod
        for i := int32(len(podList.Items)); i < podManager.Spec.Replicas; i++ {
            pod := newPodForPodManager(podManager, i)
            if err := r.Create(ctx, pod); err != nil {
                return ctrl.Result{}, err
            }
            r.Eventf(podManager, corev1.EventTypeNormal, "Created", "Created pod %s", pod.Name)
        }
    } else if int32(len(podList.Items)) > podManager.Spec.Replicas {
        // 删除多余 Pod
        for i := podManager.Spec.Replicas; i < int32(len(podList.Items)); i++ {
            if err := r.Delete(ctx, &podList.Items[i]); err != nil {
                return ctrl.Result{}, err
            }
            r.Eventf(podManager, corev1.EventTypeNormal, "Deleted", "Deleted pod %s", podList.Items[i].Name)
        }
    }

    // 6. 更新 Status
    readyCount := int32(0)
    for _, pod := range podList.Items {
        if pod.Status.Phase == corev1.PodRunning {
            readyCount++
        }
    }

    podManager.Status.ReadyReplicas = readyCount
    podManager.Status.CurrentReplicas = int32(len(podList.Items))

    if err := r.Status().Update(ctx, podManager); err != nil {
        return ctrl.Result{}, err
    }

    // 7. 重新入队
    return ctrl.Result{RequeueAfter: time.Second * 30}, nil
}
```

**要点**:
- 获取资源对象
- 处理删除（Finalizer）
- 添加 Finalizer
- 列出关联资源
- 调整期望状态
- 更新 Status
- 返回 RequeueAfter

### 3. Finalizer

```go
func (r *PodManagerReconciler) handleFinalizer(ctx context.Context, podManager *appsv1.PodManager) (ctrl.Result, error) {
    log := log.FromContext(ctx)

    if containsString(podManager.Finalizers, finalizerName) {
        // 清理关联资源
        podList := &corev1.PodList{}
        if err := r.List(ctx, podList,
            client.InNamespace(podManager.Namespace),
            client.MatchingLabels(ownerLabels(podManager)),
        ); err != nil {
            return ctrl.Result{}, err
        }

        for _, pod := range podList.Items {
            if err := r.Delete(ctx, &pod); err != nil {
                return ctrl.Result{}, err
            }
        }

        // 移除 Finalizer
        podManager.Finalizers = removeString(podManager.Finalizers, finalizerName)
        if err := r.Update(ctx, podManager); err != nil {
            return ctrl.Result{}, err
        }

        log.Info("Cleaned up resources")
    }

    return ctrl.Result{}, nil
}
```

**要点**:
- 检查 DeletionTimestamp
- 清理关联资源
- 移除 Finalizer
- 允许删除完成

### 4. OwnerReference

```go
func newPodForPodManager(podManager *appsv1.PodManager, index int32) *corev1.Pod {
    return &corev1.Pod{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("%s-%d", podManager.Name, index),
            Namespace: podManager.Namespace,
            Labels:    ownerLabels(podManager),
            OwnerReferences: []metav1.OwnerReference{
                *metav1.NewControllerRef(podManager, appsv1.GroupVersion.WithKind("PodManager")),
            },
        },
        Spec: corev1.PodSpec{
            Containers: []corev1.Container{
                {
                    Name:  "app",
                    Image: podManager.Spec.Image,
                },
            },
        },
    }
}

func ownerLabels(podManager *appsv1.PodManager) map[string]string {
    return map[string]string{
        "app":            "my-app",
        "podmanager":    podManager.Name,
        "podmanager-uid": string(podManager.UID),
    }
}
```

**要点**:
- 使用 ControllerRef 设置 OwnerReference
- 设置标签便于查询
- 级联删除自动生效

### 5. Event 记录

```go
r.Eventf(podManager, corev1.EventTypeNormal, "Created", "Created pod %s", pod.Name)
r.Eventf(podManager, corev1.EventTypeWarning, "Failed", "Failed to create pod %s: %v", pod.Name, err)
```

**要点**:
- EventTypeNormal: 正常事件
- EventTypeWarning: 警告事件
- Eventf 记录重要操作

### 6. Controller 初始化

```go
func (r *PodManagerReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&appsv1.PodManager{}).
        Owns(&corev1.Pod{}).
        Complete(r)
}
```

**要点**:
- For(): 监听主资源
- Owns(): 监听拥有的资源（自动触发 Reconcile）
- Complete(): 完成 Controller 设置

## 调试技巧

### 1. 查看 Controller 日志

```bash
# 本地运行
kubectl logs deployment/pod-operator-controller-manager -n pod-operator-system

# 使用 -v 增加日志详细程度
go run main.go -v=4
```

### 2. 查看事件

```bash
kubectl describe podmanager my-pod-manager

# 查看所有事件
kubectl get events --sort-by='.lastTimestamp'
```

### 3. 查看 Status

```bash
kubectl get podmanager my-pod-manager -o yaml

# 只查看 Status
kubectl get podmanager my-pod-manager -o jsonpath='{.status}'
```

### 4. 查看关联 Pod

```bash
kubectl get pods -l podmanager=my-pod-manager
```

## 常见问题

**Q: Pod 没有被创建？**
A:
1. 检查 Controller 日志
2. 查看事件：`kubectl describe podmanager my-pod-manager`
3. 验证 RBAC 权限

**Q: Status 没有更新？**
A:
1. 确认使用 `r.Status().Update()` 而不是 `r.Update()`
2. 检查是否有更新冲突

**Q: Pod 删除时卡住？**
A:
1. 检查 Finalizer 是否正确实现
2. 查看 Controller 是否还在运行

## 进阶扩展

### 1. 添加健康检查

```go
// 在 Reconcile 中添加
if podManager.Status.ReadyReplicas != podManager.Spec.Replicas {
    podManager.Status.Conditions = append(podManager.Status.Conditions, appsv1.PodCondition{
        Type:               "Ready",
        Status:             "False",
        LastTransitionTime: metav1.Now(),
    })
}
```

### 2. 添加指标

```go
import (
    "github.com/prometheus/client_golang/prometheus"
)

var (
    reconcileTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "podmanager_reconcile_total",
            Help: "Total number of reconcile operations",
        },
        []string{"result"},
    )
)

func init() {
    prometheus.MustRegister(reconcileTotal)
}
```

### 3. 添加多版本支持

```go
// 实现 Hub 接口
func (*PodManager) Hub() {}
```

## 参考资料

- [Kubebuilder Book](https://book.kubebuilder.io/)
- [Controller Runtime 文档](https://github.com/kubernetes-sigs/controller-runtime)
- [Kubernetes API 文档](https://kubernetes.io/docs/reference/kubernetes-api/)
