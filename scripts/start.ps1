# EasyLLM 快速启动脚本 (PowerShell - Mac / Windows / Linux)
# 用法:
#   .\scripts\start.ps1           # go run 模式（开发）
#   .\scripts\start.ps1 --build   # 先编译再运行
#   .\scripts\start.ps1 --prod    # 运行已编译的二进制

param(
    [switch]$build,
    [switch]$prod
)

# ── 切换到项目根目录 ──────────────────────────────────────────────────────────
$rootDir = Split-Path -Parent $PSScriptRoot
Set-Location $rootDir

# ── 读取端口 ──────────────────────────────────────────────────────────────────
$port = if ($env:SERVER_PORT) { $env:SERVER_PORT } else { "8022" }
if (Test-Path ".env") {
    $line = Get-Content ".env" | Where-Object { $_ -match '^\s*SERVER_PORT\s*=' } | Select-Object -Last 1
    if ($line) { $port = ($line -split '=', 2)[1].Trim() }
}

# ── 模式 ──────────────────────────────────────────────────────────────────────
$mode = if ($build) { "build" } elseif ($prod) { "prod" } else { "dev" }

# ── 颜色工具 ──────────────────────────────────────────────────────────────────
function Write-Color($text, $color = "White") { Write-Host $text -ForegroundColor $color }

Write-Host ""
Write-Color "╔══════════════════════════════════════╗" Cyan
Write-Color "║        EasyLLM  快速启动脚本        ║" Cyan
Write-Color "╚══════════════════════════════════════╝" Cyan
Write-Color "  模式  : $mode"   Yellow
Write-Color "  端口  : $port"   Yellow
Write-Color "  目录  : $rootDir" Gray
Write-Host ""

# ── 杀占用端口的进程 ──────────────────────────────────────────────────────────
function Kill-Port($p) {
    $isWin = $IsWindows -or ($PSVersionTable.PSVersion.Major -le 5)
    if ($isWin) {
        $rows = netstat -ano | Select-String ":$p\s.*LISTENING"
        foreach ($row in $rows) {
            $pid_ = ($row -split '\s+')[-1]
            if ($pid_ -match '^\d+$') {
                Write-Color "⚠  端口 $p 被 PID $pid_ 占用，正在终止..." Yellow
                Stop-Process -Id $pid_ -Force -ErrorAction SilentlyContinue
            }
        }
    } else {
        $pids = lsof -ti ":$p" 2>/dev/null
        if ($pids) {
            Write-Color "⚠  端口 $p 被占用 (PID: $pids)，正在终止..." Yellow
            $pids | ForEach-Object { kill -9 $_ 2>/dev/null }
        }
    }
    Start-Sleep -Seconds 1
    Write-Color "✓  端口 $p 已检查/释放" Green
}

# ── 加载 .env ─────────────────────────────────────────────────────────────────
function Load-Env {
    if (Test-Path ".env") {
        Write-Color "→  加载 .env" Cyan
        Get-Content ".env" | ForEach-Object {
            $line = $_.Trim()
            if ($line -and -not $line.StartsWith("#") -and $line -match "^([^=]+)=(.*)$") {
                $k = $Matches[1].Trim(); $v = $Matches[2].Trim()
                if (-not [System.Environment]::GetEnvironmentVariable($k)) {
                    [System.Environment]::SetEnvironmentVariable($k, $v, "Process")
                }
            }
        }
    } else {
        Write-Color "→  未找到 .env，使用默认配置（可 cp .env.example .env 自定义）" Yellow
    }
}

# ── 打印访问信息 ──────────────────────────────────────────────────────────────
function Print-Info {
    $p = if ($env:SERVER_PORT) { $env:SERVER_PORT } else { $port }
    Write-Host ""
    Write-Color "═══════════════════════════════════════" Green
    Write-Color "  ✓  EasyLLM 已启动" Green
    Write-Color "═══════════════════════════════════════" Green
    Write-Color "  Web UI : http://localhost:$p" Cyan
    Write-Color "  API    : http://localhost:$p/api/v1" Cyan
    Write-Color "  Pool   : http://localhost:$p/pool/status" Cyan
    Write-Color "═══════════════════════════════════════" Green
    Write-Host ""
    Write-Color "  按 Ctrl+C 停止服务" Yellow
    Write-Host ""
}

# ── 构建前端 ──────────────────────────────────────────────────────────────────
function Build-Frontend {
    Write-Color "`n→  构建前端..." Cyan
    Set-Location web
    npm install --legacy-peer-deps --silent
    npm run build
    Set-Location ..
    Write-Color "✓  前端构建完成" Green
}

# ── 构建 Go 后端 ──────────────────────────────────────────────────────────────
function Build-Backend {
    Write-Color "`n→  编译 Go 后端..." Cyan
    $bin = if ($IsWindows -or ($PSVersionTable.PSVersion.Major -le 5)) { "easyllm.exe" } else { "easyllm" }
    $env:CGO_ENABLED = "1"
    go build -ldflags="-w -s" -o $bin .
    Write-Color "✓  编译完成 → ./$bin" Green
}

# ══════════════════════════════════════════════════════════════════════════════
Kill-Port $port
Load-Env

switch ($mode) {

    "dev" {
        Write-Color "`n→  以 go run 方式启动（开发模式）" Cyan
        Print-Info
        go run main.go
    }

    "build" {
        Build-Frontend
        Build-Backend
        Write-Color "`n→  启动编译后的二进制" Cyan
        Print-Info
        $bin = if ($IsWindows -or ($PSVersionTable.PSVersion.Major -le 5)) { ".\easyllm.exe" } else { "./easyllm" }
        & $bin
    }

    "prod" {
        $bin = if ($IsWindows -or ($PSVersionTable.PSVersion.Major -le 5)) { ".\easyllm.exe" } else { "./easyllm" }
        if (-not (Test-Path ($bin -replace '\./', ''))) {
            Write-Color "✗  未找到二进制文件，请先运行: .\scripts\start.ps1 --build" Red
            exit 1
        }
        Write-Color "`n→  启动生产二进制" Cyan
        Print-Info
        & $bin
    }
}
