package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/kwo/gfa/pkg/tui"
)

type recordingUI struct {
	mu    sync.Mutex
	repos map[string]tui.RepoState
}

func newRecordingUI() *recordingUI    { return &recordingUI{repos: make(map[string]tui.RepoState)} }
func (*recordingUI) SetStatus(string) {}
func (r *recordingUI) SetRepos(repos []tui.RepoState) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, repo := range repos {
		r.repos[repo.Directory] = repo
	}
}

func (r *recordingUI) UpdateRepo(dir string, u tui.RepoUpdate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	repo := r.repos[dir]
	repo.Directory = dir
	for _, field := range []struct {
		dst *string
		src *string
	}{
		{&repo.Branch, u.Branch},
		{&repo.Status, u.Status},
		{&repo.Remote, u.Remote},
		{&repo.Action, u.Action},
		{&repo.Cleanup, u.Cleanup},
		{&repo.Error, u.Error},
	} {
		if field.src != nil {
			*field.dst = *field.src
		}
	}
	if u.Done != nil {
		repo.Done = *u.Done
	}
	r.repos[dir] = repo
}

func (r *recordingUI) get(dir string) tui.RepoState {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.repos[dir]
}

type fixture struct{ root, origin, seed, work string }

func newFixture(t *testing.T) fixture {
	t.Helper()
	root := t.TempDir()
	for key, value := range map[string]string{
		"GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": os.DevNull,
		"GIT_AUTHOR_NAME": "GFA Test", "GIT_AUTHOR_EMAIL": "gfa@example.invalid",
		"GIT_COMMITTER_NAME": "GFA Test", "GIT_COMMITTER_EMAIL": "gfa@example.invalid",
		"GIT_TERMINAL_PROMPT": "0",
	} {
		t.Setenv(key, value)
	}
	f := fixture{root: root, origin: filepath.Join(root, "origin.git"), seed: filepath.Join(root, "seed"), work: filepath.Join(root, "repos", "work")}
	mustGit(t, root, "init", "--bare", "-b", "main", f.origin)
	mustGit(t, root, "init", "-b", "main", f.seed)
	writeFile(t, filepath.Join(f.seed, "README.md"), "initial\n")
	writeFile(t, filepath.Join(f.seed, "package-lock.json"), "original lock\n")
	mustGit(t, f.seed, "add", ".")
	mustGit(t, f.seed, "commit", "-m", "initial")
	mustGit(t, f.seed, "remote", "add", "origin", f.origin)
	mustGit(t, f.seed, "push", "-u", "origin", "main")
	mustGit(t, root, "clone", f.origin, f.work)
	return f
}

func mustGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, output)
	}
	return strings.TrimSpace(string(output))
}

func writeFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func advanceOrigin(t *testing.T, f fixture) {
	t.Helper()
	writeFile(t, filepath.Join(f.seed, "upstream.txt"), "upstream change\n")
	mustGit(t, f.seed, "add", ".")
	mustGit(t, f.seed, "commit", "-m", "upstream")
	mustGit(t, f.seed, "push")
}

func process(t *testing.T, dir string, opts Options) (tui.RepoState, error) {
	t.Helper()
	ui := newRecordingUI()
	err := processRepo(context.Background(), ui, newRepoState(dir), opts)
	if err != nil {
		setRepoError(ui, dir, err)
	}
	return ui.get(dir), err
}

func TestProcessCleanFetchAndPull(t *testing.T) {
	f := newFixture(t)
	row, err := process(t, f.work, Options{})
	if err != nil || row.Status != "clean" || row.Remote != "up-to-date" || row.Action != "fetched" {
		t.Fatalf("row=%+v err=%v", row, err)
	}
	advanceOrigin(t, f)
	row, err = process(t, f.work, Options{})
	if err != nil || row.Action != "pulled" || row.Remote != "up-to-date" {
		t.Fatalf("row=%+v err=%v", row, err)
	}
	if mustGit(t, f.work, "rev-parse", "HEAD") != mustGit(t, f.seed, "rev-parse", "HEAD") {
		t.Fatal("clean repository did not pull")
	}
}

