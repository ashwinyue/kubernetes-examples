# Baremetal Webhook 示例

不依赖 Kubernetes Service 的纯手动实现 Webhook，适合离线环境或特殊网络架构。

## 功能说明

本示例展示如何在不使用 Kubernetes Service 的情况下实现 Admission Webhook：

1. **Mutating Webhook**：为 Pod 自动添加注解 `apps.onex.io/miner-type`
2. **纯 TLS + HTTP**：不依赖 Service，使用固定 IP 地址
3. **离线友好**：适合无法使用 Service 的场景

## 项目结构

```
webhook/using-byhand/by-baremetal/
├── main.go                      # Webhook 服务器入口（单文件实现）
├── gen-ca.sh                    # TLS 证书生成脚本
├── pod.yaml                     # 测试用 Pod
├── nginx-deployment.yaml        # 测试用 Deployment
├── test-mutating-webhook.yaml   # MutatingWebhookConfiguration
└── cert/                        # TLS 证书目录
    ├── ca.crt                   # CA 证书
    ├── ca.key                   # CA 私钥
    ├── server.crt               # 服务器证书
    ├── server.key               # 服务器私钥
    ├── server.csr               # 服务器证书签名请求
    ├── extfile.cnf              # SAN 配置文件
    └── ca.srl                   # CA 序列号
```

## 快速开始

### 前置条件

- Kubernetes 1.16+ 集群
- `kubectl` 已配置并连接到集群
- OpenSSL 工具
- Webhook 服务器需要可访问的 IP 地址

### 1. 生成 TLS 证书

```bash
cd webhook/using-byhand/by-baremetal

# 运行证书生成脚本
./gen-ca.sh

# 脚本会自动创建 cert 目录并生成以下文件：
# - ca.crt: CA 证书（用于验证）
# - ca.key: CA 私钥（需保密）
# - server.crt: 服务器证书
# - server.key: 服务器私钥（需保密）
```

**脚本内容**：
```bash
#!/bin/bash

mkdir cert
cd cert

# 创建 CA 证书
openssl genrsa -out ca.key 2048
openssl req -x509 -new -nodes -key ca.key -subj "/CN=superproj" -days 3650 -out ca.crt

# 创建服务器证书
openssl genrsa -out server.key 2048
openssl req -new -key server.key -subj "/CN=superproj.com" -out server.csr

# 配置 SAN（Subject Alternative Names）
echo "subjectAltName = IP:10.37.83.200" > extfile.cnf

# 使用 CA 签名服务器证书
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 365 -extfile extfile.cnf

# 输出 CA 证书的 Base64 编码（用于 WebhookConfiguration）
cat ca.crt | base64 -w 0
```

**证书说明**：
- Common Name (CN)：`superproj.com`
- SAN（Subject Alternative Name）：`IP:10.37.83.200`
- 有效期：365 天

### 2. 启动 Webhook 服务器

```bash
# 启动 Webhook 服务器
go run main.go

# 输出：Started mutating admission webhook server
# 服务器监听：https://0.0.0.0:9999
```

**服务器说明**：
- 监听端口：`9999`
- TLS 证书：`cert/server.crt` 和 `cert/server.key`
- 路径：`/mutate`

### 3. 配置 Webhook

```bash
# 1. 获取 CA Bundle（Base64 编码）
CA_BUNDLE=$(cat cert/ca.crt | base64 -w 0)
echo "CA Bundle: $CA_BUNDLE"

# 2. 修改 test-mutating-webhook.yaml 中的以下内容：
#    - clientConfig.caBundle: 替换为上面获取的 CA_BUNDLE
#    - clientConfig.url: 替换为你的 Webhook 服务器 IP 地址

# 3. 应用 Webhook 配置
kubectl apply -f test-mutating-webhook.yaml

# 4. 验证配置
kubectl get mutatingwebhookconfiguration test-mutating-webhook
```

**WebhookConfiguration 示例**：
```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: test-mutating-webhook
webhooks:
  - name: test-mutating-webhook.example.com
    clientConfig:
      caBundle: <BASE64_ENCODED_CA_CERT>
      url: https://10.37.83.200:9999/mutate  # 使用固定 IP 地址
    rules:
      - apiGroups: [""]
        apiVersions: ["v1"]
        operations: ["CREATE"]
        resources: ["pods"]
    failurePolicy: Fail
    sideEffects: None
    admissionReviewVersions: ["v1", "v1beta1"]
```

### 4. 测试 Webhook

```bash
# 创建测试 Pod
kubectl apply -f pod.yaml

# 查看 Pod 的注解（应该自动添加了 apps.onex.io/miner-type）
kubectl get pod nginx -o yaml | grep -A 5 annotations:

# 预期输出：
# annotations:
#   apps.onex.io/miner-type: S1.SMALL1

# 删除 Pod
kubectl delete pod nginx
```

