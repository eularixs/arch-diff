# arch-diff

Structural diff for Go backends. Instead of changed text lines, arch-diff shows
how the call-flow between architectural layers changes across two git refs, and
which code the change **orphaned** (newly dead) or **revived**.

It reuses the [archview](https://github.com/eularixs/archview) call-graph engine,
runs it on both refs in isolated worktrees, runs a reachability pass on each, and
diffs both the graph and the reachable-set. Output is markdown + mermaid, posted
as a PR comment.

> Status: **scaffold** (draft v0.2). See `docs/prd-arch-diff.md` for the spec and
> `docs/tasks-arch-diff.md` for the milestone breakdown. Logic is stubbed; the
> data model, config, CLI surface, and reachability pass are in place.

## Why

A unified text diff cannot show structural regressions: a handler that now calls
a repo directly, a domain package that gained an infra dependency, or a refactor
that left a function dead — tested, compiling, invisible in the diff because dead
code is the *absence* of an edge.

## Usage

```
arch-diff --base origin/main --head HEAD
arch-diff --base origin/main --head HEAD --only-crossing   # layer-crossing only
arch-diff --base origin/main --head HEAD --only-dead        # newly-dead only
arch-diff --dead --head HEAD                                # full dead-code audit
```

Report-only: exit code is always 0 in v1. Gating is arch-lint's job, not this.

## GitHub Action

Add the action to any Go repo to get a structural-diff comment on every PR,
edited in place on each push:

```yaml
# .github/workflows/arch-diff.yml
name: arch-diff
on: pull_request
permissions:
  contents: read
  pull-requests: write
jobs:
  arch-diff:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - uses: eularixs/arch-diff@v0.1.0
        with:
          base: ${{ github.event.pull_request.base.ref }}
```

See `docs/examples/arch-diff-workflow.yml`.

## Config

See `arch-diff.example.yaml` (layers, roots, ignore). Reuses archview layer rules
where possible. Built graphs are cached by commit SHA (override the location with
`ARCH_DIFF_CACHE`, disable with `ARCH_DIFF_NO_CACHE`).

## License

MIT.
