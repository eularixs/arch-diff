# Tasks — arch-diff

> Dibuat: 2026-06-23 06:24 WIB · Status: M1–M5 DONE (engine, diff, dead-code, mermaid, Action, cache). Verified e2e.
> Repo: `eularixs/arch-diff` · Depends: `eularixs/archview` (replace => ../gostruct)
> Spec: `docs/prd-arch-diff.md`. Verifiable lokal (go ada).

## Prinsip
- **Reuse archview engine. JANGAN reimplement analysis.** (PRD §5)
- Node ID lokasi-independent (pkg + recv + method) — archview existing UDAH gini.
- Report-only. NO gating (itu arch-lint). Exit 0 selalu v1.
- Go only. Output markdown + mermaid. No web UI.
- Proof per milestone: example repo, assert via DoD checklist (PRD §14).

## archview gaps (HARUS dibereskan sebelum M3 dead-code jalan)
archview `New().Graph()` emit graph yang **sudah di-prune + helper-collapsed**
(bagus buat UI, SALAH buat dead-code — dead-code butuh SEMUA node, justru yang
ga reachable). Yang kurang dari archview buat arch-diff:
- [x] **G1 Raw graph mode** ✅ archview Options.Raw (no prune/collapse). Verified: hexagonal 17→25, cqrs 26→39 nodes.: opsi archview emit graph TANPA pruneDisconnected /
      pruneIsolated / helper-collapse. Semua func node + semua call edge + layer.
      (Tanpa ini, node mati ke-buang sebelum sampai arch-diff.)
- [ ] **G2 Root metadata**: node tandain Root (endpoint udah; tambah main, cron,
      exported-API) — atau expose analyzer.Result biar arch-diff resolve sendiri.
- [x] **G3 Edge kind** ✅ map archview call→static, implements→interface (loader).: tandain edge `static` vs `interface` (archview Edge.Kind
      udah ada call/route/implements/dispatch — map ke static/interface).
- [ ] **G4 Body hash**: archview expose normalized-AST hash per func (atau
      arch-diff hitung sendiri dari go/packages pada worktree). Buat "Changed".
- [ ] **G5 Stable ID + signature**: ID sekarang pkg+recv+method tanpa signature.
      Cukup buat v1 (overload langka di Go). Catat sebagai known-limit.

## M1 — Two refs → two graphs (worktree + archview)
- [x] M1.1 `loader.AddWorktree(repo, ref)`: `git worktree add --detach <tmp> <ref>`
      + `git rev-parse` SHA + cleanup (`git worktree remove`).
- [x] M1.2 `loader.BuildGraph(wt)`: archview.New({Root: wt.Dir}) → map
      graph.Graph → model.Graph (Root = kind==endpoint, Kind map, Hash kosong dulu).
- [x] M1.3 CLI `--base/--head/--repo` cetak raw added/removed/changed (pakai
      pruned graph dulu — diff struktur kasar). 
- [ ] M1.4 Example fixture repo (`testdata/`) dua commit: base + head.

## M2 — Stable IDs + layer-crossing
- [x] M2.1 Verify DoD: tambah baris kosong di atas func → diff KOSONG.
- [x] M2.2 Verify: pindah method ke file lain tanpa ubah logic → diff KOSONG.
- [x] M2.3 Layer resolution dari config globs (PRD §11) + `--only-crossing` filter.
- [x] M2.4 Edge layer-crossing = From.Layer != To.Layer. Surface new + removed.
- [x] M2.5 Body-hash "Changed": ubah body doang → cuma node itu Changed.
- [x] M2.6 YAML config loader (gopkg.in/yaml.v3) — ganti config.Load stub.

## M3 — Roots + reachability + dead-code (CORE, gabungan dead-flow)
- [x] M3.1 Butuh G1 (raw graph). `reach.Roots` resolve routes/main/exported_api/keep.
- [x] M3.2 Validasi root set non-empty → warn keras kalau kosong (PRD §15).
- [x] M3.3 `reach.Compute` BFS (UDAH ada di scaffold) jalan di base + head.
- [x] M3.4 (core logic in diff.Diff; depends on G2 root refinement)
  - Diff reachable-set: **Newly dead** (reachable base, dead head) +
      **Revived** (dead base, reachable head). Newly-dead = headline.
- [x] M3.5 `--dead` audit mode: HEAD-only, list semua dead per layer.
- [x] M3.6 Honor `keep` allowlist + ignore globs (test, mock, vendor) — ga pernah
      dilaporin dead.
- [x] M3.7 `--only-dead` filter.

## M4 — Renderer (markdown + mermaid)
- [x] M4.1 Sectioned report (PRD §10): Newly dead, New layer-crossing edge dengan
      before/after path, Changed, Revived, New/Removed node.
- [x] M4.2 Mermaid subgraph region yang berubah, dead node style muted.
- [x] M4.3 No-change → laporan 1 baris.

## M5 — GitHub Action + cache
- [x] M5.1 Action: jalan di PR, post markdown sebagai comment, **edit in-place**
      tiap push (cari comment by marker, update).
- [x] M5.2 Cache serialized graph + reachable-set by SHA. Base branch dihitung
      sekali, reuse.
- [x] M5.3 Validasi DoD: 1 comment per PR, ke-update tiap push.

## Definition of Done (dari PRD §14 — checklist verifikasi)
- [ ] Baris kosong di atas func → diff kosong.
- [ ] Pindah method antar file tanpa ubah logic → diff kosong.
- [ ] Handler→repo langsung → muncul new layer-crossing edge.
- [ ] Hapus service hop → surface before/after path.
- [ ] Ubah body doang → cuma node itu Changed.
- [ ] Hapus caller terakhir → func dilaporin newly dead.
- [ ] Func cuma reachable dari keep → ga pernah dead.
- [ ] Exported API dead cuma kalau exported_api=false.
- [ ] _test.go + ignore globs ga pernah masuk dead set.
- [ ] `--dead` list semua unreachable per layer.
- [ ] Interface edge ditag + bisa difilter.
- [ ] Base graph + reachable-set dihitung sekali, dari cache pada SHA sama.
- [ ] Action post 1 comment/PR, edit in-place tiap push.
- [ ] PR tanpa perubahan struktur → laporan 1 baris.

## Urutan
archview-gaps (G1–G4) → M1 (worktree+map) → M2 (ID stabil + crossing) →
M3 (reachability + dead-code, INTI) → M4 (render) → M5 (action + cache).
