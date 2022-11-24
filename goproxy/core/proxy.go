package core

import (
	"log"

	"github.com/Shieldoo/shieldoo-mesh/goproxy/config"
	"github.com/Shieldoo/shieldoo-mesh/goproxy/core/server"
	"github.com/Shieldoo/shieldoo-mesh/goproxy/core/tcp"
	"github.com/Shieldoo/shieldoo-mesh/goproxy/core/udp"
)

// Proxy Agent implementation
type Proxy struct {
	Config   *config.Config
	Shutdown chan struct{}
}

// Create a new proxy instance
func New(config *config.Config) *Proxy {
	t := new(Proxy)
	t.Config = config
	t.Shutdown = make(chan struct{})
	return t
}

// Start service
func (t *Proxy) Start() {
	var s server.Server
	switch t.Config.Protocol {
	case "tcp":
		s = tcp.New(t.Config)
	case "udp":
		s = udp.New(t.Config)
	}
	go s.Start()

	<-t.Shutdown
	s.Stop()
	log.Println("proxy stopped")
}

// Stop
func (t *Proxy) Stop() {
	t.Shutdown <- struct{}{}
}

// monitor Listening system signal, restart or stop service
/*
func (t *Proxy) signal() {
	sch := make(chan os.Signal, 10)
	signal.Notify(sch, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGINT,
		syscall.SIGHUP, syscall.SIGSTOP, syscall.SIGQUIT)
	go func(ch <-chan os.Signal) {
		sig := <-ch
		log.Println("signal recieved " + sig.String() + ", at: " + time.Now().Format("2006-01-02 15:04:05"))
		t.Stop()
		if sig == syscall.SIGHUP {
			log.Println("proxy restart now...")
			procAttr := new(os.ProcAttr)
			procAttr.Files = []*os.File{nil, os.Stdout, os.Stderr}
			procAttr.Dir = os.Getenv("PWD")
			procAttr.Env = os.Environ()
			process, err := os.StartProcess(os.Args[0], os.Args, procAttr)
			if err != nil {
				log.Println("proxy restart process failed:" + err.Error())
				return
			}
			waitMsg, err := process.Wait()
			if err != nil {
				log.Println("proxy restart wait error:" + err.Error())
			}
			log.Println(waitMsg)
		} else {
			log.Println("proxy shutdown now...")
		}
	}(sch)
}
*/
