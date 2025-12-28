package main

import (
	"fmt"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/fields"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// 加载配置
	config, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	fmt.Println("=" + "========================================")
	fmt.Println("Reflector 简单测试")
	fmt.Println("=" + "========================================")
	fmt.Println()

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
	fmt.Println("等待 Reflector 同步...")
	time.Sleep(3 * time.Second)

	// 检查缓存
	keys := store.ListKeys()
	fmt.Printf("\n✅ Reflector 已启动\n")
	fmt.Printf("📊 缓存的 Pod 数量: %d\n", len(keys))
	fmt.Printf("\n缓存的 Pod 列表:\n")
	for i, key := range keys {
		if i >= 10 {
			fmt.Printf("... (还有 %d 个)\n", len(keys)-10)
			break
		}
		fmt.Printf("  %d. %s\n", i+1, key)
	}

	// 运行 10 秒后停止
	fmt.Println("\n⏱️  运行 10 秒...")
	time.Sleep(10 * time.Second)

	close(stopCh)
	time.Sleep(1 * time.Second)

	// 再次检查缓存
	keys = store.ListKeys()
	fmt.Printf("\n📊 10 秒后缓存的 Pod 数量: %d\n", len(keys))

	fmt.Println()
	fmt.Println("=" + "========================================")
	fmt.Println("✅ Reflector 测试完成")
	fmt.Println("=" + "========================================")
}
