package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	rpc "github.com/Shieldoo/shieldoo-mesh/rpc"
)

const (
	// default deskservice ping interval
	DESKSVCPINGINTERVAL int = 1000
	// inactive tunnel timeouts in minutes
	DESKSVCPTUNNELIDDLETIMEOUTMINUTES float64 = 10
)

type DeskServiceTunnelMessageCounter struct {
	MessageCounter uint64
	LastChange     time.Time
}

var deskservicePingerSuccess bool = false
var deskservicePingerNextRunTimeinterval int = DESKSVCPINGINTERVAL
var deskservicePingerQuit chan bool
var deskserviceCheckRestrictedNetworkCounter int = 0
var deskserviceTunnelArray = make(map[string]DeskServiceTunnelMessageCounter)

const DESKSERVICE_MAXRETRY_RESTRICTEDNET int = 16

func deskserviceSwitchToRestrictedNetwork() {
	log.Debug("deskservice - deskserviceSwitchToRestrictedNetwork ..")
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
	deskserviceTunnelArray = make(map[string]DeskServiceTunnelMessageCounter)
}

func deskserviceUDPCheckLighthouse() bool {
	log.Debug("deskservice - deskserviceUDPCheckLighthouse ..")
	if lighthousePublicIpPort == "" {
		return false
	}
	// create UDP connection
	udpAddr, err := net.ResolveUDPAddr("udp", lighthousePublicIpPort)
	if err != nil {
		log.Error("deskserviceUDPCheckLighthouse - resolve udp address: ", err)
		return false
	}
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		log.Error("deskserviceUDPCheckLighthouse - dial udp: ", err)
		return false
	}
	defer conn.Close()
	// send testing packet
	var testArr = [16]byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01}
	_, err = conn.Write(testArr[:])
	if err != nil {
		log.Error("deskserviceUDPCheckLighthouse - write udp: ", err)
		return false
	}
	// wait 2 seconds for response
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	// read response
	var buf [16]byte
	_, err = conn.Read(buf[:])
	if err != nil {
		log.Error("deskserviceUDPCheckLighthouse - read udp: ", err)
		return false
	}
	return true
}

func deskserviceCheckActiveNebulaTunnels() bool {
	log.Debug("deskservice - deskserviceCheckNebulaTunnels ..")
	if !localconf.Loaded || svcProcess == nil {
		return false
	}
	// check if there is any open tunnel
	list := svcProcess.nebula.ListHostmap(false)
	for _, v := range list {
		vpnip := v.VpnIp.String()
		if vpnip != lighthouseIP {
			if t, ok := deskserviceTunnelArray[vpnip]; ok {
				if t.MessageCounter != v.MessageCounter {
					deskserviceTunnelArray[vpnip] = DeskServiceTunnelMessageCounter{
						MessageCounter: v.MessageCounter,
						LastChange:     time.Now().UTC(),
					}
				}
			} else {
				deskserviceTunnelArray[vpnip] = DeskServiceTunnelMessageCounter{
					MessageCounter: v.MessageCounter,
					LastChange:     time.Now().UTC(),
				}
			}
		}
	}
	log.Debug("deskservice - deskserviceCheckNebulaTunnels - list: ", fmt.Sprintf("%+v", deskserviceTunnelArray))
	// check active tunnels
	ret := false
	for k, v := range deskserviceTunnelArray {
		if time.Now().UTC().Sub(v.LastChange).Minutes() <= DESKSVCPTUNNELIDDLETIMEOUTMINUTES {
			log.Debug("deskservice - deskserviceCheckNebulaTunnels - tunnel to ", k, " is active")
			ret = true
		} else {
			log.Debug("deskservice - deskserviceCheckNebulaTunnels - tunnel to ", k, " is inactive")
		}
	}
	return ret
}

func deskserviceSwitchBackFromRestrictedNetwork() {
	log.Debug("deskservice - deskserviceSwitchBackFromRestrictedNetwork ..")
	if !localconf.Loaded || !myconfig.RestrictedNetwork {
		return
	}
	// if there is any open established tunnel, do not switch back (except to lighthouse)
	if deskserviceCheckActiveNebulaTunnels() {
		return
	}

	// send testing UDP packet to lighthouse
	if deskserviceUDPCheckLighthouse() {
		// if there is any response, switch back to normal network (because UDP works again)
		log.Info("check restricted network - switching back to normal network")
		myconfig.RestrictedNetwork = false
		svcIsInitialized = false
		// insert into log channel empty string to initialize immediate sending after startup
		logdata <- ""
		// cleanup active tunnels
		deskserviceTunnelArray = make(map[string]DeskServiceTunnelMessageCounter)
	}
}

func deskserviceCheckRestrictedNetwork() {
	log.Debug("deskservice - deskserviceCheckRestrictedNetwork ..")
	if !localconf.Loaded {
		return
	}
	if myconfig.RestrictedNetwork {
		deskserviceSwitchBackFromRestrictedNetwork()
	} else {
		deskserviceSwitchToRestrictedNetwork()
	}
}

