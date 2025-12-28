# 项目中未列入学习路径的目录说明

本文档说明 `kubernetes-examples` 项目中未列入 6 个学习路径的目录及其功能。

## 目录概览

```
kubernetes-examples/
├── 已列入学习路径（6 个阶段）
│   ├── 阶段 1: client-go/（基础入门）
│   ├── 阶段 2: client-go/（Informer & Controller）
│   ├── 阶段 3: resourcedefinition/, pod-operator/（CRD & Operator）
│   ├── 阶段 4: webhook/（Webhook 开发）
│   ├── 阶段 5: k8s-scheduler-extender-example/（Scheduler 扩展）
│   └── 阶段 6: leader-election/, finalizer-example/, ownerreference-example/（高级主题）
│
└── 未列入学习路径
    ├── apiversioncompatibility/        # API 版本兼容性
    ├── helper/                       # 辅助工具
    ├── kind/                         # Kind 配置（学习中已提及）
    ├── kubernetes-plugins/            # Kubernetes 插件
    ├── kubescheduler-sourcetree/   # Scheduler 源码分析
    ├── pod-creation-workflow/        # Pod 创建流程
    ├── pvpvc/                        # PV/PVC 示例
    ├── setdefault/                   # 设置默认值
    └── template/                      # Rancher Local Path Provisioner
```

---

## 1. apiversioncompatibility/

**路径**: `apiversioncompatibility/`

**功能**: API 版本兼容性示例

### 说明

演示如何处理 API 版本的向后兼容性。示例展示了如何同时支持旧版参数和新版参数：

- **旧版参数**: `Param string`（单个参数）
- **新版参数**: `Params []string`（参数列表）

### 核心逻辑

```go
type Frobber struct {
    Height int      `json:"height"`
    Width  int      `json:"width"`
    Param  string   `json:"param"`   // 旧版参数
    Params []string `json:"params"` // 新版参数
}

// CreateFrobber: 创建时兼容旧版和新版参数
func CreateFrobber(ctx context.Context, frobber *Frobber) error {
    // 旧版参数 -> 新版参数转换
    if frobber.Param != "" {
        frobber.Params = append(frobber.Params, frobber.Param)
    }
    frobberStorage["frobber1"] = frobber
    return nil
}

// GetFrobber: 获取时兼容旧版和新版参数
func GetFrobber(ctx context.Context) *Frobber {
    frobber := frobberStorage["frobber1"]
    // 新版参数 -> 旧版参数迁移
    if len(frobber.Params) == 0 && frobber.Param != "" {
        frobber.Params = append(frobber.Params, frobber.Param)
    }
    return frobber
}
```

### 学习要点

1. **向后兼容策略**：保持旧 API 同时支持新 API
2. **参数迁移**：自动在旧版和新版参数之间转换
3. **存储适配**：统一存储格式
4. **验证逻辑**：确保新旧参数一致性

### 适用场景

- CRD API 升级
- 字段重命名或类型变更
- 渐进式 API 演进

---

## 2. helper/

**路径**: `helper/`

**功能**: 通用辅助工具函数

### 说明

提供常用的 Kubernetes 工具函数：

1. **Prompt()**: 交互式提示，等待用户按回车
2. **AddKubeconfigFlag()**: 添加 `--kubeconfig` 命令行参数

### 核心逻辑

```go
// Prompt: 交互式提示
func Prompt() {
    fmt.Printf("-> Press Return key to continue.")
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        break
    }
}

// AddKubeconfigFlag: 添加 kubeconfig 参数
func AddKubeconfigFlag() string {
    defaultKubeconfig := os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
    if defaultKubeconfig == "" {
        defaultKubeconfig = clientcmd.RecommendedHomeFile
    }
    
    kubeconfig := flag.String(clientcmd.RecommendedConfigPathFlag, defaultKubeconfig, "Absolute path to the kubeconfig file.")
    flag.Parse()
    
    return *kubeconfig
}
```

### 学习要点

1. **命令行参数处理**: 使用 `flag` 包
2. **环境变量优先级**: `KUBECONFIG` 环境变量 > 默认路径
3. **交互式输入**: 使用 `bufio.Scanner` 处理用户输入

