package loader_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eularixs/arch-diff/internal/config"
	"github.com/eularixs/arch-diff/internal/diff"
	"github.com/eularixs/arch-diff/internal/loader"
	"github.com/eularixs/arch-diff/internal/model"
	"github.com/eularixs/arch-diff/internal/reach"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func commit(t *testing.T, dir, msg string) string {
	t.Helper()
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", msg)
	return git(t, dir, "rev-parse", "HEAD")
}

func graphAt(t *testing.T, repo, ref string, cfg config.Config) *model.Graph {
	t.Helper()
	wt, cleanup, err := loader.AddWorktree(repo, ref)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	g, err := loader.BuildGraph(wt, cfg)
	if err != nil {
		t.Fatal(err)
	}
	reach.Compute(g, reach.Roots(g, cfg))
	return g
}

func hasSuffix(ss []string, suf string) bool {
	for _, s := range ss {
		if strings.HasSuffix(s, suf) {
			return true
		}
	}
	return false
}

const gomod = "module example.com/fix\n\ngo 1.25\n"

// base: ListUsers (handler) -> Service.List -> Repo.FindAll.
const base = `package main

import "net/http"

type Repo struct{}

func (r *Repo) FindAll() int { return 1 }

type Service struct{ repo *Repo }

func (s *Service) List() int { return s.repo.FindAll() }

type Handler struct {
	svc  *Service
	repo *Repo
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) { _ = h.svc.List() }

func main() {
	h := &Handler{svc: &Service{}, repo: &Repo{}}
	http.HandleFunc("/users", h.ListUsers)
	http.ListenAndServe(":0", nil)
}
`

func TestE2E_StructuralDiff(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "go.mod", gomod)
	write(t, dir, "main.go", base)
	git(t, dir, "init", "-q")
	baseSHA := commit(t, dir, "base")
	cfg := config.Default()

	// head1: blank line above Service.List + Repo.FindAll body change (1 -> 2).
	head1 := strings.Replace(base, "func (s *Service) List()", "\nfunc (s *Service) List()", 1)
	head1 = strings.Replace(head1, "func (r *Repo) FindAll() int { return 1 }", "func (r *Repo) FindAll() int { return 2 }", 1)
	write(t, dir, "main.go", head1)
	commit(t, dir, "head1")

	b := graphAt(t, dir, baseSHA, cfg)
	h := graphAt(t, dir, "HEAD", cfg)
	r := diff.Diff(b, h, cfg)
	if n := len(r.NewCrossing) + len(r.RemovedCrossing) + len(r.NewlyDead) + len(r.Added) + len(r.Removed); n != 0 {
		t.Fatalf("blank line + body change must not change structure: %+v", r)
	}
	if len(r.Changed) != 1 || !strings.HasSuffix(r.Changed[0], ".FindAll") {
		t.Fatalf("expected exactly Repo.FindAll Changed, got %v", r.Changed)
	}

	// head2 (from base): handler skips the service, calling the repo directly.
	head2 := strings.Replace(base, "_ = h.svc.List()", "_ = h.repo.FindAll()", 1)
	write(t, dir, "main.go", head2)
	commit(t, dir, "head2")

	h2 := graphAt(t, dir, "HEAD", cfg)
	r2 := diff.Diff(b, h2, cfg)

	foundCross := false
	for _, e := range r2.NewCrossing {
		if strings.HasSuffix(e.From, ".ListUsers") && strings.HasSuffix(e.To, ".FindAll") {
			foundCross = true
		}
	}
	if !foundCross {
		t.Fatalf("expected new crossing ListUsers->FindAll, got %v", r2.NewCrossing)
	}
	if !hasSuffix(r2.NewlyDead, ".List") {
		t.Fatalf("Service.List should be newly dead, got %v", r2.NewlyDead)
	}
}
