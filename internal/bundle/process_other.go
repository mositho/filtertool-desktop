//go:build !windows

package bundle

import "os/exec"

func hideWindow(cmd *exec.Cmd) {}
