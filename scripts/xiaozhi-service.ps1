# XiaoZhi service manager (Windows PowerShell)
# Usage: .\scripts\xiaozhi-service.ps1 start|stop|restart|status

param(
    [Parameter(Position = 0)]
    [ValidateSet("start", "stop", "restart", "status")]
    [string]$Action = "status"
)

$ErrorActionPreference = "Stop"

$RepoRoot = Split-Path -Parent $PSScriptRoot
$BundleDir = Join-Path $RepoRoot "dist\bundle\xiaozhi_server-windows-amd64-v0.6.3\xiaozhi-server-windows-amd64"
$ServerExe = Join-Path $BundleDir "xiaozhi_server.exe"
$MainConfig = Join-Path $BundleDir "main_config.yaml"
$AsrConfig = Join-Path $BundleDir "asr_server.json"
$ManagerConfig = Join-Path $BundleDir "manager.json"
$PidFile = Join-Path $BundleDir "logs\xiaozhi_server.pid"
$LogDir = Join-Path $BundleDir "logs"
$LanIP = "192.168.0.55"

function Write-Info([string]$msg) { Write-Host "[xiaozhi] $msg" }

function Get-ServiceProcess {
    if (Test-Path $PidFile) {
        $pidText = (Get-Content $PidFile -Raw).Trim()
        if ($pidText -match '^\d+$') {
            $proc = Get-Process -Id ([int]$pidText) -ErrorAction SilentlyContinue
            if ($proc -and $proc.ProcessName -eq "xiaozhi_server") {
                return $proc
            }
        }
    }
    return Get-Process -Name "xiaozhi_server" -ErrorAction SilentlyContinue | Select-Object -First 1
}

function Test-Prerequisites {
    if (-not (Test-Path $ServerExe)) {
        throw "xiaozhi_server.exe not found: $ServerExe"
    }
    if (-not (Test-Path $MainConfig)) {
        throw "main_config.yaml not found: $MainConfig"
    }
    New-Item -ItemType Directory -Force -Path $LogDir | Out-Null
}

function Start-Service {
    Test-Prerequisites
    $existing = Get-ServiceProcess
    if ($existing) {
        Write-Info "already running (PID $($existing.Id))"
        return
    }

    $args = @(
        "-c", $MainConfig,
        "-asr-enable",
        "--asr-config", $AsrConfig,
        "--manager-enable",
        "--manager-config", $ManagerConfig
    )

    Write-Info "workdir: $BundleDir"
    Write-Info "config:  $MainConfig"
    $env:XIAOZHI_REPO_ROOT = $RepoRoot
    $env:XIAOZHI_DEVICE_LOG = Join-Path $RepoRoot "logs\device.log"
    $proc = Start-Process -FilePath $ServerExe -ArgumentList $args -WorkingDirectory $BundleDir -PassThru -WindowStyle Hidden
    Start-Sleep -Seconds 2

    if ($proc.HasExited) {
        throw "process exited early, check $LogDir\server.log"
    }

    Set-Content -Path $PidFile -Value $proc.Id -Encoding ascii
    Write-Info "started (PID $($proc.Id))"
    Write-Info "console:  http://127.0.0.1:8080/"
    Write-Info "ota:      http://${LanIP}:8989/xiaozhi/ota/"
    Write-Info "ws:       ws://${LanIP}:8989/xiaozhi/v1/"
    Write-Info "mqtt:     ${LanIP}:8883 (TLS)"
    Write-Info "udp:      ${LanIP}:8990"
}

function Stop-Service {
    $proc = Get-ServiceProcess
    if (-not $proc) {
        Write-Info "not running"
        if (Test-Path $PidFile) { Remove-Item $PidFile -Force -ErrorAction SilentlyContinue }
        return
    }
    Write-Info "stopping PID $($proc.Id)..."
    Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 1
    if (Test-Path $PidFile) { Remove-Item $PidFile -Force -ErrorAction SilentlyContinue }
    Write-Info "stopped"
}

function Sync-ConsoleOTA {
    $db = Join-Path $BundleDir "data\xiaozhi.db"
    if (-not (Test-Path $db)) {
        Write-Info "sync-ota skip: db not found"
        return
    }
    $patchTool = Join-Path $RepoRoot "manager\backend\cmd\patch_ota"
    $prevToolchain = $env:GOTOOLCHAIN
    $env:GOTOOLCHAIN = "auto"
    Push-Location (Join-Path $RepoRoot "manager\backend")
    try {
        go run ./cmd/patch_ota -db $db -ip $LanIP | ForEach-Object { Write-Info $_ }
    } finally {
        Pop-Location
        if ($null -eq $prevToolchain) {
            Remove-Item Env:GOTOOLCHAIN -ErrorAction SilentlyContinue
        } else {
            $env:GOTOOLCHAIN = $prevToolchain
        }
    }
}

function Show-Status {
    Test-Prerequisites
    $proc = Get-ServiceProcess
    if ($proc) {
        Write-Info "status: running"
        Write-Info "pid:    $($proc.Id)"
        Write-Info "path:   $($proc.Path)"
    } else {
        Write-Info "status: stopped"
    }
    Write-Info "exe:    $ServerExe"
    Write-Info "config: $MainConfig"

    try {
        $r = Invoke-WebRequest -Uri "http://127.0.0.1:8080/" -UseBasicParsing -TimeoutSec 3
        Write-Info "console :8080 -> HTTP $($r.StatusCode)"
    } catch {
        Write-Info "console :8080 -> no response"
    }

    try {
        $r = Invoke-WebRequest -Uri "http://127.0.0.1:8989/xiaozhi/ota/" -Method POST -UseBasicParsing -TimeoutSec 3 `
            -Headers @{ "Device-Id" = "00:00:00:00:00:01"; "Client-Id" = "health-check" } `
            -ContentType "application/json" -Body "{}"
        Write-Info "ota :8989 -> HTTP $($r.StatusCode)"
    } catch {
        if ($_.Exception.Response) {
            $code = [int]$_.Exception.Response.StatusCode
            Write-Info "ota :8989 -> HTTP $code"
        } else {
            Write-Info "ota :8989 -> no response"
        }
    }
}

switch ($Action) {
    "start"   { Start-Service }
    "stop"    { Stop-Service }
    "restart" { Stop-Service; Start-Sleep -Seconds 1; Sync-ConsoleOTA; Start-Service }
    "sync-ota" { Sync-ConsoleOTA }
    "status"  { Show-Status }
}
