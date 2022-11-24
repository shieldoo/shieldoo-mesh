//+build windows

package main

import (
	"github.com/Microsoft/go-winio"
	"net"
	"time"
)

var testPipeName = `\\.\pipe\shieldoopipe`

func createClient() (net.Conn, error) {
	timeout := 1 * time.Second
	return winio.DialPipe(testPipeName, &timeout)
}
