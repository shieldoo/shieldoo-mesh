package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/cloudfieldcz/beeep"
	"github.com/cloudfieldcz/systray"
	rpc "github.com/shieldoo/shieldoo-mesh/rpc"
	"github.com/shieldoo/shieldoo-mesh/systray/autostart"
	icon "github.com/shieldoo/shieldoo-mesh/systray/icon"
	inputbox "github.com/shieldoo/shieldoo-mesh/systray/inputbox"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/skratchdot/open-golang/open"
)

func help(err string, out io.Writer) {
	if err != "" {
		fmt.Fprintln(out, "Error:", err)
		fmt.Fprintln(out, "")
	}

	fmt.Fprintf(out, "Usage of %s <global flags>:\n", os.Args[0])
	fmt.Fprintln(out, "  Global flags:")
	fmt.Fprintln(out, "    -debug: Run in debug mode with more detailed logging")
	fmt.Fprintln(out, "    -log: log to file HOME/.shieldoo/log.log")
	fmt.Fprintln(out, "    -h, -help: Prints this help message")
	fmt.Fprintln(out, "    -url: URL address of Shieldoo Mesh")
}

var (
	autostartApp          *autostart.App
	connectionIsConnected bool = false
	execPath              string
)

func initAutostartApp() {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}

	autostartApp = &autostart.App{
		Name:        "ShieldooMesh",
		DisplayName: "Shieldoo Mesh",
		Exec:        []string{ex},
	}
}

func cleanupWinIcons() {
	//filepath.Join(os.TempDir(), "systray_temp_icon_"+dataHash)
	if runtime.GOOS == "windows" {
		files, err := filepath.Glob(filepath.Join(os.TempDir(), "systray_temp_icon_*"))
		if err == nil {
			for _, f := range files {
				os.Remove(f)
			}
		}
	}
}

func init() {
	beeepInit()
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	execPath = filepath.Dir(ex)
}

func main() {
	onExit := func() {
	}

	flag.Usage = func() {
		help("", os.Stderr)
		os.Exit(1)
	}

	debugFlag := flag.Bool("debug", false, "Run in debug mode with more detailed logging.")
	autostartFlag := flag.Bool("autostart", false, "Enable autostart.")
	logFlag := flag.Bool("log", false, "Log to file HOME/.shieldoo/log.log.")
	urlFlag := flag.String("url", "", "Control the system service.")
	flagH := flag.Bool("h", false, "Print command line usage")

	flag.Parse()

	if *flagH {
		help("", os.Stdout)
		os.Exit(0)
	}

	cleanupWinIcons()

	//setup autostart
	if *autostartFlag {
		initAutostartApp()
		autostartApp.Enable()
		os.Exit(0)
	}

	// create config folder if not exists
	_ = os.MkdirAll(filepath.FromSlash(getConfigDir()), 0700)

	if *logFlag {
		f, err := os.OpenFile(filepath.FromSlash(getConfigDir()+"/log.log"), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			fmt.Printf("error opening file: %v", err)
		}
		log.SetOutput(f)
	}
	if *debugFlag {
		log.SetLevel(logrus.DebugLevel)
	}
	log.SetLevel(logrus.DebugLevel)
	log.Info("Starting ..")
	log.Info("Version: ", APPVERSION)
	log.Debug("Debug mode enabled")

	InitConfig()

	// set url
	if *urlFlag != "" {
		myconfig.Uri = *urlFlag
		if !strings.HasSuffix(myconfig.Uri, "/") {
			myconfig.Uri += "/"
		}
		saveClientConf()
		os.Exit(0)
	}

	if runtime.GOOS == "windows" {
		// check if application is running once
		// Windows specific code ..
		_, err := CreateMutex("NebulaClientApp")

		if err != nil {
			fmt.Printf("Error: %d - %s\n", int(err.(syscall.Errno)), err.Error())
			MessageBoxPlain("Shieldoo CLientApp Error", "shieldoo-app.exe is already running!")
			panic("cannot create global mutex")
		}
	}

	// updadte check
	go UpdManagerRun()

	systray.Run(onReady, onExit)
}

