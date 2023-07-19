//go:build linux
// +build linux

package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/matishsiao/goInfo"
)

//go:embed osupdate-script-linux.sh
var osUpdateScriptLinux string

// create osupdate.sh in tmp folder and switch on run attribute
func createOsUpdateScript() (string, error) {
	tmpfile, err := os.CreateTemp("", "osupdate-*.sh")
	if err != nil {
		return "", err
	}
	defer tmpfile.Close()
	// Change file permission to add executable permission
	err = os.Chmod(tmpfile.Name(), 0755)
	if err != nil {
		return "", err
	}
	if _, err := tmpfile.Write([]byte(osUpdateScriptLinux)); err != nil {
		return "", err
	}
	return tmpfile.Name(), nil
}

// execute script
func runOsUpdateScript(script string, param string) (string, error, int) {
	cmd := exec.Command(script, param)
	var outb bytes.Buffer
	cmd.Stdout = &outb
	err := cmd.Run()
	retcode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				retcode = status.ExitStatus()
			}
		} else {
			retcode = -1
		}
	}
	return outb.String(), err, retcode
}

func osRemoveEmptyLines(lines []string) []string {
	var ret []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			ret = append(ret, line)
		}
	}
	return ret
}

func osUpdateRun() ManagementOSAutoupdateRequest {
	gi, _ := goInfo.GetInfo()
	pretty, _ := getPrettyName()

	ret := ManagementOSAutoupdateRequest{
		Type:                 "linux",
		Name:                 gi.OS,
		Version:              gi.Core,
		Description:          pretty,
		LastUpdate:           time.Now().UTC(),
		LastUpdateOutput:     "",
		Success:              false,
		SecurityUpdatesCount: 0,
		OtherUpdatesCount:    0,
		SecurityUpdates:      []string{},
		OtherUpdates:         []string{},
	}

	// create script
	script, err := createOsUpdateScript()
	defer os.Remove(script)
	if err != nil {
		ret.LastUpdateOutput = err.Error()
		ret.Success = false
		return ret
	}

	// perform collection of update data
	log.Debug("osupdate-lnx: list security updates")
	outScripts, err, retcodeScript := runOsUpdateScript(script, "-s")
	if err != nil && retcodeScript != 5 {
		ret.LastUpdateOutput = outScripts
		ret.Success = false
		log.Error("osupdate-lnx: list security updates failed")
		return ret
	}
	log.Debug("osupdate-lnx: list other updates")
	outOther, err, _ := runOsUpdateScript(script, "-o")
	if err != nil {
		ret.LastUpdateOutput = outOther
		ret.Success = false
		log.Error("osupdate-lnx: list other updates failed")
		return ret
	}
	// parse output of script (lines)
	ret.SecurityUpdates = strings.Split(outScripts, "\n")
	ret.OtherUpdates = strings.Split(outOther, "\n")
	// cleanup arrays (remove empty lines)
	ret.SecurityUpdates = osRemoveEmptyLines(ret.SecurityUpdates)
	ret.OtherUpdates = osRemoveEmptyLines(ret.OtherUpdates)
	ret.SecurityUpdatesCount = len(ret.SecurityUpdates)
	ret.OtherUpdatesCount = len(ret.OtherUpdates)

	log.Debug("osupdate-lnx: script output security: ", outScripts)
	log.Debug("osupdate-lnx: script output other: ", outOther)
	log.Debug("osupdate-lnx: security updates count: ", ret.SecurityUpdatesCount)
	log.Debug("osupdate-lnx: other updates count: ", ret.OtherUpdatesCount)
	log.Debug("osupdate-lnx: security updates: ", ret.SecurityUpdates)
	log.Debug("osupdate-lnx: other updates: ", ret.OtherUpdates)

	// perform update if requested
	if ret.SecurityUpdatesCount > 0 || ret.OtherUpdatesCount > 0 {
		if localconf.ConfigData.OSAutoupdatePolicy.AllAutoupdateEnabled {
			log.Debug("osupdate-lnx: update all")
			outUpdate, err, _ := runOsUpdateScript(script, "-a")
			if err != nil {
				runOsUpdateScript(script, "-r")
				outUpdate, err, _ = runOsUpdateScript(script, "-a")
				if err != nil {
					ret.LastUpdateOutput = outUpdate
					ret.Success = false
					log.Error("osupdate-lnx: update all failed")
					return ret
				}
			}
			ret.OtherUpdatesCount = 0
			ret.SecurityUpdatesCount = 0
			ret.OtherUpdates = []string{}
			ret.SecurityUpdates = []string{}
		} else if localconf.ConfigData.OSAutoupdatePolicy.SecurityAutoupdateEnabled {
			log.Debug("osupdate-lnx: update security")
			outUpdate, err, _ := runOsUpdateScript(script, "-u")
			if err != nil {
				runOsUpdateScript(script, "-r")
				outUpdate, err, _ = runOsUpdateScript(script, "-u")
				if err != nil {
					ret.LastUpdateOutput = outUpdate
					ret.Success = false
					log.Error("osupdate-lnx: update security failed")
					return ret
				}
			}
			ret.SecurityUpdatesCount = 0
			ret.SecurityUpdates = []string{}
		}
		if (localconf.ConfigData.OSAutoupdatePolicy.AllAutoupdateEnabled ||
			localconf.ConfigData.OSAutoupdatePolicy.SecurityAutoupdateEnabled) &&
			localconf.ConfigData.OSAutoupdatePolicy.RestartAfterUpdate {
			// restart whole operating system
			// Execute the shutdown command to restart the system
			log.Info("osupdate-lnx: restart system")
			cmd := exec.Command("shutdown", "-r", "+2")
			cmd.Run()
		}
	}

	ret.Success = true

	return ret
}

func getPrettyName() (string, error) {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PRETTY_NAME") {
			splitLine := strings.Split(line, "=")
			if len(splitLine) != 2 {
				return "", fmt.Errorf("invalid line: %s", line)
			}

			// Remove the double quotes around the pretty name
			prettyName := strings.Trim(splitLine[1], "\"")
			return prettyName, nil
		}
	}

	if scanner.Err() != nil {
		return "", scanner.Err()
	}

	return "", fmt.Errorf("PRETTY_NAME not found")
}
