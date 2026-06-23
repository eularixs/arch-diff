// Package loader turns a git ref into a model.Graph by checking the ref out into
// an isolated worktree and running the archview call-graph engine over it
// (PRD §5). It reuses archview's Raw graph; it does not reimplement analysis.
package loader

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	archview "github.com/eularixs/archview"

	"github.com/eularixs/arch-diff/internal/config"
	"github.com/eularixs/arch-diff/internal/model"
)

// Worktree is an isolated checkout of one ref.
type Worktree struct {
	Dir string
	SHA string
}

// AddWorktree runs `git worktree add` for ref under a temp dir and returns the
// checkout plus a cleanup func that removes it.
func AddWorktree(repo, ref string) (Worktree, func(), error) {
	noop := func() {}
	sha, err := git(repo, "rev-parse", ref)
	if err != nil {
		return Worktree{}, noop, fmt.Errorf("resolve %q: %w", ref, err)
	}
	dir, err := os.MkdirTemp("", "arch-diff-*")
	if err != nil {
		return Worktree{}, noop, err
	}
	if _, err := git(repo, "worktree", "add", "--detach", "--quiet", dir, sha); err != nil {
		os.RemoveAll(dir)
		return Worktree{}, noop, fmt.Errorf("worktree add %s: %w", sha, err)
	}
	cleanup := func() {
		git(repo, "worktree", "remove", "--force", dir)
		os.RemoveAll(dir)
	}
	return Worktree{Dir: dir, SHA: sha}, cleanup, nil
}

func git(repo string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// archview layer -> arch-diff layer. Unmapped layers pass through unchanged.
var layerMap = map[string]string{
	"controller": model.LayerHandler,
	"service":    model.LayerService,
	"repository": model.LayerRepo,
}

// BuildGraph loads wt with archview's Raw (unpruned) graph and maps it into a
// model.Graph: every function is a node, call edges are static, implements edges
// are interface, and a function targeted by a route edge (or named main) is a
// root. Reachability is left to package reach; the body Hash is filled in M2.
func BuildGraph(wt Worktree, cfg config.Config) (*model.Graph, error) {
	srv, err := archview.New(archview.Options{Root: wt.Dir, Raw: true})
	if err != nil {
		return nil, err
	}
	ag := srv.Graph()

	g := &model.Graph{SHA: wt.SHA, Nodes: map[string]model.Node{}, Reachable: map[string]bool{}}
	for _, n := range ag.Nodes {
		if n.Kind != "func" {
			continue // endpoint nodes are route markers, surfaced via edges below
		}
		layer := n.Layer
		if m, ok := layerMap[layer]; ok {
			layer = m
		}
		node := model.Node{ID: n.ID, Layer: layer, Root: n.Func == "main"}
		g.Nodes[n.ID] = node
	}
	for _, e := range ag.Edges {
		switch e.Kind {
		case "route":
			// endpoint -> handler: the handler func is a root.
			if h, ok := g.Nodes[e.To]; ok {
				h.Root = true
				g.Nodes[e.To] = h
			}
		case "implements":
			g.Edges = append(g.Edges, model.Edge{From: e.From, To: e.To, Kind: model.EdgeInterface})
		case "call", "dispatch":
			if _, ok := g.Nodes[e.From]; ok {
				if _, ok := g.Nodes[e.To]; ok {
					g.Edges = append(g.Edges, model.Edge{From: e.From, To: e.To, Kind: model.EdgeStatic})
				}
			}
		}
	}
	return g, nil
}
