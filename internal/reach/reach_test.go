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

func TestRootsExportedAndKeep(t *testing.T) {
	g := &model.Graph{Nodes: map[string]model.Node{
		"m/pkg.(*H).Create":    {ID: "m/pkg.(*H).Create"},   // exported
		"m/pkg.(*H).validate":  {ID: "m/pkg.(*H).validate"}, // unexported
		"m/internal/jobs.Register": {ID: "m/internal/jobs.Register"},
	}}
	// exported_api: exported funcs are roots.
	cfg := config.Default()
	cfg.Roots = config.Roots{ExportedAPI: true}
	roots := Roots(g, cfg)
	if !contains(roots, "m/pkg.(*H).Create") || contains(roots, "m/pkg.(*H).validate") {
		t.Fatalf("exported_api roots wrong: %v", roots)
	}
	// keep: pattern matches an ID carrying the module path.
	cfg2 := config.Default()
	cfg2.Roots = config.Roots{Keep: []string{"internal/jobs.Register"}}
	if r := Roots(g, cfg2); !contains(r, "m/internal/jobs.Register") {
		t.Fatalf("keep should match jobs.Register, got %v", r)
	}
}

func contains(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}
