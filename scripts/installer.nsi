!ifndef VERSION
  !define VERSION "0.0.0"
!endif

!define APP_NAME    "YouTube Downloader"
!define PUBLISHER   "SamSkinner01"
!define INSTALL_DIR "$PROGRAMFILES64\${APP_NAME}"
!define REG_KEY     "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_NAME}"

Name            "${APP_NAME} ${VERSION}"
OutFile         "YouTube-Downloader-Setup-${VERSION}.exe"
InstallDir      "${INSTALL_DIR}"
InstallDirRegKey HKLM "Software\${APP_NAME}" "Install_Dir"
RequestExecutionLevel admin
Unicode True

!include "MUI2.nsh"

!define MUI_ICON "..\assets\icon.ico"
!define MUI_UNICON "..\assets\icon.ico"
!define MUI_WELCOMEPAGE_TITLE "Install ${APP_NAME}"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

Section "Install"
  SetOutPath "$INSTDIR"

  File "youtube-downloader-gui.exe"
  File "ffmpeg.exe"
  File "ffprobe.exe"

  WriteUninstaller "$INSTDIR\Uninstall.exe"

  ; Start Menu shortcut
  CreateDirectory "$SMPROGRAMS\${APP_NAME}"
  CreateShortcut  "$SMPROGRAMS\${APP_NAME}\${APP_NAME}.lnk" "$INSTDIR\youtube-downloader-gui.exe"
  CreateShortcut  "$SMPROGRAMS\${APP_NAME}\Uninstall.lnk"   "$INSTDIR\Uninstall.exe"

  ; Desktop shortcut
  CreateShortcut "$DESKTOP\${APP_NAME}.lnk" "$INSTDIR\youtube-downloader-gui.exe"

  ; Add/Remove Programs registry entry
  WriteRegStr HKLM "${REG_KEY}" "DisplayName"     "${APP_NAME}"
  WriteRegStr HKLM "${REG_KEY}" "DisplayVersion"  "${VERSION}"
  WriteRegStr HKLM "${REG_KEY}" "Publisher"       "${PUBLISHER}"
  WriteRegStr HKLM "${REG_KEY}" "UninstallString" "$INSTDIR\Uninstall.exe"
  WriteRegStr HKLM "${REG_KEY}" "InstallLocation" "$INSTDIR"
  WriteRegDWORD HKLM "${REG_KEY}" "NoModify" 1
  WriteRegDWORD HKLM "${REG_KEY}" "NoRepair" 1
SectionEnd

Section "Uninstall"
  Delete "$INSTDIR\youtube-downloader-gui.exe"
  Delete "$INSTDIR\ffmpeg.exe"
  Delete "$INSTDIR\ffprobe.exe"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir  "$INSTDIR"

  Delete "$SMPROGRAMS\${APP_NAME}\${APP_NAME}.lnk"
  Delete "$SMPROGRAMS\${APP_NAME}\Uninstall.lnk"
  RMDir  "$SMPROGRAMS\${APP_NAME}"
  Delete "$DESKTOP\${APP_NAME}.lnk"

  DeleteRegKey HKLM "${REG_KEY}"
  DeleteRegKey HKLM "Software\${APP_NAME}"
SectionEnd
