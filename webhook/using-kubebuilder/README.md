# Kubebuilder Webhook 示例

使用 Kubebuilder 框架开发的完整 Webhook 示例，包含 Controller、Mutating Webhook 和 Validating Webhook。

## 功能说明

本示例展示如何使用 Kubebuilder 框架开发和部署 Admission Webhook：

1. **Calculate CRD**：自定义资源，用于数学运算（加、减、乘、除）
2. **Controller**：自动计算结果并更新 Status
3. **Mutating Webhook**：为 Create 和 Update 操作设置默认值
4. **Validating Webhook**：验证除法操作时除数不为零

## 项目结构

```
webhook/using-kubebuilder/
├── api/
│   └── v1/
│       ├── calculate_types.go           # CRD 定义
│       ├── calculate_webhook.go         # Webhook 实现
│       ├── calculate_webhook_test.go    # Webhook 测试
│       ├── groupversion_info.go        # API 组和版本信息
│       └── zz_generated.deepcopy.go   # 自动生成的 DeepCopy 方法
├── cmd/
│   └── main.go                     # 入口文件
├── config/
│   ├── certmanager/                 # Cert-manager 配置
│   ├── crd/                       # CRD 定义
│   ├── default/                    # 默认配置（包含 Deployment）
│   ├── manager/                   # Manager 配置
│   ├── prometheus/                 # Prometheus 监控配置
│   ├── rbac/                      # RBAC 权限配置
│   ├── samples/                   # 测试样本
│   └── webhook/                   # WebhookConfiguration 配置
├── internal/
│   └── controller/
│       ├── calculate_controller.go     # Controller 实现
│       ├── calculate_controller_test.go
│       └── suite_test.go
├── test/
│   └── e2e/                      # E2E 测试
├── hack/
│   └── boilerplate.go.txt          # 代码模板
├── Dockerfile                      # Docker 镜像构建
├── Makefile                       # 构建和部署脚本
├── PROJECT                        # Kubebuilder 项目配置
├── go.mod                         # Go 模块定义
├── go.sum                         # Go 依赖锁定
└── README.md                      # 本文件
```

## 快速开始

### 前置条件

- Go 1.22.0+
- Docker 17.03+
- kubectl 1.11.3+
- Kubernetes 1.11.3+ 集群
- kubebuilder 工具（可选）

### 1. 本地运行（开发模式）

```bash
cd webhook/using-kubebuilder

# 1. 生成 manifests（CRD、WebhookConfiguration 等）
make manifests

# 2. 生成代码（DeepCopy 等）
make generate

# 3. 运行 controller
make run
```

**输出**：
```
2024-12-28T23:00:00.000Z	INFO	setup	starting manager
```

### 2. 本地测试（创建 CRD）

```bash
# Terminal 1：运行 controller
make run

# Terminal 2：应用 CRD
make install

# Terminal 2：创建测试资源
cat << EOF | kubectl apply -f -
apiVersion: math.superproj.com/v1
kind: Calculate
metadata:
  name: calculate-add
spec:
  action: add
  first: 10
  second: 5
EOF

# 查看结果
kubectl get calculate calculate-add
kubectl get calculate calculate-add -o yaml
```

**预期结果**：
```yaml
status:
  result: 15  # 10 + 5 = 15
```

### 3. 部署到 Kubernetes 集群

#### 3.1 构建和推送镜像

```bash
# 1. 构建镜像
make docker-build IMG=<your-registry>/webhook:latest

# 2. 推送镜像到仓库
docker push <your-registry>/webhook:latest

# 或使用本地 Kind 集群
kind load docker-image <your-registry>/webhook:latest --name onex
```

#### 3.2 安装 CRD

```bash
# 安装 CRD 到集群
make install

# 验证 CRD
kubectl get crd calculates.math.superproj.com
kubectl describe crd calculates.math.superproj.com
```

#### 3.3 部署 Controller

```bash
# 部署 controller 到集群
make deploy IMG=<your-registry>/webhook:latest

# 查看 Pod
kubectl get pods -n webhook-system

# 查看日志
kubectl logs -n webhook-system deployment/webhook-controller-manager
```

