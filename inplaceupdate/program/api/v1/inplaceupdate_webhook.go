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

package v1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var inplaceupdatelog = logf.Log.WithName("inplaceupdate-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *InplaceUpdate) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-apps-demo-cyisme-top-v1-inplaceupdate,mutating=true,failurePolicy=fail,sideEffects=None,groups=apps.demo.cyisme.top,resources=inplaceupdates,verbs=create;update,versions=v1,name=minplaceupdate.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &InplaceUpdate{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *InplaceUpdate) Default() {
	inplaceupdatelog.Info("default", "name", r.Name)
	if r.Spec.ReclaimPolicy == "" {
		r.Spec.ReclaimPolicy = ReclaimPolicyRetain
	}
	if r.Spec.FailurePolicy == "" {
		r.Spec.FailurePolicy = FailurePolicyIgnore
	}
}

//+kubebuilder:webhook:path=/validate-apps-demo-cyisme-top-v1-inplaceupdate,mutating=false,failurePolicy=fail,sideEffects=None,groups=apps.demo.cyisme.top,resources=inplaceupdates,verbs=create;update,versions=v1,name=vinplaceupdate.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &InplaceUpdate{}

func checkTargetReference(target *TargetReference) error {
	if target == nil {
		return fmt.Errorf("targetReference is required")
	}
	if target.Name == "" {
		return fmt.Errorf("targetReference.name is required")
	}
	if target.APIVersion == "" || target.Kind == "" {
		return fmt.Errorf("targetReference.apiVersion and targetReference.kind are required")
	}
	if target.APIVersion != "v1" && target.Kind != "Deployment" {
		return fmt.Errorf("targetReference.apiVersion should be v1 and targetReference.kind should be Deployment")
	}

	return nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *InplaceUpdate) ValidateCreate() (admission.Warnings, error) {
	inplaceupdatelog.Info("validate create", "name", r.Name)

	if r.Spec.TargetReference == nil {
		return admission.Warnings{}, fmt.Errorf("targetReference is required")
	}

	warnings := admission.Warnings{}
	if r.Spec.TargetReference.APIVersion == "" {
		r.Spec.TargetReference.APIVersion = "v1"
		warnings = append(warnings, "targetReference.apiVersion is empty, defaulting to v1")
	}
	if r.Spec.TargetReference.Kind == "" {
		r.Spec.TargetReference.Kind = "Deployment"
		warnings = append(warnings, "targetReference.kind is empty, defaulting to Deployment")
	}
	if err := checkTargetReference(r.Spec.TargetReference); err != nil {
		return warnings, err
	}

	return warnings, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *InplaceUpdate) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	inplaceupdatelog.Info("validate update", "name", r.Name)

	return nil, fmt.Errorf("inplaceupdate is immutable")
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *InplaceUpdate) ValidateDelete() (admission.Warnings, error) {
	inplaceupdatelog.Info("validate delete", "name", r.Name)
	return nil, nil
}
