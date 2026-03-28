package finder

import "strings"

// ClassFilter filters classes by descriptor prefix.
type ClassFilter struct {
	prefixes []string
}

// NewClassFilter creates a filter. Empty prefixes matches everything.
func NewClassFilter(prefixes []string) *ClassFilter {
	return &ClassFilter{prefixes: prefixes}
}

// Matches returns true if the class descriptor matches the filter.
func (f *ClassFilter) Matches(descriptor string) bool {
	if len(f.prefixes) == 0 {
		return true
	}
	for _, p := range f.prefixes {
		if strings.HasPrefix(descriptor, p) {
			return true
		}
	}
	return false
}