### 使用示例

```go
func main() {
    // 添加 kubeconfig 参数
    kubeconfig := helper.AddKubeconfigFlag()
    
    // 使用 kubeconfig
    config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
    
    // 提示用户
    helper.Prompt()
}
```

---

## 3. kind/

**路径**: `kind/`

**功能**: Kind 集群配置文件

### 说明

已在学习路径阶段 1 中提及，提供 Kind 集群的配置示例：

- **kind-config.yaml**: Kind 集群配置
  - 4 节点集群（1 control-plane + 3 worker）
  - 双栈网络（IPv4 + IPv6）
  - 自定义 DNS
  - kube-proxy IPVS 模式

### 学习要点

1. **Kind 配置**: 了解 Kind 集群结构
2. **双栈网络**: IPv4/IPv6 配置
3. **节点亲和性**: 多节点配置
4. **资源调度**: kube-proxy 模式选择

---

## 4. kubernetes-plugins/

**路径**: `kubernetes-plugins/`

**功能**: Kubernetes 插件开发示例

### 4.1 Aggregated API Server

**路径**: `kubernetes-plugins/api/aggregated-apiserver/`

**功能**: 实现 Aggregated API Server（聚合 API）

**说明**：
- 自定义 API Server
- 集成到 Kubernetes API Server
- 提供 `/api/v1/customresource` 端点
- 使用 TLS 加密

**核心逻辑**：
```go
// 自定义资源
type CustomResource struct {
    Message string `json:"message"`
}

// HTTP 处理器
http.HandleFunc("/api/v1/customresource", func(w http.ResponseWriter, r *http.Request) {
    response := CustomResource{Message: "Hello from Custom Resource API!"}
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
})

// HTTPS 服务器
srv := &http.Server{
    Addr:      ":8443",
    TLSConfig: tlsConfig,
}
```

**学习要点**：
- Aggregated API Server 架构
- 自定义 API 端点
- TLS 证书配置
- APIService 注册

### 4.2 Credential Plugin

**路径**: `kubernetes-plugins/client/kubectl/`

**功能**: Kubectl Credential Plugin

**说明**：
- 扩展 kubectl 认证
- 自定义认证逻辑
- `kubectl-foo` 可执行文件

### 4.3 CRD 示例

**路径**: `kubernetes-plugins/api/crd/`

**功能**: CRD 定义示例

**说明**：
- `my-crontab.yaml`: 定时任务 CRD 示例
- `resourcedefinition.yaml`: 通用资源定义示例

---

## 5. kubescheduler-sourcetree/

**路径**: `kubescheduler-sourcetree/v1.31.1/`

**功能**: Kubernetes Scheduler 源码分析文档

### 说明

提供 Kubernetes Scheduler v1.31.1 的源码树分析：

- **comprehensive.md**: 详细源码分析
- **concise.md**: 简要总结
- **superconcise.md**: 超简总结

### 学习要点

1. **Scheduler 架构**: 理解调度器设计
2. **调度流程**: Filter → Priority → Bind
3. **扩展点**: 了解可扩展的位置
4. **源码阅读**: 学习如何阅读 K8s 源码

---

## 6. pod-creation-workflow/

**路径**: `pod-creation-workflow/`

**功能**: Pod 创建工作流演示

### 说明

展示 ReplicaSet 如何创建和管理 Pod：

**文件**: `replicaset.yaml`

```yaml
apiVersion: apps/v1  
kind: ReplicaSet  
metadata:  
  namespace: default
  name: my-replicaset  
spec:  
  replicas: 2  # 指定要运行的 Pod 数量  
  selector:  
    matchLabels:  
      app: my-app  # 与 Pod 的标签匹配  
  template:  
    metadata:  
      labels:  
        app: my-app  # Pod 的标签  
    spec:  
      containers:  
      - name: my-container  
        image: nginx  
        ports:  
        - containerPort: 80
```

### 学习要点

1. **ReplicaSet 工作原理**：
   - `spec.replicas`: 指定期望 Pod 数量
   - `selector`: 标签选择器，匹配管理的 Pod
   - `template`: Pod 模板

