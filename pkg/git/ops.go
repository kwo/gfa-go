package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/kwo/gfa/pkg/tui"
)

//nolint:maintidx // Keep the safety checks in Git operation order.
func processRepo(ctx context.Context, ui Reporter, repo tui.RepoState, options Options) error {
	update := func(u tui.RepoUpdate) { ui.UpdateRepo(repo.Directory, u) }
	var cleanup []string
	addCleanup := func(message string) {
		cleanup = append(cleanup, message)
		update(tui.RepoUpdate{Cleanup: tui.StringValue(strings.Join(cleanup, "; "))})
	}
	update(tui.RepoUpdate{Status: tui.StringValue("processing"), Action: tui.StringValue("checking")})
	if !isGitRepo(repo.Directory) {
		update(tui.RepoUpdate{Branch: tui.StringValue(""), Status: tui.StringValue("no git"), Remote: tui.StringValue(""), Action: tui.StringValue("")})
		return nil
	}
	branch, err := currentBranch(ctx, repo.Directory)
	if err != nil {
		return err
	}
	update(tui.RepoUpdate{Branch: tui.StringValue(branch)})
	if options.RestorePackageLock {
		result, err := restorePackageLocks(ctx, repo.Directory)
		if err != nil {
			return err
		}
		addCleanup("package-lock " + result)
	}
	dirty, err := dirtyStatus(ctx, ui, repo.Directory)
	if err != nil {
		return err
	}
	remotes, err := runGit(ctx, repo.Directory, "remote")
	if err != nil {
		return err
	}
	if strings.TrimSpace(remotes) == "" {
		update(tui.RepoUpdate{Remote: tui.StringValue("no remotes"), Action: tui.StringValue("")})
		return nil
	}
	if _, err := runGit(ctx, repo.Directory, "fetch", "--all", "--tags", "--prune", "--prune-tags", "--quiet"); err != nil {
		return err
	}
	update(tui.RepoUpdate{Action: tui.StringValue("fetched")})

	var defaultBranch string
	if (options.SwitchDefaultBranch || options.DeleteMergedBranches) && !dirty {
		defaultBranch, err = originDefaultBranch(ctx, repo.Directory)
		if err != nil {
			return err
		}
	}
	if options.SwitchDefaultBranch {
		switch {
		case dirty:
			addCleanup("dirty; skipped switch")
		case branch == defaultBranch:
			addCleanup("on default")
		default:
			if err := switchToDefault(ctx, repo.Directory, defaultBranch); err != nil {
				return err
			}
			branch = defaultBranch
			update(tui.RepoUpdate{Branch: tui.StringValue(branch)})
			addCleanup("switched " + branch)
			// A checkout hook can change the working tree.
			dirty, err = dirtyStatus(ctx, ui, repo.Directory)
			if err != nil {
				return err
			}
		}
	}
	remote, needPull, err := remoteStatus(ctx, repo.Directory, branch)
	if err != nil {
		return err
	}
	update(tui.RepoUpdate{Remote: tui.StringValue(remote)})
	if !dirty && needPull {
		if _, err := runGit(ctx, repo.Directory, "pull", "--ff-only"); err != nil {
			return err
		}
		update(tui.RepoUpdate{Action: tui.StringValue("pulled")})
		remote, _, err = remoteStatus(ctx, repo.Directory, branch)
		if err != nil {
			return err
		}
		update(tui.RepoUpdate{Remote: tui.StringValue(remote)})
		// A merge hook can also change the working tree.
		dirty, err = dirtyStatus(ctx, ui, repo.Directory)
		if err != nil {
			return err
		}
	}
	if options.DeleteMergedBranches {
		if dirty {
			addCleanup("dirty; skipped delete")
			return nil
		}
		result, err := deleteMergedBranches(ctx, repo.Directory, branch, defaultBranch)
		if err != nil {
			return err
		}
		addCleanup(result)
	}
	return nil
}

func currentBranch(ctx context.Context, dir string) (string, error) {
	output, err := runGit(ctx, dir, "branch", "--show-current")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(output) == "" {
		return "(detached)", nil
	}
	return strings.TrimSpace(output), nil
}

func dirtyStatus(ctx context.Context, ui Reporter, dir string) (bool, error) {
	files, err := getDirtyFiles(ctx, dir)
	if err != nil {
		return false, err
	}
	status := "clean"
	if len(files) > 0 {
		status = "dirty"
	}
	ui.UpdateRepo(dir, tui.RepoUpdate{Status: tui.StringValue(status)})
	return len(files) > 0, nil
}

func remoteStatus(ctx context.Context, dir, branch string) (string, bool, error) {
	if branch == "(detached)" {
		return "no remote branch", false, nil
	}
	upstream, err := runGit(ctx, dir, "for-each-ref", "--format=%(upstream)", "refs/heads/"+branch)
	if err != nil {
		return "", false, err
	}
	upstream = strings.TrimSpace(upstream)
	if upstream == "" {
		return "no remote branch", false, nil
	}
	if _, err := runGit(ctx, dir, "show-ref", "--verify", "--quiet", upstream); err != nil {
		if exitCode(err) == 1 {
			return "gone", false, nil
		}
		return "", false, err
	}
	output, err := runGit(ctx, dir, "rev-list", "--left-right", "--count", "HEAD..."+upstream, "--")
	if err != nil {
		return "", false, err
	}
	counts := strings.Fields(output)
	if len(counts) != 2 {
		return "", false, fmt.Errorf("invalid Git ahead/behind counts: %q", output)
	}
	ahead, err := strconv.Atoi(counts[0])
	if err != nil {
		return "", false, err
	}
	behind, err := strconv.Atoi(counts[1])
	if err != nil {
		return "", false, err
	}
	var parts []string
	if ahead > 0 {
		parts = append(parts, fmt.Sprintf("ahead %d", ahead))
	}
	if behind > 0 {
		parts = append(parts, fmt.Sprintf("behind %d", behind))
	}
	if len(parts) == 0 {
		return "up-to-date", false, nil
	}
	return strings.Join(parts, ", "), behind > 0, nil
}

func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	cmd.WaitDelay = time.Second
	configureCancellation(cmd)
	// Keep stderr out of machine-readable stdout, even on success.
	var stderr strings.Builder
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return string(output), nil
}

func setRepoError(ui Reporter, directory string, err error) {
	ui.UpdateRepo(directory, tui.RepoUpdate{
		Status: tui.StringValue("error"), Action: tui.StringValue("error"), Error: tui.StringValue(err.Error()),
	})
}
