package mapper

import (
	"github.com/hashicorp/go-version"
	"k8s.io/apimachinery/pkg/runtime"
)

// IsSameStringSet returns true if both slices have the same unique elements in any order.
func IsSameStringSet(a []string, b *[]string) bool {
	if b == nil {
		return len(a) == 0
	}
	set1 := set(a)
	set2 := set(*b)
	if len(set1) != len(set2) {
		return false
	}
	for ai := range set1 {
		if _, ok := set2[ai]; !ok {
			return false
		}
	}
	return true
}

func set(s []string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, i := range s {
		m[i] = struct{}{}
	}
	return m
}

func CompareSettings(a, b runtime.RawExtension) bool {
	sa, err := ToMap(a)
	if err != nil {
		// we have to assume they're not the same
		return false
	}
	sb, err := ToMap(b)
	if err != nil {
		return false
	}
	if len(sa) != len(sb) {
		return false
	}
	for k, va := range sa {
		if sb[k] != va {
			return false
		}
	}
	return true
}

// CompareMajorVersion params should follow SemVer.
// But a version starting with 'v' is also valid, e.g v1.2.3
func CompareMajorVersion(a, b string) (bool, error) {
	va, err := version.NewVersion(a)
	if err != nil {
		return false, err
	}
	vb, err := version.NewVersion(b)
	if err != nil {
		return false, err
	}
	sva := va.Segments()
	svb := vb.Segments()

	// first segment is the major version (segment order MAJOR.MINOR.PATCH)
	return sva[0] == svb[0], nil
}
