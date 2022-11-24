package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	rpc "github.com/Shieldoo/shieldoo-mesh/rpc"
)

func main() {

	fmt.Println("start client")

	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	fmt.Println("executable: " + ex)

	flag.Usage = func() {
		fmt.Println("Usage:")
		fmt.Println(`-start -> start connection - use there json config in format: {"accessid":0,"uri":"","secret":""}`)
		fmt.Println(`-stop -> stop connection`)
		fmt.Println(`-status -> get status`)
		os.Exit(1)
	}

	startFlag := flag.String("start", "", `Start it. Send there json config in format: {"accessid":0,"uri":"","secret":""}`)
	stopFlag := flag.Bool("stop", false, "Stop it.")
	statusFlag := flag.Bool("status", false, "Get status.")

	flag.Parse()

	if *statusFlag {
		fmt.Println("status ..")
		m := rpc.RpcCommandStatus{Version: rpc.RPCVERSION}
		send(&m)
	}
	if *stopFlag {
		fmt.Println("stop ..")
		m := rpc.RpcCommandStop{Version: rpc.RPCVERSION}
		send(&m)
	}
	if *startFlag != "" {
		fmt.Println("start ..")
		fmt.Println(*startFlag)
		m := rpc.RpcCommandStart{}
		jerr := json.Unmarshal([]byte(*startFlag), &m)
		if jerr != nil {
			fmt.Printf("Error: %v\n", jerr)
			os.Exit(1)
		}
		m.Version = rpc.RPCVERSION
		send(&m)
	}
}

type CloseWriter interface {
	CloseWrite() error
}

func send(msg interface{}) {
	clientDone := make(chan bool)

	client, err := createClient()
	if err != nil {
		fmt.Println(err)
	}
	defer client.Close()

	go func() {
		// client read back
		bytes := make([]byte, 1024)
		n, e := client.Read(bytes)
		if e != nil {
			fmt.Println(e)
		}
		fmt.Printf("read bytes: %v\n", n)
		fmt.Printf("read header: %v\n", bytes[0:4])
		fmt.Printf("response: %v\n", string(bytes[4:]))
		close(clientDone)
	}()

	errs := rpc.RpcSendMessage(client, msg)
	if errs != nil {
		fmt.Println(errs)
	}
	fmt.Printf("OK send ..")
	client.(CloseWriter).CloseWrite()
	<-clientDone
}
