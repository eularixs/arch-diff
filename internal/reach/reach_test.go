package reach

import "testing"

import (
	"github.com/eularixs/arch-diff/internal/config"
	"github.com/eularixs/arch-diff/internal/model"
)

func defaultCfg() config.Config { return config.Default() }

// A -> B -> C, plus orphan D. Root A. Expect A,B,C reachable, D dead.
func TestComputeReachability(t *testing.T) {
	g := &model.Graph{
		Nodes: map[string]model.Node{
			"A": {ID: "A", Root: true}, "B": {ID: "B"}, "C": {ID: "C"}, "D": {ID: "D"},
		},
		Edges: []model.Edge{{From: "A", To: "B"}, {From: "B", To: "C"}},
	}
	Compute(g, Roots(g, defaultCfg()))
	for _, id := range []string{"A", "B", "C"} {
		if !g.Reachable[id] {
			t.Fatalf("%s should be reachable", id)
		}
	}
	if g.Reachable["D"] {
		t.Fatalf("D should be dead (no path from root)")
	}
}