**测试结果**：
- Pod 成功创建
- 自动添加了 `apps.onex.io/miner-type: S1.SMALL1` 注解
- Webhook 日志显示请求处理信息

### 5. 清理

```bash
# 删除 Webhook 配置
kubectl delete mutatingwebhookconfiguration test-mutating-webhook

# 删除测试资源
kubectl delete pod nginx

# 停止 Webhook 服务器（Ctrl+C）
```

## 代码分析

### main.go 核心逻辑

```go
func main() {
    // 1. 注册路由
    http.HandleFunc("/mutate", mutate)

    // 2. 启动 HTTPS 服务器
    fmt.Println("Started mutating admission webhook server")
    panic(http.ListenAndServeTLS(":9999", "cert/server.crt", "cert/server.key", nil))
}
```

**要点**：
- 使用 `http.ListenAndServeTLS` 启动 HTTPS 服务器
- 证书路径：`cert/server.crt` 和 `cert/server.key`
- 监听所有网络接口（`:9999`）

### mutate 函数

```go
func mutate(w http.ResponseWriter, r *http.Request) {
    fmt.Println("Received a request sent by the kube-apiserver")

    // 1. 读取请求 body
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // 2. 反序列化为 AdmissionReview
    deserializer := serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
    ar := admissionv1.AdmissionReview{}
    if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // 3. 解码 Pod 对象
    var pod corev1.Pod
    if err := json.Unmarshal(ar.Request.Object.Raw, &pod); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // 4. 修改 Pod（添加注解）
    if pod.ObjectMeta.Annotations == nil {
        pod.ObjectMeta.Annotations = make(map[string]string)
    }
    pod.ObjectMeta.Annotations["apps.onex.io/miner-type"] = "S1.SMALL1"

    // 5. 创建 JSON Patch
    patch := []patchOperation{
        {
            Op:    "add",
            Path:  "/metadata/annotations",
            Value: pod.ObjectMeta.Annotations,
        },
    }
    patchBytes, err := json.Marshal(patch)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }

    // 6. 构造 AdmissionReview 响应
    admissionReview := admissionv1.AdmissionReview{
        TypeMeta: metav1.TypeMeta{
            APIVersion: "admission.k8s.io/v1",
            Kind:       "AdmissionReview",
        },
        Response: &admissionv1.AdmissionResponse{
            UID:       ar.Request.UID,
            Allowed:   true,
            Patch:     patchBytes,
            PatchType: func() *admissionv1.PatchType {
                pt := admissionv1.PatchTypeJSONPatch
                return &pt
            }(),
        },
    }

    // 7. 返回响应
    resp, err := json.Marshal(admissionReview)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Write(resp)

    // 8. 打印日志
    fmt.Printf("[%s/%s] Resource change succeeded\n", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
}
```

**要点**：
1. 读取 HTTP 请求 body
2. 反序列化为 `AdmissionReview`
3. 解码 Pod 对象
4. 修改 Pod（添加注解）
5. 创建 JSON Patch
6. 构造 `AdmissionReview` 响应
7. 返回 JSON 响应
8. 打印日志

## 学习要点

### 1. Baremetal Webhook 特点

**与 Service 方式对比**：

| 特性 | Service 方式 | Baremetal 方式 |
|------|------------|----------------|
| 网络依赖 | 依赖 Kubernetes Service | 直接使用 IP 地址 |
| 部署灵活性 | 集群内部部署 | 可部署在集群外部 |
| 证书管理 | 证书需匹配 Service DNS | 证书需匹配 IP 地址 |
| 适用场景 | 标准集群环境 | 离线环境、特殊网络架构 |
| 网络可达性 | 需要集群内部网络可达 | 需要固定 IP 可达 |

**适用场景**：
- **离线环境**：无法使用 Kubernetes Service
- **混合云环境**：Webhook 部署在外部云
- **特殊网络架构**：需要固定 IP 访问
- **测试环境**：本地快速测试

### 2. 证书配置

**SAN 配置**：
```bash
# extfile.cnf
subjectAltName = IP:10.37.83.200
```

**关键点**：
- 必须配置 SAN（Subject Alternative Names）
- SAN 必须包含 Webhook 服务器的 IP 地址
- Common Name (CN) 可以任意设置

### 3. WebhookConfiguration 配置

**使用 URL 而非 Service**：
```yaml
clientConfig:
  # 不使用 service 字段
  # service:
  #   name: webhook-svc
  #   namespace: default
  url: https://10.37.83.200:9999/mutate  # 直接使用 URL
```

**关键点**：
- 使用 `url` 字段而非 `service` 字段
- URL 必须是 HTTPS
- IP 地址必须与证书的 SAN 匹配
- 端口号必须与服务器监听端口一致

### 4. JSON Patch 操作

**单字段修改**：
```go
patch := []patchOperation{
    {
        Op:    "add",
        Path:  "/metadata/annotations",
        Value: pod.ObjectMeta.Annotations,
    },
}
```

