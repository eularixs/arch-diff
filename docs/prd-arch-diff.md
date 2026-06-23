# PRD: arch-diff

Structural diff for Go backends. Shows how the call-flow between architectural layers changes across two git refs, and which code the change orphans or revives, instead of showing changed text lines.

Status: Draft v0.2
Owner: Dimas
Depends on: archview (existing call-graph engine)
Changelog: v0.2 merges dead-flow (reachability-based dead code detection) into the core.

---

## 1. Problem

A unified text diff (`git diff`) shows which lines changed. It does not show whether the *structure* changed. The most review-critical regressions in a layered backend are structural and invisible in a text diff:

- A handler that used to call a repo through a service now calls the repo directly.
- A domain package gains a dependency on infra.
- A refactor removes the last caller of a function, leaving it dead, fully tested and looking alive in the diff.

Reviewers catch these by reading the whole diff and holding the architecture in their head. That does not scale and is the first thing skipped under time pressure. Dead code is worse: a text diff can never show it, because dead code is defined by the *absence* of an edge, not by any changed line.

## 2. Goal

Given two git refs (base and head), produce a structural delta of the call-graph that surfaces:

1. layer-crossing changes in the flow, and
2. nodes the change made unreachable (newly dead) or reachable again (revived),

in a PR comment, so a reviewer sees architectural impact and dead code in seconds without reading every line.

## 3. Non-Goals (STRICTLY FORBIDDEN in v1)

- No policy enforcement or pass/fail gating. That is arch-lint's job. arch-diff only reports.
- No rename/move detection. A rename shows as one removed node plus one added node. Acceptable for v1.
- No path-to-sink enumeration (new path from entrypoint to a sensitive sink). Deferred to v2. Note: boolean reachability from roots IS in scope, because dead code detection needs it. Enumerating the full path to a sink is the part that is deferred.
- No support for languages other than Go.
- No web UI. Output is markdown plus mermaid only.
- No attempt to hold two type-checker sessions in one process.
- No auto-deletion of dead code. Report only. The author decides.

## 4. Users and use cases

- **PR reviewer** wants to know if this PR changed the architecture or left dead code before reading code.
- **Author** wants a self-check before requesting review.
- **Tech lead** wants a record of structural drift and dead-code accumulation over time.

Primary use case is the PR comment in CI. Local CLI is secondary.

## 5. How it works

archview already builds a call-graph for one revision. arch-diff runs that engine twice, runs a reachability pass on each, and diffs both the graph and the reachable-set.

1. `git worktree add` for each ref into a temp dir (base and head isolated).
2. Load each worktree with `go/packages`, build the graph via archview.
3. Compute the set of roots (entrypoints) for each graph from config.
4. Run reachability from roots, marking every node reachable or not.
5. Serialize each graph plus its reachable-set to JSON, keyed by node identity.
6. Set-diff the two graphs, and separately diff the two reachable-sets.
7. Render the delta as markdown plus a mermaid subgraph of the changed region.

Graphs are deterministic per commit SHA, so cache each serialized graph with the SHA as the key. The base branch rarely changes, so its graph and reachable-set are computed once and reused.

## 6. Node identity (the crux)

Diff quality depends entirely on stable, location-independent node IDs. IDs MUST NOT include file or line position, otherwise inserting one line above a function marks every node below it as changed and floods the diff with noise.

Node ID format:

```
github.com/x/eularix/internal/order/handler.(*OrderHandler).Create
```

That is: package path + receiver + method name + signature. Moving lines around does not change the ID. Only genuine structural change is visible.

## 7. Data model

```go
type Node struct {
    ID    string // stable location-independent key
    Layer string // handler | service | repo | domain | infra (resolved from path rules)
    Hash  string // hash of normalized AST body; detects "changed" nodes
    Root  bool   // is this node an entrypoint (route, main, cron, exported API)
}

type Edge struct {
    From string // Node.ID
    To   string // Node.ID
    Kind string // static | interface
}

type Graph struct {
    SHA       string
    Nodes     map[string]Node
    Edges     []Edge
    Reachable map[string]bool // node ID -> reachable from any root, computed pass
}
```

`Reachable` is derived, not authored. It is the result of a BFS/DFS from the set of root nodes over the edge set.

## 8. Dead code detection (merged from dead-flow)

A node is **dead** when it is not reachable from any root. Roots are entrypoints: HTTP routes, `main`, cron/worker registrations, and optionally exported package API. Reachability is a single graph traversal from the root set; everything not visited is dead.

Two modes:

**8.1 Diff-aware dead code (default, the novel part)**

arch-diff already has the base and head reachable-sets. The interesting signal is the delta:

- **Newly dead**: reachable in base, dead in head. The PR orphaned it. This is the headline. Example: a refactor swaps `InventoryService.Reserve` out of the flow and forgets to delete it. It still compiles, still has tests, looks alive in the text diff. arch-diff flags it as orphaned by this PR.
- **Revived**: dead in base, reachable in head. The PR wired up previously dead code. Usually fine, occasionally a sign someone resurrected something they should not have.

This ties dead-code detection to the change, which is exactly arch-diff's domain. A reviewer sees "your PR left these 2 functions with no callers" without scanning anything.

**8.2 Full audit mode (`--dead`)**

Run reachability on HEAD only and list every dead node, grouped by layer. This is a one-shot debt report, not a diff. Useful for periodic cleanup, not every PR.

**8.3 Edge cases that are NOT dead code**

- Exported symbols of a library package when configured as roots. A public API has no in-repo caller by design. Treat exported API as roots when the module is a library, controlled by config.
- Functions reached only through reflection or `init` side effects. archview cannot see these. Provide a `keep` allowlist in config so known-reflective entrypoints are treated as roots and never reported.
- Test-only helpers. Exclude `_test.go` from both the dead set and the root set unless configured otherwise.

