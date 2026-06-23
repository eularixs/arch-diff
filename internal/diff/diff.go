// Package diff set-diffs two graphs and their reachable-sets to surface
// structural change and dead-code deltas (PRD §9). It assumes both graphs were
// already run through package reach.
package diff

import (
	"sort"

	"github.com/eularixs/arch-diff/internal/config"
	"github.com/eularixs/arch-diff/internal/model"
)

// Result is the structural delta, in PRD priority order.
type Result struct {
	NewlyDead       []string     // reachable in base, dead in head (orphaned by this PR)
	NewCrossing     []model.Edge // new layer-crossing edges
	RemovedCrossing []model.Edge // removed layer-crossing edges
	Revived         []string     // dead in base, reachable in head
	Changed         []string     // body hash differs
	Added           []string     // new nodes
	Removed         []string     // removed nodes
}

// Empty reports whether nothing structural changed and nothing was orphaned.
func (r Result) Empty() bool {
	return len(r.NewlyDead)+len(r.NewCrossing)+len(r.RemovedCrossing)+len(r.Revived)+
		len(r.Changed)+len(r.Added)+len(r.Removed) == 0
}

// crossing reports whether an edge crosses a layer boundary.
func crossing(g *model.Graph, e model.Edge) bool {
	from, ok1 := g.Nodes[e.From]
	to, ok2 := g.Nodes[e.To]
	return ok1 && ok2 && from.Layer != "" && to.Layer != "" && from.Layer != to.Layer
}

func edgeKey(e model.Edge) string { return e.From + " -> " + e.To }

// Diff computes the structural delta between base and head.
func Diff(base, head *model.Graph, cfg config.Config) Result {
	var r Result

	// Node set-diff (added / removed / changed-by-hash).
	for id := range head.Nodes {
		if _, ok := base.Nodes[id]; !ok {
			r.Added = append(r.Added, id)
		}
	}
	for id, b := range base.Nodes {
		h, ok := head.Nodes[id]
		if !ok {
			r.Removed = append(r.Removed, id)
			continue
		}
		if b.Hash != "" && h.Hash != "" && b.Hash != h.Hash {
			r.Changed = append(r.Changed, id)
		}
	}

	// Reachable-set deltas: a node present in both, reachable->dead or dead->reachable.
	for id := range base.Nodes {
		if _, ok := head.Nodes[id]; !ok {
			continue // removed entirely, not "orphaned"
		}
		wasReachable := base.Reachable[id]
		isReachable := head.Reachable[id]
		switch {
		case wasReachable && !isReachable:
			r.NewlyDead = append(r.NewlyDead, id)
		case !wasReachable && isReachable:
			r.Revived = append(r.Revived, id)
		}
	}

	// Layer-crossing edge diff.
	baseEdges := map[string]model.Edge{}
	for _, e := range base.Edges {
		baseEdges[edgeKey(e)] = e
	}
	headEdges := map[string]model.Edge{}
	for _, e := range head.Edges {
		headEdges[edgeKey(e)] = e
	}
	for k, e := range headEdges {
		if _, ok := baseEdges[k]; !ok && crossing(head, e) {
			r.NewCrossing = append(r.NewCrossing, e)
		}
	}
	for k, e := range baseEdges {
		if _, ok := headEdges[k]; !ok && crossing(base, e) {
			r.RemovedCrossing = append(r.RemovedCrossing, e)
		}
	}

	r.sortAll()
	return r
}

func (r *Result) sortAll() {
	for _, s := range [][]string{r.NewlyDead, r.Revived, r.Changed, r.Added, r.Removed} {
		sort.Strings(s)
	}
	byKey := func(es []model.Edge) { sort.Slice(es, func(i, j int) bool { return edgeKey(es[i]) < edgeKey(es[j]) }) }
	byKey(r.NewCrossing)
	byKey(r.RemovedCrossing)
}
