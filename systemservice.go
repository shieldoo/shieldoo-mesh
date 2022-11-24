package main

import (
	"github.com/kardianos/service"
)

var logger service.Logger

var systemsvcIsDesktop bool = false

type program struct {
	test bool
}

func (p *program) Start(s service.Service) error {
	// Start should not block.
	log.Info("shieldoo-mesh service starting.")
	if systemsvcIsDesktop {
		go DeskserviceStart(true)
	} else {
		go SvcConnectionStart(true)
	}
	return nil
}

func (p *program) Stop(s service.Service) error {
	log.Info("shieldoo-mesh service stopping.")
	if systemsvcIsDesktop {
		DeskserviceStop()
	} else {
		SvcConnectionStop()
	}
	return nil
}

func SystemSvcDo(serviceFlag string, desktopFlag bool, debugFlag bool) (err error) {
	svcConfig := &service.Config{
		Name:        "shieldoo-mesh",
		DisplayName: "Shieldoo Mesh Network Service",
		Description: "Shieldoo Mesh network connectivity daemon for encrypted communications",
	}

	systemsvcIsDesktop = desktopFlag

	if desktopFlag {
		svcConfig.Arguments = []string{"-desktop", "-service", "run"}
	} else {
		svcConfig.Arguments = []string{"-service", "run"}
	}

	prg := &program{
		test: true,
	}

	// Here are what the different loggers are doing:
	// - `log` is the standard go log utility, meant to be used while the process is still attached to stdout/stderr
	// - `logger` is the service log utility that may be attached to a special place depending on OS (Windows will have it attached to the event log)
	// - above, in `Run` we create a `logrus.Logger` which is what nebula expects to use
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	errs := make(chan error, 256)
	logger, err = s.Logger(errs)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			err := <-errs
			if err != nil {
				// Route any errors from the system logger to stdout as a best effort to notice issues there
				log.Print(err)
			}
		}
	}()

	switch serviceFlag {
	case "run":
		HookLogerInit()
		HookLogger(log)
		err = s.Run()
		HookLogerClose()
		if err != nil {
			// Route any errors to the system logger
			log.Error(err)
		}
	default:
		err = service.Control(s, serviceFlag)
		if err != nil {
			log.Printf("Valid actions: %q\n", service.ControlAction)
			log.Fatal(err)
		}
	}
	return
}
