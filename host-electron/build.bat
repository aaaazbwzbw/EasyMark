@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

echo ============================================
echo   EasyMark 一键打包脚本
echo ============================================
echo.

cd /d "%~dp0"

:: 1. 构建 Vue 前端
echo [1/4] 构建 Vue 前端...
cd ..\frontend
call npm run build
if errorlevel 1 (
    echo 错误: Vue 前端构建失败
    pause
    exit /b 1
)
echo Vue 前端构建完成
echo.

:: 2. 编译 Go 后端
echo [2/4] 编译 Go 后端...
cd ..\backend-go
go build -ldflags "-s -w" -o easymark-backend.exe .
if errorlevel 1 (
    echo 错误: Go 后端编译失败
    pause
    exit /b 1
)
echo Go 后端编译完成
echo.

:: 3. 检查 Python 服务是否已编译
echo [3/4] 检查 Python 服务...
cd ..\host-plugins\train_python

if not exist "dist\infer_service.exe" (
    echo 警告: infer_service.exe 不存在
    echo 请先运行 pyinstaller 编译 Python 服务:
    echo   pyinstaller --onefile --name infer_service infer_service.py
    echo.
)

if not exist "dist\train_service.exe" (
    echo 警告: train_service.exe 不存在
    echo 如果需要训练功能，请编译 train_service
    echo.
)

:: 4. 打包 Electron
echo [4/4] 打包 Electron 应用...
cd ..\..\host-electron

:: 确保依赖已安装
call npm install

:: 执行打包
call npx electron-builder --win --x64
if errorlevel 1 (
    echo 错误: Electron 打包失败
    pause
    exit /b 1
)

echo.
echo ============================================
echo   打包完成！
echo   安装包位置: host-electron\dist\
echo ============================================
echo.

:: 打开输出目录
explorer dist

pause
