package inplaceupdate

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/Forget-C/demo/inplaceupdate/program/api/v1"
)

const (
	podReAddDelay = 5 * time.Second
	podRetryLimit = 3
)

func newPodUpdater(c client.Client) PodUpdater {
	return &podUpdater{Client: c}
}

type podUpdater struct {
	Client client.Client
}

func (p *podUpdater) Update(pods []*corev1.Pod) (finishedPods []*corev1.Pod, failedPods []*corev1.Pod, err error) {
	q := workqueue.NewRateLimitingQueue(&podSyncQueueRateLimit{})
	podTotal := int32(len(pods))
	finishedTotal := atomic.Int32{}
	finished := make(chan struct{})
	for _, pod := range pods {
		q.Add(pod)
	}
	complete := func(pod *corev1.Pod) {
		q.Forget(pod)
		q.Done(pod)
		finishedTotal.Add(1)
	}
	go func() {
		for {
			obj, shutdown := q.Get()
			if shutdown {
				break
			}
			pod := obj.(*corev1.Pod)
			if err := p.refreshPod(pod); err != nil {
				if q.NumRequeues(pod) >= podRetryLimit {
					complete(pod)
					failedPods = append(failedPods, pod)
				} else {
					q.AddRateLimited(pod)
				}
			} else {
				complete(pod)
				finishedPods = append(finishedPods, pod)
			}
			if finishedTotal.Load() == podTotal {
				close(finished)
				q.ShutDown()
			}
		}
	}()
	<-finished
	return
}

func (p *podUpdater) refreshPod(pod *corev1.Pod) error {
	return p.Client.Update(context.TODO(), pod)
}

type podSyncQueueRateLimit struct {
	failuresLock sync.Mutex
	failures     map[interface{}]int
}

// When anyway returns the requeue delay
func (p *podSyncQueueRateLimit) When(item interface{}) time.Duration {
	p.failuresLock.Unlock()
	defer p.failuresLock.Unlock()
	p.failures[item] = p.failures[item] + 1
	return podReAddDelay
}

func (p *podSyncQueueRateLimit) Forget(item interface{}) {
	p.failuresLock.Lock()
	defer p.failuresLock.Unlock()

	delete(p.failures, item)
}

func (p *podSyncQueueRateLimit) NumRequeues(item interface{}) int {
	p.failuresLock.Lock()
	defer p.failuresLock.Unlock()

	return p.failures[item]
}

func newStatusUpdater(c client.Client) StatusUpdater {
	return &statusUpdater{Client: c}
}

type statusUpdater struct {
	Client client.Client
}

func (s *statusUpdater) Update(obj *v1.InplaceUpdate, status *v1.InplaceUpdateStatus) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		clone := &v1.InplaceUpdate{}
		err := s.Client.Get(context.Background(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, clone)
		if err != nil {
			return err
		}
		clone.Status = *status
		return s.Client.Status().Update(context.Background(), clone)
	})
}
