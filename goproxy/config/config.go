package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/Shieldoo/shieldoo-mesh/goproxy/core/scheduler"
)

// Configure
type Config struct {
	Debug     bool
	Scheduler string
	Protocol  string
	Timeout   time.Duration // time.Millisecond
	Local     string
	Servers   []string
}

// Create configuration
func New(protocol string, local string, server string) (*Config, error) {
	t := new(Config)
	t.Scheduler = scheduler.IPHashName
	t.Timeout = 2000
	if protocol == "" {
		protocol = "tcp"
	}
	if protocol == "tcp" || protocol == "udp" {
		t.Protocol = protocol
	} else {
		return nil, fmt.Errorf("Only support tcp/udp protocol")
	}
	if local == "" {
		return nil, fmt.Errorf("Local monitoring ports cannot be empty")
	}
	t.Local = local
	servers := strings.Split(server, ",")
	if len(servers) == 0 {
		return nil, fmt.Errorf("Real server address can't be empty")
	}
	t.Servers = servers

	return t, nil
}
