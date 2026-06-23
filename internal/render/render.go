// Package render emits the structural delta as markdown (PRD §10). A mermaid
// subgraph of the changed region with muted dead-node styling is added in M4.
package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/eularixs/arch-diff/internal/diff"
	"github.com/eularixs/arch-diff/internal/model"
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

// DeadAudit renders every unreachable node in g, grouped by layer (PRD §8.2).
func DeadAudit(g *model.Graph, ref string) string {
	byLayer := map[string][]string{}
	for id, n := range g.Nodes {
		if !g.Reachable[id] {
			layer := n.Layer
			if layer == "" {
				layer = "(unclassified)"
			}
			byLayer[layer] = append(byLayer[layer], id)
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## Arch-diff dead audit: %s\n\n", ref)
	if len(byLayer) == 0 {
		b.WriteString("No dead code: every node is reachable from a root.\n")
		return b.String()
	}
	layers := make([]string, 0, len(byLayer))
	for l := range byLayer {
		layers = append(layers, l)
	}
	sort.Strings(layers)
	total := 0
	for _, l := range layers {
		ids := byLayer[l]
		sort.Strings(ids)
		total += len(ids)
		fmt.Fprintf(&b, "%s (%d)\n", l, len(ids))
		for _, id := range ids {
			fmt.Fprintf(&b, "  %s\n", id)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "%d dead node(s).\n", total)
	return b.String()
}
