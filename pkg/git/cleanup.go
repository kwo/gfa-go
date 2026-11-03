package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type dirtyFile struct {
	status string
	path   string
}

func getDirtyFiles(ctx context.Context, dir string) ([]dirtyFile, error) {
	output, err := runGit(ctx, dir, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	return parseDirtyFiles(output)
}

func parseDirtyFiles(output string) ([]dirtyFile, error) {
	var files []dirtyFile
	entries := strings.Split(output, "\x00")
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		if entry == "" {
			continue
		}
		if len(entry) < 4 || entry[2] != ' ' {
			return nil, fmt.Errorf("invalid Git status entry: %q", entry)
		}
		files = append(files, dirtyFile{status: entry[:2], path: entry[3:]})
		if strings.ContainsAny(entry[:2], "RC") {
			i++
			if i >= len(entries) || entries[i] == "" {
				return nil, fmt.Errorf("missing Git rename source")
			}
			files = append(files, dirtyFile{status: entry[:2], path: entries[i]})
		}
	}
	return files, nil
}

func restorePackageLocks(ctx context.Context, dir string) (string, error) {
	files, err := getDirtyFiles(ctx, dir)
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "unchanged", nil
	}
	var tracked, untracked []string
	for _, file := range files {
		// Renames, copies and unresolved conflicts need a deliberate manual fix.
		if filepath.Base(file.path) != "package-lock.json" || strings.ContainsAny(file.status, "RCU") || file.status == "AA" || file.status == "DD" {
			return "skipped", nil
		}
		if file.status == "??" {
			untracked = append(untracked, file.path)
		} else {
			tracked = append(tracked, file.path)
		}
	}
	if len(tracked) > 0 {
		args := append([]string{"--literal-pathspecs", "restore", "--staged", "--worktree", "--"}, tracked...)
		if _, err := runGit(ctx, dir, args...); err != nil {
			return "", err
		}
	}
	if len(untracked) > 0 {
		args := append([]string{"--literal-pathspecs", "clean", "-f", "--"}, untracked...)
		if _, err := runGit(ctx, dir, args...); err != nil {
			return "", err
		}
	}
	return "restored", nil
}

func originDefaultBranch(ctx context.Context, dir string) (string, error) {
	args := []string{"symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD"}
	output, err := runGit(ctx, dir, args...)
	if err != nil {
		if _, err := runGit(ctx, dir, "remote", "set-head", "origin", "--auto"); err != nil {
			return "", err
		}
		output, err = runGit(ctx, dir, args...)
		if err != nil {
			return "", err
		}
	}
	branch := strings.TrimSpace(strings.TrimPrefix(output, "origin/"))
	if branch == "" {
		return "", fmt.Errorf("origin default branch is empty")
	}
	return branch, nil
}

func switchToDefault(ctx context.Context, dir, branch string) error {
	_, err := runGit(ctx, dir, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	if err == nil {
		_, err = runGit(ctx, dir, "switch", "--", branch)
	} else if exitCode(err) == 1 {
		_, err = runGit(ctx, dir, "switch", "--track", "-c", branch, "origin/"+branch)
	}
	return err
}

func deleteMergedBranches(ctx context.Context, dir, current, defaultBranch string) (string, error) {
	all, err := runGit(ctx, dir, "branch", "--format=%(refname:short)")
	if err != nil {
		return "", err
	}
	merged, err := runGit(ctx, dir, "branch", "--format=%(refname:short)", "--merged", "origin/"+defaultBranch)
	if err != nil {
		return "", err
	}
	mergedSet := make(map[string]bool)
	for _, branch := range strings.Fields(merged) {
		mergedSet[branch] = true
	}
	deleted, skipped, unmerged := 0, 0, 0
	for _, branch := range strings.Fields(all) {
		if branch == current || branch == defaultBranch {
			continue
		}
		if !mergedSet[branch] {
			unmerged++
			continue
		}
		if _, err := runGit(ctx, dir, "branch", "-d", "--", branch); err != nil {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			skipped++
		} else {
			deleted++
		}
	}
	result := fmt.Sprintf("deleted %d", deleted)
	if skipped > 0 {
		result += fmt.Sprintf("; skipped %d", skipped)
	}
	return result + fmt.Sprintf("; unmerged %d", unmerged), nil
}

func exitCode(err error) int {
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode()
	}
	return -1
}