var mConnectEnabled bool = true
var mUpdate *systray.MenuItem = nil
var mConnect *systray.MenuItem = nil
var mConnectDefault *systray.MenuItem = nil
var mLogin *systray.MenuItem = nil
var mEditUrl *systray.MenuItem = nil
var mConnectSub []*systray.MenuItem = nil
var maxMenuItems int = 24
var mDisconnect *systray.MenuItem = nil
var running bool = false
var runningDisconnecting bool = false
var activeaccessid int = 0
var registering bool = false
var registeringCode string = ""
var serverMessage string = ""
var systrayToolTip string = ""
var systrayIcon []byte
var serverMessageChan chan string = make(chan string)

func systrayMenuItemDisable(itm *systray.MenuItem) {
	if !itm.Disabled() {
		itm.Disable()
	}
}

func systrayMenuItemEnable(itm *systray.MenuItem) {
	if itm.Disabled() {
		itm.Enable()
	}
}

func systraySetToolTip(text string) {
	if systrayToolTip != text {
		systrayToolTip = text
		systray.SetTooltip(text)
	}
}

func systraySetTemplateIcon(buf []byte) {
	if !bytes.Equal(systrayIcon, buf) {
		systray.SetTemplateIcon(buf, buf)
		systrayIcon = make([]byte, len(buf))
		copy(systrayIcon, buf)
	}
}

func connectNebulaUIDefult() {
	if localconfGetAccessesLen() == 1 {
		connectNebulaUI(0)
	}
}

func connectNebulaUI(index int) {
	if mDisconnect != nil {
		systrayMenuItemEnable(mDisconnect)
	}
	connectDisable()
	if mLogin != nil {
		systrayMenuItemDisable(mLogin)
	}
	if mEditUrl != nil {
		systrayMenuItemDisable(mEditUrl)
	}
	if running {
		return
	}
	c := getConfigByIndex(index)
	if c == nil {
		return
	}
	myconfig.RestrictedNetwork = false
	r := rpc.RpcCommandStart{
		Version:           rpc.RPCVERSION,
		AccessId:          c.AccessID,
		Uri:               myconfig.Uri,
		Secret:            c.Secret,
		RestrictedNetwork: myconfig.RestrictedNetwork,
		ClientID:          myconfig.ClientID,
	}
	rpcSendReceive(&r)
	running = true
	systraySetTemplateIcon(icon.IconSigned)
	systraySetToolTip("shieldoo - disconnected")
	serverMessage = ""
}

func disconnectNebulaUI() {
	if mDisconnect != nil {
		systrayMenuItemDisable(mDisconnect)
	}
	connectEnable()
	if mLogin != nil {
		systrayMenuItemEnable(mLogin)
	}
	if mEditUrl != nil {
		systrayMenuItemEnable(mEditUrl)
	}
	runningDisconnecting = true
	rpcSendReceive(&rpc.RpcCommandStop{Version: rpc.RPCVERSION})
	running = false
	runningDisconnecting = false
	systraySetTemplateIcon(icon.IconSigned)
	systraySetToolTip("shieldoo - disconnected")
	beeep.Notify(
		"DISCONNECTED", "You were disconnected from Shieldoo Mesh.",
		filepath.FromSlash(execPath+"/logo.png"))
	UpdManagerSetCheck()
	connectionIsConnected = false
}

func getConfigByIndex(idx int) *ManagementSimpleUPNResponseAccess {
	log.Debug("getConfigByIndex: ", idx)
	log.Debug("getConfigByIndex acc: ", localconf.Accesses)
	if localconf.Accesses == nil || len(*(localconf.Accesses)) < idx {
		return nil
	}
	r := (*localconf.Accesses)[idx]

	activeaccessid = r.AccessID
	return &r
}

