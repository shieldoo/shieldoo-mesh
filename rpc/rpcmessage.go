package rpc

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"net"
)

// packend send to unix socket or pipe has this format
// ----
// byte[0] - high byte length of message
// byte[1] - low byte length of message
// byte[2] - high byte command type
// byte[3] - low byte command type
// byte[4 - ..] - message serialized as json based on command type
// ----

type RpcCommandType uint16

const (
	RPCCOMMANDUNKNOWN  RpcCommandType = 0
	RPCCOMMANDSTART    RpcCommandType = 1
	RPCCOMMANDSTOP     RpcCommandType = 2
	RPCCOMMANDSTATUS   RpcCommandType = 3
	RPCCOMMANDRESPONSE RpcCommandType = 4
)

const (
	RPCVERSION string = "1.5"
)

type RpcCommandStart struct {
	Version           string `json:"version"`
	AccessId          int    `json:"accessid"`
	Uri               string `json:"uri"`
	Secret            string `json:"secret"`
	HeartbeatInterval int    `json:"heartbeatinterval"`
	RestrictedNetwork bool   `json:"restrictednetwork"`
	LighthouseRoute   bool   `json:"lighthouseroute"`
	ClientID          string `json:"clientid"`
}

type RpcCommandStop struct {
	Version string `json:"version"`
}

type RpcCommandStatus struct {
	Version string `json:"version"`
}

type RpcCommandResponse struct {
	Version           string `json:"version"`
	Status            string `json:"status"`
	AccessId          int    `json:"accessid"`
	IsConnected       bool   `json:"isconnected"`
	IsRunning         bool   `json:"isrunning"`
	Uri               string `json:"uri"`
	RestrictedNetwork bool   `json:"restrictednetwork"`
	LighthouseRoute   bool   `json:"lighthouseroute"`
	TunnelExists      bool   `json:"tunnelexists"`
	Lighthouse        string `json:"lighthouse"`
}

// Parse message header, get message type and content length
func RpcReadPacket(client net.Conn) (c RpcCommandType, ret []byte, err error) {
	// exception handling
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			c = RPCCOMMANDUNKNOWN
		}
	}()

	header := make([]byte, 4)
	var n int
	n, err = client.Read(header)
	if err != nil {
		return
	}
	if n != 4 {
		err = errors.New("malformed header")
		return
	}

	c = RpcCommandType(binary.BigEndian.Uint16(header[0:2]))
	toread := int(binary.BigEndian.Uint16(header[2:4]))

	data := make([]byte, 1024)
	for toread > 0 {
		n, err = client.Read(data)
		if err != nil {
			return
		}
		ret = append(ret, data[0:n]...)
		toread -= n
	}

	return
}

func rpcInterfaceToType(msg interface{}) RpcCommandType {
	if _, isEntity := msg.(*RpcCommandStart); isEntity {
		return RPCCOMMANDSTART
	}
	if _, isEntity := msg.(*RpcCommandStop); isEntity {
		return RPCCOMMANDSTOP
	}
	if _, isEntity := msg.(*RpcCommandStatus); isEntity {
		return RPCCOMMANDSTATUS
	}
	if _, isEntity := msg.(*RpcCommandResponse); isEntity {
		return RPCCOMMANDRESPONSE
	}
	return RPCCOMMANDUNKNOWN
}

func RpcCreateMessage(msg interface{}) []byte {
	m, _ := json.Marshal(msg)
	var ret []byte
	r := make([]byte, 2)
	binary.BigEndian.PutUint16(r, uint16(rpcInterfaceToType(msg)))
	ret = append(ret, r...)
	binary.BigEndian.PutUint16(r, uint16(len(m)))
	ret = append(ret, r...)
	ret = append(ret, m...)
	return ret
}

func RpcSendMessage(conn net.Conn, msg interface{}) error {
	buf := RpcCreateMessage(msg)
	return RpcSendData(conn, buf)
}

func RpcSendData(conn net.Conn, buf []byte) error {
	count := 0
	for count < len(buf) {
		byteSent, err := conn.Write(buf[count:])
		if err != nil {
			return err
		}
		count += byteSent
	}
	return nil
}
