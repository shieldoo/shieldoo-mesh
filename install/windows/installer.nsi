############################################################################################
#      NSIS Installation Script          
############################################################################################

!define APP_NAME "Shieldoo Secure Network"
!define COMP_NAME "shieldoo.io"
!define WEB_SITE "https://shieldoo.io"
!define VERSION "0.#APPVERSION#"
!define COPYRIGHT "shieldoo © 2022"
!define DESCRIPTION "Shieldoo Secure Network"
!define INSTALLER_NAME "../../out/asset/win10-amd64/shieldoo-mesh-setup.exe"
!define MAIN_APP_EXE "shieldoo-mesh-app.exe"
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

; StrContains
; This function does a case sensitive searches for an occurrence of a substring in a string. 
; It returns the substring if it is found. 
; Otherwise it returns null(""). 
; Written by kenglish_hi
; Adapted from StrReplace written by dandaman32
 
 
Var STR_HAYSTACK
Var STR_NEEDLE
Var STR_CONTAINS_VAR_1
Var STR_CONTAINS_VAR_2
Var STR_CONTAINS_VAR_3
Var STR_CONTAINS_VAR_4
Var STR_RETURN_VAR
Var ConfigExists
 
Function StrContains
  Exch $STR_NEEDLE
  Exch 1
  Exch $STR_HAYSTACK
  ; Uncomment to debug
  ;MessageBox MB_OK 'STR_NEEDLE = $STR_NEEDLE STR_HAYSTACK = $STR_HAYSTACK '
    StrCpy $STR_RETURN_VAR ""
    StrCpy $STR_CONTAINS_VAR_1 -1
    StrLen $STR_CONTAINS_VAR_2 $STR_NEEDLE
    StrLen $STR_CONTAINS_VAR_4 $STR_HAYSTACK
    loop:
      IntOp $STR_CONTAINS_VAR_1 $STR_CONTAINS_VAR_1 + 1
      StrCpy $STR_CONTAINS_VAR_3 $STR_HAYSTACK $STR_CONTAINS_VAR_2 $STR_CONTAINS_VAR_1
      StrCmp $STR_CONTAINS_VAR_3 $STR_NEEDLE found
      StrCmp $STR_CONTAINS_VAR_1 $STR_CONTAINS_VAR_4 done
      Goto loop
    found:
      StrCpy $STR_RETURN_VAR $STR_NEEDLE
      Goto done
    done:
   Pop $STR_NEEDLE ;Prevent "invalid opcode" errors and keep the
   Exch $STR_RETURN_VAR  
FunctionEnd
 
!macro _StrContainsConstructor OUT NEEDLE HAYSTACK
  Push `${HAYSTACK}`
  Push `${NEEDLE}`
  Call StrContains
  Pop `${OUT}`
!macroend
 
!define StrContains '!insertmacro "_StrContainsConstructor"'

Function .onInit
  ${GetParameters} $R0
  ClearErrors
  ${GetOptions} $R0 /URL= $0
  IfFileExists $PROFILE\.shieldoo\shieldoo-mesh.yaml 0 +2
  StrCpy $ConfigExists 1
FunctionEnd

!insertmacro MUI_PAGE_WELCOME

Var DialogUrl
Var LabelUrl
Var TextUrl
Var STR_RT_VAR
Var STR_URL_VAR

Page custom nsDialogsPage nsDialogsPageLeave

Function nsDialogsPage
  ${If} $ConfigExists == 1
    Abort
  ${EndIf}

	nsDialogs::Create 1018

  Pop $DialogUrl

	${If} $DialogUrl == error
		Abort
	${EndIf}

	${NSD_CreateLabel} 0 0 100% 12u "Please enter valid URL to Shieldoo Secure Network (starting with https://)"
	Pop $LabelUrl

	${NSD_CreateText} 0 13u 100% 25%u ""
	Pop $TextUrl

	nsDialogs::Show

FunctionEnd

Function nsDialogsPageLeave
    ${NSD_GetText} $TextUrl $0
    StrCpy $STR_RT_VAR $0
    ${If} $STR_RT_VAR == ""
        MessageBox MB_OK "Please enter valid URL to Shieldoo Secure Network"
        Abort
    ${EndIf}
    ${StrContains} $0 "http://" $STR_RT_VAR
    StrCmp $0 "" httpnotfound
      Goto hdone
    httpnotfound:
      ${StrContains} $0 "https://" $STR_RT_VAR
      StrCmp $0 "" httpsnotfound
        Goto hdone
      httpsnotfound:
        MessageBox MB_OK "Please enter valid URL to Shieldoo Secure Network"
        Abort
    hdone:
      StrCpy $STR_URL_VAR $STR_RT_VAR 
