package scheduler

import (
	"net"
)

// IPHash IP-based scheduler
type IPHash struct {
}

// Schedule 调度
func (strategy *IPHash) Schedule(client string, servers []string) string {
	host, _, _ := net.SplitHostPort(client)
	intIP := int(IP2Long(host))
	length := len(servers)
	server := servers[intIP%length]
	return server
}

func init() {
	Register(IPHashName, new(IPHash))
}