2. **Pod 创建流程**：
   ```
   用户创建 ReplicaSet
   → ReplicaSet Controller 检测到变化
   → 根据模板创建 Pod
   → 确保 Pod 数量等于 replicas
   ```

3. **标签匹配**：
   - Pod 必须有 `app=my-app` 标签
   - ReplicaSet 通过标签选择器管理 Pod

---

## 7. pvpvc/

**路径**: `pvpvc/`

**功能**: PV 和 PVC 的 SubPath 用法示例

### 说明

演示如何使用 SubPath 将同一个 PVC 挂载到多个容器：

**文件**: `pod-subpath.yaml`

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: pod-subpath-zltest
spec:
  containers:
  - name: ubuntu-subpath-container
    image: ubuntu
    volumeMounts:
    - mountPath: /var/lib/ubuntu      # 容器1 的挂载目录
      name: subpath-vol
      subPath: ubuntutest              # 宿主机的子目录1
  - name: nginx-subpath-container
    image: nginx
    volumeMounts:
    - mountPath: /var/www/nginx           # 容器2 的挂载目录
      name: subpath-vol
      subPath: nginxtest                 # 宿主机的子目录2 
  volumes:
  - name: subpath-vol
    persistentVolumeClaim:
      claimName: pvc-subpath          # PVC 的名字
```

**文件**: `pv-subpath.yaml`、`pvc-subpath.yaml`

### 学习要点

1. **SubPath 概念**：
   - PVC 可以包含多个子目录
   - 不同容器可以挂载同一个 PVC 的不同子目录
   - 避免数据共享和冲突

2. **使用场景**：
   - 多容器共享同一个 PVC 但隔离数据
   - 日志文件分离
   - 配置文件隔离

3. **目录结构**：
   ```
   PVC
   ├── ubuntutest/     (ubuntu 容器挂载)
   └── nginxtest/      (nginx 容器挂载)
   ```

---

## 8. setdefault/

**路径**: `setdefault/`

**功能**: HTTP API 默认值设置示例

### 说明

演示如何在 HTTP API 中设置默认值和验证参数：

**核心逻辑**：
```go
type Request struct {
    Limit  int    `form:"limit" json:"limit"`
    Offset int    `form:"offset" json:"offset"`
    Filter string `form:"filter" json:"filter"`
}

