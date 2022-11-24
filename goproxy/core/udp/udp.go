package udp

import (
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/shieldoo/shieldoo-mesh/goproxy/config"
	"github.com/shieldoo/shieldoo-mesh/goproxy/core/scheduler"
	"github.com/shieldoo/shieldoo-mesh/goproxy/core/server"
)

// UDP proxy
type UDP struct {
	config        *config.Config
	listener      *net.UDPConn
	connStore     *sync.Map
	channelServer chan message
	channelClient chan message
}

// udp message
type message struct {
	Data []byte
	Conn *net.UDPConn
	Addr *net.UDPAddr
}

type conn struct {
	Conn   *net.UDPConn
	Active time.Time
}

// New return a new udp proxy instance
func New(config *config.Config) *UDP {
	t := new(UDP)
	t.config = config
	t.connStore = new(sync.Map)
	t.channelServer = make(chan message, 1024)
	t.channelClient = make(chan message, 1024)
	return t
}

// Start listen and serve
func (t *UDP) Start() {
	addr, err := net.ResolveUDPAddr("udp", t.config.Local)
	if err != nil {
		log.Printf("UDP.Start error: %v\n", err)
		return
	}
	for i := 0; i < 16; i++ {
		t.listener, err = net.ListenUDP("udp", addr)
		if err != nil {
			log.Printf("UDP.Start attempt(%d) error: %v\n", i, err)
			time.Sleep(500 * time.Millisecond)
		} else {
			break
		}
	}
	if err != nil {
		log.Printf("UDP.Start error: %v\n", err)
		return
	}
	log.Printf("UDP.Start %v, backends: %v\n", t.config.Local, t.config.Servers)

	go t.handleServer()
	go t.handleClient()
	go t.gc()

	for {
		data := make([]byte, 4096)
		n, remoteAddr, err := t.listener.ReadFromUDP(data)
		if err != nil {
			if !strings.Contains(err.Error(), server.ErrNetClosing.Error()) {
				log.Printf("UDP.listener.Read error: %v\n", err)
			}
			break
		}
		if t.config.Debug {
			_, ok := t.connStore.Load(remoteAddr.String())
			if !ok {
				log.Printf("UDP Client is connected: %v => %v\n", remoteAddr.String(), t.listener.LocalAddr())
			}
		}
		t.channelServer <- message{
			Data: data[:n],
			Conn: t.listener,
			Addr: remoteAddr,
		}
	}
}

// Stop stop serve
func (t *UDP) Stop() {
	log.Printf("UDP.Stop %v, backends: %v\n", t.config.Local, t.config.Servers)
	close(t.channelServer)
	close(t.channelClient)
	t.listener.Close()
}

func (t *UDP) handleServer() {
	for msg := range t.channelServer {
		var err error
		var dconn *net.UDPConn
		c, ok := t.connStore.Load(msg.Addr.String())
		if ok {
			dconn = c.(*conn).Conn
			dconn.SetWriteDeadline(time.Now().Add(time.Millisecond * t.config.Timeout))
			_, err = dconn.Write(msg.Data)
			dconn.SetWriteDeadline(time.Time{})
			if err != nil {
				log.Printf("UDP Past [%v] Send data failed: %v\n", dconn.RemoteAddr().String(), err)
			} else {
				t.connStore.Store(msg.Addr.String(), &conn{Conn: dconn, Active: time.Now()})
				continue
			}
		}

		serAddr := scheduler.Get(t.config.Scheduler).Schedule(msg.Addr.String(), t.config.Servers)
		udpAddr, err := net.ResolveUDPAddr("udp", serAddr)
		dconn, err = net.DialUDP("udp", nil, udpAddr)
		if err != nil {
			log.Printf("UDP connect to the server [%v] fail: %v\n", serAddr, err)
			break
		}
		log.Printf("UDP The server is connected: %v => %v\n", dconn.LocalAddr(), serAddr)

		dconn.SetWriteDeadline(time.Now().Add(time.Millisecond * t.config.Timeout))
		_, err = dconn.Write(msg.Data)
		dconn.SetWriteDeadline(time.Time{})
		if err != nil {
			log.Printf("UDP Past [%v] Send data failed: %v\n", dconn.RemoteAddr().String(), err)
		} else {
			t.connStore.Store(msg.Addr.String(), &conn{Conn: dconn, Active: time.Now()})
		}
		go func(msg message) {
			for {
				data := make([]byte, 4096)
				n, _, err := dconn.ReadFromUDP(data)
				if err != nil {
					if !strings.Contains(err.Error(), server.ErrNetClosing.Error()) {
						log.Printf("UDP From [%v] Receive data failed: %v\n", dconn.RemoteAddr().String(), err)
					}
					break
				}
				t.channelClient <- message{
					Data: data[:n],
					Conn: msg.Conn,
					Addr: msg.Addr,
				}
			}
		}(msg)
	}
}

func (t *UDP) handleClient() {
	for msg := range t.channelClient {
		_, err := msg.Conn.WriteToUDP(msg.Data, msg.Addr)
		if err != nil {
			log.Printf("UDP Past [%v] Send data failed: %v\n", msg.Addr.String(), err)
		}
	}
}

func (t *UDP) gc() {
	for {
		time.Sleep(time.Second * 10)
		t.connStore.Range(func(key, val interface{}) bool {
			conn, ok := val.(*conn)
			if !ok {
				t.connStore.Delete(key)
				return false
			}
			if conn.Active.Add(time.Second * 30).Before(time.Now()) {
				if t.config.Debug {
					log.Printf("UDP The server is released: %v => %v\n", conn.Conn.LocalAddr().String(), conn.Conn.RemoteAddr().String())
				}
				conn.Conn.Close()
				t.connStore.Delete(key)
			}
			return true
		})
	}
}