func showServerMessage() {
	for {
		select {
		case msg := <-serverMessageChan:
			beeep.Notify(
				"SHIELDOO INFO", msg,
				filepath.FromSlash(execPath+"/logo.png"))
		}
	}
}

func restartConnection() {
	if running && localconf.Accesses != nil {
		found := false
		// find running config
		for _, v := range *localconf.Accesses {
			if v.AccessID == activeaccessid {
				found = true
			}
		}
		if !found && activeaccessid > 0 {
			// let's stop because we are running from obsolete config ...
			disconnectNebulaUI()
		}
		return
	}
}

func telemetryTaskRun() error {
	// run telemetry and config
	status, err := telemetrySend()
	if err == nil && status {
		log.Debug("redrawing menus | ", localconf)
		// need restart
		redrawConnectMenu()
		restartConnection()
	}
	return err
}

func telemetryTask() {
	for {
		var err error
		i := 1 * time.Second
		if myconfig.Secret != "" && !registering {
			err = telemetryTaskRun()
			i = 300 * time.Second
			if err != nil {
				i = 15 * time.Second
			}
		}
		time.Sleep(i)
		runtime.GC()
	}
}

func redrawConnectMenu() {
	if mConnectEnabled {
		connectEnable()
	} else {
		connectDisable()
	}
	if localconf.Accesses != nil && mConnectSub != nil {
		for i := 0; i < maxMenuItems; i++ {
			if i < len(*localconf.Accesses) && localconfGetAccessesLen() != 1 {
				mConnectSub[i].SetTitle((*localconf.Accesses)[i].Name)
				mConnectSub[i].SetTooltip((*localconf.Accesses)[i].Name)
				mConnectSub[i].Show()
				systrayMenuItemEnable(mConnectSub[i])
			} else {
				mConnectSub[i].Hide()
			}
		}
	}
}

func connectEnable() {
	if !mConnectEnabled && mConnectDefault != nil && mConnect != nil {
		if localconfGetAccessesLen() == 1 {
			mConnectDefault.Show()
			systrayMenuItemEnable(mConnectDefault)
			mConnect.SetTitle("")
			mConnect.SetTooltip("")
			mConnect.Show()
			systrayMenuItemDisable(mConnect)
		} else {
			mConnectDefault.Hide()
			mConnect.SetTitle("Connect with profile ..")
			mConnect.SetTooltip("Connect with profile ..")
			mConnect.Show()
			systrayMenuItemEnable(mConnect)
		}
	}
	mConnectEnabled = true
}

func connectDisable() {
	if mConnectEnabled && mConnectDefault != nil && mConnect != nil {
		if localconfGetAccessesLen() == 1 {
			mConnectDefault.Show()
			systrayMenuItemDisable(mConnectDefault)
			mConnect.SetTitle("")
			mConnect.SetTooltip("")
			mConnect.Show()
			systrayMenuItemDisable(mConnect)
		} else {
			mConnectDefault.Hide()
			mConnect.SetTitle("Connect with profile ..")
			mConnect.SetTooltip("Connect with profile ..")
			mConnect.Show()
			systrayMenuItemDisable(mConnect)
		}
	}
	mConnectEnabled = false
}

func localconfGetAccessesLen() int {
	var ret int = 0
	if localconf.Accesses != nil {
		ret = len(*localconf.Accesses)
	}
	return ret
}

type DeviceLoginData struct {
	UPN      string
	Provider string
	Secret   string
	URI      string
}

