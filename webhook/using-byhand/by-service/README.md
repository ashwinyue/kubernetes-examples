# Admission Webhook 示例

完整的 Webhook 开发示例，展示：
- Mutating Webhook 实现（修改资源）
- Validating Webhook 实现（验证资源）
- TLS 证书管理
- JSON Patch 操作
- WebhookConfiguration 配置

## 功能说明

本示例演示了两种 Kubernetes Admission Webhook：

1. **Mutating Webhook**：自动为 Deployment 和 Service 添加缺失的推荐标签
   - 添加 `app.kubernetes.io/*` 系列标签
   - 如果标签缺失，设置为 `not_available` 值

2. **Validating Webhook**：验证资源是否包含必需的标签
   - 检查 `app.kubernetes.io/name` 等必需标签
   - 如果标签缺失，拒绝资源创建

## 项目结构

```
webhook/using-byhand/by-service/
├── main.go                      # Webhook 服务器入口
├── webhook.go                   # Webhook 核心逻辑
├── Dockerfile                    # Docker 镜像构建
├── build.sh                     # 镜像构建和推送脚本
├── deployment/
│   ├── deployment.yaml            # Webhook Deployment
│   ├── service.yaml              # Webhook Service
│   ├── rbac.yaml                # ServiceAccount 和权限
│   ├── mutatingwebhook.yaml      # Mutating Webhook 配置
│   ├── validatingwebhook.yaml    # Validating Webhook 配置
│   ├── sleep.yaml               # 测试用例（无标签）
│   ├── sleep-with-labels.yaml   # 测试用例（有标签）
│   └── webhook-create-signed-cert.sh  # CSR 证书生成脚本
└── certs/                       # TLS 证书目录（运行时生成）
    ├── cert.pem
    └── key.pem
```

## 快速开始

### 前置条件

- Kubernetes 1.16+ 集群
- `kubectl` 已配置并连接到集群
- OpenSSL 工具（用于生成证书）

### 1. 生成 TLS 证书

```bash
cd webhook/using-byhand/by-service

# 创建证书目录
mkdir -p certs

# 生成 TLS 证书和私钥
cd certs
openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem \
  -days 365 -nodes \
  -subj "/CN=admission-webhook-example-svc.default.svc" \
  -addext "subjectAltName=DNS:admission-webhook-example-svc,DNS:admission-webhook-example-svc.default,DNS:admission-webhook-example-svc.default.svc"

# 返回上一级目录
cd ..
```

**证书说明**：
- Common Name (CN)：`admission-webhook-example-svc.default.svc`
- Subject Alternative Names (SAN)：
  - `admission-webhook-example-svc`
  - `admission-webhook-example-svc.default`
  - `admission-webhook-example-svc.default.svc`

### 2. 创建 Secret

```bash
# 创建 Secret 存储证书
kubectl create secret generic admission-webhook-example-certs \
  --from-file=cert.pem=certs/cert.pem \
  --from-file=key.pem=certs/key.pem

# 验证 Secret
kubectl get secret admission-webhook-example-certs
kubectl describe secret admission-webhook-example-certs
```

### 3. 部署 Webhook 服务

```bash
# 应用 RBAC 配置
kubectl apply -f deployment/rbac.yaml

# 应用 Service
kubectl apply -f deployment/service.yaml

# 应用 Deployment
kubectl apply -f deployment/deployment.yaml

# 等待 Pod 启动
kubectl get pods -l app=admission-webhook-example -w

# 查看日志
kubectl logs -l app=admission-webhook-example -f
```

**验证**：
- Pod 状态为 `Running`
- 日志显示 `Server started`
- TLS 证书正确加载

### 4. 配置 Webhook