**部署内容**：
- Namespace：`webhook-system`
- Deployment：`webhook-controller-manager`
- Service：`webhook-controller-manager-service`
- MutatingWebhookConfiguration：`mcalculate.kb.io`
- ValidatingWebhookConfiguration：`vcalculate.kb.io`
- RBAC：ServiceAccount、Role、RoleBinding、ClusterRole、ClusterRoleBinding

### 4. 测试 Webhook

#### 4.1 测试 Controller（运算）

```bash
# 测试加法
kubectl apply -f - << EOF
apiVersion: math.superproj.com/v1
kind: Calculate
metadata:
  name: test-add
spec:
  action: add
  first: 10
  second: 5
EOF

kubectl get calculate test-add

# 测试减法
kubectl apply -f - << EOF
apiVersion: math.superproj.com/v1
kind: Calculate
metadata:
  name: test-sub
spec:
  action: sub
  first: 10
  second: 5
EOF

# 测试乘法
kubectl apply -f - << EOF
apiVersion: math.superproj.com/v1
kind: Calculate
metadata:
  name: test-mul
spec:
  action: mul
  first: 10
  second: 5
EOF

# 测试除法
kubectl apply -f - << EOF
apiVersion: math.superproj.com/v1
kind: Calculate
metadata:
  name: test-div
spec:
  action: div
  first: 10
  second: 2
EOF
```

**预期结果**：
```
NAME       ACTION   FIRST   SECOND   RESULT
test-add   add      10       5         15
test-sub   sub      10       5         5
test-mul   mul      10       5         50
test-div   div      10       2         5
```

#### 4.2 测试 Validating Webhook（除零验证）

```bash
# 测试除法除数为零（应该失败）
kubectl apply -f - << EOF
apiVersion: math.superproj.com/v1
kind: Calculate
metadata:
  name: test-div-zero
spec:
  action: div
  first: 10
  second: 0
EOF
```

**预期错误**：
```
Error from server (BadRequest): admission webhook "vcalculate.kb.io" denied the request: Calculate.math.superproj.com "test-div-zero" is invalid: spec.second: Invalid value: 0: the divisor cannot be zero whtn action is division
```

**查看 Webhook 日志**：
```bash
kubectl logs -n webhook-system deployment/webhook-controller-manager
```

**预期日志**：
```
1.683324050000732e+09  INFO  validate create  {"name": "test-div-zero"}
```

#### 4.3 测试 Mutating Webhook（默认值）

```bash
# 测试 Mutating Webhook（当前代码中 Default 方法为空，可添加逻辑）
kubectl apply -f - << EOF
apiVersion: math.superproj.com/v1
kind: Calculate
metadata:
  name: test-default
spec:
  action: add
  first: 5
  second: 3
EOF
```

**查看日志**：
```bash
kubectl logs -n webhook-system deployment/webhook-controller-manager | grep default
```

**预期日志**：
```
1.683324050000732e+09  INFO  default  {"name": "test-default"}
```

### 5. 清理

```bash
# 删除测试资源
kubectl delete calculate --all

# 卸载 Controller
make undeploy

# 删除 CRD
make uninstall

# 删除 Namespace
kubectl delete namespace webhook-system
```

## 代码分析

### 1. CRD 定义

**文件**：`api/v1/calculate_types.go`

```go
// ActionType 定义运算类型
type ActionType string

const (
    ActionTypeAdd ActionType = ActionType("add")
    ActionTypeSub ActionType = ActionType("sub")
    ActionTypeMul ActionType = ActionType("mul")
    ActionTypeDiv ActionType = ActionType("div")
)

// CalculateSpec 定义期望状态
type CalculateSpec struct {
    Action ActionType `json:"action,omitempty"`
    First  int       `json:"first,omitempty"`
    Second int       `json:"second,omitempty"`
}

// CalculateStatus 定义观察状态
type CalculateStatus struct {
    Result int `json:"result,omitempty"`
}

// Calculate 资源定义
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Action",type="string",JSONPath=".spec.action",description="The math action type"
// +kubebuilder:printcolumn:name="First",type="integer",JSONPath=".spec.first",description="Input first number"
// +kubebuilder:printcolumn:name="Second",type="integer",JSONPath=".spec.second",description="Input second number"
// +kubebuilder:printcolumn:name="Result",type="integer",JSONPath=".status.result",description="Calculate result"
type Calculate struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   CalculateSpec   `json:"spec,omitempty"`
    Status CalculateStatus `json:"status,omitempty"`
}
```

