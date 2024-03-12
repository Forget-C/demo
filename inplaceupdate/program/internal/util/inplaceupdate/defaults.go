package inplaceupdate

import (
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	deploymentutil "k8s.io/kubernetes/pkg/controller/deployment/util"

	v1 "github.com/Forget-C/demo/inplaceupdate/program/api/v1"
	"github.com/Forget-C/demo/inplaceupdate/program/internal/util"
)

const (
	AnnotationStateKey    = "demo.cyisme.top/inplaceupdate-state"
	AnnotationFinishedKey = "demo.cyisme.top/inplaceupdate-finished"
	AnnotationFailedKey   = "demo.cyisme.top/inplaceupdate-failed"
)

func DefaultPatchPodFunc(obj *corev1.Pod, latestStatus map[string]*corev1.ContainerStatus, updateSpc *UpdateSpce) (*corev1.Pod, error) {
	clone := obj.DeepCopy()
	var containers []corev1.Container
	for _, target := range updateSpc.Args {
		container, exist := updateSpc.Containers[target.Name]
		if !exist || container.Image == target.Image {
			continue
		}
		newContainer := container.DeepCopy()
		newContainer.Image = target.Image
		containers = append(containers, *newContainer)
	}
	clone.Spec.Containers = util.ContainerMerge(clone.Spec.Containers, containers)
	state := UpdateState{
		Revision:              clone.Annotations[deploymentutil.RevisionAnnotation],
		UpdateTimestamp:       metav1.Time{},
		LastContainerStatuses: latestStatus,
	}
	stateBytes, _ := json.Marshal(state)
	if clone.Annotations == nil {
		clone.Annotations = make(map[string]string)
	}
	clone.Annotations[AnnotationStateKey] = string(stateBytes)
	return clone, nil
}

func DefaultPatchProcessFunc(obj *v1.InplaceUpdate, finishedPods, failedPods []*corev1.Pod) (*v1.InplaceUpdate, error) {
	clone := obj.DeepCopy()
	if clone.Annotations == nil {
		clone.Annotations = make(map[string]string)
	}
	clone.Annotations[AnnotationFinishedKey] = util.PodNames(finishedPods)
	clone.Annotations[AnnotationFailedKey] = util.PodNames(failedPods)
	return clone, nil
}