```bash
# 准备 CA Bundle
CA_BUNDLE=$(cat certs/cert.pem | base64 | tr -d '\n')

# 创建 Mutating Webhook 配置
cat > mutatingwebhook.yaml << EOF
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: mutating-webhook-example-cfg
webhooks:
  - name: mutating-example.qikqiak.com
    clientConfig:
      service:
        name: admission-webhook-example-svc
        namespace: default
        path: "/mutate"
      caBundle: $CA_BUNDLE
    rules:
      - operations: ["CREATE"]
        apiGroups: ["apps", ""]
        apiVersions: ["v1"]
        resources: ["deployments", "services"]
    sideEffects: None
    admissionReviewVersions: ["v1"]
    namespaceSelector:
      matchLabels:
        admission-webhook-example: enabled
EOF

# 创建 Validating Webhook 配置
cat > validatingwebhook.yaml << EOF
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: validation-webhook-example-cfg
webhooks:
  - name: required-labels.qikqiak.com
    clientConfig:
      service:
        name: admission-webhook-example-svc
        namespace: default
        path: "/validate"
      caBundle: $CA_BUNDLE
    rules:
      - operations: ["CREATE"]
        apiGroups: ["apps", ""]
        apiVersions: ["v1"]
        resources: ["deployments", "services"]
    sideEffects: None
    admissionReviewVersions: ["v1"]
    namespaceSelector:
      matchLabels:
        admission-webhook-example: enabled
EOF

# 应用 Webhook 配置
kubectl apply -f mutatingwebhook.yaml
kubectl apply -f validatingwebhook.yaml

# 验证 Webhook 配置
kubectl get mutatingwebhookconfiguration
kubectl get validatingwebhookconfiguration
```

### 5. 启用 Webhook

```bash
# 给命名空间打标签，启用 Webhook
kubectl label namespace default admission-webhook-example=enabled

# 验证标签
kubectl get namespace default --show-labels
```

**重要**：只有带有 `admission-webhook-example=enabled` 标签的命名空间才会被 Webhook 拦截。

## 测试

### 测试 1：Mutating Webhook - 添加标签

```bash
# 创建没有标签的 Deployment
kubectl apply -f deployment/sleep.yaml

# 查看 Deployment 的标签（应该自动添加了缺失的标签）
kubectl get deployment sleep -o yaml | grep -A 20 labels:

# 查看 Deployment 的注解（应该有 webhook 状态）
kubectl get deployment sleep -o yaml | grep -A 5 annotations:
```

**预期结果**：
- Deployment 成功创建
- 自动添加了 `app.kubernetes.io/*` 系列标签
- 标签值为 `not_available`
- 注解包含 `admission-webhook-example.qikqiak.com/status: mutated`

### 测试 2：Validating Webhook - 验证标签

```bash
# 尝试创建没有必需标签的 Deployment（应该失败）
kubectl apply -f deployment/sleep.yaml

# 查看错误信息
kubectl describe deployment sleep

# 或者使用 kubectl apply --dry-run 查看验证结果
kubectl apply -f deployment/sleep.yaml --dry-run=server
```

**预期结果**：
- 创建失败
- 错误消息提示缺少必需标签

### 测试 3：通过验证 - 有标签

```bash
# 创建有完整标签的 Deployment（应该成功）
kubectl apply -f deployment/sleep-with-labels.yaml

# 查看 Deployment 状态
kubectl get deployment sleep

# 查看 Webhook 日志
kubectl logs -l app=admission-webhook-example -f
```

**预期结果**：
- Deployment 成功创建
- Webhook 日志显示验证通过
- 保留原始标签值

### 清理测试资源

```bash
# 删除测试 Deployment
kubectl delete deployment sleep

# 删除 Webhook 配置
kubectl delete mutatingwebhookconfiguration mutating-webhook-example-cfg
kubectl delete validatingwebhookconfiguration validation-webhook-example-cfg

# 删除 Webhook 服务
kubectl delete -f deployment/deployment.yaml
kubectl delete -f deployment/rbac.yaml
kubectl delete -f deployment/service.yaml

# 删除 Secret
kubectl delete secret admission-webhook-example-certs

# 移除命名空间标签
kubectl label namespace default admission-webhook-example-
```

## 学习要点

### 1. Admission Webhook 架构

**AdmissionReview 协议**：

