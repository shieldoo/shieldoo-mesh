package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudfieldcz/beeep"
	"github.com/cloudfieldcz/systray"
)

const defaultSecondsToMessage = 120

var secondsToMessage = defaultSecondsToMessage
var updmanagerMenuItem *systray.MenuItem = nil

func UpdManagerInitMenuItem(menu *systray.MenuItem) {
	updmanagerMenuItem = menu
}

func UpdManagerSetCheck() {
	if secondsToMessage > defaultSecondsToMessage {
		secondsToMessage = defaultSecondsToMessage
	}
}

func UpdManagerRun() {
	for {
		time.Sleep(1 * time.Second)
		secondsToMessage--
		if secondsToMessage <= 0 {
			if forupd, ver := updManagerCheckVersionForUpdate(); forupd {
				message := fmt.Sprintf("Please update your Shieldoo client to the latest version, new version %s is available for you! ", ver)
				beeep.Notify(
					"UPDATE NOTIFICATION", message,
					filepath.FromSlash(execPath+"/logo.png"))
				if updmanagerMenuItem != nil {
					updmanagerMenuItem.Show()
				}
			} else {
				if updmanagerMenuItem != nil {
					updmanagerMenuItem.Hide()
				}
			}
			// random between 3600 anf 36000
			secondsToMessage = 3600 + (rand.Intn(32400))
		}
	}
}

func updManagerDownloadVersion() (string, error) {
	_uri := "https://download.shieldoo.io/latest/version.txt"
	response, err := http.Get(_uri)
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

func updManagerCheckVersionForUpdate() (bool, string) {
	ver, err := updManagerDownloadVersion()
	if err != nil {
		return false, ver
	}
	return APPVERSION != ver, ver
}
