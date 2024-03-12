package inplaceupdate

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	deploymentutil "k8s.io/kubernetes/pkg/controller/deployment/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/Forget-C/demo/inplaceupdate/program/api/v1"
	"github.com/Forget-C/demo/inplaceupdate/program/internal/util"
)

const defaultRequeueAfter = time.Second * 30

type RealDeploymentControl struct {
	Client           client.Client
	reconcileFunc    func(ctx context.Context, inplaceUpdate types.NamespacedName, deployment types.NamespacedName) (ctrl.Result, error)
	statusUpdater    StatusUpdater
	patchPodFunc     func(obj *corev1.Pod, latestStatus map[string]*corev1.ContainerStatus, updateSpc *UpdateSpce) (*corev1.Pod, error)
	patchProcessFunc func(obj *v1.InplaceUpdate, finishedPods, failedPods []*corev1.Pod) (*v1.InplaceUpdate, error)
	podUpdater       PodUpdater
}

func NewRealDeploymentControl(client client.Client) *RealDeploymentControl {
	controller := &RealDeploymentControl{
		Client:           client,
		statusUpdater:    newStatusUpdater(client),
		patchPodFunc:     DefaultPatchPodFunc,
		podUpdater:       newPodUpdater(client),
		patchProcessFunc: DefaultPatchProcessFunc,
	}
	controller.reconcileFunc = controller.doReconcile
	return controller
}

func (r *RealDeploymentControl) Reconcile(ctx context.Context, inplaceUpdate types.NamespacedName, deployment types.NamespacedName) (ctrl.Result, error) {
	return r.reconcileFunc(ctx, inplaceUpdate, deployment)
}

func (r *RealDeploymentControl) doReconcile(ctx context.Context, inplaceUpdate types.NamespacedName, deployment types.NamespacedName) (ctrl.Result, error) {
	i := &v1.InplaceUpdate{}
	if err := r.Client.Get(ctx, inplaceUpdate, i); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	if IsCompleted(i) {
		return ctrl.Result{}, nil
	}
	if i.Spec.Delay != nil && *i.Spec.Delay > 0 && i.Status.StartTime == nil {
		return ctrl.Result{RequeueAfter: time.Duration(*i.Spec.Delay) * time.Second}, nil
	}
	d := &appsv1.Deployment{}
	newStatus := &v1.InplaceUpdateStatus{
		StartTime: &metav1.Time{},
	}
	if err := r.Client.Get(ctx, deployment, d); err != nil {
		if apierrors.IsNotFound(err) {
			newStatus.Phase = v1.InplaceUpdatePhaseFailed
			newStatus.CompletionTime = &metav1.Time{}
			newStatus.Conditions = append(newStatus.Conditions, v1.InplaceUpdateCondition{
				Type:    v1.InplaceUpdateConditionFailedOwner,
				Status:  corev1.ConditionTrue,
				Reason:  "NotFound",
				Message: err.Error(),
			})
			return ctrl.Result{}, r.statusUpdater.Update(i, newStatus)
		}
		return ctrl.Result{}, err
	}
	if abort := r.preCheck(d, newStatus); abort {
		return ctrl.Result{RequeueAfter: defaultRequeueAfter}, r.statusUpdater.Update(i, newStatus)
	}
	var errorList []error
	for _, target := range i.Spec.Containers {
		container := util.FindContainer(target.Name, d.Spec.Template.Spec)
		if container == nil {
			errorList = append(errorList, fmt.Errorf("container %s not found in deployment %s/%s", target.Name, d.Namespace, d.Name))
			continue
		}
	}
	if len(errorList) != 0 {
		newStatus.Phase = v1.InplaceUpdatePhaseFailed
		newStatus.CompletionTime = &metav1.Time{}
		newStatus.Conditions = append(newStatus.Conditions, v1.InplaceUpdateCondition{
			Reason:  "NotFound",
			Message: utilerrors.NewAggregate(errorList).Error(),
			Type:    v1.InplaceUpdateConditionFailedPods,
			Status:  corev1.ConditionTrue,
		})
		return ctrl.Result{}, r.statusUpdater.Update(i, newStatus)
	}
	newPods, err := r.ownerRefPatchedPods(i, d, newStatus)
	if err != nil {
		newStatus.Phase = v1.InplaceUpdatePhaseFailed
		newStatus.CompletionTime = &metav1.Time{}
		newStatus.Conditions = append(newStatus.Conditions, v1.InplaceUpdateCondition{
			Type:    v1.InplaceUpdateConditionFailedPods,
			Status:  corev1.ConditionTrue,
			Reason:  "Failed",
			Message: err.Error(),
		})
		return ctrl.Result{}, r.statusUpdater.Update(i, newStatus)
	}
	syncErr := r.sync(i, newPods, newStatus)
	if newStatus.UpdatedReplicas == d.Status.Replicas {
		newStatus.Phase = v1.InplaceUpdatePhaseFinished
		newStatus.CompletionTime = &metav1.Time{}
	} else {
		newStatus.Phase = v1.InplaceUpdatePhaseRunning
	}
	err = r.statusUpdater.Update(i, newStatus)
	if err != nil {
		return ctrl.Result{}, err
	}
	if syncErr != nil {
		return ctrl.Result{RequeueAfter: defaultRequeueAfter}, nil
	}
	return ctrl.Result{}, nil
}

