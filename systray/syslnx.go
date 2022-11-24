//go:build linux || darwin
// +build linux darwin

package main

import (
	"net"
	"time"
)

const SockAddr = "/tmp/shieldoo.sock"

func createClient() (c net.Conn, err error) {
	c, err = net.Dial("unix", SockAddr)
	if err != nil {
		return
	}
	// set SetReadDeadline
	err = c.SetReadDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		c.Close()
	}
	return
}

func MessageBoxPlain(title, caption string) int {
	return 0
}

func CreateMutex(name string) (uintptr, error) {
	return 0, nil
}