**要点**：
- `+kubebuilder:object:root=true`：标记为根对象
- `+kubebuilder:subresource:status`：启用 status 子资源
- `+kubebuilder:printcolumn`：定义 `kubectl get` 输出列

### 2. Webhook 定义

**文件**：`api/v1/calculate_webhook.go`

#### Mutating Webhook

```go
// +kubebuilder:webhook:path=/mutate-math-superproj-com-v1-calculate,mutating=true,failurePolicy=fail,sideEffects=None,groups=math.superproj.com,resources=calculates,verbs=create;update,versions=v1,name=mcalculate.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Calculate{}

// Default 实现 webhook.Defaulter 接口
func (r *Calculate) Default() {
    calculatelog.Info("default", "name", r.Name)

    // TODO(user): fill in your defaulting logic.
    // 示例：设置默认的 action
    // if r.Spec.Action == "" {
    //     r.Spec.Action = ActionTypeAdd
    // }
}
```

**要点**：
- 实现 `webhook.Defaulter` 接口
- 在 Create/Update 时自动调用
- 可以设置默认值

#### Validating Webhook

```go
// +kubebuilder:webhook:path=/validate-math-superproj-com-v1-calculate,mutating=false,failurePolicy=fail,sideEffects=None,groups=math.superproj.com,resources=calculates,verbs=create;update,versions=v1,name=vcalculate.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Calculate{}

// ValidateCreate 实现验证逻辑
func (r *Calculate) ValidateCreate() (admission.Warnings, error) {
    calculatelog.Info("validate create", "name", r.Name)
    return r.validate()
}

// ValidateUpdate 实现更新验证
func (r *Calculate) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
    calculatelog.Info("validate update", "name", r.Name)
    return r.validate()
}

// ValidateDelete 实现删除验证
func (r *Calculate) ValidateDelete() (admission.Warnings, error) {
    calculatelog.Info("validate delete", "name", r.Name)
    // TODO(user): fill in your validation logic upon object deletion.
    return nil, nil
}

// validate 包含实际验证逻辑
func (r *Calculate) validate() (admission.Warnings, error) {
    allErrs := field.ErrorList{}

    specPath := field.NewPath("spec")
    // 验证除法除数不为零
    if r.Spec.Action == ActionTypeDiv && r.Spec.Second == 0 {
        allErrs = append(allErrs, field.Invalid(
            specPath.Child("second"),
            r.Spec.Second,
            "the divisor cannot be zero whtn action is division",
        ))
    }

    return nil, allErrs.ToAggregate()
}
```

**要点**：
- 实现 `webhook.Validator` 接口
- 在 Create/Update/Delete 时调用
- 使用 `field.ErrorList` 收集多个错误
- 返回 `admission.Warnings` 和 `error`

### 3. Controller 实现

**文件**：`internal/controller/calculate_controller.go`