FunctionEnd

!ifdef LICENSE_TXT
!insertmacro MUI_PAGE_LICENSE "${LICENSE_TXT}"
!endif

!ifdef REG_START_MENU
!define MUI_STARTMENUPAGE_NODISABLE
!define MUI_STARTMENUPAGE_DEFAULTFOLDER "Shieldoo Secure Network"
!define MUI_STARTMENUPAGE_REGISTRY_ROOT "${REG_ROOT}"
!define MUI_STARTMENUPAGE_REGISTRY_KEY "${UNINSTALL_PATH}"
!define MUI_STARTMENUPAGE_REGISTRY_VALUENAME "${REG_START_MENU}"
!insertmacro MUI_PAGE_STARTMENU Application $SM_Folder
!endif

!insertmacro MUI_PAGE_INSTFILES

!insertmacro MUI_UNPAGE_CONFIRM

!insertmacro MUI_UNPAGE_INSTFILES

Var AutoRunCheckBox
Function AutoRegShow
   ${NSD_CreateCheckbox} 120u 110u 100% 10u "&Register Shieldoo Secure Network for automatic startup"
   Pop $AutoRunCheckBox
   SetCtlColors $AutoRunCheckBox "" "ffffff"
   ${NSD_Check} $AutoRunCheckBox
FunctionEnd
Function AutoRegRun
    ${NSD_GetState} $AutoRunCheckBox $1
    ${If} $1 <> 0
      ExecWait '"$INSTDIR\${MAIN_APP_EXE}" -autostart'
    ${EndIf}
FunctionEnd

    # These indented statements modify settings for MUI_PAGE_FINISH
    !define MUI_FINISHPAGE_RUN
    !define MUI_FINISHPAGE_RUN_TEXT "Start Shieldoo Secure Network application"
    !define MUI_FINISHPAGE_RUN_FUNCTION "LaunchLink"

    !define MUI_PAGE_CUSTOMFUNCTION_SHOW "AutoRegShow"
    !define MUI_PAGE_CUSTOMFUNCTION_LEAVE "AutoRegRun"

  !insertmacro MUI_PAGE_FINISH

!insertmacro MUI_LANGUAGE "English"

Function LaunchLink
  ExecShell "" "$DESKTOP\${APP_NAME}.lnk"
FunctionEnd
######################################################################

Section -MainProgram
${INSTALL_TYPE}
; stop if there is previous version installed
ExecWait '"$INSTDIR\shieldoo-mesh-srv.exe" -desktop -service stop'
ExecWait 'cmd.exe /C taskkill /f /im shieldoo-mesh-app.exe'
Sleep 5000
ExecWait '"$INSTDIR\shieldoo-mesh-srv.exe" -desktop -service uninstall'

SetOverwrite ifnewer
SetOutPath "$INSTDIR"
File "./logo.png"
File "../../out/asset/win10-amd64/shieldoo-mesh-app.exe"
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

; register service
ClearErrors
ExecWait '"$INSTDIR\shieldoo-mesh-srv.exe" -desktop -service install'
IfErrors 0 noError
    ; Handle error here
noError:
; reconfigure service
ExecWait 'sc failure shieldoo-mesh reset=86400 actions=restart/1000/restart/1000/restart/1000'
; start service
ExecWait '"$INSTDIR\shieldoo-mesh-srv.exe" -desktop -service start'

; set URL for client
IfFileExists $PROFILE\.shieldoo\shieldoo-mesh.yaml +2 0
ExecWait '"$INSTDIR\${MAIN_APP_EXE}" -url $STR_URL_VAR'

; uninstall old application (old name)
DeleteRegKey ${REG_ROOT} "Software\Microsoft\Windows\CurrentVersion\Uninstall\Shieldoo Mesh"

SectionEnd

######################################################################

Section -Icons_Reg
SetOutPath "$INSTDIR"
WriteUninstaller "$INSTDIR\uninstall.exe"

