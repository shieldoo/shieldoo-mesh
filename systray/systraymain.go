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

// messages
const msgDisconnected = "shieldoo - disconnected"
const msgLogo = "/logo.png"
const msgConnectWithProfile = "Connect with profile"
const msgWaitingForSignIn = "shieldoo - waiting for sign-in"

func msgSignIn() string {
	orgname := strings.Replace(myconfig.Uri, "https://", "", 1)
	orgname = strings.Replace(orgname, "http://", "", 1)
	orgname = strings.Replace(orgname, "/", "", -1)
	return fmt.Sprintf("Sign-in to Shieldoo (%s)", orgname)
}

func main() {
	onExit := func() {
		// not needed now
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
	log.Info("build version: ", APPVERSION)
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

	// update check
	go UpdManagerRun()

	systray.Run(onReady, onExit)
}

const maxSignInTimeSeconds = 300

var mConnectEnabled bool = true
var mUpdate *systray.MenuItem = nil
var mConnect *systray.MenuItem = nil
var mFavouriteSelector *systray.MenuItem = nil
var mConnectDefault *systray.MenuItem = nil
var mSignIn *systray.MenuItem = nil
var mEditUrl *systray.MenuItem = nil
var mConnectSub []*systray.MenuItem = nil
var mFavourites []*systray.MenuItem = nil
var maxMenuItems int = 24
var mDisconnect *systray.MenuItem = nil
var running bool = false
var runningDisconnecting bool = false
var activeaccessid int = 0
var registering bool = false
var registeringStarted time.Time
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

func connectNebulaUIDefault() {
	if mConnectEnabled && localconfGetAccessesLen() == 1 {
		connectNebulaUI(0)
	}
}

func connectNebulaUI(index int) {
	if mDisconnect != nil {
		systrayMenuItemEnable(mDisconnect)
	}
	connectDisable(false)
	if mSignIn != nil {
		systrayMenuItemDisable(mSignIn)
	}
	if mEditUrl != nil {
		systrayMenuItemDisable(mEditUrl)
	}
	if mFavouriteSelector != nil {
		systrayMenuItemDisable(mFavouriteSelector)
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
	systraySetToolTip(msgDisconnected)
	serverMessage = ""
}

func disconnectNebulaUI() {
	if mDisconnect != nil {
		systrayMenuItemDisable(mDisconnect)
	}
	connectEnable(false)
	if mSignIn != nil {
		systrayMenuItemEnable(mSignIn)
	}
	if mEditUrl != nil {
		systrayMenuItemEnable(mEditUrl)
	}
	if mFavouriteSelector != nil {
		systrayMenuItemEnable(mFavouriteSelector)
	}
	runningDisconnecting = true
	rpcSendReceive(&rpc.RpcCommandStop{Version: rpc.RPCVERSION})
	running = false
	runningDisconnecting = false
	systraySetTemplateIcon(icon.IconSigned)
	systraySetToolTip(msgDisconnected)
	beeep.Notify(
		"DISCONNECTED", "You were disconnected from Shieldoo Mesh.",
		filepath.FromSlash(execPath+msgLogo))
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
				filepath.FromSlash(execPath+msgLogo))
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
	if mConnectEnabled {
		connectEnable(true)
	} else {
		connectDisable(true)
	}
}

func connectEnable(force bool) {
	if (!mConnectEnabled || force) && mConnectDefault != nil && mConnect != nil {
		if localconfGetAccessesLen() == 1 {
			mConnectDefault.Show()
			systrayMenuItemEnable(mConnectDefault)
			mConnect.Hide()
			systrayMenuItemDisable(mConnect)
		} else {
			mConnectDefault.Hide()
			mConnect.Show()
			if localconfGetAccessesLen() > 0 {
				systrayMenuItemEnable(mConnect)
			} else {
				systrayMenuItemDisable(mConnect)
			}
		}
	}
	mConnectEnabled = true
}

func connectDisable(force bool) {
	if (mConnectEnabled || force) && mConnectDefault != nil && mConnect != nil {
		if localconfGetAccessesLen() == 1 {
			mConnectDefault.Show()
			systrayMenuItemDisable(mConnectDefault)
			mConnect.Hide()
			systrayMenuItemDisable(mConnect)
		} else {
			mConnectDefault.Hide()
			mConnect.Show()
			systrayMenuItemDisable(mConnect)
		}
	}
	mConnectEnabled = false
}

func setFavouriteItems() {
	if mFavouriteSelector != nil {
		for i := 0; i < maxMenuItems; i++ {
			if i < len(myconfig.FavouriteItems) {
				mFavourites[i].SetTitle(myconfig.FavouriteItems[i].Uri)
				mFavourites[i].SetTooltip(myconfig.FavouriteItems[i].Uri)
				mFavourites[i].Show()
				systrayMenuItemEnable(mFavourites[i])
			} else {
				mFavourites[i].Hide()
			}
		}
		if len(myconfig.FavouriteItems) > 1 {
			mFavouriteSelector.Show()
		} else {
			mFavouriteSelector.Hide()
		}
	}
}

func activateFavouriteItem(idx int) {
	log.Debug("activateFavouriteItem: ", idx)
	if idx < len(myconfig.FavouriteItems) && idx >= 0 {
		log.Debug("activateFavouriteItem: ", myconfig.FavouriteItems[idx])
		myconfig.Uri = myconfig.FavouriteItems[idx].Uri
		myconfig.Upn = myconfig.FavouriteItems[idx].Upn
		myconfig.Secret = myconfig.FavouriteItems[idx].Secret
		localconf.Hash = ""
		localconf.Accesses = &[]ManagementSimpleUPNResponseAccess{}
		log.Debug("myconfig: ", myconfig)
		if mSignIn != nil {
			mSignIn.SetTitle(msgSignIn())
			mSignIn.SetTooltip(msgSignIn())
		}
		saveClientConf()
		connectDisable(false)
		redrawConnectMenu()
		if myconfig.Secret != "" {
			telemetryInvalidateToken()
			connsucc, err := telemetrySend()
			if err != nil {
				log.Error("telemetrySend error: ", err)
			}
			if connsucc {
				log.Debug("telemetrySend success")
				connectEnable(false)
				redrawConnectMenu()
				// if there is only one connection, connect to it
				connectNebulaUIDefault()
			} else {
				log.Error("telemetrySend failed")
				myconfig.Secret = ""
			}
		}
	}
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

	tmpuri := myconfig.Uri + "logindevice/" + registeringCode
	response, err := http.Get(tmpuri)
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
				systraySetToolTip(msgWaitingForSignIn)
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
			var tmpicn *[]byte
			if running {
				if !myconfig.RestrictedNetwork && r.RestrictedNetwork {
					beeep.Notify(
						"RECONNECTING", "You were switched to restrictive network mode!",
						filepath.FromSlash(execPath+msgLogo))
				}
				myconfig.RestrictedNetwork = r.RestrictedNetwork
				if r.IsConnected && !runningDisconnecting {
					systraySetToolTip("shieldoo - connected")
					if !connectionIsConnected {
						beeep.Notify(
							"CONNECTED", "You were connected to Shieldoo Mesh.",
							filepath.FromSlash(execPath+msgLogo))
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
						tmpicn = &icon.IconConnecting1
					case 2:
						tmpicn = &icon.IconConnecting2
					case 3:
						tmpicn = &icon.IconConnecting3
					case 4:
						tmpicn = &icon.IconConnecting4
					case 5:
						tmpicn = &icon.IconConnecting5
					case 6:
						tmpicn = &icon.IconConnecting6
					case 7:
						tmpicn = &icon.IconConnecting7
					default:
						tmpicn = &icon.IconConnecting8
						iconConnectingIndex = 0
					}
					iconConnectingIndex++
					systraySetTemplateIcon(*tmpicn)
					systraySetToolTip("shieldoo - connecting ..")
				}
			} else {
				if registering {
					var tmpicn *[]byte
					switch iconConnectingIndex {
					case 1:
						tmpicn = &icon.IconSigning1
					case 2:
						tmpicn = &icon.IconSigning2
					case 3:
						tmpicn = &icon.IconSigning3
					case 4:
						tmpicn = &icon.IconSigning4
					case 5:
						tmpicn = &icon.IconSigning5
					case 6:
						tmpicn = &icon.IconSigning6
					case 7:
						tmpicn = &icon.IconSigning7
					default:
						tmpicn = &icon.IconSigning8
						iconConnectingIndex = 0
					}
					iconConnectingIndex++
					systraySetTemplateIcon(*tmpicn)
					systraySetToolTip("shieldoo - signing-in ..")
					errConnI = !errConnI
					if errConnI {
						// try to register
						if _secret, _upn, _, _, err := registerToServer(); err == nil {
							log.Info("sign-in success")
							myconfig.Secret = _secret
							myconfig.Upn = _upn
							log.Debug("received secret: ", myconfig.Secret)
							connectDisable(false)
							gtelLogin = OAuthLoginResponse{}
							registering = false
							setConfigFavouriteItem(myconfig.Uri, myconfig.Upn, myconfig.Secret)
							setFavouriteItems()
							telemetryTaskRun()
							UpdManagerSetCheck()
							// we are signed in, connect to mesh if we have only one access
							connectEnable(false)
							connectNebulaUIDefault()
						} else {
							// there is registering error, check timeout
							var curTime = time.Now().Add(-time.Second * maxSignInTimeSeconds)
							if curTime.After(registeringStarted) {
								registering = false
								beeep.Notify(
									"ERROR", "Registration timeout. Please try again.",
									filepath.FromSlash(execPath+msgLogo))
								localconf.Accesses = &[]ManagementSimpleUPNResponseAccess{}
								localconf.Hash = ""
							}
						}
					}
				} else {
					if myconfig.Secret != "" {
						systraySetTemplateIcon(icon.IconSigned)
						systraySetToolTip(msgDisconnected)
						connectEnable(false)
					} else {
						systraySetTemplateIcon(icon.IconWaitForSignIn)
						systraySetToolTip(msgWaitingForSignIn)
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
	systraySetToolTip(msgWaitingForSignIn)
	systray.SetTitle("")

	// We can manipulate the systray in other goroutines
	go func() {
		systraySetTemplateIcon(icon.IconWaitForSignIn)
		systraySetToolTip(msgWaitingForSignIn)
		systray.SetTitle("")

		// enable autostart config
		initAutostartApp()

		mUpdate = systray.AddMenuItem("Update Shieldoo client", "Update Shieldoo client")
		mUpdate.Hide()
		UpdManagerInitMenuItem(mUpdate)

		mConnect = systray.AddMenuItem(msgConnectWithProfile, msgConnectWithProfile)
		mConnectSub = []*systray.MenuItem{}
		for i := 0; i < maxMenuItems; i++ {
			mConnectSub = append(mConnectSub, nil)
			mConnectSub[i] = mConnect.AddSubMenuItem("", "")
			mConnectSub[i].Hide()
		}
		mConnectDefault = systray.AddMenuItem("Connect", "Connect to mesh")
		mDisconnect = systray.AddMenuItem("Disconnect", "Disconnect from mesh")
		systray.AddSeparator()
		mSignIn = systray.AddMenuItem(msgSignIn(), msgSignIn())
		systray.AddSeparator()
		mEditUrl = systray.AddMenuItem("Edit organization name", "Edit organization name")
		mFavouriteSelector = systray.AddMenuItem("Favourite organizations", "Favourite organizations")
		mFavourites = []*systray.MenuItem{}
		for i := 0; i < maxMenuItems; i++ {
			mFavourites = append(mFavourites, nil)
			mFavourites[i] = mFavouriteSelector.AddSubMenuItem("", "")
			mFavourites[i].Hide()
		}
		mChecked := systray.AddMenuItemCheckbox("Autostart enabled", "Autostart enabled", autostartApp.IsEnabled())
		systray.AddSeparator()
		mAccess := systray.AddMenuItem("My Access Rights in Shieldoo network", "My Access Rights in Shieldoo network")
		systray.AddSeparator()
		mVersion := systray.AddMenuItem("Version: "+APPVERSION, "Version: "+APPVERSION)
		mQuit := systray.AddMenuItem("Quit", "Quit the whole app")
		systrayMenuItemDisable(mVersion)

		systrayMenuItemDisable(mDisconnect)
		connectDisable(false)
		setFavouriteItems()

		redrawConnectMenu()

		go showServerMessage()

		for {
			select {
			case <-mConnectDefault.ClickedCh:
				connectNebulaUIDefault()
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
			case <-mFavourites[0].ClickedCh:
				activateFavouriteItem(0)
			case <-mFavourites[1].ClickedCh:
				activateFavouriteItem(1)
			case <-mFavourites[2].ClickedCh:
				activateFavouriteItem(2)
			case <-mFavourites[3].ClickedCh:
				activateFavouriteItem(3)
			case <-mFavourites[4].ClickedCh:
				activateFavouriteItem(4)
			case <-mFavourites[5].ClickedCh:
				activateFavouriteItem(5)
			case <-mFavourites[6].ClickedCh:
				activateFavouriteItem(6)
			case <-mFavourites[7].ClickedCh:
				activateFavouriteItem(7)
			case <-mFavourites[8].ClickedCh:
				activateFavouriteItem(8)
			case <-mFavourites[9].ClickedCh:
				activateFavouriteItem(9)
			case <-mFavourites[10].ClickedCh:
				activateFavouriteItem(10)
			case <-mFavourites[11].ClickedCh:
				activateFavouriteItem(11)
			case <-mFavourites[12].ClickedCh:
				activateFavouriteItem(12)
			case <-mFavourites[13].ClickedCh:
				activateFavouriteItem(13)
			case <-mFavourites[14].ClickedCh:
				activateFavouriteItem(14)
			case <-mFavourites[15].ClickedCh:
				activateFavouriteItem(15)
			case <-mFavourites[16].ClickedCh:
				activateFavouriteItem(16)
			case <-mFavourites[17].ClickedCh:
				activateFavouriteItem(17)
			case <-mFavourites[18].ClickedCh:
				activateFavouriteItem(18)
			case <-mFavourites[19].ClickedCh:
				activateFavouriteItem(19)
			case <-mFavourites[20].ClickedCh:
				activateFavouriteItem(20)
			case <-mFavourites[21].ClickedCh:
				activateFavouriteItem(21)
			case <-mFavourites[22].ClickedCh:
				activateFavouriteItem(22)
			case <-mFavourites[23].ClickedCh:
				activateFavouriteItem(23)
			case <-mDisconnect.ClickedCh:
				disconnectNebulaUI()
			case <-mQuit.ClickedCh:
				if running {
					disconnectNebulaUI()
				}
				systray.Quit()
				fmt.Println("Quit now")
				return
			case <-mEditUrl.ClickedCh:
				prevUri := myconfig.Uri
				inputUri()
				mSignIn.SetTitle(msgSignIn())
				mSignIn.SetTooltip(msgSignIn())
				if prevUri != myconfig.Uri {
					connectDisable(false)
				}
			case <-mSignIn.ClickedCh:
				if myconfig.Uri == "" {
					inputUri()
					mSignIn.SetTitle(msgSignIn())
					mSignIn.SetTooltip(msgSignIn())
				}
				if myconfig.Uri != "" {
					registeringCode = GenerateRandomString(64)
					registeringStarted = time.Now()
					localconf.Accesses = &[]ManagementSimpleUPNResponseAccess{}
					localconf.Hash = ""
					registering = true
					redrawConnectMenu()
					u := myconfig.Uri
					u += "login?code=" + registeringCode
					open.Run(u)
				}
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