func deskservicePinger() {
	deskservicePingerQuit = make(chan bool)
	log.Debug("deskservice - ping started")
	deskserviceCheckRestrictedNetworkCounter = 0
	for {
		select {
		case <-deskservicePingerQuit:
			log.Debug("deskservice - quitting ping ..")
			deskservicePingerQuit = nil
			deskservicePingerSuccess = false
			return
		case <-time.After(time.Duration(deskservicePingerNextRunTimeinterval) * time.Millisecond):
			// ping loop
			deskservicePingerSuccess = NetutilsPing(lighthouseIP)
			log.Debug("deskservice - pinging ..")
			// check if we need to switch to restricted network or back
			if localconf.Loaded &&
				((!myconfig.RestrictedNetwork && !deskservicePingerSuccess) ||
					(myconfig.RestrictedNetwork && deskservicePingerSuccess)) {
				deskserviceCheckRestrictedNetworkCounter++
				if deskserviceCheckRestrictedNetworkCounter >= DESKSERVICE_MAXRETRY_RESTRICTEDNET {
					deskserviceCheckRestrictedNetworkCounter = 0
					deskserviceCheckRestrictedNetwork()
				}
			} else {
				deskserviceCheckRestrictedNetworkCounter = 0
			}
			if !deskservicePingerSuccess {
				deskservicePingerNextRunTimeinterval = DESKSVCPINGINTERVAL
			} else {
				if deskservicePingerNextRunTimeinterval < DESKSVCPINGINTERVAL*10 {
					deskservicePingerNextRunTimeinterval += DESKSVCPINGINTERVAL
				}
			}
		}
	}
}

func deskservicePingerStop() {
	if deskservicePingerQuit == nil {
		return
	}
	deskservicePingerQuit <- true
}

func deskserviceProcessor(client net.Conn) {
	defer client.Close()

	c, data, err := rpc.RpcReadPacket(client)
	if err != nil {
		log.Error("deskservice - error reading message", err)
		return
	}
	if c != rpc.RPCCOMMANDSTATUS {
		log.Debug("deskservice - read packet type: ", c)
		log.Debug("deskservice - read packet data: ", string(data))
	}

	//read packet data
	resp := rpc.RpcCommandResponse{Version: rpc.RPCVERSION, Status: "OK"}
	switch c {
	case rpc.RPCCOMMANDSTART:
		j := rpc.RpcCommandStart{}
		if err := json.Unmarshal(data, &j); err != nil {
			log.Error("deskservice - error deserializing message", err)
			return
		}
		if svcconnIsRunning {
			resp.Status = "ERROR - service already running"
		} else {
			myconfig.AccessId = j.AccessId
			myconfig.Uri = j.Uri
			myconfig.Secret = j.Secret
			myconfig.RPCClientID = j.ClientID
			if j.HeartbeatInterval >= 5 && j.HeartbeatInterval <= 300 {
				myconfig.SendInterval = j.HeartbeatInterval
			} else {
				myconfig.SendInterval = 60
			}
			deskservicePingerStop()
			removeLocalConf()
			myconfig.RestrictedNetwork = false
			go SvcConnectionStart(deskserviceEnableWinLog)
			go deskservicePinger()
		}
	case rpc.RPCCOMMANDSTOP:
		deskservicePingerStop()
		SvcConnectionStop()
		removeLocalConf()
		myconfig.RestrictedNetwork = false
		gtelLogin = OAuthLoginResponse{}
	case rpc.RPCCOMMANDSTATUS:
	default:
		resp.Status = "ERROR - unknown command"
	}

	// grab status information
	resp.IsRunning = svcconnIsRunning
	resp.IsConnected = deskservicePingerSuccess
	resp.AccessId = myconfig.AccessId
	resp.Uri = myconfig.Uri
	resp.RestrictedNetwork = myconfig.RestrictedNetwork
	// send response to client
	errs := rpc.RpcSendMessage(client, &resp)
	if err != nil {
		log.Error("deskservice - send error: ", errs)
	}
}

var deskserviceEnableWinLog bool

func DeskserviceStart(enableWinLog bool) {
	deskserviceEnableWinLog = enableWinLog
	log.Info("deskservice - starting listener ..")
	removeLocalConf()
	l, err := createCommandListener()
	if err != nil {
		log.Fatal("deskservice - listen error:", err)
		os.Exit(1)
	}
	defer l.Close()

	for {
		// Accept new connections, dispatching them to processor
		// in a goroutine.
		conn, err := l.Accept()
		if err != nil {
			log.Fatal("deskservice - accept error:", err)
		}

		go deskserviceProcessor(conn)
	}
}

func DeskserviceStop() {
	log.Info("deskservice - stopping listener ..")
}
