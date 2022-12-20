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

func ValidateUpdateVersion(oldObs, oldDes, newDes string) error {
	oldObserved, err := version.NewVersion(oldObs)
	if err != nil {
		return fmt.Errorf("set old status version '%s' failed: %w", oldObs, err)
	}
	oldDesired, err := version.NewVersion(oldDes)
	if err != nil {
		return fmt.Errorf("set old desired version '%s' failed: %w", oldDes, err)
	}
	newDesired, err := version.NewVersion(newDes)
	if err != nil {
		return fmt.Errorf("set new desired version '%s' failed: %w", newDes, err)
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

func ValidateVersions(wanted string, admittedVersions []string) error {
	if wanted == "" {
		return fmt.Errorf("version must be provided")
	}
	if !contains(admittedVersions, wanted) {
		return fmt.Errorf("version not valid")
	}
	return nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
