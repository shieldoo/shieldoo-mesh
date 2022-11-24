//go:build !windows
// +build !windows

package main

import "github.com/cloudfieldcz/beeep"

func beeepInit() {
	beeep.AppName = "Shieldoo Mesh"
}
