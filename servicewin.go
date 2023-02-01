//go:build windows
// +build windows

package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"

	"github.com/Microsoft/go-winio"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
)

func svcFirewallSetup(cidr string) {
	cmd := exec.Command("netsh", "advfirewall", "firewall", "add", "rule", "name=ShieldooMesh",
		"dir=in", "action=allow", "interfacetype=any", "protocol=any", "profile=any",
		"localip="+cidr, "remoteip="+cidr)
	log.Debug("adding firewall rule for ShieldooMesh")
	err := cmd.Run()
	if err != nil {
		log.Error("cannot execute netsh: ", err)
	}
}

func svcFirewallCleanup() {
	cmd := exec.Command("netsh", "advfirewall", "firewall", "delete", "rule", "name=ShieldooMesh")
	log.Info("deleting firewall rule for ShieldooMesh")
	err := cmd.Run()
	if err != nil {
		log.Error("cannot execute netsh: ", err)
	}
}

const (
	// This will set permissions for everyone to have full access
	AllowEveryone = "S:(ML;;NW;;;LW)D:(A;;0x12019f;;;WD)"
)

var connPipeName = `\\.\pipe\shieldoopipe`

func createCommandListener() (net.Listener, error) {
	c := winio.PipeConfig{
		MessageMode:        true,  // Use message mode so that CloseWrite() is supported
		InputBufferSize:    65536, // Use 64KB buffers to improve performance
		OutputBufferSize:   65536,
		SecurityDescriptor: AllowEveryone,
	}
	log.Debug("create listener to: ", connPipeName)
	return winio.ListenPipe(connPipeName, &c)
}

var elog *eventlog.Log

// HookLogger routes the logrus logs through the service logger so that they end up in the Windows Event Viewer
// logrus output will be discarded
func HookLogger(l *logrus.Logger) {
	l.Hooks.Add(NewHook(elog))
	//l.SetOutput(ioutil.Discard)
}

func HookLogerInit() {
	var err error
	elog, err = eventlog.Open("shieldoo")
	if err != nil {
		log.Fatal("cannot open windows log", err)
		panic(err)
	}
}

func HookLogerClose() {
	elog.Close()
}

// EventLogHook to send logs via windows log.
type EventLogHook struct {
	upstream debug.Log
}

// NewHook creates and returns a new EventLogHook wrapped around anything that implements the debug.Log interface
func NewHook(logger debug.Log) *EventLogHook {
	return &EventLogHook{upstream: logger}
}

func (hook *EventLogHook) Fire(entry *logrus.Entry) error {
	line, err := entry.String()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read entry, %v", err)
		return err
	}

	switch entry.Level {
	case logrus.PanicLevel:
		return hook.upstream.Error(3, line)
	case logrus.FatalLevel:
		return hook.upstream.Error(2, line)
	case logrus.ErrorLevel:
		return hook.upstream.Error(1, line)
	case logrus.WarnLevel:
		return hook.upstream.Warning(1, line)
	case logrus.InfoLevel:
		return hook.upstream.Info(2, line)
	case logrus.DebugLevel:
		return hook.upstream.Info(1, line)
	default:
		return nil
	}
}

func (hook *EventLogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func DetachOsProcess(cmd *exec.Cmd) {
	// Do nothing because it is not needed for windows
}