```go
// 请求结构
type AdmissionRequest struct {
    UID                string                  `json:"uid"`
    Kind               metav1.GroupVersionKind `json:"kind"`
    Resource           metav1.GroupVersionResource `json:"resource"`
    SubResource         string                   `json:"subResource,omitempty"`
    RequestKind         *AdmissionRequestKind    `json:"requestKind,omitempty"`
    RequestResource     *AdmissionRequestResource `json:"requestResource,omitempty"`
    Name               string                   `json:"name,omitempty"`
    Namespace          string                   `json:"namespace,omitempty"`
    Operation          string                   `json:"operation"`
    UserInfo           authentication.UserInfo    `json:"userInfo,omitempty"`
    Object             runtime.RawExtension      `json:"object,omitempty"`
    OldObject          runtime.RawExtension      `json:"oldObject,omitempty"`
    DryRun             bool                     `json:"dryRun,omitempty"`
    Options            runtime.RawExtension      `json:"options,omitempty"`
}

// 响应结构
type AdmissionResponse struct {
    UID     string              `json:"uid"`
    Allowed  bool                `json:"allowed"`
    Result   *metav1.Status     `json:"result,omitempty"`
    Patch    []byte              `json:"patch,omitempty"`
    PatchType *PatchType         `json:"patchType,omitempty"`
    AuditAnnotations map[string]string `json:"auditAnnotations,omitempty"`
}
```

**要点**：
- `UID`：请求的唯一标识符，必须在响应中返回
- `Allowed`：是否允许请求（true/false）
- `Patch`：JSON Patch 数据（仅用于 Mutating Webhook）
- `Result`：错误信息（仅用于 Validating Webhook）

### 2. Mutating Webhook

**文件**: `webhook.go`

```go
func (whsvr *WebhookServer) mutate(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
    req := ar.Request
    var (
        availableLabels map[string]string
        objectMeta    *metav1.ObjectMeta
    )

    // 1. 解码资源对象
    switch req.Kind.Kind {
    case "Deployment":
        var deployment appsv1.Deployment
        if err := json.Unmarshal(req.Object.Raw, &deployment); err != nil {
            return &admissionv1.AdmissionResponse{
                Result: &metav1.Status{
                    Message: err.Error(),
                },
            }
        }
        resourceName, resourceNamespace, objectMeta = deployment.Name, deployment.Namespace, &deployment.ObjectMeta
        availableLabels = deployment.Labels
    case "Service":
        // 类似处理 Service
    }

    // 2. 检查是否需要变更
    if !mutationRequired(ignoredNamespaces, objectMeta) {
        return &admissionv1.AdmissionResponse{
            Allowed: true,
        }
    }

    // 3. 创建 JSON Patch
    annotations := map[string]string{admissionWebhookAnnotationStatusKey: "mutated"}
    patchBytes, err := createPatch(availableAnnotations, annotations, availableLabels, addLabels)
    if err != nil {
        return &admissionv1.AdmissionResponse{
            Result: &metav1.Status{
                Message: err.Error(),
            },
        }
    }

    // 4. 返回响应
    return &admissionv1.AdmissionResponse{
        Allowed:   true,
        Patch:     patchBytes,
        PatchType: func() *admissionv1.PatchType {
            pt := admissionv1.PatchTypeJSONPatch
            return &pt
        }(),
    }
}
```

**要点**：
- 解码资源对象（Deployment/Service）
- 检查是否需要变更
- 生成 JSON Patch
- 返回 Allowed: true + Patch 数据

### 3. Validating Webhook

**文件**: `webhook.go`

```go
func (whsvr *WebhookServer) validate(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
    req := ar.Request
    var (
        availableLabels map[string]string
        objectMeta    *metav1.ObjectMeta
    )

    // 1. 解码资源对象
    switch req.Kind.Kind {
    case "Deployment":
        var deployment appsv1.Deployment
        if err := json.Unmarshal(req.Object.Raw, &deployment); err != nil {
            return &admissionv1.AdmissionResponse{
                Result: &metav1.Status{
                    Message: err.Error(),
                },
            }
        }
        resourceName, resourceNamespace, objectMeta = deployment.Name, deployment.Namespace, &deployment.ObjectMeta
        availableLabels = deployment.Labels
    case "Service":
        // 类似处理 Service
    }

    // 2. 检查是否需要验证
    if !validationRequired(ignoredNamespaces, objectMeta) {
        return &admissionv1.AdmissionResponse{
            Allowed: true,
        }
    }

    // 3. 检查必需标签
    allowed := true
    var result *metav1.Status
    for _, rl := range requiredLabels {
        if _, ok := availableLabels[rl]; !ok {
            allowed = false
            result = &metav1.Status{
                Status:  metav1.StatusFailure,
                Reason:  metav1.StatusReasonInvalid,
                Code:    http.StatusBadRequest,
                Message: fmt.Sprintf("missing required label: %s", rl),
            }
            break
        }
    }

    // 4. 返回响应
    return &admissionv1.AdmissionResponse{
        Allowed: allowed,
        Result:  result,
    }
}
```

