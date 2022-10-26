package mapper

import (
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// FindStatusCondition returns the condition by the given type if present.
// If not found, it returns nil.
func FindStatusCondition(c []xpv1.Condition, conditionType xpv1.ConditionType) *xpv1.Condition {
	for _, condition := range c {
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}
