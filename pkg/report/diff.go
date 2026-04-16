package report

import (
	"encoding/json"
	"fmt"
	"io"

	"dex_method_finder/pkg/finder"
)

// DumpDiffText writes diff results as colored text.
func DumpDiffText(w io.Writer, diff *finder.DiffResult, dc *DisplayConfig) {
	col := dc.Color

	if !diff.HasChanges() {
		fmt.Fprintln(w, "No differences found.")
		return
	}

	// Added methods
	if len(diff.AddedMethods) > 0 {
		fmt.Fprintf(w, "%s\n", colorDiffSection(col, fmt.Sprintf("+ %d added method(s)", len(diff.AddedMethods))))
		for _, api := range diff.AddedMethods {
			fmt.Fprintf(w, "  %s %s\n", colorDiffAdd(col, "+"), dc.FormatAPI(api))
		}
		fmt.Fprintln(w)
	}

	// Added fields
	if len(diff.AddedFields) > 0 {
		fmt.Fprintf(w, "%s\n", colorDiffSection(col, fmt.Sprintf("+ %d added field(s)", len(diff.AddedFields))))
		for _, api := range diff.AddedFields {
			fmt.Fprintf(w, "  %s %s\n", colorDiffAdd(col, "+"), dc.FormatAPI(api))
		}
		fmt.Fprintln(w)
	}

	// Removed methods
	if len(diff.RemovedMethods) > 0 {
		fmt.Fprintf(w, "%s\n", colorDiffSection(col, fmt.Sprintf("- %d removed method(s)", len(diff.RemovedMethods))))
		for _, api := range diff.RemovedMethods {
			fmt.Fprintf(w, "  %s %s\n", colorDiffRemove(col, "-"), dc.FormatAPI(api))
		}
		fmt.Fprintln(w)
	}

	// Removed fields
	if len(diff.RemovedFields) > 0 {
		fmt.Fprintf(w, "%s\n", colorDiffSection(col, fmt.Sprintf("- %d removed field(s)", len(diff.RemovedFields))))
		for _, api := range diff.RemovedFields {
			fmt.Fprintf(w, "  %s %s\n", colorDiffRemove(col, "-"), dc.FormatAPI(api))
		}
		fmt.Fprintln(w)
	}

	// Changed methods
	if len(diff.ChangedMethods) > 0 {
		fmt.Fprintf(w, "%s\n", colorDiffSection(col, fmt.Sprintf("~ %d changed method(s)", len(diff.ChangedMethods))))
		for _, e := range diff.ChangedMethods {
			fmt.Fprintf(w, "  %s %s (%d → %d refs)\n", colorDiffChange(col, "~"), dc.FormatAPI(e.API), e.OldCount, e.NewCount)
		}
		fmt.Fprintln(w)
	}

	// Changed fields
	if len(diff.ChangedFields) > 0 {
		fmt.Fprintf(w, "%s\n", colorDiffSection(col, fmt.Sprintf("~ %d changed field(s)", len(diff.ChangedFields))))
		for _, e := range diff.ChangedFields {
			fmt.Fprintf(w, "  %s %s (%d → %d refs)\n", colorDiffChange(col, "~"), dc.FormatAPI(e.API), e.OldCount, e.NewCount)
		}
		fmt.Fprintln(w)
	}

	// Summary
	summary := fmt.Sprintf("Summary: +%d added, -%d removed, ~%d changed",
		diff.TotalAdded(), diff.TotalRemoved(), diff.TotalChanged())
	fmt.Fprintln(w, colorSummary(col, summary))
}

// DumpDiffJSON writes diff results as JSON.
func DumpDiffJSON(w io.Writer, diff *finder.DiffResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(diff)
}

// --- Diff color helpers ---

func colorDiffSection(c *Colorizer, s string) string {
	if c == nil || !c.Enabled() {
		return s
	}
	return ansiBold + s + ansiReset
}

func colorDiffAdd(c *Colorizer, s string) string {
	if c == nil || !c.Enabled() {
		return s
	}
	return ansiBoldGreen + s + ansiReset
}

func colorDiffRemove(c *Colorizer, s string) string {
	if c == nil || !c.Enabled() {
		return s
	}
	return ansiBoldRed + s + ansiReset
}

func colorDiffChange(c *Colorizer, s string) string {
	if c == nil || !c.Enabled() {
		return s
	}
	return ansiYellow + s + ansiReset
}
