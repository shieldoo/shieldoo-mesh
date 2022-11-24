package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	proxyconf "github.com/shieldoo/shieldoo-mesh/goproxy/config"
	proxy "github.com/shieldoo/shieldoo-mesh/goproxy/core"
	wstunnel "github.com/shieldoo/shieldoo-mesh/wstunnel"

	"github.com/sirupsen/logrus"
	"github.com/slackhq/nebula"
	"github.com/slackhq/nebula/config"
)

type ChannelWriter struct {
	canwrite bool
}

func (p *ChannelWriter) Write(data []byte) (n int, err error) {
	if p.canwrite {
		s := string(data)
		// ignore strange messages from log
		if strings.Contains(s, `"msg":"Failed to write to tun"`) {
			// ignored messages
			fmt.Printf("NEBULA-: %s", data)
		} else {
			// collected messages
			fmt.Printf("NEBULA+: %s", data)
			logdata <- string(data)
		}
	} else {
		fmt.Printf("NEBULA#: %s", data)
	}
	return len(data), nil
}

type SvcProxyRoute struct {
	Port        int
	Protocol    string
	ForwardPort int
	ForwardHost string
	Proxy       *proxy.Proxy
}

func (r *SvcProxyRoute) Stop() {
	log.Debug("stoppping worker: ", r)
	r.Proxy.Stop()
}

func (r *SvcProxyRoute) Start(ip string) error {
	log.Debug("starting worker "+ip+": ", r)
	srvs := fmt.Sprintf("%s:%d", r.ForwardHost, r.ForwardPort)
	listen := fmt.Sprintf("%s:%d", ip, r.Port)
	config, err := proxyconf.New(r.Protocol, listen, srvs)
	if err != nil {
		log.Error("cannot start worker: ", err)
	}
	r.Proxy = proxy.New(config)
	go r.Proxy.Start()
	return nil
}

func (r *SvcProxyRoute) IsEqualToModel(m *ManagementResponseListener) bool {
	return r.ForwardHost == m.ForwardHost && r.ForwardPort == m.ForwardPort && r.Port == m.Port && r.Protocol == m.Protocol
}

func (r *SvcProxyRoute) IsInModel(m *[]ManagementResponseListener) bool {
	if m == nil {
		return false
	}
	for _, e := range *m {
		if r.IsEqualToModel(&e) {
			return true
		}
	}
	return false
}

type SvcNetworkCard struct {
	AccessID            int
	ConfigHash          string
	IPAddress           string
	Workers             []SvcProxyRoute
	nebula              *nebula.Control
	ncfg                *config.C
	log                 ChannelWriter
	nl                  *logrus.Logger
	RestrictiveNetworks bool
	PunchBack           bool
}

func (r *SvcNetworkCard) Stop() {
	// stop all listeners
	for _, i := range r.Workers {
		i.Stop()
	}

	// get tun/tap name
	ttname := r.ncfg.GetString("tun.dev", "")

	log.Debug("stopping nebula ..")
	if r.nebula != nil {
		r.nebula.Stop()
	}
	r.nebula = nil
	runtime.GC()

	// wait for a while to deallocate TUN/TAP
	time.Sleep(1000 * time.Millisecond)
	if ttname != "" {
		if runtime.GOOS == "linux" {
			for i := 1; i < 300; i++ {
				time.Sleep(1000 * time.Millisecond)

				// check if adapter exists - if yes than we have to wait to disapear
				if _, err := os.Stat("/sys/class/net/" + ttname + "/mtu"); err != nil {
					break
				}
				log.Debug("waiting for tun/tap disappear: ", ttname)

				// trying to delete interface
				if i%10 == 0 {
					cmd := exec.Command("ip", "link", "delete", ttname)
					log.Info("deleting tun/tap: ", ttname)
					err := cmd.Run()
					if err != nil {
						log.Error("cannot execute ip link delete: ", err)
					}
				}
			}
		}
	}

	log.Debug("nebula stopped")
	log.Debug("stoped nebula with ip ", r.IPAddress)
}

var svcProcess *SvcNetworkCard = nil
var svcWsTunnel wstunnel.WSTunnel
var svcIsInitialized bool = false

func svcCleanupWorkers(process *SvcNetworkCard, cfg *ManagementResponseConfig, cleanupall bool) {
	var w []SvcProxyRoute
	for _, r := range process.Workers {
		if cleanupall || (cfg != nil && !r.IsInModel(&cfg.ApplianceListeners)) {
			r.Stop()
		} else {
			w = append(w, r)
		}
	}
	process.Workers = w
}

