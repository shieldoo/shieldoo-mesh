//+build linux darwin

package main

import "net"

const SockAddr = "/tmp/shieldoo.sock"

func createClient() (net.Conn, error) {
	return net.Dial("unix", SockAddr)
}
