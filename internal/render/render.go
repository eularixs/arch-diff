// Package render emits the structural delta as markdown (PRD §10). A mermaid
// subgraph of the changed region with muted dead-node styling is added in M4.
package render

import (
	"fmt"
	"strings"

	"github.com/eularixs/arch-diff/internal/diff"
)

// Markdown renders a diff result as the PR-comment report. When the result is
// empty it returns a single-line "no structural change" report.
func Markdown(r diff.Result, prRef string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Arch-diff: %s\n\n", prRef)
	if r.Empty() {
		b.WriteString("No structural change, nothing orphaned.\n")
		return b.String()
	}
	section(&b, "Newly dead (orphaned by this PR)", r.NewlyDead)
	if len(r.NewCrossing) > 0 {
		b.WriteString("New layer-crossing edge\n")
		for _, e := range r.NewCrossing {
			fmt.Fprintf(&b, "  %s -> %s\n", e.From, e.To)
		}
		b.WriteString("\n")
	}
	if len(r.RemovedCrossing) > 0 {
		b.WriteString("Removed layer-crossing edge\n")
		for _, e := range r.RemovedCrossing {
			fmt.Fprintf(&b, "  %s -> %s\n", e.From, e.To)
		}
		b.WriteString("\n")
	}
	section(&b, "Revived", r.Revived)
	section(&b, "Changed (body)", r.Changed)
	section(&b, "New node", r.Added)
	section(&b, "Removed node", r.Removed)
	return b.String()
}

func section(b *strings.Builder, title string, ids []string) {
	if len(ids) == 0 {
		return
	}
	b.WriteString(title + "\n")
	for _, id := range ids {
		fmt.Fprintf(b, "  %s\n", id)
	}
	b.WriteString("\n")
}