```go
// CalculateReconciler 调谐器
type CalculateReconciler struct {
    client.Client
    Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=math.superproj.com,resources=calculates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=math.superproj.com,resources=calculates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=math.superproj.com,resources=calculates/finalizers,verbs=update

// Reconcile 调谐循环
func (r *CalculateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    var cal mathv1.Calculate
    if err := r.Get(ctx, req.NamespacedName, &cal); err != nil {
        klog.Error(err, "unable to fetch calculate")
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }
    klog.Infof("Found the calculate object %v", cal)

    // 根据不同的 action 类型进行运算
    switch cal.Spec.Action {
    case mathv1.ActionTypeAdd:
        cal.Status.Result = cal.Spec.First + cal.Spec.Second
    case mathv1.ActionTypeSub:
        cal.Status.Result = cal.Spec.First - cal.Spec.Second
    case mathv1.ActionTypeMul:
        cal.Status.Result = cal.Spec.First * cal.Spec.Second
    case mathv1.ActionTypeDiv:
        if cal.Spec.Second == 0 {
            return ctrl.Result{}, fmt.Errorf("the divisor cannot be zero whtn action is division")
        }
        cal.Status.Result = cal.Spec.First / cal.Spec.Second
    default:
        return ctrl.Result{}, fmt.Errorf("unknown action type")
    }

    // 更新 Status
    klog.Info("Updating the result of calculation")
    if err := r.Status().Update(ctx, &cal); err != nil {
        klog.Error(err, "Unable to update calculate status")
        return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
}

// SetupWithManager 设置 controller
func (r *CalculateReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&mathv1.Calculate{}).
        Complete(r)
}
```

**要点**：
- `Reconcile`：调谐循环，处理资源变更
- `Status().Update`：只更新 Status 子资源
- `client.IgnoreNotFound`：忽略资源已删除错误

### 4. 主程序

**文件**：`cmd/main.go`

```go
func main() {
    // 1. 命令行参数
    var metricsAddr string
    var enableLeaderElection bool
    var probeAddr string
    var secureMetrics bool
    var enableHTTP2 bool
    flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metric endpoint binds to.")
    flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
    flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager.")
    flag.BoolVar(&secureMetrics, "metrics-secure", true, "If set, metrics endpoint is served securely via HTTPS.")
    flag.BoolVar(&enableHTTP2, "enable-http2", false, "If set, HTTP/2 will be enabled for metrics and webhook servers")
    flag.Parse()

    // 2. 设置日志
    opts := zap.Options{Development: true}
    opts.BindFlags(flag.CommandLine)
    ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

    // 3. 配置 TLS 选项
    tlsOpts := []func(*tls.Config){}
    if !enableHTTP2 {
        tlsOpts = append(tlsOpts, func(c *tls.Config) {
            c.NextProtos = []string{"http/1.1"}
        })
    }

    // 4. 创建 Webhook 服务器
    webhookServer := webhook.NewServer(webhook.Options{
        TLSOpts: tlsOpts,
    })

    // 5. 创建 Manager
    mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
        Scheme: scheme,
        Metrics: metricsserver.Options{
            BindAddress:   metricsAddr,
            SecureServing: secureMetrics,
            TLSOpts:       tlsOpts,
        },
        WebhookServer:          webhookServer,
        HealthProbeBindAddress: probeAddr,
        LeaderElection:         enableLeaderElection,
        LeaderElectionID:       "53f2a732.superproj.com",
    })

    // 6. 注册 Controller
    if err = (&controller.CalculateReconciler{
        Client: mgr.GetClient(),
        Scheme: mgr.GetScheme(),
    }).SetupWithManager(mgr); err != nil {
        setupLog.Error(err, "unable to create controller", "controller", "Calculate")
        os.Exit(1)
    }

    // 7. 注册 Webhook
    if os.Getenv("ENABLE_WEBHOOKS") != "false" {
        if err = (&mathv1.Calculate{}).SetupWebhookWithManager(mgr); err != nil {
            setupLog.Error(err, "unable to create webhook", "webhook", "Calculate")
            os.Exit(1)
        }
    }

    // 8. 健康检查
    if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
        setupLog.Error(err, "unable to set up health check")
        os.Exit(1)
    }
    if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
        setupLog.Error(err, "unable to set up ready check")
        os.Exit(1)
    }

    // 9. 启动 Manager
    setupLog.Info("starting manager")
    if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
        setupLog.Error(err, "problem running manager")
        os.Exit(1)
    }
}
```

**要点**：
- Webhook 服务器自动配置 TLS
- Manager 管理 Controller 和 Webhook
- 健康检查：`/healthz` 和 `/readyz`
- 优雅关闭

## 学习要点

### 1. Kubebuilder 框架

**核心概念**：

