package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	NATDETECTION_NOTSET   int = 0
	NATDETECTION_DISABLED int = 1
	NATDETECTION_ENABLED  int = 2
	NATDETECTION_AUTO     int = 3
)

var myconfig *NebulaClientYamlConfig
var localconf NebulaLocalYamlConfig = NebulaLocalYamlConfig{ConfigData: &ManagementResponseConfig{}}
var dnsconf ManagementResponseDNS

const MYCONFIG_FILENAME = "myconfig.yaml"

var execPath string

func WSTunnelCredentials() (usr string, pwd string, wss string) {
	cred := strings.Split(localconf.ConfigData.WebSocketUsernamePassword, ":")
	usr = cred[0]
	pwd = ""
	if len(cred) > 1 {
		pwd = cred[1]
	}
	wss = strings.TrimSpace(localconf.ConfigData.WebSocketUrl)
	return
}

func execPathCreate(p string) string {
	if runtime.GOOS == "darwin" {
		return filepath.FromSlash("/Library/Preferences/ShieldooMesh/" + p)
	} else {
		return filepath.FromSlash(execPath + "/config/" + p)
	}
}

func InitExecPath() {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	execPath = filepath.Dir(ex)
}

func CreateConfigFromBase64(str string) (err error) {
	InitExecPath()

	// create config folder if not exists
	_ = os.MkdirAll(execPathCreate(""), 0700)

	var data []byte
	data, err = base64.StdEncoding.DecodeString(str)
	if err != nil {
		return
	}
	err = saveFile(MYCONFIG_FILENAME, data)
	return
}

func UpdateConfigSetDisableHostsEdit(disableEdit bool) error {
	InitConfig(false)

	myconfig.DisableHostsEdit = disableEdit

	// marshal yaml
	data, err := yaml.Marshal(myconfig)
	if err != nil {
		log.Error("cannot marshal yaml: ", err)
		return err
	}
	// save file
	err = saveFile(MYCONFIG_FILENAME, data)
	if err != nil {
		log.Error("cannot save file: ", err)
		return err
	}
	return nil
}

func InitConfig(isDesktop bool) {
	InitExecPath()

	// create config folder if not exists
	_ = os.MkdirAll(execPathCreate(""), 0700)

	log.Debug("Loading configs ..")
	// read myconfig.yaml
	mc, err := readClientConf(MYCONFIG_FILENAME)
	if err != nil && !isDesktop {
		log.Info("cannot find "+execPathCreate(MYCONFIG_FILENAME)+" file or configuration file is corrupted: ", err)
	}
	myconfig = mc
	// sanitize config
	if myconfig.SendInterval <= 1 || myconfig.SendInterval > 3600 {
		myconfig.SendInterval = 60
	}
	if myconfig.AutoUpdateIntervalMinutes <= 1 || myconfig.AutoUpdateIntervalMinutes > 1440 {
		myconfig.AutoUpdateIntervalMinutes = 720
	}
	if !strings.HasSuffix(myconfig.Uri, "/") {
		myconfig.Uri += "/"
	}
	if myconfig.LocalUDPPort == 0 {
		myconfig.LocalUDPPort = 24242
	}
	if myconfig.AutoUpdateChannel != "latest" && myconfig.AutoUpdateChannel != "beta" {
		myconfig.AutoUpdateChannel = "latest"
	}
}

func removeLocalConf() {
	dnsconf = ManagementResponseDNS{}
	localconf = NebulaLocalYamlConfig{ConfigData: &ManagementResponseConfig{}}
}

func readClientConf(filename string) (*NebulaClientYamlConfig, error) {
	c := &NebulaClientYamlConfig{}
	buf, err := ioutil.ReadFile(execPathCreate(filename))
	if err != nil {
		return c, err
	}

	err = yaml.Unmarshal(buf, c)
	if err != nil {
		return c, fmt.Errorf("in file %q: %v", filename, err)
	}

	return c, nil
}

func saveTextFile(filename string, text string) error {
	return saveFile(filename, []byte(text))
}

func saveFile(filename string, data []byte) error {

	file, err := os.OpenFile(execPathCreate(filename), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)

	return err
}
