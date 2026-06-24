package git

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/kwo/gitterbug/pkg/tui"
)

func processRepo(ctx context.Context, ui *tui.TUI, repo tui.RepoState) {
	ui.UpdateRepo(repo.Directory, tui.RepoUpdate{
		Status: tui.StringValue("processing"),
		Action: tui.StringValue("checking"),
	})

	if !isGitRepo(repo.Directory) {
		ui.UpdateRepo(repo.Directory, tui.RepoUpdate{
			Branch: tui.StringValue(""),
			Status: tui.StringValue("no git"),
			Remote: tui.StringValue(""),
			Action: tui.StringValue(""),
		})
		return
	}

	branch, err := runGit(ctx, repo.Directory, "branch", "--show-current")
	if err != nil {
		setRepoError(ui, repo.Directory, err)
		return
	}
	ui.UpdateRepo(repo.Directory, tui.RepoUpdate{Branch: tui.StringValue(strings.TrimSpace(branch))})

	status, err := runGit(ctx, repo.Directory, "status", "--porcelain")
	if err != nil {
		setRepoError(ui, repo.Directory, err)
		return
	}
	dirty := strings.TrimSpace(status) != ""
	if dirty {
		ui.UpdateRepo(repo.Directory, tui.RepoUpdate{Status: tui.StringValue("dirty")})
	} else {
		ui.UpdateRepo(repo.Directory, tui.RepoUpdate{Status: tui.StringValue("clean")})
	}

	remotes, err := runGit(ctx, repo.Directory, "remote")
	if err != nil {
		setRepoError(ui, repo.Directory, err)
		return
	}
	if strings.TrimSpace(remotes) == "" {
		ui.UpdateRepo(repo.Directory, tui.RepoUpdate{
			Remote: tui.StringValue("no remotes"),
			Action: tui.StringValue(""),
		})
		return
	}

	_, err = runGit(ctx, repo.Directory, "fetch", "--all", "--tags", "--prune", "--prune-tags", "--quiet")
	if err != nil {
		setRepoError(ui, repo.Directory, err)
		return
	}
	ui.UpdateRepo(repo.Directory, tui.RepoUpdate{Action: tui.StringValue("fetched")})

	branchStatus, err := runGit(ctx, repo.Directory, "--no-pager", "branch", "-vv")
	if err != nil {
		setRepoError(ui, repo.Directory, err)
		return
	}
	remote, hasRemoteBranch := parseRemoteStatus(branchStatus)
	if !hasRemoteBranch {
		ui.UpdateRepo(repo.Directory, tui.RepoUpdate{Remote: tui.StringValue("no remote branch")})
		return
	}
	ui.UpdateRepo(repo.Directory, tui.RepoUpdate{Remote: tui.StringValue(remote)})

	if dirty || !strings.Contains(remote, "behind") {
		return
	}

	_, err = runGit(ctx, repo.Directory, "pull", "--ff-only")
	if err != nil {
		setRepoError(ui, repo.Directory, err)
		return
	}
	ui.UpdateRepo(repo.Directory, tui.RepoUpdate{Action: tui.StringValue("pulled")})
}

func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(output))
		if text == "" {
			return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
		}
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, text)
	}
	return string(output), nil
}

func setRepoError(ui *tui.TUI, directory string, err error) {
	ui.UpdateRepo(directory, tui.RepoUpdate{
		Status: tui.StringValue("error"),
		Action: tui.StringValue(err.Error()),
	})
}

var remoteStatusPattern = regexp.MustCompile(`\[\s*([^:\]]+)(:\s*([^\]]+)\s*)?\]`)

func parseRemoteStatus(output string) (string, bool) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "*") {
			continue
		}

		match := remoteStatusPattern.FindStringSubmatch(line)
		if match == nil {
			return "", false
		}

		status := strings.TrimSpace(match[3])
		if status == "" {
			return "up-to-date", true
		}
		return status, true
	}

	return "", false
}
