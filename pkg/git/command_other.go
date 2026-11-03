//go:build !unix

package git

import "os/exec"

func configureCancellation(_ *exec.Cmd) {}