**要点**：
- 解码资源对象
- 检查必需标签
- 返回 Allowed: false + Result（错误信息）
- 不使用 Patch 字段

### 4. JSON Patch 操作

**文件**: `webhook.go`

```go
type patchOperation struct {
    Op    string      `json:"op"`              // add、replace、remove
    Path  string      `json:"path"`            // JSON Pointer 路径
    Value interface{} `json:"value,omitempty"` // 要设置的新值
}

func createPatch(availableAnnotations map[string]string, annotations map[string]string, availableLabels map[string]string, labels map[string]string) ([]byte, error) {
    var patch []patchOperation

    // 添加注解
    patch = append(patch, updateAnnotation(availableAnnotations, annotations)...)

    // 添加标签
    patch = append(patch, updateLabels(availableLabels, labels)...)

    return json.Marshal(patch)
}

func updateLabels(availableLabels map[string]string, labels map[string]string) []patchOperation {
    var patch []patchOperation
    for k, v := range labels {
        if _, ok := availableLabels[k]; !ok {
            patch = append(patch, patchOperation{
                Op:    "add",
                Path:  "/metadata/labels",
                Value: map[string]string{k: v},
            })
        }
    }
    return patch
}
```

**JSON Patch 示例**：

```json
[
  {
    "op": "add",
    "path": "/metadata/annotations",
    "value": {
      "admission-webhook-example.qikqiak.com/status": "mutated"
    }
  },
  {
    "op": "add",
    "path": "/metadata/labels",
    "value": {
      "app.kubernetes.io/name": "not_available",
      "app.kubernetes.io/version": "not_available"
    }
  }
]
```

**要点**：
- RFC 6902 标准（JSON Patch）
- `op` 字段：add、replace、remove、move、copy、test
- `path` 字段：JSON Pointer 格式（如 `/metadata/labels`）
- `value` 字段：要设置的新值

### 5. HTTP 服务器

**文件**: `main.go`

```go
func main() {
    var parameters WhSvrParameters

    // 1. 获取命令行参数
    flag.IntVar(&parameters.port, "port", 443, "Webhook server port.")
    flag.StringVar(&parameters.certFile, "tlsCertFile", "/etc/webhook/certs/cert.pem", "TLS certificate file")
    flag.StringVar(&parameters.keyFile, "tlsKeyFile", "/etc/webhook/certs/key.pem", "TLS private key file")
    flag.Parse()

    // 2. 加载 TLS 证书
    pair, err := tls.LoadX509KeyPair(parameters.certFile, parameters.keyFile)
    if err != nil {
        log.Errorf("Failed to load key pair: %v", err)
        return
    }

    // 3. 创建 HTTP 服务器
    whsvr := &WebhookServer{
        server: &http.Server{
            Addr:      fmt.Sprintf(":%v", parameters.port),
            TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
        },
    }

    // 4. 注册路由
    mux := http.NewServeMux()
    mux.HandleFunc("/mutate", whsvr.serve)
    mux.HandleFunc("/validate", whsvr.serve)
    whsvr.server.Handler = mux

    // 5. 启动服务器
    go func() {
        if err := whsvr.server.ListenAndServeTLS("", ""); err != nil {
            log.Errorf("Failed to listen and serve webhook server: %v", err)
        }
    }()

    // 6. 等待关闭信号
    signalChan := make(chan os.Signal, 1)
    signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
    <-signalChan

    whsvr.server.Shutdown(context.Background())
}
```

