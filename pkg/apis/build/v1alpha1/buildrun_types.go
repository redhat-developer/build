package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	LabelBuildRun           = "buildrun.build.dev/name"
	LabelBuildRunGeneration = "buildrun.build.dev/generation"
)

// BuildRunSpec defines the desired state of BuildRun
type BuildRunSpec struct {

	// BuildRef refers to the Build
	BuildRef *BuildRef `json:"buildRef"`

	// Compute Resources required by the build container
	// which can overwrite the configuration in Build.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// ServiceAccount refers to the kubernetes serviceaccount
	// which is used for resource control.
	// Default serviceaccount will be set if it is empty
	// +optional
	ServiceAccount *ServiceAccount `json:"serviceAccount,omitempty"`

	// Timeout defines the maximum run time of this build run.
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// Output refers to the location where the generated
	// image would be pushed to. It will overwrite the output image in build spec
	// +optional
	Output *Image `json:"output,omitempty"`
}

// BuildRunStatus defines the observed state of BuildRun
type BuildRunStatus struct {

	// The Succeeded status of the TaskRun
	// +optional
	Succeeded corev1.ConditionStatus `json:"succeeded,omitempty"`

	// The Succeeded reason of the TaskRun
	// +optional
	Reason string `json:"reason,omitempty"`

	// PodName is the name of the pod responsible for executing this task's steps.
	// +optional
	LatestTaskRunRef *string `json:"latestTaskRunRef,omitempty"`

	// StartTime is the time the build is actually started.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the time the build completed.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
}

// BuildRef can be used to refer to a specific instance of a Build.
type BuildRef struct {
	// Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names
	Name string `json:"name"`
	// API version of the referent
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`
}

// ServiceAccount can be used to refer to a specific ServiceAccount.
type ServiceAccount struct {
	// Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names
	// +optional
	Name *string `json:"name,omitempty"`
	// If generates a new ServiceAccount for the build
	// +optional
	Generate bool `json:"generate,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BuildRun is the Schema for the buildruns API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=buildruns,scope=Namespaced,shortName=br;brs
// +kubebuilder:printcolumn:name="Succeeded",type="string",JSONPath=".status.succeeded",description="The Succeeded status of the TaskRun"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.reason",description="The Succeeded reason of the TaskRun"
// +kubebuilder:printcolumn:name="StartTime",type="date",JSONPath=".status.startTime",description="The start time of this BuildRun"
// +kubebuilder:printcolumn:name="CompletionTime",type="date",JSONPath=".status.completionTime",description="The completion time of this BuildRun"
type BuildRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildRunSpec   `json:"spec,omitempty"`
	Status BuildRunStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BuildRunList contains a list of BuildRun
type BuildRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BuildRun `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BuildRun{}, &BuildRunList{})
}