## 9. Functional requirements (MVP)

1. Accept two refs (`--base`, `--head`), default base to the merge target branch.
2. Build both graphs by reusing the archview engine. No reimplementation.
3. Resolve roots from config and run reachability on each graph.
4. Diff with stable node IDs. Surface, in priority order:
   - **Newly dead** nodes (orphaned by this PR). Highest signal alongside layer-crossing.
   - New layer-crossing edges (e.g. handler to repo directly).
   - Removed layer-crossing edges (e.g. a service hop that disappeared).
   - **Revived** nodes (dead -> reachable).
   - Changed nodes (body hash differs).
   - New and removed nodes.
5. Tag interface edges with `Kind: interface` so they can be filtered. Interface edges inherit archview's call-resolution imprecision.
6. Honor a `keep` allowlist and a vendor/codegen ignore-glob so reflective entrypoints and generated code are not falsely reported as dead.
7. Render a markdown report plus a mermaid subgraph of the changed region, with dead nodes styled distinctly.
8. Provide `--dead` full audit mode (HEAD-only reachability, grouped by layer).
9. Ship a GitHub Action that posts the report as a PR comment and updates it in place on new pushes.
10. Cache serialized graphs and reachable-sets by SHA.

## 10. Output spec

```markdown
## Arch-diff: #142

Newly dead (orphaned by this PR)
  InventoryService.Reserve   (was reached via OrderHandler.Create)
  InventoryService.rollback

New layer-crossing edge
  OrderHandler.Create -> InventoryRepo.Decrement
  was: OrderHandler.Create -> InventoryService.Reserve -> InventoryRepo.Decrement

Changed (body)
  PaymentService.Charge

Revived
  LegacyMigrator.Run   (now reached via AdminHandler.Migrate)

New node
  WebhookHandler.Handle
```

Followed by a mermaid block of the affected subgraph, with dead nodes rendered in a muted style.

When there is no structural change and nothing was orphaned, the report says so in one line.

## 11. Layer and root resolution

Resolved from a config file, reused from archview where possible.

```yaml
layers:
  handler:  "internal/**/handler"
  service:  "internal/**/service"
  repo:     "internal/**/repo"
  domain:   "internal/domain/**"
  infra:    "internal/infra/**"

roots:
  routes: true            # treat registered HTTP routes as roots
  main:   true            # treat func main as root
  exported_api: false     # set true for library modules
  keep:                   # allowlist for reflective/known entrypoints
    - "internal/jobs/*.Register"

ignore:
  - "**/*_test.go"
  - "**/mock_*.go"
  - "vendor/**"
```

An edge is "layer-crossing" when From.Layer and To.Layer differ. A node is dead when not reachable from any resolved root.

## 12. CLI surface

```
arch-diff --base origin/main --head HEAD
arch-diff --base <sha> --head <sha> --format markdown
arch-diff --base origin/main --head HEAD --only-crossing   # filter to layer-crossing
arch-diff --base origin/main --head HEAD --only-dead        # filter to newly-dead only
arch-diff --dead --head HEAD                                # full dead-code audit, no diff
```

Exit code is always 0 in v1 (report-only, no gating).

## 13. Milestones

- **M1** Two refs to two graphs via worktree plus archview, serialized to JSON. Local CLI prints raw added/removed/changed.
- **M2** Stable node IDs verified: line shifts produce empty diff. Layer resolution and layer-crossing filter.
- **M3** Root resolution plus reachability pass. `--dead` audit mode. Newly-dead and revived in the diff.
- **M4** Markdown plus mermaid renderer with dead-node styling.
- **M5** GitHub Action with in-place PR comment, SHA cache for graphs and reachable-sets.

## 14. Definition of Done

- [ ] Adding a blank line above a function produces an empty structural diff.
- [ ] Moving a method to a different file with no logic change produces an empty diff.
- [ ] Introducing a handler-to-repo direct call surfaces as a new layer-crossing edge.
- [ ] Removing a service hop surfaces the before/after path.
- [ ] Changing only a function body marks exactly that node as changed and nothing else.
- [ ] A PR that removes the last caller of a function reports that function as newly dead.
- [ ] A function reachable only from a `keep` allowlist entry is never reported as dead.
- [ ] Exported API is reported as dead only when `exported_api` is false.
- [ ] `_test.go` and ignored globs never appear in the dead set.
- [ ] `--dead` lists all unreachable nodes grouped by layer with no diff.
- [ ] Interface edges are tagged and filterable.
- [ ] Base graph and reachable-set are computed once and served from cache on the second run with the same SHA.
- [ ] GitHub Action posts one comment per PR and edits it in place on subsequent pushes.
- [ ] A PR with no structural change and no orphaning yields a single-line report.

## 15. Risks

- **Interface edge precision** follows archview's call resolution. CHA/RTA over-approximates dynamic dispatch. Over-approximation is safe for dead code (it under-reports dead, never falsely kills live code) but can hide truly-dead interface impls. Acceptable, documented.
- **Reflection and init side effects** are invisible to static analysis and can cause false dead reports. Mitigation: `keep` allowlist.
- **Rename appears as delete plus add** in v1. Accepted. Similarity matching is a v2 concern.
- **Build cost** of two full type-checks. Mitigation: SHA cache and base-branch reuse.
- **Root misconfiguration** is the main failure mode for dead code. If routes are not detected as roots, half the codebase looks dead. M3 must validate that the root set is non-empty and warn loudly otherwise.