!ifdef REG_START_MENU
!insertmacro MUI_STARTMENU_WRITE_BEGIN Application
CreateDirectory "$SMPROGRAMS\$SM_Folder"
CreateShortCut "$SMPROGRAMS\$SM_Folder\${APP_NAME}.lnk" "$INSTDIR\${MAIN_APP_EXE}"
CreateShortCut "$DESKTOP\${APP_NAME}.lnk" "$INSTDIR\${MAIN_APP_EXE}"
!ifdef WEB_SITE
WriteIniStr "$INSTDIR\${APP_NAME} website.url" "InternetShortcut" "URL" "${WEB_SITE}"
CreateShortCut "$SMPROGRAMS\$SM_Folder\${APP_NAME} Website.lnk" "$INSTDIR\${APP_NAME} website.url"
!endif
!insertmacro MUI_STARTMENU_WRITE_END
!endif

!ifndef REG_START_MENU
CreateDirectory "$SMPROGRAMS\Shieldoo Secure Network"
CreateShortCut "$SMPROGRAMS\Shieldoo Secure Network\${APP_NAME}.lnk" "$INSTDIR\${MAIN_APP_EXE}"
CreateShortCut "$DESKTOP\${APP_NAME}.lnk" "$INSTDIR\${MAIN_APP_EXE}"
!ifdef WEB_SITE
WriteIniStr "$INSTDIR\${APP_NAME} website.url" "InternetShortcut" "URL" "${WEB_SITE}"
CreateShortCut "$SMPROGRAMS\Shieldoo Secure Network\${APP_NAME} Website.lnk" "$INSTDIR\${APP_NAME} website.url"
!endif
!endif

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
ExecWait '"$INSTDIR\shieldoo-mesh-srv.exe" -desktop -service stop'
ExecWait '"$INSTDIR\shieldoo-mesh-srv.exe" -desktop -service uninstall'
ExecWait 'cmd.exe /C taskkill /f /im shieldoo-mesh-app.exe'

Delete "$INSTDIR\logo.png"
Delete "$INSTDIR\${MAIN_APP_EXE}"
Delete "$INSTDIR\shieldoo-mesh-srv.exe"
Delete "$INSTDIR\dist\windows\wintun\LICENSE.txt"
Delete "$INSTDIR\dist\windows\wintun\README.md"
Delete "$INSTDIR\dist\windows\wintun\include\wintun.h"
Delete "$INSTDIR\dist\windows\wintun\bin\x86\wintun.dll"
Delete "$INSTDIR\dist\windows\wintun\bin\arm64\wintun.dll"
Delete "$INSTDIR\dist\windows\wintun\bin\arm\wintun.dll"
Delete "$INSTDIR\dist\windows\wintun\bin\amd64\wintun.dll"
 
RmDir "$INSTDIR\dist\windows\wintun\bin\amd64"
RmDir "$INSTDIR\dist\windows\wintun\bin\arm"
RmDir "$INSTDIR\dist\windows\wintun\bin\arm64"
RmDir "$INSTDIR\dist\windows\wintun\bin\x86"
RmDir "$INSTDIR\dist\windows\wintun\bin"
RmDir "$INSTDIR\dist\windows\wintun\include"
RmDir "$INSTDIR\dist\windows\wintun"
RmDir "$INSTDIR\dist\windows"
RmDir "$INSTDIR\dist"
 
Delete "$INSTDIR\uninstall.exe"
!ifdef WEB_SITE
Delete "$INSTDIR\${APP_NAME} website.url"
!endif

RmDir "$INSTDIR"

Delete "$PROFILE\.shieldoo\shieldoo-mesh.yaml"
RmDir "$PROFILE\.shieldoo"

!ifdef REG_START_MENU
!insertmacro MUI_STARTMENU_GETFOLDER "Application" $SM_Folder
Delete "$SMPROGRAMS\$SM_Folder\${APP_NAME}.lnk"
!ifdef WEB_SITE
Delete "$SMPROGRAMS\$SM_Folder\${APP_NAME} Website.lnk"
!endif
Delete "$DESKTOP\${APP_NAME}.lnk"

RmDir "$SMPROGRAMS\$SM_Folder"
!endif

!ifndef REG_START_MENU
Delete "$SMPROGRAMS\Shieldoo Secure Network\${APP_NAME}.lnk"
!ifdef WEB_SITE
Delete "$SMPROGRAMS\Shieldoo Secure Network\${APP_NAME} Website.lnk"
!endif
Delete "$DESKTOP\${APP_NAME}.lnk"

RmDir "$SMPROGRAMS\Shieldoo Secure Network"
!endif

DeleteRegKey ${REG_ROOT} "${REG_APP_PATH}"
DeleteRegKey ${REG_ROOT} "${UNINSTALL_PATH}"
SectionEnd

######################################################################

