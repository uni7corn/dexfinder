package report

import (
	"encoding/json"
	"io"

	"dex_method_finder/pkg/dex"
	"dex_method_finder/pkg/finder"
	"dex_method_finder/pkg/hiddenapi"
)

// SARIF 2.1.0 types (minimal subset)

type sarifLog struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules,omitempty"`
}

type sarifRule struct {
	ID               string         `json:"id"`
	Name             string         `json:"name,omitempty"`
	ShortDescription sarifMessage   `json:"shortDescription"`
	HelpURI          string         `json:"helpUri,omitempty"`
	Properties       map[string]any `json:"properties,omitempty"`
}

type sarifResult struct {
	RuleID    string           `json:"ruleId"`
	Level     string           `json:"level"` // error, warning, note
	Message   sarifMessage     `json:"message"`
	Locations []sarifLocation  `json:"locations,omitempty"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifLocation struct {
	PhysicalLocation *sarifPhysicalLocation `json:"physicalLocation,omitempty"`
	LogicalLocations []sarifLogicalLocation `json:"logicalLocations,omitempty"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifLogicalLocation struct {
	FullyQualifiedName string `json:"fullyQualifiedName"`
	Kind               string `json:"kind"` // "function", "type"
}

func newSarifLog() *sarifLog {
	return &sarifLog{
		Version: "2.1.0",
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/main/sarif-2.1/schema/sarif-schema-2.1.0.json",
	}
}

func newSarifRun() sarifRun {
	return sarifRun{
		Tool: sarifTool{
			Driver: sarifDriver{
				Name:           "dexfinder",
				Version:        "dev",
				InformationURI: "https://github.com/JuneLeGency/dexfinder",
			},
		},
	}
}

// DumpScanSARIF writes scan results in SARIF format.
func DumpScanSARIF(w io.Writer, result *finder.ScanResult, dexFiles []*dex.DexFile, query string, scope finder.QueryScope, dc *DisplayConfig) error {
	var opts []finder.QueryOption
	if dc != nil && dc.Mapping != nil {
		opts = append(opts, finder.QueryOption{Mapping: dc.Mapping})
	}
	qr := finder.Query(result, dexFiles, query, scope, opts...)

	log := newSarifLog()
	run := newSarifRun()
	run.Tool.Driver.Rules = []sarifRule{
		{ID: "dexfinder/method-ref", Name: "MethodReference", ShortDescription: sarifMessage{Text: "Method reference found matching query"}},
		{ID: "dexfinder/field-ref", Name: "FieldReference", ShortDescription: sarifMessage{Text: "Field reference found matching query"}},
		{ID: "dexfinder/string-ref", Name: "StringReference", ShortDescription: sarifMessage{Text: "String constant found matching query"}},
	}

	// Methods
	for api, refs := range qr.MatchedMethods {
		callers := aggregateMethodCallers(refs, dexFiles, dc)
		for _, c := range callers {
			run.Results = append(run.Results, sarifResult{
				RuleID: "dexfinder/method-ref",
				Level:  "note",
				Message: sarifMessage{
					Text: api + " called by " + c.name,
				},
				Locations: []sarifLocation{{
					LogicalLocations: []sarifLogicalLocation{{
						FullyQualifiedName: c.name,
						Kind:               "function",
					}},
				}},
			})
		}
	}

	// Fields
	for api, refs := range qr.MatchedFields {
		callers := aggregateFieldCallers(refs, dexFiles, dc)
		for _, c := range callers {
			run.Results = append(run.Results, sarifResult{
				RuleID: "dexfinder/field-ref",
				Level:  "note",
				Message: sarifMessage{
					Text: api + " accessed by " + c.name,
				},
				Locations: []sarifLocation{{
					LogicalLocations: []sarifLogicalLocation{{
						FullyQualifiedName: c.name,
						Kind:               "function",
					}},
				}},
			})
		}
	}

	// Strings
	for str, refs := range qr.MatchedStrings {
		callers := aggregateStringCallers(refs, dexFiles, dc)
		for _, c := range callers {
			run.Results = append(run.Results, sarifResult{
				RuleID: "dexfinder/string-ref",
				Level:  "note",
				Message: sarifMessage{
					Text: `"` + str + `" used by ` + c.name,
				},
				Locations: []sarifLocation{{
					LogicalLocations: []sarifLogicalLocation{{
						FullyQualifiedName: c.name,
						Kind:               "function",
					}},
				}},
			})
		}
	}

	log.Runs = []sarifRun{run}
	return writeSARIF(w, log)
}

// DumpTraceSARIF writes trace results in SARIF format.
func DumpTraceSARIF(w io.Writer, result *finder.ScanResult, dexFiles []*dex.DexFile, query string, maxDepth int, dc *DisplayConfig) error {
	var opts []finder.QueryOption
	if dc != nil && dc.Mapping != nil {
		opts = append(opts, finder.QueryOption{Mapping: dc.Mapping})
	}
	qr := finder.Query(result, dexFiles, query, finder.ScopeCallee, opts...)
	cg := finder.BuildCallGraph(result, dexFiles)

	log := newSarifLog()
	run := newSarifRun()
	run.Tool.Driver.Rules = []sarifRule{
		{ID: "dexfinder/call-chain", Name: "CallChain", ShortDescription: sarifMessage{Text: "Call chain to target API"}},
	}

	allAPIs := append(sortedKeys(qr.MatchedMethods), sortedFieldKeys(qr.MatchedFields)...)
	for _, api := range allAPIs {
		tree := cg.TraceCallers(api, maxDepth)
		chains := finder.FlatCallerChains(tree, 100)
		for _, chain := range chains {
			root := ""
			if len(chain) > 0 {
				root = dc.FormatNode(chain[len(chain)-1])
			}
			run.Results = append(run.Results, sarifResult{
				RuleID: "dexfinder/call-chain",
				Level:  "note",
				Message: sarifMessage{
					Text: dc.FormatHeader(api) + " ← " + root,
				},
				Locations: []sarifLocation{{
					LogicalLocations: []sarifLogicalLocation{{
						FullyQualifiedName: root,
						Kind:               "function",
					}},
				}},
			})
		}
	}

	log.Runs = []sarifRun{run}
	return writeSARIF(w, log)
}

// DumpHiddenAPISARIF writes hidden API findings in SARIF format.
func DumpHiddenAPISARIF(w io.Writer, result *finder.ScanResult, dexFiles []*dex.DexFile, db *hiddenapi.Database) error {
	log := newSarifLog()
	run := newSarifRun()
	run.Tool.Driver.Rules = []sarifRule{
		{ID: "dexfinder/hidden-api-linking", Name: "HiddenAPILinking", ShortDescription: sarifMessage{Text: "Direct linking to hidden/restricted API"}},
		{ID: "dexfinder/hidden-api-reflection", Name: "HiddenAPIReflection", ShortDescription: sarifMessage{Text: "Potential reflection-based hidden API access"}},
	}

	// Linking: methods
	for api, refs := range result.MethodRefs {
		apiList := db.GetApiList(api)
		level := hiddenAPILevel(apiList)
		callers := aggregateMethodCallers(refs, dexFiles, nil)
		for _, c := range callers {
			run.Results = append(run.Results, sarifResult{
				RuleID:  "dexfinder/hidden-api-linking",
				Level:   level,
				Message: sarifMessage{Text: "Linking " + apiList.String() + " " + api},
				Locations: []sarifLocation{{
					LogicalLocations: []sarifLogicalLocation{{
						FullyQualifiedName: c.name,
						Kind:               "function",
					}},
				}},
			})
		}
	}

	// Linking: fields
	for api, refs := range result.FieldRefs {
		apiList := db.GetApiList(api)
		level := hiddenAPILevel(apiList)
		callers := aggregateFieldCallers(refs, dexFiles, nil)
		for _, c := range callers {
			run.Results = append(run.Results, sarifResult{
				RuleID:  "dexfinder/hidden-api-linking",
				Level:   level,
				Message: sarifMessage{Text: "Linking " + apiList.String() + " " + api},
				Locations: []sarifLocation{{
					LogicalLocations: []sarifLogicalLocation{{
						FullyQualifiedName: c.name,
						Kind:               "function",
					}},
				}},
			})
		}
	}

	// Reflection
	reflections := result.FindPotentialReflection(db)
	for _, ref := range reflections {
		apiList := db.GetApiList(ref.Signature)
		level := hiddenAPILevel(apiList)
		callers := aggregateStringCallers(ref.StringRef, dexFiles, nil)
		for _, c := range callers {
			run.Results = append(run.Results, sarifResult{
				RuleID:  "dexfinder/hidden-api-reflection",
				Level:   level,
				Message: sarifMessage{Text: "Reflection " + apiList.String() + " " + ref.Signature},
				Locations: []sarifLocation{{
					LogicalLocations: []sarifLogicalLocation{{
						FullyQualifiedName: c.name,
						Kind:               "function",
					}},
				}},
			})
		}
	}

	log.Runs = []sarifRun{run}
	return writeSARIF(w, log)
}

func hiddenAPILevel(apiList hiddenapi.ApiList) string {
	switch apiList {
	case hiddenapi.Blocked:
		return "error"
	case hiddenapi.Unsupported:
		return "warning"
	default:
		return "note"
	}
}

func writeSARIF(w io.Writer, log *sarifLog) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(log)
}
