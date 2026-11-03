# Go port implementation

The Go CLI supports the TypeScript GFA workflow and cleanup flags. The implementation keeps the installed Git executable as its Git backend.

## Lifecycle

The TUI owns the worker context. Workers send repository updates through a queue. A completion message follows all updates in that queue.

Normal completion exits Bubble Tea without cancellation. The CLI then prints the final report and returns the worker result. Cancellation stops the workers before the CLI exits.

On Unix, each Git subprocess has its own process group. Cancellation kills that group, including child processes such as SSH and credential helpers.

## Display

Column widths depend on the terminal width, not repository state. A repository update cannot move the columns.

The renderer measures terminal cells rather than UTF-8 bytes. Table cells contain one line without terminal control codes. Long cells use an ellipsis.

The live viewport responds to terminal resize events and keyboard scrolling. The final report includes all rows. Full errors and cleanup results appear outside the table.

Redirected output bypasses Bubble Tea and prints one plain report. Color selection does not require terminal queries.

## Git operations

Repository discovery recognizes `.git` directories and files. It processes either the current repository or immediate child directories.

A worker pool limits processing to eight repositories. Errors remain local to their rows. The command returns an error after the other repositories finish.

Git status uses NUL-delimited porcelain output. Rename sources remain separate from destination paths. Restore commands treat paths as literals.

Upstream detection uses Git reference fields. Ahead/behind counts come from `git rev-list`, not human-readable branch descriptions.

## Cleanup and safety

The CLI supports `-R`, `-s`, `-d`, and packed forms such as `-Rsd`. The `-v` alias prints the version. Repeatable `--verbose` flags retain the prototype verbosity levels.

Lock-file restore skips changes to other files, renames, copies, and conflicts. It restores tracked lock files and removes untracked lock files only after that check.

Branch switching and branch deletion require a clean working tree. GFA checks the working tree again after a switch or pull because hooks can change files.

Merged-branch cleanup compares local branches with the origin default branch. It excludes the current and default branches and uses `git branch -d` only.

## Deliberate differences from TypeScript

- Pulls always use `--ff-only`.
- Linked worktrees are supported.
- Concurrency is limited to eight repositories.
- Repository errors produce exit code `1`.
- Cancellation produces exit code `130`.
- Unknown flags and positional arguments produce an error.
- The display supports terminal resizing and scrolling.

## Build tools

The project requires Go 1.27.0 or later. `Taskfile.yml` provides the build interface, with Task and golangci-lint pinned in the root Go module.

The default task runs lint, tests, and compilation in order. `build-go` compiles without checks. No task installs the executable.

Tool dependencies share the application module graph. Two replacement directives retain the tested Bubble Tea and Lip Gloss versions. Newer transitive versions caused incompatible cell-buffer APIs and terminal startup queries.

Tool updates require the terminal regression tests as well as the normal build checks. The tooling packages are not part of the application executable.

## Verification

Tests cover table sizing, Unicode, stable columns, multiline errors, scrolling, completion, and cancellation.

Temporary Git fixtures cover clean, dirty, behind, diverged, detached, missing-upstream, and deleted-upstream states. Cleanup tests cover lock files, default-branch discovery, safe deletion, and linked worktrees.

CLI tests build the executable and examine help, version aliases, packed flags, redirected output, final results, and error exits.

## Release boundary

Local implementation and verification do not publish a release or replace an installed executable. The existing release tooling targets GitHub, not Forgejo. Select the release destination before publication.