// Validate: 验证参数并设置默认值
func (req *Request) Validate() error {
    // 设置默认值
    if req.Limit <= 0 {
        req.Limit = 10 // 默认值
    }
    if req.Offset < 0 {
        req.Offset = 0 // 默认值
    }
    if req.Filter == "" {
        req.Filter = "all" // 默认值
    }
    
    return nil
}
```

### 学习要点

1. **默认值设置**：在 Validate 中设置
2. **参数验证**：检查参数合法性
3. **Gin 框架**：使用 `ShouldBindQuery` 解析查询参数
4. **RESTful API**：GET 请求参数处理

### 适用场景

- CRD Webhook 的 Default 逻辑
- Kubernetes Operator 的默认值设置
- RESTful API 开发

---

## 9. template/

**路径**: `template/examples-form-rancher-local-path-provisioner/`

**功能**: Rancher Local Path Provisioner 模板

### 说明

提供 Rancher Local Path Provisioner 的部署模板和示例：

**内容**：
1. **distroless/**: 无操作系统依赖的 Provisioner
2. **Pod 示例**：
   - `pod/`: 基础 Pod
   - `pod-with-local-volume/`: 使用本地存储
   - `pod-with-node-affinity/`: 节点亲和性
   - `pod-with-rwop-volume/`: 读写访问模式
   - `pod-with-security-context/`: 安全上下文
   - `pod-with-subpath/`: SubPath 挂载
3. **PVC 示例**：
   - `pvc/`: 基础 PVC
   - `pvc-with-local-volume/`: 使用本地存储
   - `pvc-with-node/`: 指定节点
   - `pvc-with-rwop-access-mode/`: 读写访问模式
4. **Quota 示例**：资源配额
5. **配置文件**：
   - `kind.yaml`: Kind 集群配置
   - `local-path-storage.yaml`: Local Path StorageClass
   - `kustomization.yaml`: Kustomize 配置

### 学习要点

1. **Local Path Provisioner**：
   - 本地存储自动提供
   - 无需外部存储系统
   - 适合测试和开发环境

2. **StorageClass 配置**：
   ```yaml
   apiVersion: storage.k8s.io/v1
   kind: StorageClass
   metadata:
     name: local-path
   provisioner: rancher.io/local-path
   volumeBindingMode: WaitForFirstConsumer
   reclaimPolicy: Delete
   ```

3. **Kustomize 使用**：
   - 基础配置管理
   - Overlay 和 Patch 机制
   - 环境差异化配置

---

## 总结

### 可学习的高级主题

| 目录 | 主题 | 复杂度 | 推荐学习顺序 |
|------|--------|----------|--------------|
| apiversioncompatibility | API 兼容性 | ⭐⭐ | 阶段 3 后 |
| helper | 工具函数 | ⭐ | 随时学习 |
| kind | Kind 配置 | ⭐ | 阶段 1 |
| kubernetes-plugins | 插件开发 | ⭐⭐⭐⭐ | 阶段 5 后 |
| kubescheduler-sourcetree | Scheduler 源码 | ⭐⭐⭐⭐ | 阶段 5 后 |
| pod-creation-workflow | 工作流理解 | ⭐⭐ | 阶段 1 |
| pvpvc | PV/PVC 高级用法 | ⭐⭐ | 阶段 1 |
| setdefault | 默认值处理 | ⭐⭐ | 阶段 4 |
| template | Local Storage | ⭐⭐⭐ | 阶段 1 后 |

### 建议扩展学习路径

#### 基础扩展（阶段 0-1 后）
- **kind/**: Kind 集群配置
- **pod-creation-workflow/**: Pod 创建流程
- **pvpvc/**: PV/PVC 高级用法
- **template/**: Local Path Provisioner

#### 进阶扩展（阶段 3 后）
- **apiversioncompatibility/**: API 兼容性
- **setdefault/**: 默认值处理

#### 高级扩展（阶段 5 后）
- **kubernetes-plugins/**: 插件开发
  - Aggregated API Server
  - Credential Plugin
  - CRD 定义
- **kubescheduler-sourcetree/**: Scheduler 源码分析

#### 工具函数（随时学习）
- **helper/**: 通用工具函数

---

## 如何使用这些目录

### 1. 按需学习

根据实际需求选择对应目录：

- **需要本地测试环境**：学习 `kind/` 和 `template/`
- **需要理解 Pod 流程**：学习 `pod-creation-workflow/`
- **需要存储方案**：学习 `pvpvc/` 和 `template/`
- **需要 API 兼容性**：学习 `apiversioncompatibility/`
- **需要插件开发**：学习 `kubernetes-plugins/`
- **需要 Scheduler 深入**：学习 `kubescheduler-sourcetree/`

### 2. 辅助工具

所有示例都可能用到 `helper/` 中的工具函数：

```go
import "github.com/ashwinyue/kubernetes-examples/helper"

func main() {
    // 添加 kubeconfig 参数
    kubeconfig := helper.AddKubeconfigFlag()
    
    // 交互式提示
    helper.Prompt()
    
    // ... 其他逻辑
}
```

### 3. 最佳实践

1. **循序渐进**：先掌握基础知识，再学习高级主题
2. **动手实践**：每个示例都应该实际运行
3. **阅读源码**：深入理解实现原理
4. **阅读文档**：结合 Kubernetes 官方文档学习

---

## 相关资源

- [Kubernetes 官方文档](https://kubernetes.io/docs/home/)
- [Kubernetes API 概念](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/)
- [Aggregated API Server](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/apiserver-aggregation/)
- [Storage Classes](https://kubernetes.io/docs/concepts/storage/storage-classes/)
- [Local Path Provisioner](https://github.com/rancher/local-path-provisioner)

---

**最后更新**: 2025-12-28
**维护者**: kubernetes-examples 项目团队
