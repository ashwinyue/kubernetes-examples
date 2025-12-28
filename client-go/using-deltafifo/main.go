package main

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

func main() {
	// 创建一个DeltaFIFO对象（使用新 API）
	fifo := cache.NewDeltaFIFOWithOptions(cache.DeltaFIFOOptions{
		KeyFunction: cache.MetaNamespaceKeyFunc,
	})

	dep1 := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "dep1", Namespace: metav1.NamespaceDefault}}
	dep2 := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "dep2", Namespace: metav1.NamespaceDefault}}

	fmt.Println("========================================")
	fmt.Println("DeltaFIFO 示例")
	fmt.Println("========================================")
	fmt.Println()

	// 1. 将对象添加事件放入 DeltaFIFO 中
	fmt.Println("步骤 1: 添加 dep1 和 dep2")
	_ = fifo.Add(dep1)
	_ = fifo.Add(dep2)
	fmt.Printf("Keys 数量: %d\n", len(fifo.ListKeys()))
	fmt.Println()

	// 2. 将对象变更事件放入 DeltaFIFO 中
	fmt.Println("步骤 2: 更新 dep1 为 dep1-modified")
	dep1.Name = "dep1-modified"
	_ = fifo.Update(dep1)
	fmt.Printf("Keys 数量: %d\n", len(fifo.ListKeys()))
	fmt.Println()

	// 3. 以列表形式返回所有 Key
	fmt.Println("步骤 3: 列出所有 Keys")
	fmt.Printf("Keys: %v\n", fifo.ListKeys())
	fmt.Println()

	// 4. 将对象删除事件放入 DeltaFIFO 中
	fmt.Println("步骤 4: 删除 dep1")
	_ = fifo.Delete(dep1)
	fmt.Printf("Keys 数量: %d\n", len(fifo.ListKeys()))
	fmt.Println()

	// 5. 从 DeltaFIFO 中 Pop 所有资源对象
	fmt.Println("步骤 5: Pop 处理所有事件")
	fmt.Println("----------------------------------------")

	// 循环处理所有事件，直到队列为空
	for len(fifo.ListKeys()) > 0 {
		_, _ = fifo.Pop(func(obj any, isInInitialList bool) error {
			for _, delta := range obj.(cache.Deltas) {
				deploy := delta.Object.(*appsv1.Deployment)

				// 区分不同事件，执行不同回调
				switch delta.Type {
				case cache.Added:
					fmt.Printf("Added:    %s/%s\n", deploy.Namespace, deploy.Name)
				case cache.Updated:
					fmt.Printf("Updated:  %s/%s\n", deploy.Namespace, deploy.Name)
				case cache.Deleted:
					fmt.Printf("Deleted:  %s/%s\n", deploy.Namespace, deploy.Name)
				}
			}

			return nil
		})
	}

	fmt.Println("----------------------------------------")
	fmt.Printf("Keys 数量: %d\n", len(fifo.ListKeys()))
	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("✅ DeltaFIFO 示例完成")
	fmt.Println("========================================")
}
