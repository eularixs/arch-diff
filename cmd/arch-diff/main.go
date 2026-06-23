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

	headWt, cleanupHead, err := loader.AddWorktree(repo, head)
	if err != nil {
		return err
	}
	defer cleanupHead()
	headG, err := loader.BuildGraph(headWt, cfg)
	if err != nil {
		return err
	}
	reach.Compute(headG, reach.Roots(headG, cfg))

	if deadAudit {
		// TODO(M3): list every unreachable node grouped by layer, no diff.
		fmt.Println("## Arch-diff dead audit\n\n(not implemented — M3)")
		return nil
	}

	baseWt, cleanupBase, err := loader.AddWorktree(repo, base)
	if err != nil {
		return err
	}
	defer cleanupBase()
	baseG, err := loader.BuildGraph(baseWt, cfg) // TODO(M5): served from SHA cache
	if err != nil {
		return err
	}
	reach.Compute(baseG, reach.Roots(baseG, cfg))

	res := diff.Diff(baseG, headG, cfg)
	fmt.Println(render.Markdown(res, head))
	return nil
}
