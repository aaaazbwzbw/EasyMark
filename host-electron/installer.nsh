; EasyMark Custom NSIS Installer Script
; 自定义安装逻辑

!include "FileFunc.nsh"
!include "LogicLib.nsh"
!include "WordFunc.nsh"

; 注册表路径
!define INSTALL_REG_KEY "Software\EasyMark"
!define UNINSTALL_REG_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\EasyMark"

; ========== 最早期初始化 ==========
!macro preInit
  ; 设置默认安装路径
  ${If} ${FileExists} "D:\"
    StrCpy $INSTDIR "D:\EasyMark"
  ${Else}
    StrCpy $INSTDIR "C:\EasyMark"
  ${EndIf}
!macroend

; ========== 安装前初始化 ==========
!macro customInit
  ; 检查是否已安装
  ReadRegStr $0 HKCU "${INSTALL_REG_KEY}" "InstallPath"
  ${If} $0 != ""
    ${AndIf} ${FileExists} "$0\EasyMark.exe"
      StrCpy $INSTDIR $0
      Goto initDone
  ${EndIf}
  
  ; 确保默认路径正确
  ${If} ${FileExists} "D:\"
    StrCpy $INSTDIR "D:\EasyMark"
  ${Else}
    StrCpy $INSTDIR "C:\EasyMark"
  ${EndIf}
  
  initDone:
!macroend

; ========== 目录验证函数 ==========
!macro customHeader
  Var /GLOBAL EASYMARK_SUFFIX_ADDED
  
  ; 目录验证回调
  Function .onVerifyInstDir
    ; 防止重复追加
    StrCmp $EASYMARK_SUFFIX_ADDED "1" done 0
    
    ; 检查路径最后一个目录
    ${GetFileName} $INSTDIR $0
    StrCmp $0 "EasyMark" done 0
    StrCmp $0 "easymark" done 0
    StrCmp $0 "EASYMARK" done 0
    
    ; 标记已追加
    StrCpy $EASYMARK_SUFFIX_ADDED "1"
    StrCpy $INSTDIR "$INSTDIR\EasyMark"
    
    done:
    ; 重置标记，允许下次验证
    StrCpy $EASYMARK_SUFFIX_ADDED ""
  FunctionEnd
!macroend

; ========== 安装完成后 ==========
!macro customInstall
  ; 保存安装路径到注册表
  WriteRegStr HKCU "${INSTALL_REG_KEY}" "InstallPath" "$INSTDIR"
  
  ; 读取用户数据路径（如果存在）
  ReadRegStr $0 HKCU "${INSTALL_REG_KEY}" "DataPath"
  ${If} $0 == ""
    ; 默认数据路径
    ${If} ${FileExists} "D:\"
      WriteRegStr HKCU "${INSTALL_REG_KEY}" "DataPath" "D:\EasyMark\Data"
    ${Else}
      WriteRegStr HKCU "${INSTALL_REG_KEY}" "DataPath" "C:\EasyMark\Data"
    ${EndIf}
  ${EndIf}
!macroend

; ========== 删除程序文件 ==========
; 这个宏在卸载和更新时都会被调用
; 更新时使用静默模式，我们跳过删除；正常卸载时执行删除
!macro customRemoveFiles
  ${If} ${Silent}
    ; 更新模式（静默）：不删除文件，只覆盖
    Goto skip_remove
  ${EndIf}
  
  ; 正常卸载：删除程序文件
  RMDir /r "$INSTDIR\resources"
  RMDir /r "$INSTDIR\locales"
  Delete "$INSTDIR\*.exe"
  Delete "$INSTDIR\*.dll"
  Delete "$INSTDIR\*.pak"
  Delete "$INSTDIR\*.bin"
  Delete "$INSTDIR\*.dat"
  Delete "$INSTDIR\*.json"
  Delete "$INSTDIR\LICENSE*"
  Delete "$INSTDIR\version"
  Delete "$INSTDIR\*.log"
  
  skip_remove:
!macroend

; ========== 卸载初始化 ==========
!macro customUnInit
  ; 卸载前询问是否删除数据
!macroend

; ========== 卸载完成后 ==========
!macro customUnInstall
  ; 检测是否是静默卸载（更新时会使用静默模式）
  ; 如果是静默卸载，说明是更新安装，不删除任何数据，也不清理注册表
  ${If} ${Silent}
    ; 更新模式：什么都不做，保留所有数据和注册表
    Goto uninstall_done
  ${EndIf}
  
  ; 真正的卸载：询问是否删除用户数据
  MessageBox MB_YESNO|MB_ICONQUESTION "是否删除 EasyMark 的用户数据？$\n$\n这将删除您的项目数据、配置和模型文件。" IDNO skip_data_delete
  
  ; 二次确认
  MessageBox MB_YESNO|MB_ICONEXCLAMATION "警告：此操作不可恢复！$\n$\n确定要删除所有用户数据吗？" IDNO skip_data_delete
  
  ; 读取数据路径并删除
  ReadRegStr $0 HKCU "${INSTALL_REG_KEY}" "DataPath"
  ${If} $0 != ""
  ${AndIf} ${FileExists} "$0"
    RMDir /r "$0"
  ${EndIf}
  
  ; 同时删除安装目录下的 Data 子目录（如果存在）
  ${If} ${FileExists} "$INSTDIR\Data"
    RMDir /r "$INSTDIR\Data"
  ${EndIf}
  
  skip_data_delete:
  
  ; 只有真正卸载时才清理注册表
  DeleteRegKey HKCU "${INSTALL_REG_KEY}"
  
  uninstall_done:
!macroend
