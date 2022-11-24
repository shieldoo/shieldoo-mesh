############################################################################################
#      NSIS Installation Script          
############################################################################################

!define APP_NAME "Shieldoo Mesh"
!define COMP_NAME "shieldoo.io"
!define WEB_SITE "https://shieldoo.io"
!define VERSION "0.#APPVERSION#"
!define COPYRIGHT "shieldoo © 2022"
!define DESCRIPTION "Shieldoo Mesh Secure Network"
!define INSTALLER_NAME "../../out/asset/win10-amd64/shieldoo-mesh-svc-setup.exe"
!define MAIN_APP_EXE "shieldoo-mesh-srv.exe"
!define INSTALL_TYPE "SetShellVarContext all"
!define REG_ROOT "HKLM"
!define REG_APP_PATH "Software\Microsoft\Windows\CurrentVersion\App Paths\${MAIN_APP_EXE}"
!define UNINSTALL_PATH "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_NAME}"
!define MUI_WELCOMEFINISHPAGE_BITMAP "banner.bmp" #164x314
!define MUI_ICON installer.ico
!define MUI_UNICON installer.ico 

######################################################################

VIProductVersion  "${VERSION}"
VIAddVersionKey "ProductName"  "${APP_NAME}"
VIAddVersionKey "CompanyName"  "${COMP_NAME}"
VIAddVersionKey "LegalCopyright"  "${COPYRIGHT}"
VIAddVersionKey "FileDescription"  "${DESCRIPTION}"
VIAddVersionKey "FileVersion"  "${VERSION}"

######################################################################

SetCompressor ZLIB
Name "${APP_NAME}"
Caption "${APP_NAME}"
OutFile "${INSTALLER_NAME}"
XPStyle on
BrandingText "${APP_NAME}"
InstallDirRegKey "${REG_ROOT}" "${REG_APP_PATH}" ""
InstallDir "$PROGRAMFILES64\Shieldoo Mesh"

######################################################################

!include "MUI2.nsh"
!include nsDialogs.nsh
!include LogicLib.nsh

!define MUI_ABORTWARNING
!define MUI_UNABORTWARNING

!include FileFunc.nsh
!insertmacro GetParameters
!insertmacro GetOptions

Function .onInit
  ${GetParameters} $R0
  ClearErrors
  ${GetOptions} $R0 /DATA= $0
FunctionEnd

!insertmacro MUI_PAGE_WELCOME

Var DialogData
Var LabelData
Var TextData

Page custom nsDialogsPage nsDialogsPageLeave

Function nsDialogsPage

	nsDialogs::Create 1018
	Pop $DialogData

	${If} $DialogData == error
		Abort
	${EndIf}

	${NSD_CreateLabel} 0 0 100% 12u "Please enter valid DATA for Shieldoo Mesh"
	Pop $LabelData

	${NSD_CreateText} 0 13u 100% -13u ""
	Pop $TextData

	nsDialogs::Show

FunctionEnd

Function nsDialogsPageLeave
    ${NSD_GetText} $TextData $0
    ${If} $0 == ""
        MessageBox MB_OK "Please enter valid DATA for Shieldoo Mesh"
        Abort
    ${EndIf}
FunctionEnd

!ifdef LICENSE_TXT
!insertmacro MUI_PAGE_LICENSE "${LICENSE_TXT}"
!endif

!ifdef REG_START_MENU
!define MUI_STARTMENUPAGE_NODISABLE
!define MUI_STARTMENUPAGE_DEFAULTFOLDER "Shieldoo Mesh"
!define MUI_STARTMENUPAGE_REGISTRY_ROOT "${REG_ROOT}"
!define MUI_STARTMENUPAGE_REGISTRY_KEY "${UNINSTALL_PATH}"
!define MUI_STARTMENUPAGE_REGISTRY_VALUENAME "${REG_START_MENU}"
!insertmacro MUI_PAGE_STARTMENU Application $SM_Folder
!endif

!insertmacro MUI_PAGE_INSTFILES

!insertmacro MUI_UNPAGE_CONFIRM

!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_LANGUAGE "English"

