//go:build windows
// +build windows

package main

import "time"

func osUpdateRun() ManagementOSAutoupdateRequest {
	return ManagementOSAutoupdateRequest{
		Type:             "windows",
		Name:             "windows",
		LastUpdate:       time.Now().UTC(),
		LastUpdateOutput: "windows is not supported for OS updates.",
		Success:          false,
	}
}
