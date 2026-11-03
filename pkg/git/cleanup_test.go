package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRestorePackageLocks(t *testing.T) {
	for _, mode := range []string{"unchanged", "unstaged", "staged", "deleted", "untracked", "staged-new", "mixed", "renamed", "literal-path"} {
		t.Run(mode, func(t *testing.T) {
			f := newFixture(t)
			lock := filepath.Join(f.work, "package-lock.json")
			want := "restored"
			switch mode {
			case "unchanged":
				want = "unchanged"
			case "unstaged":
				writeFile(t, lock, "changed\n")
			case "staged":
				writeFile(t, lock, "changed\n")
				mustGit(t, f.work, "add", "package-lock.json")
			case "deleted":
				if err := os.Remove(lock); err != nil {
					t.Fatal(err)
				}
			case "untracked":
				writeFile(t, filepath.Join(f.work, "nested dir", "package-lock.json"), "untracked\n")
			case "staged-new":
				writeFile(t, filepath.Join(f.work, "nested dir", "package-lock.json"), "staged new\n")
				mustGit(t, f.work, "add", ".")
			case "mixed":
				want = "skipped"
				writeFile(t, lock, "preserve lock\n")
				writeFile(t, filepath.Join(f.work, "README.md"), "preserve other change\n")
			case "renamed":
				want = "skipped"
				mustGit(t, f.work, "mv", "package-lock.json", "renamed.json")
			case "literal-path":
				writeFile(t, filepath.Join(f.work, "[abc]*\n dir", "package-lock.json"), "untracked\n")
			}
			before := mustGit(t, f.work, "status", "--porcelain")
			got, err := restorePackageLocks(context.Background(), f.work)
			if err != nil || got != want {
				t.Fatalf("result=%q err=%v, want %q", got, err, want)
			}
			after := mustGit(t, f.work, "status", "--porcelain")
			if want == "skipped" && before != after {
				t.Fatal("skipped restore changed working tree")
			}
			if want != "skipped" && after != "" {
				t.Fatalf("restore left dirty files: %s", after)
			}
		})
	}
}

func TestParseDirtyFilesHandlesNULPathsAndRenames(t *testing.T) {
	files, err := parseDirtyFiles(" M nested dir/package-lock.json\x00?? line\nbreak/package-lock.json\x00R  new name\x00old name\x00")
	if err != nil || len(files) != 4 {
		t.Fatalf("files=%+v err=%v", files, err)
	}
	if files[1].path != "line\nbreak/package-lock.json" || files[2].path != "new name" || files[3].path != "old name" {
		t.Fatalf("paths corrupted: %+v", files)
	}
	for _, invalid := range []string{"M x\x00", "R  target\x00"} {
		if _, err := parseDirtyFiles(invalid); err == nil {
			t.Fatalf("accepted invalid status %q", invalid)
		}
	}
}

func TestRestoreUnmergedLockIsSkipped(t *testing.T) {
	f := newFixture(t)
	mustGit(t, f.work, "switch", "-c", "other")
	writeFile(t, filepath.Join(f.work, "package-lock.json"), "other\n")
	mustGit(t, f.work, "commit", "-am", "other")
	mustGit(t, f.work, "switch", "main")
	writeFile(t, filepath.Join(f.work, "package-lock.json"), "main\n")
	mustGit(t, f.work, "commit", "-am", "main")
	// Git must report a conflict for this fixture.
	if _, err := runGit(context.Background(), f.work, "merge", "other"); err == nil {
		t.Fatal("expected merge conflict")
	}
	before := mustGit(t, f.work, "status", "--porcelain")
	if !strings.HasPrefix(before, "UU") {
		t.Fatalf("expected unmerged lock: %s", before)
	}
	result, err := restorePackageLocks(context.Background(), f.work)
	if err != nil || result != "skipped" || mustGit(t, f.work, "status", "--porcelain") != before {
		t.Fatalf("result=%s err=%v", result, err)
	}
}
