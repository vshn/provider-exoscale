package webhook

import (
	"fmt"

	"github.com/hashicorp/go-version"
	"github.com/vshn/provider-exoscale/operator/mapper"

	"k8s.io/apimachinery/pkg/runtime"
)

func ValidateRawExtension(raw runtime.RawExtension) error {
	m, err := mapper.ToMap(raw)
	if err != nil {
		return fmt.Errorf("mapper.ToMap(%q): %w", raw, err)
	}
	for k, v := range m {
		switch v.(type) {
		case string:
		case int64:
		case float64:
		case bool:
			continue
		default:
			return fmt.Errorf("validate: value of key %q is not a supported type (only strings, boolean and numbers): %v", k, v)
		}
	}
	return nil
}

func ValidateVersion(oldObs, oldDes, newDes string) error {
	if oldObs == "" {
		return nil
	}
	oldObserved, err := version.NewVersion(oldObs)
	if err != nil {
		return fmt.Errorf("set old status version failed: %w", err)
	}
	oldDesired, err := version.NewVersion(oldDes)
	if err != nil {
		return fmt.Errorf("set old desired version failed: %w", err)
	}
	newDesired, err := version.NewVersion(newDes)
	if err != nil {
		return fmt.Errorf("set new desired version: %w", err)
	}

	c, err := version.NewConstraint(fmt.Sprintf("<= %s", oldObserved.String()))
	if err != nil {
		return fmt.Errorf("set version constraint failed: %w", err)
	}

	if newDesired.GreaterThan(oldDesired) && !c.Check(newDesired) {
		// update "14" -> "15.1" not allowed if observed version is < "15.1"
		return fmt.Errorf("field is immutable after creation: %s (old), %s (changed)", oldDesired, newDesired)
	}
	// we only allow version change if it matches the observed version
	return nil
}
