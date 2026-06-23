package diff

import (
	"testing"

	"github.com/eularixs/arch-diff/internal/config"
	"github.com/eularixs/arch-diff/internal/model"
)

func has(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

// base: Handler -> Service -> Repo (all reachable).
// head: Handler -> Repo directly; Service now orphaned (newly dead) and the
// new Handler->Repo edge crosses a layer boundary.
func TestDiffNewlyDeadAndCrossing(t *testing.T) {
	mk := func(edges []model.Edge, reach map[string]bool) *model.Graph {
		return &model.Graph{
			Nodes: map[string]model.Node{
				"H": {ID: "H", Layer: model.LayerHandler, Root: true},
				"S": {ID: "S", Layer: model.LayerService},
				"R": {ID: "R", Layer: model.LayerRepo},
			},
			Edges:     edges,
			Reachable: reach,
		}
	}
	base := mk(
		[]model.Edge{{From: "H", To: "S", Kind: "static"}, {From: "S", To: "R", Kind: "static"}},
		map[string]bool{"H": true, "S": true, "R": true},
	)
	head := mk(
		[]model.Edge{{From: "H", To: "R", Kind: "static"}},
		map[string]bool{"H": true, "R": true}, // S no longer reachable
	)

	r := Diff(base, head, config.Default())
	if !has(r.NewlyDead, "S") {
		t.Fatalf("S should be newly dead, got %v", r.NewlyDead)
	}
	if len(r.NewCrossing) != 1 || r.NewCrossing[0].From != "H" || r.NewCrossing[0].To != "R" {
		t.Fatalf("expected new crossing H->R, got %v", r.NewCrossing)
	}
	if len(r.RemovedCrossing) == 0 {
		t.Fatalf("S->R (or H->S) should be a removed crossing")
	}
}

// Changing only a body hash marks exactly that node Changed.
func TestDiffChangedByHash(t *testing.T) {
	base := &model.Graph{Nodes: map[string]model.Node{"A": {ID: "A", Hash: "x"}, "B": {ID: "B", Hash: "y"}}}
	head := &model.Graph{Nodes: map[string]model.Node{"A": {ID: "A", Hash: "x"}, "B": {ID: "B", Hash: "z"}}}
	r := Diff(base, head, config.Default())
	if len(r.Changed) != 1 || r.Changed[0] != "B" {
		t.Fatalf("only B should be changed, got %v", r.Changed)
	}
}

// No structural change yields an empty result.
func TestDiffEmpty(t *testing.T) {
	g := &model.Graph{
		Nodes:     map[string]model.Node{"A": {ID: "A", Layer: "handler"}},
		Reachable: map[string]bool{"A": true},
	}
	if !Diff(g, g, config.Default()).Empty() {
		t.Fatal("identical graphs should diff empty")
	}
}
