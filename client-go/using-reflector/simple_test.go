package main

import (
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/fields"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

// TestReflectorSimple 测试 Reflector 的基本功能
func TestReflectorSimple(t *testing.T) {
	// 加载配置
	config, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		t.Skipf("无法加载 kubeconfig: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("创建 clientset 失败: %v", err)
	}

	t.Log("开始测试 Reflector 基本功能")

	// 创建 ListWatch
	lw := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		"pods",
		metav1.NamespaceAll,
		fields.Everything(),
	)

	// 创建 Store
	store := cache.NewStore(cache.MetaNamespaceKeyFunc)

	// 创建 Reflector，10 秒 Resync
	reflector := cache.NewReflector(lw, &metav1.Pod{}, store, 10*time.Second)

	// 启动 Reflector
	stopCh := make(chan struct{})
	go reflector.Run(stopCh)

	// 等待同步
	t.Log("等待 Reflector 同步...")
	time.Sleep(3 * time.Second)

	// 检查缓存
	keys := store.ListKeys()
	t.Logf("Reflector 已启动，缓存的 Pod 数量: %d", len(keys))

	// 验证至少缓存了一些 Pod
	if len(keys) == 0 {
		t.Error("期望缓存至少一个 Pod，但缓存为空")
	}

	// 验证 Key 格式
	for _, key := range keys {
		if key == "" {
			t.Error("缓存的 Key 不应为空")
		}
	}

	// 运行一段时间后停止
	time.Sleep(2 * time.Second)

	close(stopCh)
	time.Sleep(1 * time.Second)

	t.Log("Reflector 测试完成")
}
