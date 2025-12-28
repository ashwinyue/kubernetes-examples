package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	defaultKubeconfig := os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
	if defaultKubeconfig == "" {
		defaultKubeconfig = clientcmd.RecommendedHomeFile
	}

	kubeconfig := flag.String(clientcmd.RecommendedConfigPathFlag, defaultKubeconfig,
		"Absolute path to the kubeconfig file")
	namespace := flag.String("namespace", metav1.NamespaceAll, "Namespace to watch")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// 创建 ListWatch
	lw := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		"pods",
		*namespace,
		fields.Everything(),
	)

	// 创建 Store
	store := cache.NewStore(cache.MetaNamespaceKeyFunc)

	// 创建 Reflector，30 秒 Resync
	reflector := cache.NewReflector(lw, &corev1.Pod{}, store, 30*time.Second)

	fmt.Println("=" + "========================================")
	fmt.Println("Reflector 高级示例")
	fmt.Println("=" + "========================================")
	fmt.Printf("监听命名空间: %s\n", *namespace)
	fmt.Printf("Resync 周期: %d 秒\n", 30)
	fmt.Println()

	// 设置信号处理
	stopCh := make(chan struct{})
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n收到终止信号，停止 Reflector...")
		close(stopCh)
	}()

	// 启动 Reflector
	go reflector.Run(stopCh)

	// 等待 Reflector 同步
	fmt.Println("等待 Reflector 同步...")
	time.Sleep(2 * time.Second)

	// 统计协程
	var stats Stats
	stats.Store = store
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				stats.Print()
			}
		}
	}()

	// 事件处理协程
	go func() {
		for {
			select {
			case <-stopCh:
				return
			default:
				pods := store.List()
				if len(pods) > 0 {
					pod := pods[0].(*corev1.Pod)
					stats.UpdateLastSeen(pod.Namespace, pod.Name)
				}
				time.Sleep(2 * time.Second)
			}
		}
	}()

	// 等待终止信号
	<-sigCh
	time.Sleep(1 * time.Second)
	stats.Print()
	fmt.Println("=" + "========================================")
	fmt.Println("Reflector 已停止")
	fmt.Println("=" + "========================================")
}

// Stats 统计结构
type Stats struct {
	mu        sync.RWMutex
	store     cache.Store
	lastSeen  map[string]string // namespace/name -> timestamp
	podCount  map[string]int    // namespace -> count
}

func (s *Stats) UpdateLastSeen(namespace, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lastSeen == nil {
		s.lastSeen = make(map[string]string)
		s.podCount = make(map[string]int)
	}

	key := fmt.Sprintf("%s/%s", namespace, name)
	s.lastSeen[key] = time.Now().Format("15:04:05")
	s.podCount[namespace]++
}

func (s *Stats) Print() {
	s.mu.Lock()
	defer s.mu.Unlock()

	keys := s.store.ListKeys()
	fmt.Printf("\n[%s] 统计信息:\n", time.Now().Format("15:04:05"))
	fmt.Println("----------------------------------------")
	fmt.Printf("总 Pod 数量: %d\n", len(keys))
	fmt.Println("\n按命名空间分布:")
	for ns, count := range s.podCount {
		fmt.Printf("  %s: %d 个 Pod\n", ns, count)
	}
	fmt.Println("\n最近的 5 个 Pod:")
	for i := 0; i < 5 && i < len(keys); i++ {
		fmt.Printf("  %s\n", keys[i])
		if lastSeen, ok := s.lastSeen[keys[i]]; ok {
			fmt.Printf("    最后更新: %s\n", lastSeen)
		}
	}
	fmt.Println("----------------------------------------")
}
