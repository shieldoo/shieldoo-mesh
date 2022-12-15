package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

type OAuthUPNLoginRequest struct {
	Upn       string `json:"upn"`
	Timestamp int64  `json:"timestamp"`
	Key       string `json:"key"`
}

type OAuthLoginResponse struct {
	JWTToken string    `json:"jwt"`
	ValidTo  time.Time `json:"valid_to"`
}

type ManagementUPNRequest struct {
	ConfigHash    string    `json:"confighash"`
	Timestamp     time.Time `json:"timestamp"`
	LogData       string    `json:"log_data"`
	ClientVersion string    `json:"clientversion"`
}

type ManagementSimpleUPNResponseAccess struct {
	AccessID int    `json:"accessid"`
	Name     string `json:"name"`
	Secret   string `json:"secret"`
}

type ManagementSimpleUPNResponse struct {
	Status        string                               `json:"status"`
	Hash          string                               `json:"hash"`
	Accesses      *[]ManagementSimpleUPNResponseAccess `json:"accesses"`
	ServerMessage string                               `json:"servermessage"`
}

const msgLoginError = "Login error: "

var localconf ManagementSimpleUPNResponse

var gtelLogin OAuthLoginResponse

func telemetryInvalidateToken() {
	gtelLogin.ValidTo = time.Now().UTC().Add(-1000 * time.Hour)
}

func telemetryLogin() error {
	if gtelLogin.ValidTo.UTC().Local().Add(-300 * time.Second).Before(time.Now().UTC()) {
		uri := myconfig.Uri + "api/oauth/authorizeupn"
		log.Info("Login  to management server: ", uri)
		timst := time.Now().UTC().Unix()
		keymaterial := strconv.FormatInt(timst, 10) + "|" + myconfig.Secret
		hash := sha256.Sum256([]byte(keymaterial))
		req := OAuthUPNLoginRequest{
			Upn:       myconfig.Upn,
			Timestamp: timst,
			Key:       base64.URLEncoding.EncodeToString(hash[:]),
		}
		jsonReq, _ := json.Marshal(req)
		response, err := http.Post(uri, "application/json; charset=utf-8", bytes.NewBuffer(jsonReq))
		if err != nil {
			log.Error(msgLoginError, err)
			return err
		}
		log.Debug("Login http status: ", response.Status)
		if response.StatusCode != 200 {
			return errors.New("http error")
		}
		bodyBytes, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Error(msgLoginError, err)
			return err
		}
		err = json.Unmarshal(bodyBytes, &gtelLogin)
		if err != nil {
			log.Error(msgLoginError, err)
		}
		return err
	}
	return nil
}

func telemetrySend() (ret bool, err error) {
	// exception handling
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			log.Error("telemetrySend() telemetry error: ", err)
			ret = false
		}
	}()

	ret = false

	if e := telemetryLogin(); e == nil {
		uri := myconfig.Uri + "api/management/configupn"
		log.Info("Sending telemetry to: ", uri)
		request := ManagementUPNRequest{
			ConfigHash:    localconf.Hash,
			Timestamp:     time.Now().UTC(),
			LogData:       "",
			ClientVersion: APPVERSION,
		}
		jsonReq, _ := json.Marshal(request)
		log.Debug("jsonReq: ", string(jsonReq))

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
		log.Debug("body: ", string(bodyBytes))
		resp := ManagementSimpleUPNResponse{}
		err = json.Unmarshal(bodyBytes, &resp)
		if err != nil {
			panic(err)
		}
		if resp.Hash != localconf.Hash {
			log.Info("new config data")
			log.Debug("new config data: ", resp)
			localconf = resp
			ret = true
		}
		if resp.ServerMessage != serverMessage {
			serverMessage = resp.ServerMessage
			if serverMessage != "" {
				log.Info("server message: ", serverMessage)
				serverMessageChan <- serverMessage
			}
		}
	}
	return
}
