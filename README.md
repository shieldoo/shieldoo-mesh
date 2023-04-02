# Shieldoo Secure Network client
[![Build](https://github.com/shieldoo/shieldoo-mesh/actions/workflows/build.yml/badge.svg)](https://github.com/shieldoo/shieldoo-mesh/actions/workflows/build.yml) 
[![Release](https://img.shields.io/github/v/release/shieldoo/shieldoo-mesh?logo=GitHub&style=flat-square)](https://github.com/shieldoo/shieldoo-mesh/releases/latest) 
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=shieldoo_shieldoo-mesh&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=shieldoo_shieldoo-mesh) 
[![Bugs](https://sonarcloud.io/api/project_badges/measure?project=shieldoo_shieldoo-mesh&metric=bugs)](https://sonarcloud.io/summary/new_code?id=shieldoo_shieldoo-mesh) 
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=shieldoo_shieldoo-mesh&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=shieldoo_shieldoo-mesh)

# What is shieldoo-mesh client applications

Shieldoo-mesh application is wrapper application around incredible network mesh solution *nebula*. 
Nebula ia maintained by Slack and is actively developed and supported on github, nebula itself is open source software published via MIT license.

Nebula is a scalable overlay networking tool with a focus on performance, simplicity and security. It lets you seamlessly connect computers anywhere in the world. Nebula is portable, and runs on Linux, OSX, Windows, iOS, and Android. It can be used to connect a small number of computers, but is also able to connect tens of thousands of computers.

Original github repository can be found there https://github.com/slackhq/nebula (Shieldoo Secure Network is referencing this repository like golang package).

# Installation instructions

## Desktop client installation

Desktop client application is solution for end users which contains two binaries - system service which is responsible for connection management and communication and client application running in sys-tray and is used by user for managing connections.

### Windows

For the installation we have to collect necessary configuration - it is URL where the web management portal runs - for example in case of our demo portal it is `https://demo.shieldoo.net`, you can find this value in your management portal.   

#### Graphical installation

Download installation package `windows-amd64-shieldoo-mesh-setup.exe` and run installation. On the second screen of installation wizard provide URL of web management portal - for example for demo portal `https://demo.shieldoo.net` and finish installation.
Shieldoo Secure Network client application will run in sys-tray, please follow user manual for details.

#### Automated installation from command line

For installation from command line - for example for remote installation done by administration: download installation package `windows-amd64-shieldoo-mesh-setup.exe` and run this command from command prompt (use your URL value which correspond to your web portal instance):

```
windows-amd64-shieldoo-mesh-setup.exe  /S /URL="https://demo.shieldoo.net"
```
### Linux

Shieldoo Secure Network service will run like system service with name shieldoo-mesh and logs are printed to STDOUT which can be seen by standard linux management commands.   

#### Installation from command line

*Quick setup - if you know what your doing*

For the installation we have to collect necessary configuration - it is URL where the web management portal runs - for example in case of our demo portal it is `https://demo.shieldoo.net`, you can find this value in your management portal. And than run this command from command line:

```bash
wget -qO- "https://download.shieldoo.io/latest/linux-amd64-install.sh" | sudo bash -s -- "$USER" "https://<YOUR URL>"
```

*Setup steps - if you want to do installation manually*

Download installation script package `linux-amd64-install.sh` and than process these installation steps, assuming that installation will be done to target directory `/opt/shieldoo-mesh` (use your URL value which correspond to your web management portal):

```bash
# change script permission to be able to execute it
chmod +x linux-amd64-install.sh

# now check content of script - if it is doing what you are expecting

# and run installation
sudo ./linux-amd64-install.sh "$USER" "https://<YOUR URL>"
```

#### Tweaking on Centos

Because Centos has no direct support for System tray icons we have to use few tweaks:

* install Tweak application to be able to enable desktop icons (Shieldoo Secure Network than will appear on desktop)
* install Gnome extension to support systray on Wayland desktop
  * download App indicator - https://extensions.gnome.org/extension/615/appindicator-support/
  * activate app indicator by copying extracted directory to `~/.local/share/gnome-shell/extensions/`
  * install indicator support: `sudo yum install epel-release` and `sudo yum install libappindicator-gtk3`
  * restart your operating system

## Server installation installation

Server side application is solution for servers in your organization which contains one binnariy - system service which is responsible for connection management and communication.

For the installation we have to collect necessary configuration - it is CONFIGURATION DATA from web management portal for concrete server instance - this configuration data is Base64 encoded configuration file with unique access token for server, configuration data has to be handled like secure asset with sensitive data.

### Windows

Shieldoo Secure Network service will run like system service with name shieldoo-mesh and logs are stored in Windows event log.   

#### Graphical installation

Download installation package `windows-amd64-shieldoo-mesh-svc-setup.exe` and run installation. On the second screen of installation wizard provide CONFIGURATION DATA collected from web management portal and finish installation.

#### Automated installation from command line

For installation from command line - for example for remote installation done by administration: download installation package `windows-amd64-shieldoo-mesh-svc-setup.exe` and run this command from command prompt (use your CONFIGURATION DATA value which correspond to your server in web management portal):

```
windows-amd64-shieldoo-mesh-svc-setup.exe  /S /DATA="<BASE64 CONFIGURATION DATA>"
```

### Linux

Shieldoo Secure Network service will run like system service with name shieldoo-mesh and logs are printed to STDOUT which can be seen by standard linux management commands.   

#### Installation from command line

*Quick setup - if you know what your doing*

Prepare your CONFIGURATION DATA value which correspond to your server in web management portal and run this command from command line:

```bash
wget -qO- "https://download.shieldoo.io/latest/linux-amd64-install-svc.sh" | sudo bash -s -- "<BASE64 CONFIGURATION DATA>"
```

*Setup steps - if you want to do installation manually*

Download installation package `linux-amd64-shieldoo-mesh-svc-setup.tar.gz` and than process these installation steps, assuming that installation will be done to target directory `/opt/shieldoo-mesh` (use your CONFIGURATION DATA value which correspond to your server in web management portal):

```bash
# if mesh is already running than you must stop
sudo /opt/shieldoo-mesh/shieldoo-mesh-srv -service stop

# create directory
mkdir -p /opt/shieldoo-mesh

# installation steps
cat ./linux-amd64-shieldoo-mesh-svc-setup.tar.gz | sudo tar -xvz -C /opt/shieldoo-mesh
sudo chmod 755 /opt/shieldoo-mesh/shieldoo-mesh-srv

# create configuration file from configuration data 
/opt/shieldoo-mesh/shieldoo-mesh-srv -createconfig "<BASE64 CONFIGURATION DATA>"
# install service
/opt/shieldoo-mesh/shieldoo-mesh-srv -service install
/opt/shieldoo-mesh/shieldoo-mesh-srv -service start
```

# Build and development instruction

## simplified build steps

### build main service 

#### linux

for build we need some prerequirements (cgo):
```bash
sudo apt-get install gcc libgtk-3-dev libappindicator3-dev
sudo apt-get install --no-install-recommends -y nsis nsis-doc nsis-pluginapi
```

build:
```bash
env GOOS=linux GOARCH=amd64 go build -ldflags "-X main.APPVERSION=0.0.0 -X main.ARCHITECTURE=amd64" -o out/shieldoo-mesh-srv
env GOOS=linux GOARCH=arm GOARM=7 go build -ldflags "-X main.APPVERSION=0.0.0 -X main.ARCHITECTURE=arm7" -o out/shieldoo-mesh-srv-arm7
```

#### apple

```bash
#darwin-arm64 (apple silicon)
env GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.APPVERSION=0.0.0 -X main.ARCHITECTURE=x64" -o out/shieldoo-mesh-srv-arm64

#darwin-amd64 (intel)
env GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.APPVERSION=0.0.0 -X main.ARCHITECTURE=x64" -o out/shieldoo-mesh-srv-amd64

#build universal binary
lipo -create -output out/shieldoo-mesh-srv out/shieldoo-mesh-srv-amd64 out/shieldoo-mesh-srv-arm64
rm out/shieldoo-mesh-srv-amd64 out/shieldoo-mesh-srv-arm64
```

#### windows
`env GOOS=windows GOARCH=amd64 go build -ldflags "-X main.APPVERSION=0.0.0 -X main.ARCHITECTURE=amd64" -o out/shieldoo-mesh-srv.exe`

### build testcli

#### linux
`env GOOS=linux GOARCH=amd64 go build -o out/testcli ./test`

#### apple
`env GOOS=darwin GOARCH=amd64 go build -o out/testcli ./test`

#### windows
`env GOOS=windows GOARCH=amd64 go build -o out/testcli.exe ./test`

### build systray app

#### linux

`env GOOS=linux GOARCH=amd64 go build -tags=legacy_appindicator -ldflags "-X main.APPVERSION=0.0.0 -X main.ARCHITECTURE=amd64" -o out/shieldoo-mesh-app ./systray`

apple twwek:
docker run -it --platform=linux/amd64 -v ~/go/src/github.com/shieldoo/shieldoo-mesh:/go/src/github.com/shieldoo/shieldoo-mesh go

#### apple

Simpliest way to build - MACOSX machine with XCODE installed:

```bash
#darwin-arm64 (apple silicon)
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 SDKROOT=$(xcrun --sdk macosx --show-sdk-path)  go build -ldflags "-X main.APPVERSION=0.0.0 -X main.ARCHITECTURE=x64" -o out/shieldoo-mesh-app-arm64 ./systray

#darwin-amd64 (intel)
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 SDKROOT=$(xcrun --sdk macosx --show-sdk-path)  go build -ldflags "-X main.APPVERSION=0.0.0 -X main.ARCHITECTURE=amd64" -o out/shieldoo-mesh-app-amd64 ./systray

#build universal binary
lipo -create -output out/shieldoo-mesh-app out/shieldoo-mesh-app-amd64 out/shieldoo-mesh-app-arm64
rm out/shieldoo-mesh-app-amd64 out/shieldoo-mesh-app-arm64
```

#### windows

One time action - logo for app:
```
go install github.com/tc-hib/go-winres@latest
cd systray
go-winres simply --icon ../install/windows/logo.png
```

build:

`env GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -ldflags "-H=windowsgui -X main.APPVERSION=$MYTAG -X main.ARCHITECTURE=amd64" -o out/shieldoo-mesh-app.exe ./systray`

### build windows MSI packages

Installed toolchain: `apt-get install --no-install-recommends -y nsis nsis-doc nsis-pluginapi`

#### Desktop installer

build command: 
```
cd install/windows
makensis installer.nsi
```

run installer in silent mode:
```
shieldoo-mesh-setup.exe  /S /URL="https://mycompany.shieldoo.net"
```

#### Service installer

build command: 
```
cd install/windows
makensis svcinstaller.nsi
```

run installer in silent mode:
```
shieldoo-mesh-svc-setup.exe  /S /DATA="base64-encoded-data"
```

# Run server in container
Start container with NET_ADMIN cap and mount `myconfig.yaml` to `/app/config/myconfig.yaml`, containing base64 decoded configuration data from Admin UI.
```
docker run --cap-add NET_ADMIN --volume /shieldoo-mesh/config:/app/config shieldoo:latest
```