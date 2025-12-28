package main

import (
	"fmt"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	configLoader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	rc, err := configLoader.ClientConfig()
	if err != nil {
		panic(err)
	}

	dc, err := discovery.NewDiscoveryClientForConfig(rc)
	if err != nil {
		panic(err)
	}

	lists, err := dc.ServerPreferredResources()
	if err != nil {
		fmt.Printf("Warning: %v\n", err)
	}

	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println("Kubernetes API 资源发现报告")
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println()

	// 统计信息
	totalResources := 0
	groups := make(map[string]int)
	verbs := make(map[string]int)

	// 遍历所有资源
	for _, list := range lists {
		if len(list.APIResources) == 0 {
			continue
		}

		groupVersion := list.GroupVersion
		groups[groupVersion] += len(list.APIResources)

		for _, res := range list.APIResources {
			totalResources++
			for _, verb := range res.Verbs {
				verbs[verb]++
			}
		}
	}

	// 打印统计信息
	fmt.Printf("📊 总计发现 %d 个 API 资源\n\n", totalResources)

	fmt.Println("🏗️  API 版本分布:")
	fmt.Println("-" + strings.Repeat("-", 79))
	for gv, count := range groups {
		fmt.Printf("  %-40s %d 个资源\n", gv, count)
	}
	fmt.Println()

	fmt.Println("🎯 操作类型统计:")
	fmt.Println("-" + strings.Repeat("-", 79))
	for verb, count := range verbs {
		fmt.Printf("  %-15s %d 个资源支持\n", verb, count)
	}
	fmt.Println()

	// 按类别展示常用资源
	fmt.Println("🔍 常用资源示例:")
	fmt.Println("-" + strings.Repeat("-", 79))

	for _, list := range lists {
		if len(list.APIResources) == 0 {
			continue
		}

		gv := list.GroupVersion
		for _, res := range list.APIResources {
			// 只展示一些常见资源
			if isCommonResource(res.Name) {
				fmt.Printf("  %-25s %-40s %v\n", res.Kind, gv, res.Verbs)
			}
		}
	}
	fmt.Println()

	// 检查特定资源
	fmt.Println("🔧 资源支持检查:")
	fmt.Println("-" + strings.Repeat("-", 79))
	checkResource(dc, "Pod", "v1")
	checkResource(dc, "Deployment", "apps/v1")
	checkResource(dc, "CronJob", "batch/v1")
	checkResource(dc, "Ingress", "networking.k8s.io/v1")

	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println("✅ 发现完成")
	fmt.Println("=" + strings.Repeat("=", 79))

	aggregate := errors.NewAggregate([]error{})
	if len(aggregate.Errors()) > 0 {
		os.Exit(1)
	}
}

func isCommonResource(name string) bool {
	commonResources := []string{
		"pods", "deployments", "services", "configmaps", "secrets",
		"namespaces", "nodes", "persistentvolumes", "persistentvolumeclaims",
		"statefulsets", "daemonsets", "jobs", "cronjobs", "ingresses",
		"replicasets", "events", "endpoints", "serviceaccounts",
	}
	for _, r := range commonResources {
		if name == r {
			return true
		}
	}
	return false
}

func checkResource(dc *discovery.DiscoveryClient, kind, groupVersion string) {
	resources, err := dc.ServerResourcesForGroupVersion(groupVersion)
	if err != nil {
		fmt.Printf("  ❌ %-20s %-25s (错误: %v)\n", kind, groupVersion, err)
		return
	}

	for _, res := range resources.APIResources {
		if res.Kind == kind {
			fmt.Printf("  ✅ %-20s %-25s 支持: %v\n", kind, groupVersion, res.Verbs)
			return
		}
	}
	fmt.Printf("  ⚠️  %-20s %-25s 未找到\n", kind, groupVersion)
}
