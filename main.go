package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// global log data which are send to server during telemtry collection
var logdata chan string

// IP of lighthouse
var lighthouseIP string = ""
var lighthousePublicIpPort string = ""

// global logging
var log *logrus.Logger

func help(err string, out io.Writer) {
	if err != "" {
		fmt.Fprintln(out, "Error:", err)
		fmt.Fprintln(out, "")
	}

	fmt.Fprintf(out, "Usage of %s <global flags>:\n", os.Args[0])
	fmt.Fprintln(out, "  Global flags:")
	fmt.Fprintln(out, "    -debug: Run in debug mode with more detailed logging")
	fmt.Fprintln(out, "    -nofwcontrol: Disable control windows firewall rules")
	fmt.Fprintln(out, "    -version: Prints the version")
	fmt.Fprintln(out, "    -h, -help: Prints this help message")
	fmt.Fprintln(out, "    -log: Log to file in ./config directory")
	fmt.Fprintln(out, "    -desktop: Run service in desktop mode (for interaction with tray icon app)")
	fmt.Fprintln(out, "    -service: configure service [run, start, stop, restart, install, uninstall]")
	fmt.Fprintln(out, "    -createconfig: create configuration file from base64 input string")
}

func main() {
	// initialize logrus
	log = logrus.New()

	flag.Usage = func() {
		help("", os.Stderr)
		os.Exit(1)
	}

	runFlag := flag.Bool("run", false, "Run in CLI mode.")
	debugFlag := flag.Bool("debug", false, "Run in debug mode with more detailed logging.")
	noFwFlag := flag.Bool("nofwcontrol", false, "Disable control Windows firewall.")
	desktopFlag := flag.Bool("desktop", false, "Run in desktop service mode for interact with tray app.")
	serviceFlag := flag.String("service", "", "Control the system service.")
	printVersion := flag.Bool("version", false, "Print version")
	flagHelp := flag.Bool("help", false, "Print command line usage")
	flagLog := flag.Bool("log", false, "Log to file in ./config directory")
	flagH := flag.Bool("h", false, "Print command line usage")
	flagCreateConfig := flag.String("createconfig", "", "Create configuration file from base64 input string")
	disableHostsEdit := flag.String("disablehostsedit", "", "Disable hosts file editing [true, false]")
	printUsage := false

	flag.Parse()

	if *flagH || *flagHelp {
		printUsage = true
	}

	if *flagCreateConfig != "" {
		if err := CreateConfigFromBase64(*flagCreateConfig); err != nil {
			fmt.Printf("cannot create config: %v\n", err)
			os.Exit(1)
		} else {
			fmt.Printf("config created")
			os.Exit(0)
		}
	}

	if *disableHostsEdit != "" {
		disableEdit := *disableHostsEdit == "true"
		if err := UpdateConfigSetDisableHostsEdit(disableEdit); err != nil {
			fmt.Printf("cannot set disable hosts edit: %v\n", err)
			os.Exit(1)
		} else {
			fmt.Printf("disable hosts edit set to %v", disableEdit)
			os.Exit(0)
		}
	}

	//exception to running app without argument
	if *serviceFlag == "" && !(*debugFlag || *printVersion || *flagH || *runFlag) {
		fmt.Printf("Version: %v / %v\n", APPVERSION, ARCHITECTURE)
		os.Exit(0)
	}

	if *printVersion {
		fmt.Printf("Version: %v / %v\n", APPVERSION, ARCHITECTURE)
		os.Exit(0)
	}

	if printUsage {
		help("", os.Stdout)
		os.Exit(0)
	}

	if *flagLog {
		ex, err := os.Executable()
		if err != nil {
			panic(err)
		}
		expath := filepath.Dir(ex)
		f, err := os.OpenFile(filepath.FromSlash(filepath.FromSlash(expath+"/config/"+"shieldoo-log.log")), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			fmt.Printf("error opening file: %v", err)
		}
		log.SetOutput(f)
	}

	log.SetLevel(logrus.InfoLevel)
	log.Info("shieldoo-mesh version: ", APPVERSION)
	log.Debug("OS Args: ", os.Args)

	logdata = make(chan string, 10000)

	InitConfig(*desktopFlag)

	if *debugFlag {
		log.SetLevel(logrus.DebugLevel)
	}
	if myconfig.Debug {
		log.SetLevel(logrus.DebugLevel)
	}
	myconfig.WindowsFW = !(*noFwFlag)

	myconfig.RunAsDeskServiceRPC = *desktopFlag

	// cleanup DNS records in hosts file
	SvcCleanupDNS()

	if *serviceFlag != "" {
		// start service
		err := SystemSvcDo(*serviceFlag, *desktopFlag, *debugFlag)
		if *serviceFlag == "uninstall" || *serviceFlag == "stop" || err == nil {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	// start desktop service or standard server service in CLI mode
	if *desktopFlag {
		DeskserviceStart(false)
	} else {
		go ServiceCheckPinger()
		SvcConnectionStart(false)
	}
}
