// Package reach resolves the root set and computes reachability (PRD §8). A node
// is dead when it is not reachable from any root. Reachability is a single BFS
// from the roots over the edge set; everything unvisited is dead.
package reach

import (
	"strings"
	"unicode"

	"github.com/eularixs/arch-diff/internal/config"
	"github.com/eularixs/arch-diff/internal/model"
)

// Roots returns the root node IDs per config (routes, main, exported API, keep).
//
// TODO(M3): resolve from cfg.Roots over g.Nodes. Validate non-empty and warn
// loudly otherwise — an empty root set makes the whole codebase look dead
// (PRD §15).
func Roots(g *model.Graph, cfg config.Config) []string {
	var roots []string
	for id, n := range g.Nodes {
		switch {
		case n.Root:
			// routes / main, already marked by the loader.
		case cfg.Roots.ExportedAPI && exported(id):
			// library mode: an exported symbol is an entrypoint by design.
		case keepMatch(id, cfg.Roots.Keep):
			// reflective/known entrypoint kept alive by config.
		default:
			continue
		}
		roots = append(roots, id)
	}
	return roots
}

// exported reports whether the node's function name is exported (capitalized).
// The name is the final dotted segment of the ID, e.g. "(*H).Create" -> "Create".
func exported(id string) bool {
	name := id[strings.LastIndex(id, ".")+1:]
	for _, r := range name {
		return unicode.IsUpper(r)
	}
	return false
}

// keepMatch reports whether the node ID matches any keep pattern. Patterns are
// doublestar globs over the node ID; a "**/" prefix is tried so a relative
// pattern (internal/jobs/*.Register) matches an ID carrying the module path.
func keepMatch(id string, keep []string) bool {
	for _, pat := range keep {
		if config.MatchGlob(pat, id) || config.MatchGlob("**/"+pat, id) {
			return true
		}
	}
	return false
}

// Compute fills g.Reachable with a BFS from roots over the edge set.
func Compute(g *model.Graph, roots []string) {
	out := make(map[string][]string, len(g.Edges))
	for _, e := range g.Edges {
		out[e.From] = append(out[e.From], e.To)
	}
	g.Reachable = make(map[string]bool, len(g.Nodes))
	queue := make([]string, 0, len(roots))
	for _, r := range roots {
		if !g.Reachable[r] {
			g.Reachable[r] = true
			queue = append(queue, r)
		}
	}
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		for _, m := range out[n] {
			if !g.Reachable[m] {
				g.Reachable[m] = true
				queue = append(queue, m)
			}
		}
	}
}
