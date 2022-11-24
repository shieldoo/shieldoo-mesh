package main

import (
	"encoding/json"
	"errors"

	rpc "github.com/Shieldoo/shieldoo-mesh/rpc"
)

type CloseWriter interface {
	CloseWrite() error
}

func rpcSendReceive(msg interface{}) (ret *rpc.RpcCommandResponse, err error) {
	ret = nil
	var errread error = nil
	err = nil
	clientDone := make(chan bool)

	client, e := createClient()

	if e != nil {
		err = e
		return
	}
	defer client.Close()

	go func() {
		// read response
		ct, data, e := rpc.RpcReadPacket(client)
		errread = e
		if e == nil {
			if ct == rpc.RPCCOMMANDRESPONSE {
				j := rpc.RpcCommandResponse{}
				if errread = json.Unmarshal(data, &j); errread == nil {
					ret = &j
				}
			} else {
				errread = errors.New("wrong command type")
			}
		}
		close(clientDone)
	}()

	errs := rpc.RpcSendMessage(client, msg)
	if errs != nil {
		err = errs
	}
	client.(CloseWriter).CloseWrite()
	<-clientDone
	if err == nil {
		err = errread
	}
	return
}