func (r *RealDeploymentControl) preCheck(d *appsv1.Deployment, status *v1.InplaceUpdateStatus) (abort bool) {
	status.Phase = v1.InplaceUpdatePhasePending
	condition := v1.InplaceUpdateCondition{
		Type:   v1.InplaceUpdateConditionFailedOwner,
		Status: corev1.ConditionTrue,
		Reason: "OwnerUnavailable",
	}
	if d.DeletionTimestamp != nil {
		condition.Message = "deployment is being deleted"
		status.Conditions = append(status.Conditions, condition)
		return true
	}
	if !deploymentutil.DeploymentComplete(d, &d.Status) {
		condition.Message = "deployment is not complete"
		status.Conditions = append(status.Conditions, condition)
		return true
	}
	return false
}

func (r *RealDeploymentControl) sync(i *v1.InplaceUpdate, pods []*corev1.Pod, status *v1.InplaceUpdateStatus) error {
	finishedPods, failedPods, err := r.podUpdater.Update(pods)
	if err != nil {
		return err
	}
	containerNames := ContainerNames(i.Spec)
	for _, pod := range finishedPods {
		finishedContainers := util.FindContainers(containerNames, pod.Spec)
		status.UpdatedContainerNumber += int32(len(finishedContainers))
	}

	newObj, err := r.patchProcessFunc(i, finishedPods, failedPods)
	if err != nil {
		return err
	}
	if err := r.Client.Patch(context.Background(), newObj, client.MergeFrom(i)); err != nil {
		return err
	}
	status.UpdatedReplicas += int32(len(finishedPods))
	return nil

}

func (r *RealDeploymentControl) ownerRefPatchedPods(i *v1.InplaceUpdate, d *appsv1.Deployment, status *v1.InplaceUpdateStatus) ([]*corev1.Pod, error) {
	accusedReplicaSets, err := r.getReplicaSetsForDeployment(d)
	if err != nil {
		return nil, err
	}
	curRs := deploymentutil.FindNewReplicaSet(d, accusedReplicaSets)
	if curRs == nil {
		return nil, fmt.Errorf("deployment %s/%s has no new replicaset", d.Namespace, d.Name)
	}
	if *curRs.Spec.Replicas != *d.Spec.Replicas {
		return nil, fmt.Errorf("deployment %s/%s has updated replicas, expect %d, got %d", d.Namespace, d.Name, *d.Spec.Replicas, *curRs.Spec.Replicas)
	}
	if !d.Spec.Paused {
		deployPaused := d.DeepCopy()
		deployPaused.Spec.Paused = true
		err = r.Client.Patch(context.Background(), d, client.MergeFrom(deployPaused))
		if err != nil {
			return nil, err
		}
		defer func() {
			resumed := deployPaused.DeepCopy()
			resumed.Spec.Paused = false
			if err := r.Client.Patch(context.Background(), resumed, client.MergeFrom(deployPaused)); err != nil {
				log.Log.Error(err, "failed to resume deployment", "deployment", resumed)
			}
		}()
	}
	accusedPods, err := r.getPodsForReplicaSet(curRs)
	if err != nil {
		return nil, err
	}
	if len(accusedPods) != int(curRs.Status.Replicas) {
		return nil, fmt.Errorf("replicaset %s/%s has %d pods, expect %d", curRs.Namespace, curRs.Name, len(accusedPods), curRs.Status.Replicas)
	}
	newPods := make([]*corev1.Pod, 0, len(accusedPods))
	containerNames := ContainerNames(i.Spec)
	var errorList []error
	for _, pod := range accusedPods {
		latestStatus := util.GetLatestContainerStatusMap(pod.Status.ContainerStatuses, containerNames...)
		accusedContainers := util.FindContainers(containerNames, pod.Spec)
		status.ContainerNumber += int32(len(accusedContainers))
		newPod, err := r.patchPodFunc(pod, latestStatus, &UpdateSpce{Args: i.Spec.Containers, Containers: accusedContainers})
		if err != nil {
			errorList = append(errorList, err)
			status.UnavailableReplicas++
			continue
		}
		newPods = append(newPods, newPod)
	}
	status.Replicas = int32(len(newPods))
	if len(errorList) != 0 && i.Spec.FailurePolicy == v1.FailurePolicyAbort {
		return nil, utilerrors.NewAggregate(errorList)
	}
	return newPods, nil
}

func (r *RealDeploymentControl) getReplicaSetsForDeployment(deploy *appsv1.Deployment) ([]*appsv1.ReplicaSet, error) {
	deploySelector, err := metav1.LabelSelectorAsSelector(deploy.Spec.Selector)
	if err != nil {
		return nil, fmt.Errorf("deployment %s/%s has invalid selector: %v", deploy.Namespace, deploy.Name, err)
	}
	rsList := appsv1.ReplicaSetList{}
	if err := r.Client.List(context.TODO(), &rsList, &client.ListOptions{LabelSelector: deploySelector}); err != nil {
		return nil, err
	}
	var accused []*appsv1.ReplicaSet
	for _, rs := range rsList.Items {
		controllerRef := metav1.GetControllerOf(&rs)
		if controllerRef != nil && controllerRef.UID == deploy.UID && rs.DeletionTimestamp == nil {
			accused = append(accused, &rs)
		}
	}
	return accused, nil
}

func (r *RealDeploymentControl) getPodsForReplicaSet(rs *appsv1.ReplicaSet) ([]*corev1.Pod, error) {
	selector, err := metav1.LabelSelectorAsSelector(rs.Spec.Selector)
	if err != nil {
		return nil, err
	}
	podList := corev1.PodList{}
	if err := r.Client.List(context.TODO(), &podList, &client.ListOptions{LabelSelector: selector}); err != nil {
		return nil, err
	}
	var accused []*corev1.Pod
	for _, pod := range podList.Items {
		controllerRef := metav1.GetControllerOf(&pod)
		if controllerRef != nil && controllerRef.UID == rs.UID && pod.DeletionTimestamp == nil {
			accused = append(accused, &pod)
		}
	}
	return accused, nil
}
