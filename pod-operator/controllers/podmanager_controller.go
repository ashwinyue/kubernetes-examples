package controllers

import (
	"context"
	"fmt"
	"time"

	appsv1 "github.com/ashwinyue/kubernetes-examples/pod-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const finalizerName = "podmanager.mycompany.com/finalizer"

// PodManagerReconciler reconciles a PodManager object
type PodManagerReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=apps.mycompany.com,resources=podmanagers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.mycompany.com,resources=podmanagers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.mycompany.com,resources=podmanagers/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;patch

// Reconcile is the main reconciliation loop
func (r *PodManagerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// 1. Get the PodManager
	podManager := &appsv1.PodManager{}
	if err := r.Get(ctx, req.NamespacedName, podManager); err != nil {
		if errors.IsNotFound(err) {
			log.Info("PodManager resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get PodManager")
		return ctrl.Result{}, err
	}

	// 2. Handle Finalizer (deletion)
	if !podManager.DeletionTimestamp.IsZero() {
		return r.handleFinalizer(ctx, podManager)
	}

	// 3. Add Finalizer
	if !controllerutil.ContainsFinalizer(podManager, finalizerName) {
		controllerutil.AddFinalizer(podManager, finalizerName)
		if err := r.Update(ctx, podManager); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("Added finalizer")
		return ctrl.Result{Requeue: true}, nil
	}

	// 4. List Pods owned by this PodManager
	podList := &corev1.PodList{}
	if err := r.List(ctx, podList,
		client.InNamespace(req.Namespace),
		client.MatchingLabels(ownerLabels(podManager)),
	); err != nil {
		log.Error(err, "Failed to list Pods")
		return ctrl.Result{}, err
	}

	// 5. Adjust Pod count
	desiredReplicas := podManager.Spec.Replicas
	currentReplicas := int32(len(podList.Items))

	if currentReplicas < desiredReplicas {
		// Create Pods
		for i := currentReplicas; i < desiredReplicas; i++ {
			pod := newPodForPodManager(podManager, i)
			if err := r.Create(ctx, pod); err != nil {
				log.Error(err, "Failed to create Pod", "pod", pod.Name)
				r.Recorder.Eventf(podManager, corev1.EventTypeWarning, "Failed", "Failed to create pod %s: %v", pod.Name, err)
				return ctrl.Result{}, err
			}
			r.Recorder.Eventf(podManager, corev1.EventTypeNormal, "Created", "Created pod %s", pod.Name)
		}
	} else if currentReplicas > desiredReplicas {
		// Delete extra Pods
		for i := desiredReplicas; i < currentReplicas; i++ {
			if err := r.Delete(ctx, &podList.Items[i]); err != nil {
				log.Error(err, "Failed to delete Pod", "pod", podList.Items[i].Name)
				r.Recorder.Eventf(podManager, corev1.EventTypeWarning, "Failed", "Failed to delete pod %s: %v", podList.Items[i].Name, err)
				return ctrl.Result{}, err
			}
			r.Recorder.Eventf(podManager, corev1.EventTypeNormal, "Deleted", "Deleted pod %s", podList.Items[i].Name)
		}
	}

	// 6. Update Status
	readyCount := int32(0)
	for _, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodRunning {
			// Check if Pod is ready
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
					readyCount++
					break
				}
			}
		}
	}

	podManager.Status.ReadyReplicas = readyCount
	podManager.Status.CurrentReplicas = currentReplicas

	// Update condition
	if readyCount == desiredReplicas && desiredReplicas > 0 {
		podManager.Status.SetCondition(appsv1.PodCondition{
			Type:               "Ready",
			Status:             "True",
			LastTransitionTime: podManager.Status.Conditions[0].LastTransitionTime,
			Message:            "All Pods are ready",
		})
	} else {
		podManager.Status.SetCondition(appsv1.PodCondition{
			Type:               "Ready",
			Status:             "False",
			LastTransitionTime: podManager.Status.Conditions[0].LastTransitionTime,
			Message:            fmt.Sprintf("%d/%d Pods are ready", readyCount, desiredReplicas),
		})
	}

	if err := r.Status().Update(ctx, podManager); err != nil {
		log.Error(err, "Failed to update PodManager status")
		return ctrl.Result{}, err
	}

	// 7. Requeue for next reconciliation
	return ctrl.Result{RequeueAfter: time.Second * 30}, nil
}

// handleFinalizer handles the finalizer when the PodManager is being deleted
func (r *PodManagerReconciler) handleFinalizer(ctx context.Context, podManager *appsv1.PodManager) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(podManager, finalizerName) {
		// List all Pods owned by this PodManager
		podList := &corev1.PodList{}
		if err := r.List(ctx, podList,
			client.InNamespace(podManager.Namespace),
			client.MatchingLabels(ownerLabels(podManager)),
		); err != nil {
			log.Error(err, "Failed to list Pods for cleanup")
			return ctrl.Result{}, err
		}

		// Delete all owned Pods
		for _, pod := range podList.Items {
			if err := r.Delete(ctx, &pod); err != nil {
				if !errors.IsNotFound(err) {
					log.Error(err, "Failed to delete Pod", "pod", pod.Name)
					return ctrl.Result{}, err
				}
			} else {
				r.Recorder.Eventf(podManager, corev1.EventTypeNormal, "Deleting", "Deleting pod %s", pod.Name)
			}
		}

		// Remove finalizer
		controllerutil.RemoveFinalizer(podManager, finalizerName)
		if err := r.Update(ctx, podManager); err != nil {
			log.Error(err, "Failed to remove finalizer")
			return ctrl.Result{}, err
		}

		log.Info("Finalizer processed, cleaned up resources")
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *PodManagerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("podmanager-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.PodManager{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}

// newPodForPodManager returns a new Pod for a PodManager
func newPodForPodManager(podManager *appsv1.PodManager, index int32) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", podManager.Name, index),
			Namespace: podManager.Namespace,
			Labels:    ownerLabels(podManager),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(podManager, appsv1.GroupVersion.WithKind("PodManager")),
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: podManager.Spec.Image,
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 80,
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyAlways,
		},
	}
}

// ownerLabels returns the labels used to identify Pods owned by a PodManager
func ownerLabels(podManager *appsv1.PodManager) map[string]string {
	return map[string]string{
		"app":            "my-app",
		"podmanager":     podManager.Name,
		"podmanager-uid": string(podManager.UID),
	}
}