func TestProcessDirtyNeverPulls(t *testing.T) {
	f := newFixture(t)
	before := mustGit(t, f.work, "rev-parse", "HEAD")
	advanceOrigin(t, f)
	writeFile(t, filepath.Join(f.work, "README.md"), "my uncommitted work\n")
	row, err := process(t, f.work, Options{SwitchDefaultBranch: true, DeleteMergedBranches: true, RestorePackageLock: true})
	if err != nil || row.Status != "dirty" || row.Remote != "behind 1" || row.Action != "fetched" {
		t.Fatalf("row=%+v err=%v", row, err)
	}
	if !strings.Contains(row.Cleanup, "skipped switch") || !strings.Contains(row.Cleanup, "skipped delete") {
		t.Fatalf("missing dirty safeguards: %+v", row)
	}
	if mustGit(t, f.work, "rev-parse", "HEAD") != before {
		t.Fatal("dirty repository changed HEAD")
	}
}

func TestProcessDivergedNeverMerges(t *testing.T) {
	f := newFixture(t)
	advanceOrigin(t, f)
	writeFile(t, filepath.Join(f.work, "local.txt"), "local\n")
	mustGit(t, f.work, "add", ".")
	mustGit(t, f.work, "commit", "-m", "local")
	before := mustGit(t, f.work, "rev-parse", "HEAD")
	row, err := process(t, f.work, Options{})
	if err == nil || row.Remote != "ahead 1, behind 1" || row.Status != "error" {
		t.Fatalf("row=%+v err=%v", row, err)
	}
	if mustGit(t, f.work, "rev-parse", "HEAD") != before {
		t.Fatal("diverged repository changed HEAD")
	}
	if row.Action != "error" || row.Error == "" {
		t.Fatal("error summary and details were not separated")
	}
}

func TestProcessNoUpstreamDetachedAndGone(t *testing.T) {
	f := newFixture(t)
	mustGit(t, f.work, "branch", "--unset-upstream")
	// A bracketed commit subject must not be mistaken for an upstream.
	mustGit(t, f.work, "commit", "--allow-empty", "-m", "[ticket-123] no upstream")
	row, err := process(t, f.work, Options{})
	if err != nil || row.Remote != "no remote branch" {
		t.Fatalf("row=%+v err=%v", row, err)
	}
	mustGit(t, f.work, "switch", "--detach")
	row, err = process(t, f.work, Options{})
	if err != nil || row.Branch != "(detached)" || row.Remote != "no remote branch" {
		t.Fatalf("row=%+v err=%v", row, err)
	}
	mustGit(t, f.seed, "push", "origin", "main:temporary")
	mustGit(t, f.work, "fetch", "origin")
	mustGit(t, f.work, "switch", "--track", "-c", "temporary", "origin/temporary")
	mustGit(t, f.seed, "push", "origin", "--delete", "temporary")
	row, err = process(t, f.work, Options{})
	if err != nil || row.Remote != "gone" {
		t.Fatalf("row=%+v err=%v", row, err)
	}
}

func TestCombinedCleanupFlags(t *testing.T) {
	f := newFixture(t)
	mustGit(t, f.work, "branch", "merged")
	mustGit(t, f.work, "switch", "-c", "unmerged")
	writeFile(t, filepath.Join(f.work, "unmerged.txt"), "keep\n")
	mustGit(t, f.work, "add", ".")
	mustGit(t, f.work, "commit", "-m", "unmerged")
	mustGit(t, f.work, "switch", "-c", "feature", "main")
	writeFile(t, filepath.Join(f.work, "package-lock.json"), "discard lock\n")
	mustGit(t, f.work, "add", "package-lock.json")
	// Exercise discovery when origin/HEAD is absent.
	mustGit(t, f.work, "symbolic-ref", "--delete", "refs/remotes/origin/HEAD")
	row, err := process(t, f.work, Options{RestorePackageLock: true, SwitchDefaultBranch: true, DeleteMergedBranches: true})
	if err != nil || row.Branch != "main" || row.Status != "clean" {
		t.Fatalf("row=%+v err=%v", row, err)
	}
	for _, part := range []string{"package-lock restored", "switched main", "deleted 2", "unmerged 1"} {
		if !strings.Contains(row.Cleanup, part) {
			t.Fatalf("missing %q in %q", part, row.Cleanup)
		}
	}
	if got := mustGit(t, f.work, "branch", "--format=%(refname:short)"); got != "main\nunmerged" {
		t.Fatalf("unsafe branch cleanup: %s", got)
	}
	if got := mustGit(t, f.work, "status", "--porcelain"); got != "" {
		t.Fatalf("lock restore left dirty files: %s", got)
	}
}

