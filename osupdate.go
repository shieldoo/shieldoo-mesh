package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

var OSUpdateLastCheck time.Time = time.Now().UTC().Add(-time.Hour * 24 * 7)
var osUpdaterQuit chan bool

func OSUpdateCheckStart() {
	osUpdaterQuit = make(chan bool)
	log.Info("osupdater - started")
	for {
		select {
		case <-osUpdaterQuit:
			log.Debug("osupdater - quitting ..")
			osUpdaterQuit = nil
			return
		case <-time.After(time.Duration(1) * time.Minute):
			log.Debug("osupdater - checking for updates ..")
			osUpdateProcess()
		}
	}
}

func osUpdateProcess() {
	if !localconf.Loaded {
		log.Debug("osupdater - localconf not loaded")
		return
	}
	if localconf.ConfigData == nil {
		log.Debug("osupdater - localconf.ConfigData not loaded")
		return
	}
	if !localconf.ConfigData.OSAutoupdatePolicy.Enabled {
		log.Debug("osupdater - os autoupdate not enabled")
		return
	}
	if OSUpdateLastCheck.After(time.Now().UTC()) {
		log.Debug("osupdater - last check not older than 6 hours - ", OSUpdateLastCheck)
		return
	}
	if localconf.ConfigData.OSAutoupdatePolicy.UpdateHour != 0 &&
		localconf.ConfigData.OSAutoupdatePolicy.UpdateHour != time.Now().UTC().Hour() {
		log.Debug("osupdater - not update hour")
		return
	}
	log.Debug("osupdater - try update ..")
	// check for updates
	updReq := osUpdateRun()
	// send to server
	if e := telemetryLogin(); e == nil {
		uri := myconfig.Uri + "api/management/autoupdate"
		log.Debug("Sending autoupdate to: ", uri)
		jsonReq, _ := json.Marshal(updReq)
		log.Debug("http req: ", string(jsonReq))

		req, _ := http.NewRequest("POST", uri, bytes.NewBuffer(jsonReq))
		req.Header.Set("Authorization", "Bearer "+gtelLogin.JWTToken)
		req.Header.Add("Accept", "application/json; charset=utf-8")
		client := &http.Client{}
		response, err := client.Do(req)
		if err == nil {
			log.Debug("http resp: ", response.Status)
			if response.StatusCode == 401 {
				gtelLogin.ValidTo = time.Now().UTC().Add(-1000 * time.Hour)
			} else if response.StatusCode != 200 {
				log.Error("status code from management API != 200: ", response.Status)
			}
		}
	}
	// update last check time
	OSUpdateLastCheck = time.Now().UTC().Add(time.Hour * 6)
}

func OSUpdateCheckStop() {
	if osUpdaterQuit == nil {
		return
	}
	log.Info("osupdater - stopping os updater ..")
	osUpdaterQuit <- true
}
