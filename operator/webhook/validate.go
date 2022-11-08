package webhook

import (
	"fmt"

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
