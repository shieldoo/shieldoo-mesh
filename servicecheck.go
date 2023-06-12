package main

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

const SERVICECHECK_MAXRETRY_RESTRICTEDNET int = 16

const (
	// default servicecheck ping interval
	SVCCHECKPINGINTERVAL int = 1000
	// inactive tunnel timeouts in minutes
	SVCCHECKPTUNNELIDDLETIMEOUTMINUTES float64 = 10
)

type ServiceCheckTunnelMessageCounter struct {
	MessageCounter uint64
	LastChange     time.Time
}

var servicecheckPingerSuccess bool = false
var servicecheckPingerNextRunTimeinterval int = SVCCHECKPINGINTERVAL
var servicecheckPingerQuit chan bool
var servicecheckTestRestrictedNetworkCounter int = 0
var servicecheckTunnelArray = make(map[string]ServiceCheckTunnelMessageCounter)
var ServicecheckExistingTunnels bool = false

func ServiceCheckGetPingerSuccess() bool {
	return servicecheckPingerSuccess
}

func servicecheckSwitchToRestrictedNetwork() {
	log.Debug("servicecheckSwitchToRestrictedNetwork ..")
	if !localconf.Loaded || myconfig.RestrictedNetwork {
		return
	}
	// create credentials for restricted network
	_usr, _pwd, _wss := WSTunnelCredentials()
	if _usr == "" || _pwd == "" || _wss == "" {
		log.Error("wstunnel address or credentials is not provided, cannot start")
		return
	}
	auth := base64.StdEncoding.EncodeToString([]byte(_usr + ":" + _pwd))
	_wss = strings.Replace(_wss, "wss://", "https://", 1)
	url := fmt.Sprintf("%s/api/health", _wss)
	log.Debug("check restricted network url: ", url)
	// create http request to check if we can connect to restricted network
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error("check restricted network request: ", err)
		return
	}
	req.Header.Add("Authorization", "Basic "+auth)
	client := http.Client{Timeout: 5 * time.Second}
	// send request
	resp, err := client.Do(req)
	if err != nil {
		log.Debug("check restricted network response: ", err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		log.Debug("check restricted network response: ", resp.StatusCode)
		return
	}
	// we can connect to restricted network, switch to it
	log.Info("check restricted network - switching to restricted network")
	myconfig.RestrictedNetwork = true
	svcIsInitialized = false
	// insert into log channel empty string to initialize immediate sending after startup
	logdata <- ""
	// cleanup active tunnels
	servicecheckTunnelArray = make(map[string]ServiceCheckTunnelMessageCounter)
}

func servicecheckUDPCheckLighthouse() bool {
	log.Debug("servicecheckUDPCheckLighthouse ..")
	if lighthousePublicIpPort == "" {
		return false
	}
	// create UDP connection
	udpAddr, err := net.ResolveUDPAddr("udp", lighthousePublicIpPort)
	if err != nil {
		log.Error("servicecheckUDPCheckLighthouse - resolve udp address: ", err)
		return false
	}
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		log.Error("servicecheckUDPCheckLighthouse - dial udp: ", err)
		return false
	}
	defer conn.Close()
	// send testing packet
	var testArr = [16]byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}
	_, err = conn.Write(testArr[:])
	if err != nil {
		log.Error("servicecheckUDPCheckLighthouse - write udp: ", err)
		return false
	}
	// wait 2 seconds for response
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	// read response
	var buf [16]byte
	_, err = conn.Read(buf[:])
	if err != nil {
		log.Error("servicecheckUDPCheckLighthouse - read udp: ", err)
		return false
	}
	return true
}

func servicecheckTestActiveNebulaTunnels() bool {
	log.Debug("servicecheckTestActiveNebulaTunnels ..")
	if !localconf.Loaded || svcProcess == nil {
		return false
	}
	// check if there is any open tunnel
	list := svcProcess.nebula.ListHostmap(false)
	for _, v := range list {
		vpnip := v.VpnIp.String()
		if vpnip != lighthouseIP {
			if t, ok := servicecheckTunnelArray[vpnip]; ok {
				if t.MessageCounter != v.MessageCounter {
					servicecheckTunnelArray[vpnip] = ServiceCheckTunnelMessageCounter{
						MessageCounter: v.MessageCounter,
						LastChange:     time.Now().UTC(),
					}
				}
			} else {
				servicecheckTunnelArray[vpnip] = ServiceCheckTunnelMessageCounter{
					MessageCounter: v.MessageCounter,
					LastChange:     time.Now().UTC(),
				}
			}
		}
	}
	log.Debug("servicecheckTestActiveNebulaTunnels - list: ", fmt.Sprintf("%+v", servicecheckTunnelArray))
	// check active tunnels
	ret := false
	for k, v := range servicecheckTunnelArray {
		if time.Now().UTC().Sub(v.LastChange).Minutes() <= SVCCHECKPTUNNELIDDLETIMEOUTMINUTES {
			log.Debug("servicecheckTestActiveNebulaTunnels - tunnel to ", k, " is active")
			ret = true
		} else {
			log.Debug("servicecheckTestActiveNebulaTunnels - tunnel to ", k, " is inactive")
		}
	}
	return ret
}