**说明**：
- `Op`: 操作类型（add、replace、remove）
- `Path`: JSON Pointer 路径
- `Value`: 要设置的新值

## 调试技巧

### 1. 本地测试

```bash
# 启动 Webhook 服务器
go run main.go

# 在另一个终端测试（需要发送 AdmissionReview 格式的 JSON）
curl -k https://localhost:9999/mutate -X POST -H "Content-Type: application/json" -d @test-admission-review.json
```

**测试用例（test-admission-review.json）**：
```json
{
  "apiVersion": "admission.k8s.io/v1",
  "kind": "AdmissionReview",
  "request": {
    "uid": "test-uid",
    "kind": {"group":"","version":"v1","kind":"Pod"},
    "resource": {"group":"","version":"v1","resource":"pods"},
    "name": "test-pod",
    "namespace": "default",
    "operation": "CREATE",
    "object": {
      "metadata": {
        "name": "test-pod",
        "namespace": "default"
      },
      "spec": {
        "containers": [{
          "name": "nginx",
          "image": "nginx"
        }]
      }
    }
  }
}
```

### 2. 查看日志

```bash
# Webhook 服务器日志（标准输出）
Started mutating admission webhook server
Received a request sent by the kube-apiserver
[default/test-pod] Resource change succeeded
```

### 3. 验证证书

```bash
# 查看 SAN 配置
openssl x509 -in cert/server.crt -text -noout | grep -A 1 "Subject Alternative Name"

# 测试 TLS 连接
openssl s_client -connect 10.37.83.200:9999 -showcerts
```

### 4. 查看 Webhook 调用

```bash
# 查看 API Server 日志
kubectl logs -n kube-system -l component=kube-apiserver | grep webhook

# 查看 Webhook 调用事件
kubectl get events --field-selector involvedObject.kind=MutatingWebhookConfiguration
```

## 常见问题

**Q: Webhook 没有被调用？**
A:
1. 检查 Webhook 服务器是否启动
2. 检查 IP 地址是否可达：`ping 10.37.83.200`
3. 检查端口是否开放：`telnet 10.37.83.200 9999`
4. 检查证书 SAN 是否包含正确的 IP 地址
5. 查看 API Server 日志

**Q: TLS 握手失败？**
A:
1. 检查证书 SAN 是否包含 Webhook 服务器 IP
2. 检查证书是否过期：`openssl x509 -in cert/server.crt -noout -dates`
3. 检查 caBundle 是否正确编码（Base64）
4. 测试 TLS 连接：`openssl s_client -connect 10.37.83.200:9999`

**Q: context deadline exceeded？**
A:
1. 检查网络连接是否正常
2. 检查防火墙规则
3. 检查 Webhook 服务器是否在监听
4. 增加超时时间（在 WebhookConfiguration 中配置 `timeoutSeconds`）

**Q: 如何在 Kind 集群中测试？**
A:
Kind 集群运行在 Docker 容器中，无法直接访问宿主机 IP。建议：
1. 使用 `using-byhand/by-service` 方式（推荐）
2. 使用 hostNetwork 运行 Webhook Pod
3. 使用 `kubectl port-forward` 转发端口

**Q: 如何使用域名而非 IP？**
A:
1. 修改证书 SAN：`DNS:webhook.example.com`
2. 确保 DNS 解析正确
3. 修改 WebhookConfiguration 的 URL

## 与 using-byhand/by-service 的区别

| 特性 | by-service | by-baremetal |
|------|-----------|--------------|
| 网络依赖 | Kubernetes Service | 固定 IP 地址 |
| 部署方式 | Deployment + Service | 独立运行 |
| 证书要求 | 证书需匹配 Service DNS | 证书需匹配 IP |
| 复杂度 | 中等 | 简单 |
| 适用场景 | 标准集群环境 | 离线、特殊网络 |
| 学习价值 | 了解 Service 集成 | 了解底层原理 |

**学习建议**：
- 先学习 `using-byhand/by-service`（标准方式）
- 再学习 `using-byhand/by-baremetal`（了解原理）
- 最后学习 `using-kubebuilder`（生产级框架）

## 安全考虑

### 1. 证书管理

- CA 私钥必须保密
- 定期轮换证书（建议 90 天）
- 不要将私钥提交到 Git

### 2. 网络安全

- 使用防火墙限制访问
- 配置 NetworkPolicy（如果使用 Service）
- 使用 TLS 1.2+

### 3. 输入验证

- 验证所有输入参数
- 防止注入攻击
- 限制资源大小

## 参考资源

- [Kubernetes Admission Webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#admission-webhooks)
- [JSON Patch (RFC 6902)](https://tools.ietf.org/html/rfc6902)
- [TLS Certificate Requirements](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#tls-certificates)

---

**最后更新**: 2025-12-28
**维护者**: kubernetes-examples 项目团队
