package tcp

import (
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/Shieldoo/shieldoo-mesh/goproxy/config"
	"github.com/Shieldoo/shieldoo-mesh/goproxy/core/scheduler"
	"github.com/Shieldoo/shieldoo-mesh/goproxy/core/server"
)

// TCP proxy
type TCP struct {
	config   *config.Config
	listener net.Listener
}

// New return a new tcp proxy instance
func New(config *config.Config) *TCP {
	t := new(TCP)
	t.config = config
	return t
}

// Start listen and serve
func (t *TCP) Start() {
	var err error
	for i := 0; i < 16; i++ {
		t.listener, err = net.Listen("tcp", t.config.Local)
		if err != nil {
			log.Printf("TCP.Start attempt(%d) error: %v\n", i, err)
			time.Sleep(500 * time.Millisecond)
		} else {
			break
		}
	}
	if err != nil {
		log.Printf("TCP.Start error: %v\n", err)
		return
	}

	defer t.Stop()
	log.Printf("TCP.Start %v, backends: %v\n", t.config.Local, t.config.Servers)

	for {
		conn, err := t.listener.Accept()
		if err != nil {
			if !strings.Contains(err.Error(), server.ErrNetClosing.Error()) {
				log.Printf("TCP.Listener.accept error: %v\n", err)
			}
			break
		}
		if t.config.Debug {
			log.Printf("TCP Client is connected: %v => %v\n", conn.RemoteAddr(), conn.LocalAddr())
		}
		go t.handle(conn)
	}
}

// Stop stop serve
func (t *TCP) Stop() {
	log.Printf("TCP.Stop %v, backends: %v\n", t.config.Local, t.config.Servers)
	lic, ok := t.listener.(*net.TCPListener)
	if ok {
		lic.Close()
	}
}

func (t *TCP) handle(sconn net.Conn) {
	defer func() {
		log.Printf("TCP Client is closed: %v => %v\n", sconn.RemoteAddr(), sconn.LocalAddr())
		sconn.Close()
	}()

	addr := scheduler.Get(t.config.Scheduler).Schedule(sconn.RemoteAddr().String(), t.config.Servers)
	dconn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Printf("TCP connect to the server [%v] fail: %v\n", addr, err)
		return
	}
	defer func() {
		log.Printf("TCP The server is closed: %v => %v\n", dconn.LocalAddr(), addr)
		dconn.Close()
	}()
	log.Printf("TCP The server is connected: %v => %v\n", dconn.LocalAddr(), addr)

	closeChan := make(chan struct{}, 1)
	go func(sconn net.Conn, dconn net.Conn, closeChan chan struct{}) {
		_, err := io.Copy(dconn, sconn)
		if err != nil {
			if err == io.EOF {
				// Read after reading
			} else if strings.Contains(err.Error(), server.ErrNetClosing.Error()) {
				// log.Printf("TCP Past [%v] Send data failed: Connection has been closed %v\n", addr, err)
			} else {
				log.Printf("TCP Past [%v] Send data failed: %v\n", addr, err)
			}
		}
		closeChan <- struct{}{}
	}(sconn, dconn, closeChan)

	go func(sconn net.Conn, dconn net.Conn, closeChan chan struct{}) {
		_, err := io.Copy(sconn, dconn)
		if err != nil {
			if err == io.EOF {
				// Read after reading
			} else if strings.Contains(err.Error(), server.ErrNetClosing.Error()) {
				// log.Printf("TCP From [%v] Receive data failed: Connection has been closed %v\n", ip, err)
			} else {
				log.Printf("TCP From [%v] Receive data failed: %v\n", addr, err)
			}
		}
		closeChan <- struct{}{}
	}(sconn, dconn, closeChan)

	<-closeChan
}