func servicecheckSwitchBackFromRestrictedNetwork() {
	log.Debug("servicecheckSwitchBackFromRestrictedNetwork ..")
	if !localconf.Loaded || !myconfig.RestrictedNetwork {
		return
	}
	// if there is any open established tunnel, do not switch back (except to lighthouse)
	if servicecheckTestActiveNebulaTunnels() {
		return
	}

	// send testing UDP packet to lighthouse
	if servicecheckUDPCheckLighthouse() {
		// if there is any response, switch back to normal network (because UDP works again)
		log.Info("check restricted network - switching back to normal network")
		myconfig.RestrictedNetwork = false
		svcIsInitialized = false
		// insert into log channel empty string to initialize immediate sending after startup
		logdata <- ""
		// cleanup active tunnels
		servicecheckTunnelArray = make(map[string]ServiceCheckTunnelMessageCounter)
	}
}

func servicecheckTestRestrictedNetwork() {
	log.Debug("servicecheckTestRestrictedNetwork ..")
	if !localconf.Loaded {
		return
	}
	if myconfig.RestrictedNetwork {
		servicecheckSwitchBackFromRestrictedNetwork()
	} else {
		servicecheckSwitchToRestrictedNetwork()
	}
}

func servicecheckHandleWakeUp() {
	log.Debug("servicecheckHandleWakeUp ..")
	if !localconf.Loaded {
		return
	}
	if svcProcess == nil {
		return
	}
	// force exchange IP configuration with lighthouse
	log.Info("servicecheck - wake-up from sleep - force exchange IP configuration with lighthouse")
	svcProcess.nebula.RebindUDPServer()
}

func ServiceCheckPinger() {
	servicecheckPingerQuit = make(chan bool)
	log.Info("servicecheck - ping started")
	servicecheckTestRestrictedNetworkCounter = 0
	// cleanup active tunnels
	servicecheckTunnelArray = make(map[string]ServiceCheckTunnelMessageCounter)
	for {
		currentTime := time.Now().UTC()
		select {
		case <-servicecheckPingerQuit:
			log.Debug("servicecheck - quitting ping ..")
			servicecheckPingerQuit = nil
			servicecheckPingerSuccess = false
			return
		case <-time.After(time.Duration(servicecheckPingerNextRunTimeinterval) * time.Millisecond):
			// check if system was in sleep mode
			if time.Now().UTC().Sub(currentTime).Milliseconds() >= int64(2*servicecheckPingerNextRunTimeinterval) {
				servicecheckHandleWakeUp()
			}
			// check if tunnels are active
			ServicecheckExistingTunnels = servicecheckTestActiveNebulaTunnels()
			// ping loop
			servicecheckPingerSuccess = NetutilsPing(lighthouseIP)
			// check if we need to switch to restricted network or back
			if localconf.Loaded &&
				((!myconfig.RestrictedNetwork && !servicecheckPingerSuccess) ||
					(myconfig.RestrictedNetwork && servicecheckPingerSuccess)) {
				servicecheckTestRestrictedNetworkCounter++
				if servicecheckTestRestrictedNetworkCounter >= SERVICECHECK_MAXRETRY_RESTRICTEDNET {
					servicecheckTestRestrictedNetworkCounter = 0
					servicecheckTestRestrictedNetwork()
				}
			} else {
				servicecheckTestRestrictedNetworkCounter = 0
			}
			if !servicecheckPingerSuccess {
				servicecheckPingerNextRunTimeinterval = SVCCHECKPINGINTERVAL
			} else {
				if servicecheckPingerNextRunTimeinterval < SVCCHECKPINGINTERVAL*10 {
					servicecheckPingerNextRunTimeinterval += SVCCHECKPINGINTERVAL
				}
			}
		}
	}
}

func ServiceCheckPingerStop() {
	if servicecheckPingerQuit == nil {
		return
	}
	log.Info("servicecheck - stopping ping ..")
	servicecheckPingerQuit <- true
}
