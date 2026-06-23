// Command arch-diff reports the structural delta of a Go backend's call-flow
// between two git refs: layer-crossing changes and the code a change orphaned
// (newly dead) or revived. Report-only; exit code is always 0 in v1 (PRD §12).
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/eularixs/arch-diff/internal/config"
	"github.com/eularixs/arch-diff/internal/diff"
	"github.com/eularixs/arch-diff/internal/loader"
	"github.com/eularixs/arch-diff/internal/model"
	"github.com/eularixs/arch-diff/internal/reach"
	"github.com/eularixs/arch-diff/internal/render"
)

func main() {
	var (
		base        = flag.String("base", "origin/main", "base ref (merge target)")
		head        = flag.String("head", "HEAD", "head ref")
		repo        = flag.String("repo", ".", "git repository path")
		cfgPath     = flag.String("config", "", "path to arch-diff.yaml (defaults to built-in)")
		format      = flag.String("format", "markdown", "output format: markdown")
		onlyCross   = flag.Bool("only-crossing", false, "filter to layer-crossing changes")
		onlyDead    = flag.Bool("only-dead", false, "filter to newly-dead nodes")
		deadAudit   = flag.Bool("dead", false, "full dead-code audit on head (no diff)")
	)
	flag.Parse()
	_, _, _ = *format, *onlyCross, *onlyDead

	if err := run(*repo, *base, *head, *cfgPath, *deadAudit); err != nil {
		fmt.Fprintln(os.Stderr, "arch-diff:", err)
		os.Exit(1) // wiring errors only; structural findings never fail the build (v1)
	}
}

func run(repo, base, head, cfgPath string, deadAudit bool) error {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	headG, err := buildAndReach(repo, head, cfg)
	if err != nil {
		return err
	}

	if deadAudit {
		fmt.Println(render.DeadAudit(headG, head))
		return nil
	}

	baseG, err := buildAndReach(repo, base, cfg) // TODO(M5): served from SHA cache
	if err != nil {
		return err
	}

	res := diff.Diff(baseG, headG, cfg)
	fmt.Println(render.Markdown(res, head, headG))
	return nil
}

// buildAndReach checks out ref, builds its graph, and computes reachability.
// It warns loudly when the root set is empty — the main dead-code failure mode
// (PRD §15): with no roots, every node looks dead.
func buildAndReach(repo, ref string, cfg config.Config) (*model.Graph, error) {
	wt, cleanup, err := loader.AddWorktree(repo, ref)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	g, err := loader.BuildGraphCached(wt, cfg)
	if err != nil {
		return nil, err
	}
	roots := reach.Roots(g, cfg)
	if len(roots) == 0 {
		fmt.Fprintf(os.Stderr,
			"arch-diff: warning: no roots resolved for %s — every node will look dead. "+
				"Check roots config (routes/main/exported_api/keep).\n", ref)
	}
	reach.Compute(g, roots)
	return g, nil
}