######################################################################

Section -MainProgram
${INSTALL_TYPE}
; stop if there is previous version installed
ExecWait '"$INSTDIR\shieldoo-mesh-srv.exe" -service stop'
Sleep 5000
ExecWait '"$INSTDIR\shieldoo-mesh-srv.exe" -service uninstall'

SetOverwrite ifnewer
SetOutPath "$INSTDIR"
File "../../out/asset/win10-amd64/shieldoo-mesh-srv.exe"
SetOutPath "$INSTDIR\dist\windows\wintun"
File "artifacts/dist/windows/wintun/LICENSE.txt"
File "artifacts/dist/windows/wintun/README.md"
SetOutPath "$INSTDIR\dist\windows\wintun\include"
File "artifacts/dist/windows/wintun/include/wintun.h"
SetOutPath "$INSTDIR\dist\windows\wintun\bin\x86"
File "artifacts/dist/windows/wintun/bin/x86/wintun.dll"
SetOutPath "$INSTDIR\dist\windows\wintun\bin\arm64"
File "artifacts/dist/windows/wintun/bin/arm64/wintun.dll"
SetOutPath "$INSTDIR\dist\windows\wintun\bin\arm"
File "artifacts/dist/windows/wintun/bin/arm/wintun.dll"
SetOutPath "$INSTDIR\dist\windows\wintun\bin\amd64"
File "artifacts/dist/windows/wintun/bin/amd64/wintun.dll"

; create config file
ClearErrors
ExecWait '"$INSTDIR\shieldoo-mesh-srv.exe" -createconfig "$0"'
IfErrors 0 noError
    ; Handle error here
noError:

; register service
ClearErrors
ExecWait '"$INSTDIR\shieldoo-mesh-srv.exe" -service install'
IfErrors 0 noError2
    ; Handle error here
noError2:
; reconfigure service
ExecWait 'sc failure shieldoo-mesh reset=86400 actions=restart/1000/restart/1000/restart/1000'
; start service
ExecWait '"$INSTDIR\shieldoo-mesh-srv.exe" -service start'

SectionEnd

######################################################################

Section -Icons_Reg
SetOutPath "$INSTDIR"
WriteUninstaller "$INSTDIR\uninstall.exe"

WriteRegStr ${REG_ROOT} "${REG_APP_PATH}" "" "$INSTDIR\${MAIN_APP_EXE}"
WriteRegStr ${REG_ROOT} "${UNINSTALL_PATH}"  "DisplayName" "${APP_NAME}"
WriteRegStr ${REG_ROOT} "${UNINSTALL_PATH}"  "UninstallString" "$INSTDIR\uninstall.exe"
WriteRegStr ${REG_ROOT} "${UNINSTALL_PATH}"  "DisplayIcon" "$INSTDIR\${MAIN_APP_EXE}"
WriteRegStr ${REG_ROOT} "${UNINSTALL_PATH}"  "DisplayVersion" "${VERSION}"
WriteRegStr ${REG_ROOT} "${UNINSTALL_PATH}"  "Publisher" "${COMP_NAME}"

!ifdef WEB_SITE
WriteRegStr ${REG_ROOT} "${UNINSTALL_PATH}"  "URLInfoAbout" "${WEB_SITE}"
!endif
SectionEnd

######################################################################

Section Uninstall
${INSTALL_TYPE}

; unregister service
ExecWait '"$INSTDIR\shieldoo-mesh-srv.exe" -service stop'
ExecWait '"$INSTDIR\shieldoo-mesh-srv.exe" -service uninstall'

Delete "$INSTDIR\shieldoo-mesh-srv.exe"
RmDir /r "$INSTDIR\dist"
RmDir /r "$INSTDIR\config"
 
Delete "$INSTDIR\uninstall.exe"
!ifdef WEB_SITE
Delete "$INSTDIR\${APP_NAME} website.url"
!endif

RmDir "$INSTDIR"

DeleteRegKey ${REG_ROOT} "${REG_APP_PATH}"
DeleteRegKey ${REG_ROOT} "${UNINSTALL_PATH}"
SectionEnd

######################################################################