func svcCleanupProcesses(cfg *NebulaLocalYamlConfig) {
	if svcProcess != nil {
		if svcProcess.AccessID != cfg.ConfigData.AccessID /* accessID changed */ ||
			svcProcess.IPAddress != cfg.ConfigData.ConfigData.IPAddress /* IP address of tun/tap changed */ ||
			svcProcess.RestrictiveNetworks != myconfig.RestrictedNetwork /* if restrictive network changed */ {
			// there is change in config which will recreate network adapter
			svcStopProcess()
			// cleanup changes to windows firewall
			if myconfig.WindowsFW {
				svcFirewallCleanup()
			}
		}
	}
}

func svcStopProcess() {
	log.Debug("stopping service ..")
	// stop standard nebula layer
	if svcProcess != nil {
		log.Debug("stopping service: ", svcProcess.IPAddress)
		svcCleanupWorkers(svcProcess, nil, true)
		svcProcess.Stop()
		svcProcess = nil
		runtime.GC()
	}
}

func svcCancelableWait(periodSeconds int) {
	log.Debug("svcCancelableWait() waiting for ", periodSeconds, " seconds")
	for i := 0; i < periodSeconds*10; i++ {
		if svcconnCancel {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func svcNewProcess(c *ManagementResponseConfig, enableWinLog bool) (SvcNetworkCard, error) {
	ret := SvcNetworkCard{
		AccessID:            c.AccessID,
		ConfigHash:          c.ConfigData.Hash,
		IPAddress:           c.ConfigData.IPAddress,
		PunchBack:           c.NebulaPunchBack,
		RestrictiveNetworks: myconfig.RestrictedNetwork,
	}

	log.Debug("create service: ", c.ConfigData.IPAddress)
	maxNebulaRetries := 4

	// ### start process

	// create config file
	cfgtext, lhIP, err := NebulaConfigCreate(
		c.ConfigData.Data,
		ret.PunchBack,
		myconfig.RestrictedNetwork)
	if err != nil {
		return ret, err
	}
	lighthousePublicIpPort = lhIP

	ret.log.canwrite = false
	ret.nl = logrus.New()
	ret.nl.Out = &ret.log
	if enableWinLog {
		HookLogger(ret.nl)
	}
	ret.ncfg = config.NewC(ret.nl)
	err = ret.ncfg.LoadString(cfgtext)

	//get lighthouse from config
	lgths := ret.ncfg.GetStringSlice("lighthouse.hosts", []string{})
	if len(lgths) >= 1 {
		lighthouseIP = lgths[0]
	}
	log.Debug("lighthouse IP: ", lighthouseIP)

	if err != nil {
		log.Error("failed to load config: ", err)
		return ret, err
	}

	for i := 1; i <= maxNebulaRetries; i++ {
		ctrl, err := nebula.Main(ret.ncfg, false, APPVERSION, ret.nl, nil)
		if err == nil {
			ret.nebula = ctrl
			break
		}
		if err != nil && (i == maxNebulaRetries || svcconnCancel) {
			log.Error("failed to start nebula: ", err)
			return ret, err
		}
		ctrl = nil
		log.Error("repeating start of nebula: ", err)
		svcCancelableWait(i)
		if runtime.GOOS == "windows" &&
			err.Error() == "create Wintun interface failed, create TUN device failed: Error creating interface: The system cannot find the file specified." {
			cmd := exec.Command("pnputil", "/remove-device", "ROOT\\WINTUN\\0000")
			log.Info("deleting tun/tap: ROOT\\WINTUN\\0000")
			err := cmd.Run()
			if err != nil {
				log.Error("cannot execute pnputil /remove-device: ", err)
			}
			svcCancelableWait(i)
		}
	}
	ret.log.canwrite = true
	log.Debug("start nebula with ip ", ret.IPAddress)
	ret.nebula.Start()

	// configure windows firewall
	if myconfig.WindowsFW && len(c.NebulaCIDR) > 0 {
		log.Debug("configuring windows firewall for cidr: ", c.NebulaCIDR)
		svcFirewallSetup(c.NebulaCIDR)
	}

	// wait for a while to create TUN/TAP
	time.Sleep(500 * time.Millisecond)
	return ret, nil
}

func svcFindWorker(c *ManagementResponseListener, proc *SvcNetworkCard) *SvcProxyRoute {
	for i, r := range proc.Workers {
		if r.IsEqualToModel(c) {
			return &proc.Workers[i]
		}
	}
	return nil
}

func svcUpdateWorkers(netw *ManagementResponseConfig, proc *SvcNetworkCard) bool {
	var ret bool = true

	//stop workers where we have change
	svcCleanupWorkers(proc, netw, false)

	// start new workers
	for _, w := range netw.ApplianceListeners {
		if svcFindWorker(&w, proc) == nil {
			worker := SvcProxyRoute{
				Port:        w.Port,
				Protocol:    w.Protocol,
				ForwardPort: w.ForwardPort,
				ForwardHost: w.ForwardHost,
			}
			if worker.Start(proc.IPAddress) != nil {
				ret = false
			} else {
				proc.Workers = append(proc.Workers, worker)
			}
		}
	}

	return ret
}

func svcUpdateProcesses(cfg *NebulaLocalYamlConfig, enableWinLog bool) bool {
	log.Debug("updating services ..")
	if cfg.ConfigData != nil {
		// create new nebula process if needed
		if svcProcess == nil {
			svcProcess = nil
			// standard proccess
			newp, err := svcNewProcess(cfg.ConfigData, enableWinLog)
			if err != nil {
				newp.Stop()
				return false
			}
			svcProcess = &newp
		} else {
			// update properties of running nebula
			if svcProcess.ConfigHash != cfg.ConfigHash {
				svcProcess.PunchBack = cfg.ConfigData.NebulaPunchBack
				// create config files
				cfgtext, lhIP, err := NebulaConfigCreate(
					cfg.ConfigData.ConfigData.Data,
					svcProcess.PunchBack,
					myconfig.RestrictedNetwork)
				if err != nil {
					log.Error("failed to create config: ", err)
					return false
				}
				lighthousePublicIpPort = lhIP
				log.Debug("updating services ..")
				err = svcProcess.ncfg.ReloadConfigString(cfgtext)
				if err != nil {
					log.Error("failed to reload config: ", err)
					svcStopProcess()
					return false
				}
				log.Debug("reload config for nebula with ip ", svcProcess.AccessID)
				svcProcess.ConfigHash = cfg.ConfigHash
			}
		}
		// create reverse proy threads if needed (not for underlay connection)
		if !svcUpdateWorkers(cfg.ConfigData, svcProcess) {
			return false
		}
	}
	return true
}

func configureServices(enableWinLog bool) bool {
	log.Debug("create service..")
	var ret bool = true
	// cleanup not existing network configs or changed ..
	svcCleanupProcesses(&localconf)
	// create new processes if needed and update workers..
	if !svcUpdateProcesses(&localconf, enableWinLog) {
		ret = false
	}
	return ret
}

var svcconnCancel bool = false
var svcconnIsRunning bool = false
var svcconnStopped chan bool

func svcConnectWstunnel(accessid int, upn string) {
	log.Debug("svcConnectWstunnel - starting wstunnel")
	if !svcWsTunnel.IsRunning() {
		_usr, _pwd, _wss := WSTunnelCredentials()
		if _usr == "" || _pwd == "" || _wss == "" {
			log.Error("wstunnel address or credentials is not provided, cannot start")
			return
		}
		svcWsTunnel.Start(myconfig.LocalUDPPort, _wss, _usr, _pwd, accessid, upn)
	}
}

func svcDisconnectWstunnel() {
	log.Debug("svcDisconnectWstunnel - stopping wstunnel")
	if !svcWsTunnel.IsRunning() {
		return
	}
	svcWsTunnel.Stop()
}

func SvcConnectionStart(enableWinLog bool) {
	log.Debug("svcconnection starting ..")
	if svcconnIsRunning {
		return
	}
	log.Debug("svcconnection starting ....")
	svcconnCancel = false
	svcconnStopped = make(chan bool)
	// insert into log channel empty string to initialize immediate sending after startup
	logdata <- ""
	svcconnIsRunning = true
	for {
		// run telemetry and config
		log.Debug("waiting for next telemetry send ..")
		if telemetrySend() ||
			!svcIsInitialized {
			if localconf.Loaded {
				if myconfig.RestrictedNetwork {
					svcConnectWstunnel(localconf.ConfigData.AccessID, localconf.ConfigData.UPN)
				}
				if !myconfig.RestrictedNetwork {
					svcDisconnectWstunnel()
				}
				//dns
				loadDNS()
				// need restart or its first time
				svcIsInitialized = configureServices(enableWinLog)
			}
		}
		if svcconnCancel {
			// stop services
			svcStopProcess()
			// send stop signal
			svcconnStopped <- true
			break
		}
	}
	svcconnIsRunning = false
}

func SvcConnectionStop() {
	log.Debug("svcconnection stopping ..")
	if svcconnCancel || !svcconnIsRunning {
		return
	}
	log.Debug("svcconnection stopping ....")

	svcconnCancel = true
	// invoke break of waiting loop in telemtrySend
	logdata <- ""

	// wait for stop nebula connections
	if svcconnStopped != nil {
		<-svcconnStopped
	}
	// stoppping wstunnel if exists
	svcDisconnectWstunnel()
	svcconnCancel = false

	// cleanup configs
	removeLocalConf()

	// cleanup DNS
	loadDNS()

	// cleanup windows firewall
	if myconfig.WindowsFW {
		svcFirewallCleanup()
	}
}