func TestSwitchCreatesMissingLocalDefault(t *testing.T) {
	f := newFixture(t)
	mustGit(t, f.work, "branch", "-m", "topic")
	row, err := process(t, f.work, Options{SwitchDefaultBranch: true})
	if err != nil || row.Branch != "main" {
		t.Fatalf("row=%+v err=%v", row, err)
	}
	if got := mustGit(t, f.work, "rev-parse", "--abbrev-ref", "@{upstream}"); got != "origin/main" {
		t.Fatalf("wrong upstream: %s", got)
	}
}

func TestBranchCleanupProtectsCurrentDefaultAndWorktrees(t *testing.T) {
	f := newFixture(t)
	mustGit(t, f.work, "switch", "-c", "current")
	mustGit(t, f.work, "branch", "in-worktree")
	mustGit(t, f.work, "worktree", "add", filepath.Join(f.root, "linked"), "in-worktree")
	row, err := process(t, f.work, Options{DeleteMergedBranches: true})
	if err != nil || !strings.Contains(row.Cleanup, "skipped 1") {
		t.Fatalf("row=%+v err=%v", row, err)
	}
	for _, branch := range []string{"current", "main", "in-worktree"} {
		mustGit(t, f.work, "show-ref", "--verify", "refs/heads/"+branch)
	}
}

func TestLinkedWorktreeIsProcessed(t *testing.T) {
	f := newFixture(t)
	linked := filepath.Join(f.root, "linked")
	mustGit(t, f.work, "worktree", "add", "-b", "linked", linked)
	row, err := process(t, linked, Options{})
	if err != nil || row.Status != "clean" || row.Action != "fetched" {
		t.Fatalf("row=%+v err=%v", row, err)
	}
}

func TestRunReportsFailuresAndContinuesOtherRepositories(t *testing.T) {
	f := newFixture(t)
	root := filepath.Dir(f.work)
	bad := filepath.Join(root, "bad")
	mustGit(t, root, "clone", f.origin, bad)
	mustGit(t, bad, "remote", "set-url", "origin", filepath.Join(f.root, "missing.git"))
	if err := os.Mkdir(filepath.Join(root, "not-git"), 0o700); err != nil {
		t.Fatal(err)
	}
	noRemote := filepath.Join(root, "no-remote")
	mustGit(t, root, "init", "-b", "main", noRemote)
	t.Chdir(root)
	ui := newRecordingUI()
	err := Run(context.Background(), ui, Options{})
	if err == nil || err.Error() != "1 repositories failed" {
		t.Fatalf("wrong aggregate error: %v", err)
	}
	if row := ui.get("work"); row.Action != "fetched" || !row.Done {
		t.Fatalf("healthy repository did not finish: %+v", row)
	}
	if row := ui.get("bad"); row.Status != "error" || row.Error == "" || !row.Done {
		t.Fatalf("missing repository error: %+v", row)
	}
	if row := ui.get("no-remote"); row.Remote != "no remotes" || !row.Done {
		t.Fatalf("wrong no-remote state: %+v", row)
	}
	if row := ui.get("not-git"); row.Status != "no git" || !row.Done {
		t.Fatalf("wrong non-repo state: %+v", row)
	}
}
