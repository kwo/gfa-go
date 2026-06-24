# Porting GFA to Gitterbug

This document tracks the work needed to bring `gitterbug` up to parity with `gfa`.

## Current state

`gitterbug` is a Go/Charm prototype. It has:

- a Go CLI entrypoint
- `--help`, `--version`, and configurable log level
- Bubble Tea / Lip Gloss / Bubbles dependencies
- signal-aware cancellation
- a status line and per-directory spinner lines
- immediate-child `.git` directory detection
- concurrent processing scaffolding

It does not yet implement the core `gfa` Git workflow.

## GFA parity target

The TypeScript `gfa` currently provides:

- scan current directory
- if current directory is a Git repository, process `.`
- otherwise process immediate child directories
- detect Git repositories via `.git`
- display a table with columns:
  - directory
  - branch
  - working-tree status
  - remote tracking status
  - action
- run Git operations per repository:
  - `git branch --show-current`
  - `git status --porcelain`
  - `git remote`
  - `git fetch --all --tags --prune --prune-tags --quiet`
  - `git --no-pager branch -vv`
  - `git pull` when safe
- avoid pulling dirty repositories
- avoid pulling repositories without remotes or upstream branches
- process repositories in parallel
- color-code output states
- quit with `q`, Escape, or Ctrl-C
- provide release/build/Homebrew tooling

## Porting phases

### Phase 0: Shore up the Bubble Tea model

Status: completed in the first cleanup pass. External TUI calls now dispatch Tea messages through the update loop instead of mutating model fields directly.

Before porting behavior, the TUI should follow Bubble Tea ownership rules:

- no direct mutation of Bubble Tea model state from worker goroutines
- external updates should be sent as Tea messages
- process rows should be added through model updates
- status changes should flow through the update loop
- per-process status changes should flow through the update loop
- the public TUI API may remain small, but it should dispatch messages rather than mutate state directly

Success criteria:

- `go test ./...` passes
- no externally-called method mutates `model.status` or `model.processes` directly
- `git.Run` can continue using `ui.SetStatus`, `ui.AddProcess`, and `line.SetStatus`

### Phase 1: Introduce repository state

Status: completed in the first porting pass. The TUI now owns repository rows as state and renders a table from `RepoState` values. Worker updates are addressable by directory.

Add a repository model equivalent to GFA's `RepoState`:

```go
type RepoState struct {
    Directory string
    Branch    string
    Status    string
    Remote    string
    Action    string
}
```

Success criteria:

- TUI renders repository rows from state rather than free-form progress strings
- row updates are addressable by directory or stable ID

### Phase 2: Match GFA scanning behavior

Status: completed in the second porting pass. Scanning now returns `.` when the current directory is a Git repository, otherwise immediate child directories only.

Implement scanner behavior:

- read current working directory
- if current directory contains `.git`, process `.` only
- otherwise process immediate child directories
- preserve the existing note that recursive scanning can be added later, but do not silently change behavior beyond GFA parity

Success criteria:

- running in a Git repository shows one row for `.`
- running in a parent directory shows one row per immediate child directory

### Phase 3: Port Git operations

Port the GFA per-repository workflow:

1. check whether the directory is a Git repository
2. read current branch
3. check dirty/clean status
4. check whether remotes exist
5. fetch all remotes/tags with prune flags
6. parse remote tracking state from `git --no-pager branch -vv`
7. pull only when the repository is clean and behind upstream

Success criteria:

- clean up-to-date repository shows `clean`, `up-to-date`, `fetched`
- dirty repository shows `dirty` and never pulls
- repository without remote shows `no remotes`
- repository without upstream shows `no remote branch`
- behind clean repository pulls
- Git errors are captured per row without aborting all processing

### Phase 4: Replace spinner output with table output

Render a GFA-like table in Bubble Tea/Lip Gloss:

- dynamic column widths
- header row
- color-coded status/remote/action cells
- stable row ordering
- final status line

Success criteria:

- visible output is functionally equivalent to GFA
- no ANSI formatting leaks into business logic

### Phase 5: Build and release parity

Add Go-native release tooling:

- build script or Makefile target
- cross-compilation notes
- GitHub Actions release workflow
- checksums
- Homebrew formula generation/update instructions

Success criteria:

- tagged release can produce installable binaries
- Homebrew tap can install `gitterbug` or renamed `gfa` replacement

## Open naming decision

Before release work, decide whether the Go port remains `gitterbug` or becomes the Go implementation of `gfa`.

For now, keep the code and module named `gitterbug` and focus on behavioral parity.