func registerToServer() (secret string, upn string, provider string, uri string, err error) {
	// exception handling
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			log.Error("registerToServer() telemetry error: ", err)
		}
	}()

	_uri := myconfig.Uri + "logindevice/" + registeringCode
	response, err := http.Get(_uri)
	if err != nil {
		panic(err)
	}

	log.Debug("http resp: ", response.Status)
	if response.StatusCode == http.StatusNotFound {
		panic(errors.New("unauthorized call (404)"))
	} else if response.StatusCode != 200 {
		panic(errors.New("status code from management API != 200: " + response.Status))
	}
	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}
	resp := DeviceLoginData{}
	err = json.Unmarshal(bodyBytes, &resp)
	if err != nil {
		panic(err)
	}
	secret = resp.Secret
	upn = resp.UPN
	uri = resp.URI
	provider = resp.Provider
	return
}

var iconConnectingIndex int = 1

func checkConnectionStatus() {
	time.Sleep(1000 * time.Millisecond)
	errConnI := false
	for {
		r, e := rpcSendReceive(&rpc.RpcCommandStatus{Version: rpc.RPCVERSION})
		if e == nil {
			if myconfig.Secret == "" {
				systraySetTemplateIcon(icon.IconWaitForSignIn)
				systraySetToolTip("shieldoo - waiting for sign-in")
			} else {
				if !running && r.IsRunning {
					connectNebulaUI(0)
					activeaccessid = r.AccessId
					running = true
				}
				if running && !r.IsRunning {
					disconnectNebulaUI()
				}
			}
			var _icn *[]byte
			if running {
				if !myconfig.RestrictedNetwork && r.RestrictedNetwork {
					beeep.Notify(
						"RECONNECTING", "You were switched to restrictive network mode!",
						filepath.FromSlash(execPath+"/logo.png"))
				}
				myconfig.RestrictedNetwork = r.RestrictedNetwork
				if r.IsConnected && !runningDisconnecting {
					systraySetToolTip("shieldoo - connected")
					if !connectionIsConnected {
						beeep.Notify(
							"CONNECTED", "You were connected to Shieldoo Mesh.",
							filepath.FromSlash(execPath+"/logo.png"))
						UpdManagerSetCheck()
						connectionIsConnected = true
						systraySetTemplateIcon(icon.IconConnected1)
						time.Sleep(200 * time.Millisecond)
						systraySetTemplateIcon(icon.IconConnected2)
						time.Sleep(200 * time.Millisecond)
						if myconfig.RestrictedNetwork {
							systraySetTemplateIcon(icon.IconConnected4)
						} else {
							systraySetTemplateIcon(icon.IconConnected3)
						}
					} else {
						if myconfig.RestrictedNetwork {
							systraySetTemplateIcon(icon.IconConnected4)
						} else {
							systraySetTemplateIcon(icon.IconConnected3)
						}
					}
				} else {
					switch iconConnectingIndex {
					case 1:
						_icn = &icon.IconConnecting1
					case 2:
						_icn = &icon.IconConnecting2
					case 3:
						_icn = &icon.IconConnecting3
					case 4:
						_icn = &icon.IconConnecting4
					case 5:
						_icn = &icon.IconConnecting5
					case 6:
						_icn = &icon.IconConnecting6
					case 7:
						_icn = &icon.IconConnecting7
					default:
						_icn = &icon.IconConnecting8
						iconConnectingIndex = 0
					}
					iconConnectingIndex++
					systraySetTemplateIcon(*_icn)
					systraySetToolTip("shieldoo - connecting ..")
				}
			} else {
				if registering {
					var _icn *[]byte
					switch iconConnectingIndex {
					case 1:
						_icn = &icon.IconSigning1
					case 2:
						_icn = &icon.IconSigning2
					case 3:
						_icn = &icon.IconSigning3
					case 4:
						_icn = &icon.IconSigning4
					case 5:
						_icn = &icon.IconSigning5
					case 6:
						_icn = &icon.IconSigning6
					case 7:
						_icn = &icon.IconSigning7
					default:
						_icn = &icon.IconSigning8
						iconConnectingIndex = 0
					}
					iconConnectingIndex++
					systraySetTemplateIcon(*_icn)
					systraySetToolTip("shieldoo - signing-in ..")
					errConnI = !errConnI
					if errConnI {
						// try to register
						if _secret, _upn, _, _, err := registerToServer(); err == nil {
							myconfig.Secret = _secret
							myconfig.Upn = _upn
							log.Debug("received secret: ", myconfig.Secret)
							connectDisable()
							gtelLogin = OAuthLoginResponse{}
							telemetryTaskRun()
							registering = false
							UpdManagerSetCheck()
						}
					}
				} else {
					if myconfig.Secret != "" {
						systraySetTemplateIcon(icon.IconSigned)
						systraySetToolTip("shieldoo - disconnected")
						connectEnable()
					} else {
						systraySetTemplateIcon(icon.IconWaitForSignIn)
						systraySetToolTip("shieldoo - waiting for sign-in")
					}
				}
			}
		} else {
			systraySetTemplateIcon(icon.IconError)
			systraySetToolTip("shieldoo - ERROR - shieldoo-mesh service is not running")
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func inputUri() {
	got, ok := inputbox.InputBox("Shieldoo mesh - your company name", "Company name", myconfig.Uri)
	if ok && got != "" {
		log.Debug("you entered:", got)
		if !strings.HasSuffix(got, "/") {
			got += "/"
		}
		// if there is change we must enforce new login
		if got != myconfig.Uri {
			myconfig.Secret = ""
		}
		myconfig.Uri = got
		saveClientConf()
	} else {
		log.Debug("No value entered")
	}
}

func onReady() {
	systraySetTemplateIcon(icon.IconWaitForSignIn)
	systraySetToolTip("shieldoo - waiting for sign-in")
	systray.SetTitle("")

	// We can manipulate the systray in other goroutines
	go func() {
		systraySetTemplateIcon(icon.IconWaitForSignIn)
		systraySetToolTip("shieldoo - waiting for sign-in")
		systray.SetTitle("")

		// enable autostart config
		initAutostartApp()

		mUpdate = systray.AddMenuItem("Update Shieldoo client ..", "Update Shieldoo client ..")
		mUpdate.Hide()
		UpdManagerInitMenuItem(mUpdate)

		mConnect = systray.AddMenuItem("Connect with profile ..", "Connect with profile ..")
		mConnectSub = []*systray.MenuItem{}
		for i := 0; i < maxMenuItems; i++ {
			mConnectSub = append(mConnectSub, nil)
			mConnectSub[i] = mConnect.AddSubMenuItem("", "")
			mConnectSub[i].Hide()
		}
		mConnectDefault = systray.AddMenuItem("Connect ..", "Connect to mesh")
		mDisconnect = systray.AddMenuItem("Disconnect..", "Disconnect from mesh")
		systray.AddSeparator()
		mLogin = systray.AddMenuItem("Sign-in to Shieldoo", "Sign-in to Shieldoo")
		systray.AddSeparator()
		mEditUrl = systray.AddMenuItem("Edit organization name", "Edit organization name")
		mChecked := systray.AddMenuItemCheckbox("Autostart enabled", "Autostart enabled", autostartApp.IsEnabled())
		systray.AddSeparator()
		mWeb := systray.AddMenuItem("Go to shieldoo portal: "+myconfig.Uri, "Go to shieldoo portal: "+myconfig.Uri)
		mAccess := systray.AddMenuItem("Devices in mesh which I can access.", "Devices in mesh which I can access.")
		systray.AddSeparator()
		mVersion := systray.AddMenuItem("version: "+APPVERSION, "version: "+APPVERSION)
		mQuit := systray.AddMenuItem("Quit", "Quit the whole app")
		systrayMenuItemDisable(mVersion)

		systrayMenuItemDisable(mDisconnect)
		connectDisable()

		redrawConnectMenu()

		go showServerMessage()

		for {
			select {
			case <-mConnectDefault.ClickedCh:
				connectNebulaUIDefult()
			// UGLY !!!
			case <-mConnectSub[0].ClickedCh:
				connectNebulaUI(0)
			case <-mConnectSub[1].ClickedCh:
				connectNebulaUI(1)
			case <-mConnectSub[2].ClickedCh:
				connectNebulaUI(2)
			case <-mConnectSub[3].ClickedCh:
				connectNebulaUI(3)
			case <-mConnectSub[4].ClickedCh:
				connectNebulaUI(4)
			case <-mConnectSub[5].ClickedCh:
				connectNebulaUI(5)
			case <-mConnectSub[6].ClickedCh:
				connectNebulaUI(6)
			case <-mConnectSub[7].ClickedCh:
				connectNebulaUI(7)
			case <-mConnectSub[8].ClickedCh:
				connectNebulaUI(8)
			case <-mConnectSub[9].ClickedCh:
				connectNebulaUI(9)
			case <-mConnectSub[10].ClickedCh:
				connectNebulaUI(10)
			case <-mConnectSub[11].ClickedCh:
				connectNebulaUI(11)
			case <-mConnectSub[12].ClickedCh:
				connectNebulaUI(12)
			case <-mConnectSub[13].ClickedCh:
				connectNebulaUI(13)
			case <-mConnectSub[14].ClickedCh:
				connectNebulaUI(14)
			case <-mConnectSub[15].ClickedCh:
				connectNebulaUI(15)
			case <-mConnectSub[16].ClickedCh:
				connectNebulaUI(16)
			case <-mConnectSub[17].ClickedCh:
				connectNebulaUI(17)
			case <-mConnectSub[18].ClickedCh:
				connectNebulaUI(18)
			case <-mConnectSub[19].ClickedCh:
				connectNebulaUI(19)
			case <-mConnectSub[20].ClickedCh:
				connectNebulaUI(20)
			case <-mConnectSub[21].ClickedCh:
				connectNebulaUI(21)
			case <-mConnectSub[22].ClickedCh:
				connectNebulaUI(22)
			case <-mConnectSub[23].ClickedCh:
				connectNebulaUI(23)
			case <-mDisconnect.ClickedCh:
				disconnectNebulaUI()
			case <-mQuit.ClickedCh:
				systray.Quit()
				fmt.Println("Quit now..")
				return
			case <-mEditUrl.ClickedCh:
				prevUri := myconfig.Uri
				inputUri()
				mWeb.SetTitle("Go to shieldoo portal: " + myconfig.Uri)
				mWeb.SetTooltip("Go to shieldoo portal: " + myconfig.Uri)
				if prevUri != myconfig.Uri {
					connectDisable()
				}
			case <-mLogin.ClickedCh:
				if myconfig.Uri == "" {
					inputUri()
					mWeb.SetTitle("Go to shieldoo portal: " + myconfig.Uri)
					mWeb.SetTooltip("Go to shieldoo portal: " + myconfig.Uri)
				}
				if myconfig.Uri != "" {
					registeringCode = GenerateRandomString(64)
					registering = true
					localconf.Accesses = &[]ManagementSimpleUPNResponseAccess{}
					localconf.Hash = ""
					redrawConnectMenu()
					u := myconfig.Uri
					u += "login?code=" + registeringCode
					open.Run(u)
				}
			case <-mWeb.ClickedCh:
				open.Run(myconfig.Uri)
			case <-mUpdate.ClickedCh:
				open.Run(myconfig.Uri + "connect-me")
			case <-mAccess.ClickedCh:
				open.Run(myconfig.Uri + "access-rights")
			case <-mChecked.ClickedCh:
				if autostartApp.IsEnabled() {
					autostartApp.Disable()
					mChecked.Uncheck()
				} else {
					autostartApp.Enable()
					mChecked.Check()
				}
			}
		}
	}()

	// run background ping
	go checkConnectionStatus()

	// run background telemetry and config
	go telemetryTask()

}
