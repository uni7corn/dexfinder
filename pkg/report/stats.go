package report

import "dex_method_finder/pkg/hiddenapi"

// Stats accumulates hidden API usage statistics.
type Stats struct {
	Count           uint32
	LinkingCount    uint32
	ReflectionCount uint32
	ApiCounts       map[hiddenapi.ApiList]uint32
}

// NewStats creates a new Stats.
func NewStats() *Stats {
	return &Stats{
		ApiCounts: make(map[hiddenapi.ApiList]uint32),
	}
}
