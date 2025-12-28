package main

import (
	"context"
	"fmt"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	// 加载配置
	home := homedir.HomeDir()
	kubeconfig := filepath.Join(home, ".kube", "config")

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err)
	}

	// 创建 Dynamic Client
	dc, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	fmt.Println("========================================")
	fmt.Println("Dynamic Client 高级示例")
	fmt.Println("========================================")
	fmt.Println()

	// 示例 1：列出 Pod
	fmt.Println("示例 1：列出 default 命名空间的 Pod")
	fmt.Println("----------------------------------------")
	listPods(dc, "default")
	fmt.Println()

	// 示例 2：创建 ConfigMap
	fmt.Println("示例 2：创建 ConfigMap")
	fmt.Println("----------------------------------------")
	createConfigMap(dc)
	fmt.Println()

	// 示例 3：更新 ConfigMap
	fmt.Println("示例 3：更新 ConfigMap")
	fmt.Println("----------------------------------------")
	updateConfigMap(dc)
	fmt.Println()

	// 示例 4：删除 ConfigMap
	fmt.Println("示例 4：删除 ConfigMap")
	fmt.Println("----------------------------------------")
	deleteConfigMap(dc)

	fmt.Println("========================================")
	fmt.Println("✅ 所有示例执行完成")
	fmt.Println("========================================")
}

func listPods(dc dynamic.Interface, namespace string) {
	gvr := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "pods",
	}

	list, err := dc.Resource(gvr).Namespace(namespace).
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}

	if len(list.Items) == 0 {
		fmt.Println("没有找到 Pod")
		return
	}

	for i, item := range list.Items {
		fmt.Printf("%d. %s\n", i+1, item.GetName())
	}
}

func createConfigMap(dc dynamic.Interface) {
	gvr := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "configmaps",
	}

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      "example-config",
				"namespace": "default",
			},
			"data": map[string]any{
				"app.name":    "dynamic-client",
				"app.version": "1.0.0",
			},
		},
	}

	result, err := dc.Resource(gvr).
		Namespace("default").
		Create(context.TODO(), obj, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("创建失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 创建 ConfigMap: %s\n", result.GetName())
	fmt.Printf("   数据: %v\n", result.Object["data"])
}

func updateConfigMap(dc dynamic.Interface) {
	gvr := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "configmaps",
	}

	// 获取 ConfigMap
	obj, err := dc.Resource(gvr).Namespace("default").
		Get(context.TODO(), "example-config", metav1.GetOptions{})
	if err != nil {
		fmt.Printf("获取失败: %v\n", err)
		return
	}

	fmt.Printf("更新前: %v\n", obj.Object["data"])

	// 更新数据
	data := obj.Object["data"].(map[string]any)
	data["app.version"] = "2.0.0"
	obj.Object["data"] = data

	// 更新
	result, err := dc.Resource(gvr).Namespace("default").
		Update(context.TODO(), obj, metav1.UpdateOptions{})
	if err != nil {
		fmt.Printf("更新失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 更新 ConfigMap: %s\n", result.GetName())
	fmt.Printf("   更新后: %v\n", result.Object["data"])
}

func deleteConfigMap(dc dynamic.Interface) {
	gvr := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "configmaps",
	}

	err := dc.Resource(gvr).Namespace("default").
		Delete(context.TODO(), "example-config", metav1.DeleteOptions{})
	if err != nil {
		fmt.Printf("删除失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 删除 ConfigMap: example-config\n")
}
