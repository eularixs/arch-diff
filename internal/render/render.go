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

// Markdown renders a diff result as the PR-comment report, followed by a mermaid
// subgraph of the changed region. When the result is empty it returns a
// single-line "no structural change" report.
func Markdown(r diff.Result, prRef string, head *model.Graph) string {
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
	b.WriteString(mermaid(r, head))
	return b.String()
}

// shortLabel trims a node ID to its package-tail plus receiver.method.
func shortLabel(id string) string {
	if i := strings.LastIndex(id, "/"); i >= 0 {
		return id[i+1:]
	}
	return id
}

// mermaid draws the changed region: every node touched by the diff, the new
// (solid) and removed (dotted) layer-crossing edges between them, with newly
// dead, revived, changed, added, and removed nodes styled distinctly.
func mermaid(r diff.Result, head *model.Graph) string {
	involved := map[string]string{} // id -> class
	set := func(id, class string) {
		if _, ok := involved[id]; !ok {
			involved[id] = class
		}
	}
	for _, id := range r.NewlyDead {
		set(id, "dead")
	}
	for _, id := range r.Revived {
		set(id, "revived")
	}
	for _, id := range r.Changed {
		set(id, "changed")
	}
	for _, id := range r.Added {
		set(id, "added")
	}
	for _, id := range r.Removed {
		set(id, "removed")
	}
	for _, e := range r.NewCrossing {
		set(e.From, "")
		set(e.To, "")
	}
	for _, e := range r.RemovedCrossing {
		set(e.From, "")
		set(e.To, "")
	}
	if len(involved) == 0 {
		return ""
	}

	// Stable mermaid node ids.
	ids := make([]string, 0, len(involved))
	for id := range involved {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	nid := map[string]string{}
	for i, id := range ids {
		nid[id] = fmt.Sprintf("n%d", i)
	}

	var b strings.Builder
	b.WriteString("```mermaid\ngraph LR\n")
	for _, id := range ids {
		line := fmt.Sprintf("  %s[\"%s\"]", nid[id], shortLabel(id))
		if c := involved[id]; c != "" {
			line += ":::" + c
		}
		b.WriteString(line + "\n")
	}
	for _, e := range r.NewCrossing {
		fmt.Fprintf(&b, "  %s --> %s\n", nid[e.From], nid[e.To])
	}
	for _, e := range r.RemovedCrossing {
		fmt.Fprintf(&b, "  %s -.-> %s\n", nid[e.From], nid[e.To])
	}
	b.WriteString("  classDef dead fill:#3a1212,stroke:#f87171,color:#fca5a5;\n")
	b.WriteString("  classDef revived fill:#0f2a1a,stroke:#34d399,color:#6ee7b7;\n")
	b.WriteString("  classDef changed fill:#2a230f,stroke:#fbbf24,color:#fcd34d;\n")
	b.WriteString("  classDef added fill:#10233a,stroke:#60a5fa,color:#93c5fd;\n")
	b.WriteString("  classDef removed fill:#241024,stroke:#a78bfa,color:#c4b5fd;\n")
	b.WriteString("```\n")
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
