package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var serviceupdaterQuit chan bool

// download version file from server
func serviceupdaterDownloadVersion() (string, error) {
	tmpuri := "https://download.shieldoo.io/" + myconfig.AutoUpdateChannel + "/version.txt"
	response, err := http.Get(tmpuri)
	if err != nil {
		return "", err
	}
	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	ret := string(bodyBytes)
	ret = strings.TrimSpace(ret)
	return ret, nil
}

// download large file over http
func serviceupdaterDownloadLargeFile(filepath string, url string) (err error) {

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// download installation file from server
func serviceupdaterDownloadInstall() (string, string, error) {
	log.Debug("serviceupdaterDownloadInstall ..")
	tmpuri := "https://download.shieldoo.io/" + myconfig.AutoUpdateChannel + "/"
	fname := ""
	// create download package name
	switch runtime.GOOS {
	case "windows":
		fname = "windows-amd64-shieldoo-mesh-svc-setup.exe"
	case "linux":
		if ARCHITECTURE == "arm7" {
			fname = "linux-arm7-shieldoo-mesh-svc-setup.tar.gz"
		} else {
			fname = "linux-amd64-shieldoo-mesh-svc-setup.tar.gz"
		}
	case "darwin":
		fname = "darwin-x64-shieldoo-mesh-svc-setup.pkg"
	default:
		return "", "", errors.New("Unsupported OS")
	}
	dname := filepath.FromSlash(os.TempDir() + "/shieldoo-mesh-install")
	os.RemoveAll(dname)
	err := os.Mkdir(dname, 0700)
	if err != nil {
		log.Error("serviceupdaterDownloadInstall - cannot create temp dir: ", err)
		return "", "", err
	}
	// download large file, create temp path for storing
	fpath := filepath.FromSlash(dname + "/" + fname)
	log.Debug("serviceupdaterDownloadInstall - downloading file: ", fpath)
	// download large file
	err = serviceupdaterDownloadLargeFile(fpath, tmpuri+fname)
	if err != nil {
		log.Error("serviceupdaterDownloadInstall - cannot download file: ", err)
		return dname, "", err
	}
	log.Info("serviceupdaterDownloadInstall - downloaded file: ", fpath)
	return dname, fpath, nil
}

// linux install
// on linux we have problem with systemd, systemd during shutdown kills all processes
// in process group so we need to run installation process detached from current process.
// we use at command for this purpose
func serviceupdaterInstallLinux(fpath string) error {
	log.Debug("serviceupdaterInstallLinux ..")
	// create  install script
	scriptname1 := fpath + ".1.sh"
	script1 := "#!/bin/sh\n" +
		"rm -f /opt/shieldoo-mesh/shieldoo-mesh-srv.bak\n" +
		"mv /opt/shieldoo-mesh/shieldoo-mesh-srv /opt/shieldoo-mesh/shieldoo-mesh-srv.bak\n" +
		"tar -xf " + fpath + " -C /opt/shieldoo-mesh\n" +
		"chmod 755 /opt/shieldoo-mesh/shieldoo-mesh-srv\n" +
		"/opt/shieldoo-mesh/shieldoo-mesh-srv -service restart\n" +
		"\n"
	err := ioutil.WriteFile(scriptname1, []byte(script1), 0755)
	if err != nil {
		log.Error("serviceupdaterInstallLinux - cannot create script1: ", err)
		return err
	}
	// stop service, unpack and reinstall
	log.Info("serviceupdaterInstallLinux - running installer ..")
	cmd := exec.Command("/usr/bin/sh", scriptname1)
	err = cmd.Run()
	if err != nil {
		log.Error("serviceupdaterInstallLinux - cannot run installer: ", err)
		return err
	}
	return nil
}

// windows install del "C:\Program Files\Shieldoo Mesh\shieldoo-mesh-srv.exe"
// on windows we are using standard application installer
func serviceupdaterInstallWindows(fpath string) error {
	log.Debug("serviceupdaterInstallWindows ..")
	scriptname1 := fpath + ".1.cmd"
	script1 := "net stop shieldoo-mesh\r\n" +
		"del \"C:\\Program Files\\Shieldoo Mesh\\shieldoo-mesh-srv.exe\"\r\n" +
		fpath + " /S\r\n"
	err := ioutil.WriteFile(scriptname1, []byte(script1), 0755)
	if err != nil {
		log.Error("serviceupdaterInstallWindows - cannot create script1: ", err)
		return err
	}
	log.Info("serviceupdaterInstallWindows - running installer ..")
	cmd := exec.Command("cmd.exe", "/C", "start", "/B", scriptname1)
	DetachOsProcess(cmd)
	err = cmd.Start()
	if err != nil {
		log.Error("serviceupdaterInstallWindows - cannot run installer: ", err)
	}
	return nil
}

// darwin install
// on darwin we are using standard system installer
func serviceupdaterInstallDarwin(fpath string) error {
	log.Debug("serviceupdaterInstallDarwin ..")

	// create flag file
	flagpath := "/Library/Preferences/ShieldooMesh/unattended-install"
	f, err := os.Create(flagpath)
	if err != nil {
		log.Error("serviceupdaterInstallDarwin - cannot create flag file: ", err)
		return err
	}
	f.Close()

	// run installer
	log.Info("serviceupdaterInstallDarwin - running installer ..")
	cmd := exec.Command("/usr/sbin/installer", "-pkg", fpath, "-target", "/")
	DetachOsProcess(cmd)
	err = cmd.Run()
	log.Debug("serviceupdaterInstallDarwin - installer finished: ", err)
	return err
}

// process update on various OS
func serviceupdaterProcess() error {
	var err error
	var fpath string
	var dname string
	log.Debug("serviceupdaterProcess ..")
	// download update package
	dname, fpath, err = serviceupdaterDownloadInstall()
	if err != nil {
		if dname != "" {
			os.RemoveAll(dname)
		}
		return err
	}
	// process update
	switch runtime.GOOS {
	case "windows":
		err = serviceupdaterInstallWindows(fpath)
	case "linux":
		err = serviceupdaterInstallLinux(fpath)
	case "darwin":
		err = serviceupdaterInstallDarwin(fpath)
	default:
		log.Error("serviceupdaterProcess - unsupported OS")
		err = errors.New("Unsupported OS")
	}
	if err != nil {
		os.RemoveAll(dname)
	}
	return err
}

func serviceupdaterCheck() {
	log.Debug("serviceupdaterCheck ..")
	ver, err := serviceupdaterDownloadVersion()
	if err != nil {
		log.Error("serviceupdaterCheck: ", err)
		return
	}
	if ver != APPVERSION {
		log.Info("serviceupdaterCheck: new version available: ", ver)
		err = serviceupdaterProcess()
		if err != nil {
			log.Error("serviceupdaterCheck: ", err)
			return
		}
	}
}

func ServiceUpdaterStart() {
	log.Info("Service updater starting with interval: ", myconfig.AutoUpdateIntervalMinutes, " minutes")
	log.Info("Service updater starting with channel: ", myconfig.AutoUpdateChannel)
	serviceupdaterQuit = make(chan bool)

	for {
		select {
		case <-serviceupdaterQuit:
			log.Debug("Service updater quitting ..")
			serviceupdaterQuit = nil
			return
		case <-time.After(time.Duration(myconfig.AutoUpdateIntervalMinutes) * time.Second * 60):
			if myconfig.AutoUpdate {
				serviceupdaterCheck()
			}
		}
	}
}

func ServiceUpdaterStop() {
	if serviceupdaterQuit == nil {
		return
	}
	log.Info("Service updater stopping.")
	serviceupdaterQuit <- true
}