**要点**：
- HTTPS 服务器（TLS 必须）
- 443 端口（标准 Webhook 端口）
- 路由：`/mutate` 和 `/validate`
- 优雅关闭处理

### 6. WebhookConfiguration

**MutatingWebhookConfiguration**：

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: mutating-webhook-example-cfg
webhooks:
  - name: mutating-example.qikqiak.com
    clientConfig:
      service:
        name: admission-webhook-example-svc
        namespace: default
        path: "/mutate"
      caBundle: <BASE64_ENCODED_CERT>
    rules:
      - operations: ["CREATE", "UPDATE"]
        apiGroups: ["apps", ""]
        apiVersions: ["v1"]
        resources: ["deployments", "services"]
    sideEffects: None
    admissionReviewVersions: ["v1"]
    namespaceSelector:
      matchLabels:
        admission-webhook-example: enabled
```

**ValidatingWebhookConfiguration**：

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: validation-webhook-example-cfg
webhooks:
  - name: required-labels.qikqiak.com
    clientConfig:
      service:
        name: admission-webhook-example-svc
        namespace: default
        path: "/validate"
      caBundle: <BASE64_ENCODED_CERT>
    rules:
      - operations: ["CREATE", "UPDATE"]
        apiGroups: ["apps", ""]
        apiVersions: ["v1"]
        resources: ["deployments", "services"]
    sideEffects: None
    admissionReviewVersions: ["v1"]
    namespaceSelector:
      matchLabels:
        admission-webhook-example: enabled
```

**关键字段**：
- `clientConfig.service`：Webhook Service 地址
- `clientConfig.caBundle`：CA 证书的 Base64 编码
- `rules`：触发规则（operations、apiGroups、apiVersions、resources）
- `namespaceSelector`：命名空间选择器
- `admissionReviewVersions`：支持的 AdmissionReview 版本

### 7. TLS 证书

**证书要求**：
- X.509 证书格式
- 有效的 CN（Common Name）
- 正确的 SAN（Subject Alternative Names）
- 证书必须被 Kubernetes API Server 信任

**SAN 配置**：
```
DNS.1 = admission-webhook-example-svc
DNS.2 = admission-webhook-example-svc.default
DNS.3 = admission-webhook-example-svc.default.svc
```

**要点**：
- SAN 必须包含所有可能的 Service 地址
- 证书有效期建议 1 年
- 私钥必须保密（不应提交到 Git）

## 调试技巧

### 1. 查看 Webhook 日志

```bash
# 查看 Pod 日志
kubectl logs -l app=admission-webhook-example -f

# 查看 Webhook 收到的请求
kubectl logs -l app=admission-webhook-example | grep "AdmissionReview"

# 查看 JSON Patch 生成
kubectl logs -l app=admission-webhook-example | grep "AdmissionResponse: patch"
```

### 2. 测试 Webhook 连接

```bash
# 从集群内测试 HTTPS 连接
kubectl run debug --rm -it --image=curlimages/curl --restart=Never -- \
  curl -v https://admission-webhook-example-svc.default.svc:443/mutate \
  --cacert /etc/ssl/certs/ca.crt

# 查看 TLS 握手信息
kubectl logs -l app=admission-webhook-example | grep "TLS"
```

### 3. 查看事件

```bash
# 查看创建失败的事件
kubectl get events --field-selector involvedObject.kind=Deployment --sort-by='.lastTimestamp'

# 查看特定资源的事件
kubectl describe deployment <name>
```

### 4. 调试 JSON Patch

```bash
# 使用 kubectl patch --dry-run 测试 Patch
kubectl patch deployment <name> --type=json \
  --patch='[{"op": "add", "path": "/metadata/labels/app.kubernetes.io/name", "value": "test"}]' \
  --dry-run=server -o yaml
```

### 5. 查看请求详情

```bash
# 在 Webhook 代码中添加详细日志
glog.Infof("Received request: %+v", ar.Request)

# 查看解码的资源
glog.Infof("Decoded object: %+v", deployment)
```

## 常见问题

