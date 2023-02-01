package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/matishsiao/goInfo"
)

var gtelLogin OAuthLoginResponse

func telemetryLogin() error {
	if gtelLogin.ValidTo.UTC().Local().Add(-300 * time.Second).Before(time.Now().UTC()) {
		gi, _ := goInfo.GetInfo()
		uri := myconfig.Uri + "api/oauth/authorize"
		log.Info("Login  to management server: ", uri)
		timst := time.Now().UTC().Unix()
		keymaterial := strconv.FormatInt(timst, 10) + "|" + myconfig.Secret
		hash := sha256.Sum256([]byte(keymaterial))
		var jsonReq []byte
		req := OAuthLoginRequest{
			AccessID:      myconfig.AccessId,
			Timestamp:     timst,
			Key:           base64.URLEncoding.EncodeToString(hash[:]),
			ClientID:      myconfig.RPCClientID,
			ClientOS:      runtime.GOOS + ", " + gi.OS + ", " + gi.Core,
			ClientInfo:    gi.Hostname,
			ClientVersion: APPVERSION,
		}
		jsonReq, _ = json.Marshal(req)
		log.Debug("Login message: ", string(jsonReq))
		response, err := http.Post(uri, "application/json; charset=utf-8", bytes.NewBuffer(jsonReq))
		if err != nil {
			log.Error("Login error - post: ", err)
			time.Sleep(1000 * time.Millisecond)
			return err
		}
		log.Debug("Login http status: ", response.Status)
		if response.StatusCode != 200 {
			return errors.New("http error")
		}
		bodyBytes, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Error("Login error - post/read: ", err)
			return err
		}
		log.Debug("login response bytes: ", string(bodyBytes))
		err = json.Unmarshal(bodyBytes, &gtelLogin)
		if err != nil {
			log.Error("Login error - post/unmarshal: ", err)
			log.Error("Login error - post/unmarshal - body: ", string(bodyBytes))
		}
		return err
	}
	return nil
}

func telemetryProcessChanges(cfg *ManagementResponseConfig) {
	// save configs and certs
	localconf.ConfigHash = cfg.ConfigData.Hash
	localconf.ConfigData = cfg
	localconf.Loaded = true
	myconfig.AutoUpdate = cfg.Autoupdate
}

func telemetryCollectLogData() string {
	tmplog := ""

	// collect telemtry data
	logreading := true
	// in first step we will try to read any data from channel, if there is nothing we will wait for defined time
	select {
	case l := <-logdata:
		if l != "" {
			tmplog += l + "\n"
		}
	case <-time.After(time.Duration(myconfig.SendInterval) * 1000 * time.Millisecond):
	}
	// there we will try to read rest of data from channel
	for logreading {
		select {
		case l := <-logdata:
			if l != "" {
				tmplog += l + "\n"
			}
		case <-time.After(100 * time.Millisecond):
			logreading = false
		}
	}
	return tmplog
}

func telemetrySend() (ret bool) {
	tmplog := ""

	// exception handling
	defer func() {
		if r := recover(); r != nil {
			err := r.(error)
			log.Error("telemetrySend() telemetry error: ", err)
			ret = false
			// return log data to memory for next time
			// if logdata are extremly big forgot them
			if len(tmplog) < 16384 {
				logdata <- tmplog
			}
			// because there was a error, lets wait for a while
			time.Sleep(2500 * time.Millisecond)
		}
	}()

	// collect telemtry data
	tmplog = telemetryCollectLogData()

	ret = false
	// sned telemetry
	if e := telemetryLogin(); e == nil {
		uri := myconfig.Uri + "api/management/message"
		log.Debug("Sending telemetry to: ", uri)
		request := ManagementRequest{
			AccessID:      myconfig.AccessId,
			ClientID:      myconfig.RPCClientID,
			ConfigHash:    localconf.ConfigHash,
			DnsHash:       dnsconf.DnsHash,
			Timestamp:     time.Now().UTC(),
			LogData:       tmplog,
			OverWebSocket: myconfig.RestrictedNetwork,
			IsConnected:   NetutilsPing(lighthouseIP),
		}
		jsonReq, _ := json.Marshal(request)
		log.Debug("http req: ", string(jsonReq))

		req, _ := http.NewRequest("POST", uri, bytes.NewBuffer(jsonReq))
		req.Header.Set("Authorization", "Bearer "+gtelLogin.JWTToken)
		req.Header.Add("Accept", "application/json; charset=utf-8")
		client := &http.Client{}
		response, err := client.Do(req)
		if err != nil {
			panic(err)
		}

		log.Debug("http resp: ", response.Status)
		if response.StatusCode == 401 {
			gtelLogin.ValidTo = time.Now().UTC().Add(-1000 * time.Hour)
			panic(errors.New("unauthorized call to management API (401)"))
		} else if response.StatusCode != 200 {
			panic(errors.New("status code from management API != 200: " + response.Status))
		}
		bodyBytes, err := ioutil.ReadAll(response.Body)
		if err != nil {
			panic(err)
		}
		resp := ManagementResponse{}
		err = json.Unmarshal(bodyBytes, &resp)
		if err != nil {
			panic(err)
		}
		if resp.Dns != nil {
			log.Info("Save new DNS config data")
			dnsconf = *resp.Dns
			ret = true
		}
		if resp.ConfigData != nil {
			log.Info("Save new config data")
			telemetryProcessChanges(resp.ConfigData)
			ret = true
		}
	}
	return
}