- **Manager**：管理 Controller 和 Webhook 的生命周期
- **Controller**：监听资源变更并调谐状态
- **Webhook**：处理 AdmissionReview 请求
- **Scheme**：管理类型注册和转换

**工作流程**：

```
┌─────────────┐
│ API Server │
└──────┬──────┘
       │
       ├─► AdmissionRequest
       │   (Create/Update/Delete)
       │
       ▼
┌──────────────────┐
│ Mutating Webhook │ ◄─── Default()
└──────────────────┘
       │
       │ (Allowed + Patch)
       ▼
┌──────────────────┐
│ Validating Webhook│ ◄─── ValidateCreate/Update/Delete()
└──────────────────┘
       │
       │ (Allowed + Result)
       ▼
┌─────────────┐
│ Controller  │ ◄─── Reconcile()
└─────────────┘
       │
       │ (Update Status)
       ▼
┌─────────────┐
│ API Server │
└─────────────┘
```

### 2. Webhook 注解

**Mutating Webhook 注解**：

```go
// +kubebuilder:webhook:path=/mutate-math-superproj-com-v1-calculate,mutating=true,failurePolicy=fail,sideEffects=None,groups=math.superproj.com,resources=calculates,verbs=create;update,versions=v1,name=mcalculate.kb.io,admissionReviewVersions=v1
```

**参数说明**：
- `path`：Webhook 路径（自动生成）
- `mutating=true`：Mutating Webhook
- `failurePolicy=fail`：失败时拒绝请求
- `sideEffects=None`：无副作用
- `groups=math.superproj.com`：API 组
- `resources=calculates`：资源名称
- `verbs=create;update`：操作类型
- `name`：Webhook 名称
- `admissionReviewVersions=v1`：支持的 AdmissionReview 版本

**Validating Webhook 注解**：

```go
// +kubebuilder:webhook:path=/validate-math-superproj-com-v1-calculate,mutating=false,failurePolicy=fail,sideEffects=None,groups=math.superproj.com,resources=calculates,verbs=create;update,versions=v1,name=vcalculate.kb.io,admissionReviewVersions=v1
```

**参数说明**：
- `mutating=false`：Validating Webhook
- 其他参数同上

### 3. RBAC 注解

```go
// +kubebuilder:rbac:groups=math.superproj.com,resources=calculates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=math.superproj.com,resources=calculates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=math.superproj.com,resources=calculates/finalizers,verbs=update
```

**生成资源**：
- ClusterRole：包含所有权限
- ClusterRoleBinding：绑定到 ServiceAccount

### 4. CRD 注解

```go
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Action",type="string",JSONPath=".spec.action",description="The math action type"
// +kubebuilder:printcolumn:name="First",type="integer",JSONPath=".spec.first",description="Input first number"
// +kubebuilder:printcolumn:name="Second",type="integer",JSONPath=".spec.second",description="Input second number"
// +kubebuilder:printcolumn:name="Result",type="integer",JSONPath=".status.result",description="Calculate result"
```

**生成内容**：
- CRD Schema
- Status 子资源
- kubectl get 输出列

### 5. Makefile 命令

**开发命令**：

```bash
make manifests     # 生成 CRD、WebhookConfiguration、RBAC
make generate      # 生成 DeepCopy 等代码
make fmt          # 格式化代码
make vet          # 运行 go vet
make test         # 运行测试
make run          # 本地运行
```

**构建命令**：

```bash
make build        # 构建 manager 二进制
make docker-build # 构建 Docker 镜像
make docker-push  # 推送 Docker 镜像
```

**部署命令**：

```bash
make install       # 安装 CRD
make uninstall    # 删除 CRD
make deploy       # 部署 Controller
make undeploy    # 卸载 Controller
```

## 调试技巧

### 1. 本地运行

```bash
# 1. 运行 controller
make run

# 2. 在另一个终端应用 CRD
make install

# 3. 创建资源
kubectl apply -f config/samples/

# 4. 查看日志
# Controller 会直接在终端输出日志
```

### 2. 查看集群日志

