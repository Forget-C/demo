package inplaceupdate

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/Forget-C/demo/inplaceupdate/program/api/v1"
)

type PodUpdater interface {
	Update(pods []*corev1.Pod) (finishedPods []*corev1.Pod, failedPods []*corev1.Pod, err error)
}

type UpdateSpce struct {
	Args       []v1.InplaceUpdateArgs
	Containers map[string]*corev1.Container
}

type UpdateState struct {
	// Revision is the updated revision hash.
	Revision string `json:"revision"`

	// UpdateTimestamp is the start time when the in-place update happens.
	UpdateTimestamp metav1.Time `json:"updateTimestamp"`
	// LastContainerStatuses records the before-in-place-update container statuses. It is a map from ContainerName
	LastContainerStatuses map[string]*corev1.ContainerStatus `json:"lastContainerStatuses"`
}

type StatusUpdater interface {
	Update(obj *v1.InplaceUpdate, status *v1.InplaceUpdateStatus) error
}
