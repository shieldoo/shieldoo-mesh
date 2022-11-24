//go:build linux || darwin
// +build linux darwin

package main

import (
	"os/exec"
	"syscall"
)

func ExecCmdAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func CreateMutex(name string) (uintptr, error) {
	return 0, nil
}

func processSigTerm(pid int) error {
	return syscall.Kill(pid, syscall.SIGTERM)
}
