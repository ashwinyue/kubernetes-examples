package main

import (
	"context"
	"fmt"
	"time"

	appsv1 "github.com/ashwinyue/kubernetes-examples/finalizer-example/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const finalizerName = "simpleapp.example.com/finalizer"

// SimpleAppReconciler 演示 Finalizer 的使用
type SimpleAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=example.com,resources=simpleapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=example.com,resources=simpleapps/finalizers,verbs=update
// +kubebuilder:rbac:groups=example.com,resources=simpleapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete

func (r *SimpleAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// 1. 获取 SimpleApp 资源
	app := &appsv1.SimpleApp{}
	if err := r.Get(ctx, req.NamespacedName, app); err != nil {
		if errors.IsNotFound(err) {
			log.Info("SimpleApp resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get SimpleApp")
		return ctrl.Result{}, err
	}

	// 2. 处理删除（Finalizer 逻辑）
	if !app.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, app)
	}

	// 3. 添加 Finalizer
	// 确保资源有 Finalizer，这样删除时不会立即被删除
	if !controllerutil.ContainsFinalizer(app, finalizerName) {
		controllerutil.AddFinalizer(app, finalizerName)
		if err := r.Update(ctx, app); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		log.Info("Added finalizer")
		return ctrl.Result{Requeue: true}, nil
	}

	// 4. 正常 Reconcile 逻辑
	// 确保存在对应数量的 Pod
	if err := r.reconcilePods(ctx, app); err != nil {
		log.Error(err, "Failed to reconcile pods")
		return ctrl.Result{}, err
	}

	// 5. 更新 Status
	app.Status.Ready = true
	app.Status.ObservedGeneration = app.Generation

	if err := r.Status().Update(ctx, app); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	// 6. 定期重新入队
	return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
}

// handleDeletion 处理资源删除，执行清理逻辑
func (r *SimpleAppReconciler) handleDeletion(ctx context.Context, app *appsv1.SimpleApp) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// 检查是否有我们的 Finalizer
	if !controllerutil.ContainsFinalizer(app, finalizerName) {
		// 没有 Finalizer，直接返回，让 Kubernetes 删除资源
		return ctrl.Result{}, nil
	}

	log.Info("Processing deletion", "app", app.Name)

	// 1. 查找关联的 Pod
	podList := &corev1.PodList{}
	if err := r.List(ctx, podList,
		client.InNamespace(app.Namespace),
		client.MatchingLabels(map[string]string{
			"app":        app.Name,
			"managed-by": "simpleapp-controller",
		}),
	); err != nil {
		log.Error(err, "Failed to list pods")
		return ctrl.Result{}, err
	}

	// 2. 删除所有 Pod
	for _, pod := range podList.Items {
		if err := r.Delete(ctx, &pod); err != nil {
			if !errors.IsNotFound(err) {
				log.Error(err, "Failed to delete pod", "pod", pod.Name)
				return ctrl.Result{}, err
			}
		}
		log.Info("Deleted pod", "pod", pod.Name)
	}

	// 3. 等待所有 Pod 被删除
	if len(podList.Items) > 0 {
		log.Info("Waiting for pods to be deleted", "count", len(podList.Items))
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	// 4. 清理外部资源（示例）
	if err := r.cleanupExternalResources(ctx, app); err != nil {
		log.Error(err, "Failed to cleanup external resources")
		return ctrl.Result{}, err
	}

	// 5. 移除 Finalizer
	controllerutil.RemoveFinalizer(app, finalizerName)
	if err := r.Update(ctx, app); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	log.Info("Finalizer processed, resource will be deleted")
	return ctrl.Result{}, nil
}

// reconcilePods 确保存在正确数量的 Pod
func (r *SimpleAppReconciler) reconcilePods(ctx context.Context, app *appsv1.SimpleApp) error {
	// 列出当前 Pod
	podList := &corev1.PodList{}
	if err := r.List(ctx, podList,
		client.InNamespace(app.Namespace),
		client.MatchingLabels(map[string]string{
			"app":        app.Name,
			"managed-by": "simpleapp-controller",
		}),
	); err != nil {
		return err
	}

	desiredReplicas := int32(1)
	currentReplicas := int32(len(podList.Items))

	// 调整 Pod 数量
	if currentReplicas < desiredReplicas {
		// 创建 Pod
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-pod", app.Name),
				Namespace: app.Namespace,
				Labels: map[string]string{
					"app":        app.Name,
					"managed-by": "simpleapp-controller",
				},
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(app, appsv1.GroupVersion.WithKind("SimpleApp")),
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "app",
						Image: app.Spec.Image,
					},
				},
			},
		}
		if err := r.Create(ctx, pod); err != nil {
			return err
		}
	}

	return nil
}

// cleanupExternalResources 清理外部资源（示例）
func (r *SimpleAppReconciler) cleanupExternalResources(ctx context.Context, app *appsv1.SimpleApp) error {
	// 这里可以清理外部资源，例如：
	// - 删除云服务（负载均衡器、存储卷等）
	// - 关闭数据库连接
	// - 释放 IP 地址
	// - 删除 DNS 记录

	// 示例：记录日志
	log.FromContext(ctx).Info("Cleaning up external resources", "app", app.Name)

	return nil
}
