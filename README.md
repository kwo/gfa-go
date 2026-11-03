# gfa

GFA fetches and updates Git repositories in the current directory. It shows repository states in a live terminal table.

The Go implementation uses Bubble Tea and Lip Gloss. It runs the installed `git` executable and does not require Node.js.

## Usage

Run GFA inside a repository or its parent directory:

```sh
gfa
```

Inside a repository, GFA processes that repository only. Otherwise, it processes immediate child directories. It recognizes ordinary repositories and linked worktrees. It does not scan recursively.

GFA processes up to eight repositories at once. For each repository, it reads the branch and working-tree state, then fetches all remotes:

```sh
git fetch --all --tags --prune --prune-tags --quiet
```

If the working tree is clean and the branch is behind its upstream, GFA runs `git pull --ff-only`. Diverged branches produce an error instead of a merge or rebase.

CAUTION: The fetch command prunes remote-tracking branches and local tags that no longer exist on the remotes.

## Cleanup flags

All cleanup flags are optional:

| Flag | Short form | Behavior |
| --- | --- | --- |
| `--restore-package-lock` | `-R` | Discard lock-file changes only when every dirty file is named `package-lock.json` |
| `--switch-default-branch` | `-s` | Switch clean repositories to the default branch of `origin` |
| `--delete-merged-branches` | `-d` | Delete local branches merged into `origin/<default-branch>` with `git branch -d` |
| `--version` | `-v` | Print the version and exit |
| `--help` | `-h` | Print command help |

Combine the short cleanup flags:

```sh
gfa -Rsd
```

CAUTION: The `-R` flag discards staged and unstaged lock-file changes. It also removes untracked lock files. Renames, copies, conflicts, and changes to other files cause GFA to skip the restore.

Cleanup runs in this order: restore lock files, fetch, switch branches, pull, then delete merged branches. Dirty repositories still fetch, but GFA skips branch switching, pulling, and branch deletion.

GFA reads the default branch from `origin/HEAD`. If that reference is absent, it asks `origin` for its default branch. Repositories without an `origin` cannot use default-branch cleanup.

Branch deletion protects the current branch and default branch. It never uses force deletion. Git also protects branches that other worktrees use. The cleanup report counts deleted, skipped, and unmerged branches.

## Terminal output

The normal table has four columns:

```text
REPO          BRANCH         SYNC   OP
gfa-go        main           =      F
baserss       main*          ↓2     F
homelab       main           =      P
old-project   feature/login  ↑1 ↓2  !
```

| Column | Meaning |
| --- | --- |
| `REPO` | Repository directory |
| `BRANCH` | Branch name, with `*` for a dirty working tree |
| `SYNC` | `=` up to date, `↑2` ahead two commits, `↓3` behind three commits |
| `OP` | `F` fetched, `P` pulled, `!` error, `…` pending or in progress |

In `SYNC`, `—` means no upstream and `?` means an unavailable or deleted upstream. In `OP`, `—` means no Git operation was required.

Cleanup flags add a fifth column:

```text
REPO     BRANCH  SYNC  OP  CLEANUP
gfa-go   main    =     F   R S D2 U1
```

Cleanup codes are `R` for restored locks, `S` for a branch switch, `D2` for two deleted branches, and `U1` for one unmerged branch. A `?` suffix marks a skipped action, such as `R?` or `S?`. `D?2` means Git refused to delete two branches. `—` means no change.

The table uses one-space gaps and stable widths. Long cells use an ellipsis, but a dirty branch retains its `*` marker. Colors reinforce the symbols but are not required to read them.

During processing, use these keys:

| Keys | Action |
| --- | --- |
| Up / Down, `k` / `j` | Scroll one repository |
| Page Up / Page Down | Scroll one page |
| Home / End | Show the first or last page |
| `q`, Escape, Ctrl-C | Cancel the run and stop Git subprocesses |

The footer shows completed/total repositories and counts for the `F`, `P`, and `!` results. Non-Git directories appear only as a footer count.

The final report includes every repository, including rows outside the live viewport, followed by a short legend. Full errors and cleanup skip reasons appear below the table. Successful cleanup is not repeated unless its compact result exceeds the available width. Full sync counts also appear below the table when they do not fit.

Redirected output contains one final report without animation or terminal control codes. Set `NO_COLOR=1` to disable colors in a terminal. GFA does not enable interactive Git credential prompts.

Exit codes are `0` for success, `1` for errors, and `130` for cancellation. One repository error does not stop other repositories.

## Compatibility

This implementation supports the TypeScript CLI cleanup flags and the `-v` version alias. Unlike the TypeScript implementation, pulls always require a fast-forward.

The previous Go prototype used `-v` for verbosity. Verbosity now uses the repeatable `--verbose` flag.

## Development

Requirements are Git and Go 1.27.0 or later. The module pins Task and golangci-lint. No separate tool installation is required.

Run lint and tests, then build the executable:

```sh
go tool task build
./dist/gfa --version
```

The default command, `go tool task`, runs the same checked build. It does not install the executable.

Run individual tasks:

```sh
go tool task --list
go tool task test
go tool task lint
go tool task build-go VERSION=v1.2.3
go tool task clean
```

The test task runs the suite twice. Git tests use temporary local repositories and do not contact remote servers. The first tool invocation can download Go modules.

See [PORTING.md](PORTING.md) for implementation notes and [BUILD.md](BUILD.md) for build and release instructions.
