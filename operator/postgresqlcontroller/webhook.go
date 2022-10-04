package postgresqlcontroller

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
)

// Validator validates admission requests.
type Validator struct {
	log logr.Logger
}

// ValidateCreate implements admission.CustomValidator.
func (v *Validator) ValidateCreate(_ context.Context, obj runtime.Object) error {
	//	instance := obj.(*exoscalev1.PostgreSQL)
	v.log.V(1).Info("Validate create")

	return nil
}

// ValidateUpdate implements admission.CustomValidator.
func (v *Validator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) error {
	//	newInstance := newObj.(*exoscalev1.PostgreSQL)
	//	oldInstance := oldObj.(*exoscalev1.PostgreSQL)
	v.log.V(1).Info("Validate update")

	return nil
}

// ValidateDelete implements admission.CustomValidator.
func (v *Validator) ValidateDelete(_ context.Context, obj runtime.Object) error {
	//	instance := obj.(*exoscalev1.PostgreSQL)
	v.log.V(1).Info("Validate delete (noop)")
	return nil
}
