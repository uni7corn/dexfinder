package finder

import (
	"sort"

	"dex_method_finder/pkg/dex"
)

// DiffResult holds the differences between two scan results.
type DiffResult struct {
	// APIs only in the new scan (added)
	AddedMethods []string `json:"added_methods,omitempty"`
	AddedFields  []string `json:"added_fields,omitempty"`

	// APIs only in the old scan (removed)
	RemovedMethods []string `json:"removed_methods,omitempty"`
	RemovedFields  []string `json:"removed_fields,omitempty"`

	// APIs in both but with different caller counts
	ChangedMethods []DiffEntry `json:"changed_methods,omitempty"`
	ChangedFields  []DiffEntry `json:"changed_fields,omitempty"`
}

// DiffEntry represents a changed API reference.
type DiffEntry struct {
	API      string `json:"api"`
	OldCount int    `json:"old_count"`
	NewCount int    `json:"new_count"`
}

// DiffScans compares two scan results and returns the differences.
// If query is non-empty, only matching APIs are compared.
func DiffScans(oldResult, newResult *ScanResult, oldDex, newDex []*dex.DexFile, query string, scope QueryScope) *DiffResult {
	dr := &DiffResult{}

	// Optionally filter by query
	var oldMethods map[string][]MethodRef
	var newMethods map[string][]MethodRef
	var oldFields map[string][]FieldRef
	var newFields map[string][]FieldRef

	if query != "" {
		oldQR := Query(oldResult, oldDex, query, scope)
		newQR := Query(newResult, newDex, query, scope)
		oldMethods = oldQR.MatchedMethods
		newMethods = newQR.MatchedMethods
		oldFields = oldQR.MatchedFields
		newFields = newQR.MatchedFields
	} else {
		oldMethods = oldResult.MethodRefs
		newMethods = newResult.MethodRefs
		oldFields = oldResult.FieldRefs
		newFields = newResult.FieldRefs
	}

	// Compare methods
	for api, refs := range newMethods {
		if oldRefs, ok := oldMethods[api]; !ok {
			dr.AddedMethods = append(dr.AddedMethods, api)
		} else if len(refs) != len(oldRefs) {
			dr.ChangedMethods = append(dr.ChangedMethods, DiffEntry{
				API: api, OldCount: len(oldRefs), NewCount: len(refs),
			})
		}
	}
	for api := range oldMethods {
		if _, ok := newMethods[api]; !ok {
			dr.RemovedMethods = append(dr.RemovedMethods, api)
		}
	}

	// Compare fields
	for api, refs := range newFields {
		if oldRefs, ok := oldFields[api]; !ok {
			dr.AddedFields = append(dr.AddedFields, api)
		} else if len(refs) != len(oldRefs) {
			dr.ChangedFields = append(dr.ChangedFields, DiffEntry{
				API: api, OldCount: len(oldRefs), NewCount: len(refs),
			})
		}
	}
	for api := range oldFields {
		if _, ok := newFields[api]; !ok {
			dr.RemovedFields = append(dr.RemovedFields, api)
		}
	}

	// Sort for deterministic output
	sort.Strings(dr.AddedMethods)
	sort.Strings(dr.AddedFields)
	sort.Strings(dr.RemovedMethods)
	sort.Strings(dr.RemovedFields)
	sort.Slice(dr.ChangedMethods, func(i, j int) bool { return dr.ChangedMethods[i].API < dr.ChangedMethods[j].API })
	sort.Slice(dr.ChangedFields, func(i, j int) bool { return dr.ChangedFields[i].API < dr.ChangedFields[j].API })

	return dr
}

// HasChanges returns true if any differences were found.
func (dr *DiffResult) HasChanges() bool {
	return len(dr.AddedMethods) > 0 || len(dr.AddedFields) > 0 ||
		len(dr.RemovedMethods) > 0 || len(dr.RemovedFields) > 0 ||
		len(dr.ChangedMethods) > 0 || len(dr.ChangedFields) > 0
}

// TotalAdded returns total added APIs.
func (dr *DiffResult) TotalAdded() int {
	return len(dr.AddedMethods) + len(dr.AddedFields)
}

// TotalRemoved returns total removed APIs.
func (dr *DiffResult) TotalRemoved() int {
	return len(dr.RemovedMethods) + len(dr.RemovedFields)
}

// TotalChanged returns total changed APIs.
func (dr *DiffResult) TotalChanged() int {
	return len(dr.ChangedMethods) + len(dr.ChangedFields)
}
