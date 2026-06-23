package loader

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/eularixs/arch-diff/internal/config"
	"github.com/eularixs/arch-diff/internal/model"
)

// Graphs are deterministic per commit SHA and config, so a built graph is cached
// by (SHA, config fingerprint). The base branch rarely changes, so its graph is
// computed once and reused across runs (PRD §5). The derived reachable-set is
// cheap and recomputed, so it is not cached.

func cacheDir() string {
	if d := os.Getenv("ARCH_DIFF_CACHE"); d != "" {
		return d
	}
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "arch-diff", "graphs")
}

func cfgFingerprint(cfg config.Config) string {
	b, _ := json.Marshal(cfg)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:6])
}

// BuildGraphCached returns the graph for wt, served from the SHA cache when
// present. A bare SHA (no worktree) cannot be rebuilt, so callers still pass a
// live worktree; on a cache miss it is built and stored. Caching is skipped when
// ARCH_DIFF_NO_CACHE is set or the SHA is unknown.
func BuildGraphCached(wt Worktree, cfg config.Config) (*model.Graph, error) {
	if wt.SHA == "" || os.Getenv("ARCH_DIFF_NO_CACHE") != "" {
		return BuildGraph(wt, cfg)
	}
	key := wt.SHA + "-" + cfgFingerprint(cfg) + ".json"
	path := filepath.Join(cacheDir(), key)
	if data, err := os.ReadFile(path); err == nil {
		var g model.Graph
		if json.Unmarshal(data, &g) == nil && g.Nodes != nil {
			return &g, nil
		}
	}
	g, err := BuildGraph(wt, cfg)
	if err != nil {
		return nil, err
	}
	if data, err := json.Marshal(g); err == nil {
		_ = os.MkdirAll(cacheDir(), 0o755)
		_ = os.WriteFile(path, data, 0o644)
	}
	return g, nil
}
