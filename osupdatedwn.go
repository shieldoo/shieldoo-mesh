//go:build darwin
// +build darwin

package main

import (
	"time"

	"github.com/matishsiao/goInfo"
)

func osUpdateRun() ManagementOSAutoupdateRequest {
	gi, _ := goInfo.GetInfo()

	return ManagementOSAutoupdateRequest{
		Type:             "darwin",
		Name:             gi.OS,
		Description:      gi.Core,
		LastUpdate:       time.Now().UTC(),
		LastUpdateOutput: "darwin is not supported for OS updates.",
		Success:          false,
	}
}
