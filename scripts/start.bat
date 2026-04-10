@echo off
:: EasyLLM 快速启动脚本 (Windows CMD)
:: 用法:
::   scripts\start.bat          -- go run 模式（开发）
::   scripts\start.bat --build  -- 先编译再运行
::   scripts\start.bat --prod   -- 运行已编译的 easyllm.exe

setlocal EnableDelayedExpansion

:: ── 切换到项目根目录 ──────────────────────────────────────────────────────────
cd /d "%~dp0\.."

:: ── 默认参数 ──────────────────────────────────────────────────────────────────
set "MODE=dev"
set "PORT=8022"

:: 读取 .env 里的 SERVER_PORT
if exist ".env" (
    for /f "usebackq tokens=1,* delims==" %%A in (".env") do (
        set "_key=%%A"
        set "_val=%%B"
        set "_key=!_key: =!"
        if /i "!_key!"=="SERVER_PORT" set "PORT=!_val: =!"
    )
)

:: 解析命令行参数
for %%A in (%*) do (
    if /i "%%A"=="--build" set "MODE=build"
    if /i "%%A"=="--prod"  set "MODE=prod"
)

echo.
echo  ╔══════════════════════════════════════╗
echo  ║        EasyLLM  快速启动脚本        ║
echo  ╚══════════════════════════════════════╝
echo   模式  : %MODE%
echo   端口  : %PORT%
echo   目录  : %CD%
echo.

:: ── 杀掉占用端口的进程 ────────────────────────────────────────────────────────
:kill_port
echo [*] 检查端口 %PORT%...
for /f "tokens=5" %%P in ('netstat -aon ^| findstr ":%PORT% " ^| findstr "LISTENING" 2^>nul') do (
    echo [!] 端口 %PORT% 被 PID %%P 占用，正在终止...
    taskkill /F /PID %%P >nul 2>&1
    timeout /t 1 /nobreak >nul
    echo [OK] 端口 %PORT% 已释放
    goto :port_done
)
echo [OK] 端口 %PORT% 空闲
:port_done

:: ── 加载 .env ─────────────────────────────────────────────────────────────────
if exist ".env" (
    echo [*] 加载 .env...
    for /f "usebackq tokens=1,* delims==" %%A in (".env") do (
        set "_k=%%A"
        set "_k=!_k: =!"
        if not "!_k:~0,1!"=="#" (
            if not "%%B"=="" set "%%A=%%B"
        )
    )
) else (
    echo [!] 未找到 .env，使用默认配置
)

:: ── 构建函数 ──────────────────────────────────────────────────────────────────
if "%MODE%"=="build" goto do_build
if "%MODE%"=="prod"  goto do_prod
goto do_dev

:do_build
echo.
echo [*] 构建前端...
cd web
call npm install --legacy-peer-deps --silent
call npm run build
cd ..
echo [OK] 前端构建完成

echo.
echo [*] 编译 Go 后端...
set CGO_ENABLED=1
go build -ldflags="-w -s" -o easyllm.exe .
echo [OK] 编译完成 =^> easyllm.exe

:do_prod
if not exist "easyllm.exe" (
    echo [ERROR] 未找到 easyllm.exe，请先运行: scripts\start.bat --build
    pause
    exit /b 1
)
goto print_info_prod

:do_dev
goto print_info_dev

:: ── 打印访问信息并启动 ────────────────────────────────────────────────────────
:print_info_dev
echo.
echo  ═══════════════════════════════════════
echo    EasyLLM 启动中 (go run 模式)
echo  ═══════════════════════════════════════
echo    Web UI : http://localhost:%PORT%
echo    API    : http://localhost:%PORT%/api/v1
echo    Pool   : http://localhost:%PORT%/pool/status
echo  ═══════════════════════════════════════
echo.
echo   按 Ctrl+C 停止服务
echo.
go run main.go
goto :eof

:print_info_prod
echo.
echo  ═══════════════════════════════════════
echo    EasyLLM 启动中 (二进制模式)
echo  ═══════════════════════════════════════
echo    Web UI : http://localhost:%PORT%
echo    API    : http://localhost:%PORT%/api/v1
echo    Pool   : http://localhost:%PORT%/pool/status
echo  ═══════════════════════════════════════
echo.
echo   按 Ctrl+C 停止服务
echo.
easyllm.exe
goto :eof
