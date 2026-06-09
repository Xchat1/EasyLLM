param(
    [string]$Version = $env:EASYLLM_VERSION,
    [string]$Arch = $env:GOARCH,
    [string]$OutDir = "build/release"
)

$ErrorActionPreference = "Stop"

$rootDir = Split-Path -Parent $PSScriptRoot
Set-Location $rootDir

if (-not $Version) {
    $Version = "dev"
}
if (-not $Arch) {
    $Arch = "amd64"
}

$packageName = "EasyLLM-$Version-windows-$Arch"
$workDir = Join-Path $OutDir $packageName
$zipPath = Join-Path $OutDir "$packageName.zip"

Write-Host "=== Packaging $packageName ==="

if (Test-Path $workDir) {
    Remove-Item -Recurse -Force $workDir
}
if (Test-Path $zipPath) {
    Remove-Item -Force $zipPath
}
New-Item -ItemType Directory -Force $workDir | Out-Null

Write-Host "-> Building frontend"
Push-Location web
npm ci --legacy-peer-deps
npm run build
Pop-Location

Write-Host "-> Building backend"
$env:CGO_ENABLED = "1"
$env:GOOS = "windows"
$env:GOARCH = $Arch
go build -trimpath -ldflags="-w -s" -o (Join-Path $workDir "easyllm.exe") .

Write-Host "-> Copying runtime files"
New-Item -ItemType Directory -Force (Join-Path $workDir "web") | Out-Null
Copy-Item -Recurse "web/dist" (Join-Path $workDir "web/dist")
Copy-Item ".env.example" $workDir
Copy-Item "README.md" $workDir
Copy-Item "LICENSE" $workDir

$scriptsDir = Join-Path $workDir "scripts"
New-Item -ItemType Directory -Force $scriptsDir | Out-Null
Copy-Item "scripts/start.bat" $scriptsDir
Copy-Item "scripts/start.ps1" $scriptsDir

@"
@echo off
cd /d "%~dp0"
scripts\start.bat --prod
"@ | Set-Content -Encoding ASCII (Join-Path $workDir "start-easyllm.bat")

@"
# EasyLLM Windows Release

1. Optional: copy `.env.example` to `.env` and adjust settings.
2. Double-click `start-easyllm.bat`, or run:

```powershell
.\scripts\start.ps1 -prod
```

Then open:

```text
http://localhost:8022
```
"@ | Set-Content -Encoding UTF8 (Join-Path $workDir "README-Windows.md")

Write-Host "-> Creating zip"
Compress-Archive -Path $workDir -DestinationPath $zipPath -Force

Write-Host ""
Write-Host "=== Windows package complete ==="
Write-Host "Package: $zipPath"
