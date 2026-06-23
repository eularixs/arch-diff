package render

import (
	"strings"
	"testing"

	"github.com/eularixs/arch-diff/internal/diff"
	"github.com/eularixs/arch-diff/internal/model"
)

func TestMarkdownMermaidAndEmpty(t *testing.T) {
	head := &model.Graph{Nodes: map[string]model.Node{
		"m/h.Create": {ID: "m/h.Create", Layer: "handler"},
		"m/r.Save":   {ID: "m/r.Save", Layer: "repo"},
	}}
	r := diff.Result{
		NewlyDead:   []string{"m/s.Do"},
		NewCrossing: []model.Edge{{From: "m/h.Create", To: "m/r.Save"}},
	}
	out := Markdown(r, "#1", head)
	for _, want := range []string{"Newly dead", "```mermaid", ":::dead", ":::", "classDef dead"} {
		if !strings.Contains(out, want) {
			t.Fatalf("report missing %q\n%s", want, out)
		}
	}
	// empty diff -> single line, no mermaid.
	empty := Markdown(diff.Result{}, "#1", head)
	if strings.Contains(empty, "mermaid") || !strings.Contains(empty, "No structural change") {
		t.Fatalf("empty report wrong: %s", empty)
	}
}
