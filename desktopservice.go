package main

import (
	"encoding/json"
	"net"
	"os"

	rpc "github.com/shieldoo/shieldoo-mesh/rpc"
)

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
			ServiceCheckPingerStop()
			removeLocalConf()
			myconfig.RestrictedNetwork = false
			go SvcConnectionStart(deskserviceEnableWinLog)
			go ServiceCheckPinger()
		}
	case rpc.RPCCOMMANDSTOP:
		ServiceCheckPingerStop()
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
	resp.IsConnected = ServiceCheckGetPingerSuccess()
	resp.AccessId = myconfig.AccessId
	resp.Uri = myconfig.Uri
	resp.RestrictedNetwork = myconfig.RestrictedNetwork
	resp.TunnelExists = ServicecheckExistingTunnels
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
