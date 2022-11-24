package scheduler

import (
	"time"
)

// Random Random scheduler
type Random struct {
}

// Schedule Dispatch
func (strategy *Random) Schedule(client string, servers []string) string {
	length := len(servers)
	server := servers[int(time.Now().UnixNano())%length]
	return server
}

func init() {
	Register(RandomName, new(Random))
}
