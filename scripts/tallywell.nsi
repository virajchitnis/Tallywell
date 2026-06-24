; Tallywell Windows installer
; Built with makensis (NSIS 3.x) on Linux in CI.
; Usage: makensis -DAPP_VERSION=1.2.3 scripts/tallywell.nsi
;
; Installs to %LOCALAPPDATA%\Tallywell — no UAC prompt required.

!ifndef APP_VERSION
  !define APP_VERSION "dev"
!endif

!define APP_NAME   "Tallywell"
!define APP_EXE    "tallywell-windows-amd64.exe"
!define PUBLISHER  "Viraj Chitnis"
!define REG_ROOT   HKCU
!define REG_APP    "Software\${APP_NAME}"
!define REG_UNINST "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_NAME}"

Name              "${APP_NAME} ${APP_VERSION}"
OutFile           "dist/Tallywell-${APP_VERSION}-Setup.exe"
InstallDir        "$LOCALAPPDATA\${APP_NAME}"
InstallDirRegKey  ${REG_ROOT} "${REG_APP}" "InstallDir"
RequestExecutionLevel user
SetCompressor     /SOLID lzma
Unicode           true

!include "MUI2.nsh"

!define MUI_ICON              "dist/tallywell.ico"
!define MUI_UNICON            "dist/tallywell.ico"
!define MUI_ABORTWARNING

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_RUN          "$INSTDIR\${APP_EXE}"
!define MUI_FINISHPAGE_RUN_TEXT     "Launch Tallywell now"
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

; ── Install ──────────────────────────────────────────────────────────────────

Section "Install"
  SetOutPath "$INSTDIR"
  File "dist/${APP_EXE}"
  File "dist/tallywell.ico"

  ; Start Menu shortcut
  CreateDirectory "$SMPROGRAMS\${APP_NAME}"
  CreateShortcut  "$SMPROGRAMS\${APP_NAME}\${APP_NAME}.lnk" \
                  "$INSTDIR\${APP_EXE}" "" "$INSTDIR\tallywell.ico" 0
  CreateShortcut  "$SMPROGRAMS\${APP_NAME}\Uninstall.lnk" \
                  "$INSTDIR\Uninstall.exe"

  ; Registry — install location + Add/Remove Programs entry
  WriteRegStr   ${REG_ROOT} "${REG_APP}"    "InstallDir"      "$INSTDIR"
  WriteRegStr   ${REG_ROOT} "${REG_UNINST}" "DisplayName"     "${APP_NAME}"
  WriteRegStr   ${REG_ROOT} "${REG_UNINST}" "DisplayVersion"  "${APP_VERSION}"
  WriteRegStr   ${REG_ROOT} "${REG_UNINST}" "Publisher"       "${PUBLISHER}"
  WriteRegStr   ${REG_ROOT} "${REG_UNINST}" "DisplayIcon"     "$INSTDIR\tallywell.ico"
  WriteRegStr   ${REG_ROOT} "${REG_UNINST}" "UninstallString" "$INSTDIR\Uninstall.exe"
  WriteRegDWORD ${REG_ROOT} "${REG_UNINST}" "NoModify"        1
  WriteRegDWORD ${REG_ROOT} "${REG_UNINST}" "NoRepair"        1

  WriteUninstaller "$INSTDIR\Uninstall.exe"
SectionEnd

; ── Uninstall ─────────────────────────────────────────────────────────────────

Section "Uninstall"
  Delete "$INSTDIR\${APP_EXE}"
  Delete "$INSTDIR\tallywell.ico"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir  "$INSTDIR"

  Delete "$SMPROGRAMS\${APP_NAME}\${APP_NAME}.lnk"
  Delete "$SMPROGRAMS\${APP_NAME}\Uninstall.lnk"
  RMDir  "$SMPROGRAMS\${APP_NAME}"

  DeleteRegKey ${REG_ROOT} "${REG_APP}"
  DeleteRegKey ${REG_ROOT} "${REG_UNINST}"
SectionEnd
