package hiddenapi

// ApiListFilter determines which API lists should be reported.
type ApiListFilter struct {
	// excluded contains API list values that should NOT be reported
	excluded map[ApiList]bool
}

// NewApiListFilter creates a filter that excludes the named API lists.
// An empty excludeNames means report everything except Sdk.
func NewApiListFilter(excludeNames []string) *ApiListFilter {
	f := &ApiListFilter{
		excluded: make(map[ApiList]bool),
	}

	if len(excludeNames) == 0 {
		// Default: exclude SDK (public APIs are not interesting)
		f.excluded[Sdk] = true
		return f
	}

	for _, name := range excludeNames {
		name = trimSpace(name)
		if val, ok := nameToApiList[name]; ok {
			f.excluded[val] = true
		}
	}
	return f
}

// Matches returns true if the given API list should be reported.
func (f *ApiListFilter) Matches(list ApiList) bool {
	if list == Invalid {
		return false
	}
	return !f.excluded[list]
}
