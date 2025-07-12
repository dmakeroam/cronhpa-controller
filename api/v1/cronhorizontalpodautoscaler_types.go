package v1

import (
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CronHorizontalPodAutoscalerSpec defines the desired state of CronHorizontalPodAutoscaler
type CronHorizontalPodAutoscalerSpec struct {
	ScaleTargetRef autoscalingv2.CrossVersionObjectReference `json:"scaleTargetRef"`
	Jobs           []Job                                     `json:"jobs"`
}

type Job struct {
	Name        string `json:"name"`
	Schedule    string `json:"schedule"`
	Timezone    string `json:"timezone,omitempty"`
	MinReplicas int32  `json:"minReplicas"`
	RunOnce     bool   `json:"runOnce"`
}

// CronHorizontalPodAutoscalerStatus defines the observed state of CronHorizontalPodAutoscaler
type CronHorizontalPodAutoscalerStatus struct {
	// Insert additional status fields here if needed (e.g., last scale time).
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// CronHorizontalPodAutoscaler is the Schema for the cronhorizontalpodautoscalers API
type CronHorizontalPodAutoscaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CronHorizontalPodAutoscalerSpec   `json:"spec,omitempty"`
	Status CronHorizontalPodAutoscalerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CronHorizontalPodAutoscalerList contains a list of CronHorizontalPodAutoscaler
type CronHorizontalPodAutoscalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CronHorizontalPodAutoscaler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CronHorizontalPodAutoscaler{}, &CronHorizontalPodAutoscalerList{})
}