**Q: Webhook 没有被调用？**
A:
1. 检查 WebhookConfiguration 是否正确创建
2. 检查 namespaceSelector 是否匹配
3. 检查 Service 是否可达
4. 查看 API Server 日志：`kubectl logs -n kube-system -l component=kube-apiserver`

**Q: TLS 握手失败？**
A:
1. 检查证书的 SAN 是否正确
2. 检查 caBundle 是否正确编码（Base64）
3. 检查 Service 的 port 是否为 443
4. 使用 `openssl s_client -connect` 测试证书

**Q: JSON Patch 失败？**
A:
1. 检查 JSON Pointer 格式是否正确
2. 检查 Patch 是否有效（使用 JSON Patch 测试工具）
3. 查看错误消息：`kubectl describe deployment <name>`

**Q: 资源被拒绝但没有错误消息？**
A:
1. 检查 Validating Webhook 的 `Result` 字段
2. 确保错误消息清晰明确
3. 查看 Webhook 日志确认验证逻辑

**Q: Webhook 响应超时？**
A:
1. 检查 Webhook 处理逻辑是否过慢
2. 增加超时时间（配置在 WebhookConfiguration 中）
3. 优化 JSON 编解码性能

## 高级特性

### 1. Webhook 优先级

```yaml
webhooks:
  - name: high-priority-webhook
    objectSelector:
      matchLabels:
        priority: "high"
  - name: low-priority-webhook
    objectSelector:
      matchLabels:
        priority: "low"
```

**说明**：通过 `objectSelector` 控制 Webhook 执行顺序。

### 2. Failure Policy

```yaml
webhooks:
  - name: important-webhook
    failurePolicy: Fail  # Webhook 失败时拒绝请求
  - name: optional-webhook
    failurePolicy: Ignore  # Webhook 失败时忽略错误
```

**选项**：
- `Fail`：Webhook 失败时拒绝请求（默认）
- `Ignore`：Webhook 失败时忽略错误
- `NoOpinions`：Webhook 失败时不影响决策

### 3. Side Effects

```yaml
webhooks:
  - name: side-effect-webhook
    sideEffects: None  # 无副作用（默认）
  - name: side-effect-webhook
    sideEffects: NoneOnDryRun  # Dry Run 时无副作用
```

**选项**：
- `None`：无副作用
- `NoneOnDryRun`：Dry Run 时无副作用
- `Some`：有副作用

### 4. Reinvocation Policy

```yaml
webhooks:
  - name: reinvocation-webhook
    reinvocationPolicy: IfNeeded  # 根据需要重新调用
```

**选项**：
- `Never`：从不重新调用
- `IfNeeded`：根据需要重新调用（默认）
- `BeforePolicyExecution`：在策略执行前重新调用

## 安全考虑

### 1. 证书管理

- 使用 Kubernetes CSR（Certificate Signing Request）生成证书
- 定期轮换证书（建议 90 天）
- 使用 cert-manager 自动管理证书

### 2. 权限最小化

```yaml
rules:
  - apiGroups: [""]
    resources: ["configmaps", "secrets"]
    verbs: ["get", "list", "watch"]
```

**说明**：只授予必要的权限。

### 3. 输入验证

- 验证所有输入参数
- 防止注入攻击
- 限制资源大小

### 4. 审计日志

- 记录所有 Webhook 调用
- 记录请求和响应
- 使用 Kubernetes Audit 日志

## 参考资源

- [Kubernetes Admission Webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#admission-webhooks)
- [JSON Patch (RFC 6902)](https://tools.ietf.org/html/rfc6902)
- [Kubernetes API - Admission](https://kubernetes.io/docs/reference/kubernetes-api/working-resources/common-definitions/#admissionreview-v1)
- [Webhook 认证](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#webhook-configuration)

## 附录：Docker 镜像构建

如果需要构建和推送 Docker 镜像：

```bash
# 构建镜像
./build.sh REGISTRY_PREFIX=<your-registry>

# 或者手动构建
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o admission-webhook-example
docker build -t <registry>/admission-webhook-example:v1 .

# 推送镜像
docker push <registry>/admission-webhook-example:v1

# 更新 deployment.yaml 中的镜像地址
```

---

**最后更新**: 2025-12-28
**维护者**: kubernetes-examples 项目团队
