package main

import (
	"fmt"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

type NebulaYamlConfigFW struct {
	Port   string   `yaml:"port"`
	Proto  string   `yaml:"proto"`
	Host   string   `yaml:"host,omitempty"`
	Groups []string `yaml:"groups,omitempty"`
}

type NebulaYamlConfigUnsafeRoutes struct {
	Route string `yaml:"route"`
	Via   string `yaml:"via"`
}

type NebulaYamlConfig struct {
	Pki struct {
		Ca        string   `yaml:"ca"`
		Cert      string   `yaml:"cert"`
		Key       string   `yaml:"key"`
		Blocklist []string `yaml:"blocklist"`
	} `yaml:"pki"`
	StaticHostMap map[string][]string `yaml:"static_host_map"`
	Lighthouse    struct {
		AmLighthouse bool     `yaml:"am_lighthouse"`
		Interval     int      `yaml:"interval"`
		Hosts        []string `yaml:"hosts"`
	} `yaml:"lighthouse"`
	Listen struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"listen"`
	Punchy struct {
		Punch   bool `yaml:"punch"`
		Respond bool `yaml:"respond"`
	} `yaml:"punchy"`
	Relay struct {
		Relays    []string `yaml:"relays"`
		AmRelay   bool     `yaml:"am_relay"`
		UseRelays bool     `yaml:"use_relays"`
	} `yaml:"relay"`
	Tun struct {
		Disabled           bool                           `yaml:"disabled"`
		Dev                string                         `yaml:"dev"`
		DropLocalBroadcast bool                           `yaml:"drop_local_broadcast"`
		DropMulticast      bool                           `yaml:"drop_multicast"`
		TxQueue            int                            `yaml:"tx_queue"`
		Mtu                int                            `yaml:"mtu"`
		Routes             interface{}                    `yaml:"routes"`
		UnsafeRoutes       []NebulaYamlConfigUnsafeRoutes `yaml:"unsafe_routes"`
	} `yaml:"tun"`
	Logging struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
	} `yaml:"logging"`
	Firewall struct {
		Conntrack struct {
			TCPTimeout     string `yaml:"tcp_timeout"`
			UDPTimeout     string `yaml:"udp_timeout"`
			DefaultTimeout string `yaml:"default_timeout"`
			MaxConnections int    `yaml:"max_connections"`
		} `yaml:"conntrack"`
		Outbound []NebulaYamlConfigFW `yaml:"outbound"`
		Inbound  []NebulaYamlConfigFW `yaml:"inbound"`
	} `yaml:"firewall"`
}

func NebulaConfigGetLighthouseIP(configdata string) (lighthouse string, err error) {
	c := &NebulaYamlConfig{}
	buf := []byte(configdata)
	err = yaml.Unmarshal(buf, c)
	if err != nil {
		log.Debug("Error deserialize nebula config: ", err)
		return "", err
	}
	// read light house IP and port from hostmap
	for _, k := range c.StaticHostMap {
		// get first IP:port for lighthouse
		if len(k) > 0 {
			// parse IP from IP:port syntax
			lighthouse = strings.Split(k[0], ":")[0]
		}
		break
	}
	return
}

func NebulaConfigCreate(configdata string, punchback bool, isrestrictednetwork bool) (string, string, error) {
	c := &NebulaYamlConfig{}
	var err error
	lhIP := ""
	lhInternalIP := ""
	buf := []byte(configdata)
	err = yaml.Unmarshal(buf, c)
	if err != nil {
		log.Debug("Error deserialize nebula config: ", err)
		return "", "", err
	}
	c.Punchy.Respond = punchback
	c.Relay.UseRelays = true
	// read light house IP and port from hostmap
	for kkey, k := range c.StaticHostMap {
		// parse IP and port
		if len(k) > 0 {
			lhIP = k[0]
		}
		lhInternalIP = kkey
		break
	}
	if isrestrictednetwork {
		// change host map for WSS style of communication - for first lighthouse
		for k, _ := range c.StaticHostMap {
			c.StaticHostMap[k] = []string{fmt.Sprintf("127.0.0.1:%d", myconfig.LocalUDPPort)}
			break
		}
	}
	// exception for darwin (get from GOOS), ignore Dev name
	if runtime.GOOS == "darwin" {
		c.Tun.Dev = ""
	}
	// sanitize HOST, PORT config
	c.Listen.Host = "0.0.0.0"
	c.Listen.Port = 0

	// if there is enabled LighthouseRoute add there routes via lighthouse
	if myconfig.LighthouseRoute {
		if len(ServicecheckServiceDNSIPsData) > 0 {
			// generate route list
			log.Debug("ServicecheckServiceDNSIPsData: ", ServicecheckServiceDNSIPsData)
			routeList := NetutilsGenerateRoutes(ServicecheckServiceDNSIPsData)
			// add routes to lighthouse
			for _, v := range routeList {
				c.Tun.UnsafeRoutes = append(c.Tun.UnsafeRoutes, NebulaYamlConfigUnsafeRoutes{Route: v, Via: lhInternalIP})
			}
		}
	}

	buf, err = yaml.Marshal(&c)
	if err != nil {
		log.Debug("Error serialize nebula config: ", err)
		return "", lhIP, err
	}

	return string(buf), lhIP, err
}