```bash
# 查看 Pod 日志
kubectl logs -n webhook-system deployment/webhook-controller-manager -f

# 查看 Webhook 调用
kubectl logs -n webhook-system deployment/webhook-controller-manager | grep "validate"
kubectl logs -n webhook-system deployment/webhook-controller-manager | grep "default"

# 查看错误
kubectl logs -n webhook-system deployment/webhook-controller-manager | grep "Error"
```

### 3. 查看 WebhookConfiguration

```bash
# 查看 Mutating WebhookConfiguration
kubectl get mutatingwebhookconfiguration mcalculate.kb.io -o yaml

# 查看 Validating WebhookConfiguration
kubectl get validatingwebhookconfiguration vcalculate.kb.io -o yaml
```

### 4. 查看事件

```bash
# 查看资源事件
kubectl describe calculate <name>

# 查看所有事件
kubectl get events --sort-by='.lastTimestamp'
```

### 5. 本地调试 Webhook

```bash
# 使用 kubectl proxy
kubectl proxy --port=8080

# 测试 Webhook
curl -k http://localhost:8080/validate-math-superproj-com-v1-calculate \
  -X POST \
  -H "Content-Type: application/json" \
  -d @test-request.json
```

## 常见问题

**Q: CRD 安装失败？**
A:
1. 检查 `make manifests` 是否成功
2. 检查 `make install` 的输出
3. 查看 API Server 日志

**Q: Webhook 没有被调用？**
A:
1. 检查 WebhookConfiguration 是否正确创建
2. 检查 Service 是否可达
3. 查看 Webhook Pod 日志
4. 查看 API Server 日志

**Q: Validating Webhook 错误消息不清晰？**
A:
1. 修改 `validate()` 方法中的错误消息
2. 使用 `field.Invalid()` 指定具体字段
3. 返回 `admission.Warnings` 提供警告

**Q: Controller 不更新 Status？**
A:
1. 检查是否使用 `r.Status().Update()` 而非 `r.Update()`
2. 检查 `+kubebuilder:subresource:status` 注解
3. 查看 Controller 日志

**Q: 如何添加新的验证规则？**
A:
1. 在 `validate()` 方法中添加新的验证逻辑
2. 使用 `field.ErrorList` 收集错误
3. 使用 `allErrs.ToAggregate()` 返回错误

**Q: 如何设置默认值？**
A:
1. 在 `Default()` 方法中添加逻辑
2. 检查字段是否为空值
3. 设置默认值

**Q: 如何禁用 Webhook？**
A:
```bash
# 环境变量
export ENABLE_WEBHOOKS=false
make run

# 或删除 WebhookConfiguration
kubectl delete mutatingwebhookconfiguration mcalculate.kb.io
kubectl delete validatingwebhookconfiguration vcalculate.kb.io
```

## 高级特性

### 1. Leader Election

```bash
# 启用 Leader Election
make run -- --leader-elect

# 或者在部署时启用
kubectl set env deployment/webhook-controller-manager \
  --namespace=webhook-system \
  --containers=manager \
  -- LEADER_ELECTION=true
```

### 2. 监控

```bash
# Metrics 端点
http://localhost:8443/metrics

# 常用指标
- controller_runtime_reconcile_total
- workqueue_depth
- webhook_request_total
```

### 3. 健康检查

```bash
# 健康检查
curl http://localhost:8081/healthz

# 就绪检查
curl http://localhost:8081/readyz
```

### 4. 多副本部署

```yaml
# config/manager/manager.yaml
apiVersion: v1
kind: Deployment
metadata:
  name: webhook-controller-manager
spec:
  replicas: 3  # 多副本
  selector:
    matchLabels:
      control-plane: controller-manager
```

## 参考资源

- [Kubebuilder 文档](https://book.kubebuilder.io/)
- [Kubebuilder GitHub](https://github.com/kubernetes-sigs/kubebuilder)
- [Controller Runtime](https://pkg.go.dev/sigs.k8s.io/controller-runtime)
- [Kubernetes CRD](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)

---

**最后更新**: 2025-12-28
**维护者**: kubernetes-examples 项目团队
