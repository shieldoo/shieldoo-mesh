package wstunnel

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

const WSST_MAXREADINACTIVITY float64 = 10
const WSST_MAXTIMEOUTS int = 5

type WSTunnel struct {
	locaddr       net.Addr
	udpconn       net.PacketConn
	udplock       sync.Mutex
	conn          *websocket.Conn
	url           string
	auth          string
	isrunning     bool
	localport     int
	lastWrite     time.Time
	lastRead      time.Time
	timeoutsCount int
}

func (t *WSTunnel) receiveHandler() {
	for {
		mt, msg, err := t.conn.ReadMessage()
		t.lastRead = time.Now()
		t.timeoutsCount = 0
		if err != nil {
			log.Debug("Error in receive:", err)
			if t.isrunning {
				log.Info("wstunnel reconnect ..")
				go t.reconnectWs()
			} else {
				log.Info("wstunnel close")
			}
			return
		}
		if mt == websocket.BinaryMessage {
			if _, err := t.udpconn.WriteTo(msg, t.locaddr); err != nil {
				log.Error("wstunnel error in send udp back to client:", err)
			}
		}
	}
}

func (t *WSTunnel) udpResendToWs(buf []byte) {
	t.udplock.Lock()
	defer t.udplock.Unlock()
	if t.conn != nil {
		if err := t.conn.WriteMessage(websocket.BinaryMessage, buf); err != nil {
			log.Debug("wstunnel send error: ", err)
		}
		t.lastWrite = time.Now()
		if t.lastWrite.Sub(t.lastRead).Seconds() > WSST_MAXREADINACTIVITY {
			log.Debug("wstunnel send TIMEOUT - retry: ", t.timeoutsCount)
			t.timeoutsCount++
			if t.timeoutsCount > WSST_MAXTIMEOUTS {
				t.conn.Close()
				log.Info("wstunnel send TIMEOUT - closing conn")
			}
		}
	}
}

func (t *WSTunnel) udpCreate() error {
	// listen to incoming udp packets
	log.Info("wstunnel udp create: ", fmt.Sprintf("127.0.0.1:%d", t.localport))
	var err error
	t.udpconn, err = net.ListenPacket("udp", fmt.Sprintf("127.0.0.1:%d", t.localport))
	if err != nil {
		log.Error("wstunnel udp listen error", err)
		return err
	}
	return nil
}

func (t *WSTunnel) udpServe() {
	if t.udpconn == nil {
		return
	}
	log.Debug("wstunnel udp start")

	defer t.udpconn.Close()

	for {
		buf := make([]byte, 2048)
		n, addr, err := t.udpconn.ReadFrom(buf)
		if err != nil {
			if !t.isrunning {
				log.Info("wstunnel udp close")
				return
			}
			continue
		}
		t.locaddr = addr
		go t.udpResendToWs(buf[:n])
	}
}

func (t *WSTunnel) IsRunning() bool {
	return t.isrunning
}

func (t *WSTunnel) reconnectWs() error {
	if t.conn != nil {
		t.conn.Close()
		t.conn = nil
	}
	err := t.connectWs()
	log.Println("ws connect:", err)
	if err != nil && t.isrunning {
		time.Sleep(1000 * time.Millisecond)
		if t.isrunning {
			go t.reconnectWs()
		}
	}
	t.lastRead = time.Now()
	t.lastWrite = time.Now()
	t.timeoutsCount = 0
	return err
}

func (t *WSTunnel) connectWs() error {
	h := http.Header{"Authorization": []string{"Basic " + t.auth}}
	con, _, err := websocket.DefaultDialer.Dial(t.url, h)
	if err != nil {
		if con != nil {
			con.Close()
		}
		t.conn = nil
		return err
	}
	t.conn = con
	go t.receiveHandler()
	return nil
}

func (t *WSTunnel) Start(UdpLocalPort int, Url string, Username string, Password string, AccessId int, UPN string) error {
	if t.isrunning {
		return nil
	}
	t.url = fmt.Sprintf("%s/wstunnel/udp/%s/%d", Url, UPN, AccessId)
	log.Info("wstunnel start: ", t.url)
	t.auth = base64.StdEncoding.EncodeToString([]byte(Username + ":" + Password))
	log.Info("wstunnel auth: ", t.auth)
	t.localport = UdpLocalPort
	err := t.udpCreate()
	if err != nil {
		log.Error("wstunnel cannot create udp server: ", err)
		return err
	}
	t.isrunning = true
	go t.reconnectWs()
	go t.udpServe()
	return nil
}

func (t *WSTunnel) Stop() error {
	t.isrunning = false
	if t.conn != nil {
		err := t.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Error("wstunnel error during closing websocket:", err)
		}
		t.isrunning = false
		t.conn.Close()
	}
	if t.udpconn != nil {
		t.udpconn.Close()
	}
	t.udpconn = nil
	t.conn = nil
	t.isrunning = false
	return nil
}
