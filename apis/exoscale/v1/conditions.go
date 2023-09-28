package v1

import (
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Updating returns a Ready condition where the service is updating.
// Crossplane's runtine doesn't provide a pre-defined update condition for some
// reason.
func Updating() xpv1.Condition {
	return xpv1.Condition{
		Type:               xpv1.TypeReady,
		Status:             corev1.ConditionFalse,
		Reason:             "Updating",
		Message:            "The service is being updated",
		LastTransitionTime: metav1.Now(),
	}
}
