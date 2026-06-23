// Package model is the language-neutral structural graph arch-diff compares
// across two git refs. It mirrors the PRD §7 data model. Node identity is
// location-independent (package path + receiver + method) so that line shifts
// produce no diff; only genuine structural change is visible.
package model

// Node is one function/method in the call graph.
type Node struct {
	ID    string // stable location-independent key, e.g. pkg/path.(*OrderHandler).Create
	Layer string // handler | service | repo | domain | infra (resolved from path rules)
	Hash  string // hash of the normalized AST body; detects "changed" nodes
	Root  bool   // entrypoint (route, main, cron, exported API)
}

// Edge is a call from one node to another.
type Edge struct {
	From string // Node.ID
	To   string // Node.ID
	Kind string // static | interface
}

// Graph is one revision's structural graph plus its derived reachable-set.
type Graph struct {
	SHA       string
	Nodes     map[string]Node
	Edges     []Edge
	Reachable map[string]bool // node ID -> reachable from any root (computed by package reach)
}

// Edge kinds.
const (
	EdgeStatic    = "static"
	EdgeInterface = "interface"
)

// Layers.
const (
	LayerHandler = "handler"
	LayerService = "service"
	LayerRepo    = "repo"
	LayerDomain  = "domain"
	LayerInfra   = "infra"
)
