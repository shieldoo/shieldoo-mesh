//go:build linux || darwin
// +build linux darwin

package main

import (
	"net"
	"os"
	"os/exec"
	"syscall"

	"github.com/sirupsen/logrus"
)

const connPipeName = "/tmp/shieldoo.sock"

func svcFirewallCleanup() {
	// Do nothing because it is not needed for linux
}

func svcFirewallSetup(cidr string) {
	// Do nothing because it is not needed for linux
}

func createCommandListener() (l net.Listener, err error) {
	log.Debug("create listener to: ", connPipeName)
	os.Remove(connPipeName)
	l, err = net.Listen("unix", connPipeName)
	if err != nil {
		return
	}
	fi, err := os.Stat(connPipeName)
	if err != nil {
		return
	}
	err = os.Chmod(connPipeName, fi.Mode()|0066)
	return
}

func HookLogger(l *logrus.Logger) {
	// Do nothing, let the logs flow to stdout/stderr
}

func HookLogerInit() {
	// Do nothing because it is not needed for linux
}

func HookLogerClose() {
	// Do nothing because it is not needed for linux
}

func DetachOsProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}
