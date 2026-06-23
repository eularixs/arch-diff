// Package reach resolves the root set and computes reachability (PRD §8). A node
// is dead when it is not reachable from any root. Reachability is a single BFS
// from the roots over the edge set; everything unvisited is dead.
package reach

import (
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
		if n.Root {
			roots = append(roots, id)
		}
	}
	return roots
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
