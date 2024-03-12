/*
Copyright 2024 extreme.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"time"

	errors2 "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/Forget-C/demo/inplaceupdate/program/api/v1"
	"github.com/Forget-C/demo/inplaceupdate/program/internal/util/inplaceupdate"
)

const defaultRequeueAfter = time.Second * 10

// InplaceUpdateReconciler reconciles a InplaceUpdate object
type InplaceUpdateReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=apps.demo.cyisme.top,resources=inplaceupdates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.demo.cyisme.top,resources=inplaceupdates/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.demo.cyisme.top,resources=inplaceupdates/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the InplaceUpdate object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *InplaceUpdateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	obj := &v1.InplaceUpdate{}
	err := r.Client.Get(ctx, req.NamespacedName, obj)
	if errors2.IsNotFound(err) {
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}
	switch obj.Spec.TargetReference.Kind {
	case "Deployment":
		reconcile := inplaceupdate.NewRealDeploymentControl(r.Client)
		return reconcile.Reconcile(ctx, req.NamespacedName, types.NamespacedName{Namespace: req.Namespace, Name: obj.Spec.TargetReference.Name})
	default:
		// never reach here
		return ctrl.Result{}, nil
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *InplaceUpdateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.InplaceUpdate{}).
		Complete(r)
}
