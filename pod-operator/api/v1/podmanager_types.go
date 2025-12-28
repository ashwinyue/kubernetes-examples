package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodCondition describes the state of a PodManager at a certain point.
type PodCondition struct {
	// Type of condition.
	Type string `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status string `json:"status"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	// Human-readable message indicating details about last transition.
	Message string `json:"message"`
}

// PodManagerSpec defines the desired state of PodManager
type PodManagerSpec struct {
	// Replicas is the desired number of Pod replicas.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=3
	Replicas int32 `json:"replicas"`

	// Image is the container image to use for Pods.
	Image string `json:"image"`
}

// PodManagerStatus defines the observed state of PodManager
type PodManagerStatus struct {
	// ReadyReplicas is the number of Pods that are ready.
	ReadyReplicas int32 `json:"readyReplicas"`

	// CurrentReplicas is the total number of Pods.
	CurrentReplicas int32 `json:"currentReplicas"`

	// Conditions represent the latest available observations of PodManager's state.
	// +optional
	Conditions []PodCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=pm;pmgr

// PodManager is the Schema for the podmanagers API
type PodManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PodManagerSpec   `json:"spec,omitempty"`
	Status PodManagerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PodManagerList contains a list of PodManager
type PodManagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PodManager `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PodManager{}, &PodManagerList{})
}

// SetCondition sets a condition on the PodManager status.
func (m *PodManagerStatus) SetCondition(cond PodCondition) {
	for i, c := range m.Conditions {
		if c.Type == cond.Type {
			if c.Status != cond.Status {
				m.Conditions[i] = cond
			}
			return
		}
	}
	m.Conditions = append(m.Conditions, cond)
}

// RemoveCondition removes a condition from the PodManager status.
func (m *PodManagerStatus) RemoveCondition(condType string) {
	for i, c := range m.Conditions {
		if c.Type == condType {
			m.Conditions = append(m.Conditions[:i], m.Conditions[i+1:]...)
			break
		}
	}
}
