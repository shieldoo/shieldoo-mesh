package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	configFileName = "shieldoo-mesh.yaml"
)

type NebulaClientUPNYamlConfig struct {
	Upn               string `yaml:"upn"`
	Uri               string `yaml:"uri"`
	Secret            string `yaml:"-"`
	RestrictedNetwork bool   `yaml:"-"`
	ClientID          string `yaml:"clientid"`
}

var myconfig *NebulaClientUPNYamlConfig

func getConfigDir() string {
	mydir := "/.shieldoo"
	if runtime.GOOS == "darwin" {
		mydir = "/Library/ShieldooMesh"
	}
	return GetHomeDir() + mydir
}

func cleanupConfig() {
	if myconfig.Uri != "" {
		if !strings.HasSuffix(myconfig.Uri, "/") {
			myconfig.Uri += "/"
		}
		saveClientConf()
	}
	if myconfig.ClientID == "" {
		myconfig.ClientID = GenerateRandomString(52)
		saveClientConf()
	}
}

func InitConfig() {

	log.Info("Loading config ..")
	// read myconfig.yaml
	mc, err := readClientConf()
	if err != nil {
		log.Error("cannot find shieldoo-mesh.yaml file or configuration file is corrupted (loading defaults): ", err)
	}
	myconfig = mc
	cleanupConfig()
}

func saveClientConf() error {
	file, err := os.OpenFile(
		filepath.FromSlash(getConfigDir()+"/"+configFileName),
		os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := yaml.NewEncoder(file)

	err = enc.Encode(myconfig)
	if err != nil {
		return err
	}
	return nil
}

func readClientConf() (ret *NebulaClientUPNYamlConfig, err error) {
	ret = &NebulaClientUPNYamlConfig{}
	buf, e := ioutil.ReadFile(filepath.FromSlash(getConfigDir() + "/" + configFileName))
	if e != nil {
		err = e
		return
	}
	err = yaml.Unmarshal(buf, ret)
	return
}
