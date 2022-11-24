package scheduler

const (
	IPHashName = "iphash"
	RandomName = "random"
)

var store = make(map[string]Scheduler)

// scheduler
type Scheduler interface {
	Schedule(client string, servers []string) string
}

// Get scheduler
func Get(name string) Scheduler {
	return store[name]
}

// Registration scheduler
func Register(name string, handle Scheduler) {
	store[name] = handle
}
