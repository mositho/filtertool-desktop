Unicode true
!include "MUI2.nsh"
!include "LogicLib.nsh"

!ifndef VERSION
  !error "Pass /DVERSION matching package.json when building the installer"
!endif
Name "Filtertool"
OutFile "..\..\bin\Filtertool-Setup.exe"
InstallDir "$LOCALAPPDATA\Programs\Filtertool"
RequestExecutionLevel user
SetCompressor /SOLID lzma
!define MUI_ICON "icon.ico"
!define MUI_UNICON "icon.ico"
!define MUI_ABORTWARNING
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Section "Filtertool"
  SetShellVarContext current
  ReadRegStr $0 HKCU "Software\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" "pv"
  ${If} $0 == ""
    SetRegView 32
    ReadRegStr $0 HKLM "Software\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" "pv"
  ${EndIf}
  ${If} $0 == ""
    InitPluginsDir
    SetOutPath "$PLUGINSDIR"
    File "MicrosoftEdgeWebview2Setup.exe"
    ExecWait '"$PLUGINSDIR\MicrosoftEdgeWebview2Setup.exe" /silent /install' $1
    ${If} $1 != 0
      MessageBox MB_ICONSTOP "WebView2 could not be installed. Install Microsoft Edge WebView2, then run setup again."
      Abort
    ${EndIf}
  ${EndIf}
  SetOutPath "$INSTDIR"
  File "..\..\bin\Filtertool.exe"
  File "..\..\docs\THIRD_PARTY.md"
  File "..\..\LICENSE"
  WriteUninstaller "$INSTDIR\Uninstall.exe"
  CreateShortcut "$SMPROGRAMS\Filtertool.lnk" "$INSTDIR\Filtertool.exe"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Filtertool" "DisplayName" "Filtertool"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Filtertool" "DisplayVersion" "${VERSION}"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Filtertool" "Publisher" "mositho"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Filtertool" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Filtertool" "UninstallString" '$\"$INSTDIR\Uninstall.exe$\"'
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Filtertool" "DisplayIcon" "$INSTDIR\Filtertool.exe"
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Filtertool" "NoModify" 1
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Filtertool" "NoRepair" 1
SectionEnd

Section "Uninstall"
  SetShellVarContext current
  Delete "$SMPROGRAMS\Filtertool.lnk"
  Delete "$INSTDIR\Filtertool.exe"
  Delete "$INSTDIR\THIRD_PARTY.md"
  Delete "$INSTDIR\LICENSE"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"
  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Filtertool"
  ; Deliberately preserve %APPDATA%\Filtertool and all game exports.
SectionEnd
