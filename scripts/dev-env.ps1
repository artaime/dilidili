# DILI local dev environment (Windows PowerShell)
#
# Load into the current PowerShell session (env changes persist):
#   . .\scripts\dev-env.ps1
#
# From cmd.exe, use the wrapper instead (do NOT double-click .ps1):
#   scripts\dev-go.cmd run .\cmd\server -c config\config.yaml
#
# Or explicitly invoke PowerShell:
#   powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\dev-go.ps1 run .\cmd\server -c config\config.yaml
#
# ONNX path can also be set via env: $env:DILI_ONNX_ROOT = "..."

param(
    [switch]$FullBuild,
    [switch]$Check,
    [switch]$ForceImport,
    [string]$MingwRoot = "C:\msys64\mingw64",
    [string]$Toolchain = "go1.24.11",
    [string]$OnnxRoot = $env:DILI_ONNX_ROOT
)

$ErrorActionPreference = "Stop"

function Write-DevInfo([string]$msg) { Write-Host "[dev-env] $msg" }
function Write-DevWarn([string]$msg) { Write-Warning "[dev-env] $msg" }

function Test-Directory([string]$Path) {
    return -not [string]::IsNullOrWhiteSpace($Path) -and (Test-Path $Path)
}

function Add-FrontPath([string]$Dir) {
    if (-not (Test-Directory $Dir)) {
        return $false
    }
    if ($env:PATH -split ';' | Where-Object { $_ -eq $Dir } | Select-Object -First 1) {
        return $true
    }
    $env:PATH = "$Dir;$env:PATH"
    return $true
}

function Ensure-PathContains([string]$Dir) {
    if (-not (Test-Directory $Dir)) {
        return $false
    }
    if ($env:PATH -split ';' | Where-Object { $_ -eq $Dir } | Select-Object -First 1) {
        return $true
    }
    $env:PATH = "$env:PATH;$Dir"
    return $true
}

function Get-MachineEnv([string]$Name) {
    $user = [Environment]::GetEnvironmentVariable($Name, "User")
    if (-not [string]::IsNullOrWhiteSpace($user)) {
        return $user
    }
    return [Environment]::GetEnvironmentVariable($Name, "Machine")
}

function Test-DevEnv {
    Write-DevInfo "checking environment..."

    $machineGoroot = Get-MachineEnv "GOROOT"
    if ($machineGoroot) {
        Write-DevWarn "system/user GOROOT is set to '$machineGoroot'. Remove it to avoid go toolchain mismatch."
    }

    $mingwBin = Join-Path $MingwRoot "bin"
    if (Test-Directory $mingwBin) {
        Write-DevInfo "mingw64: $mingwBin"
    } else {
        Write-DevWarn "mingw64 not found: $mingwBin"
    }

    $gcc = Join-Path $mingwBin "gcc.exe"
    if (Test-Path $gcc) {
        Write-DevInfo "gcc: $(& $gcc --version | Select-Object -First 1)"
    } else {
        Write-DevWarn "gcc not found under $mingwBin"
    }

    $pkgConfigDir = Join-Path $MingwRoot "lib\pkgconfig"
    $opusFilePc = Join-Path $pkgConfigDir "opusfile.pc"
    if (-not $FullBuild) {
        Write-DevInfo "default build tags: nolibopusfile (silero_vad enabled; use -FullBuild to disable default tags)"
    } elseif (-not (Test-Path $opusFilePc)) {
        Write-DevWarn "FullBuild enabled but opusfile.pc not found. Install with: pacman -S mingw-w64-x86_64-opus mingw-w64-x86_64-opusfile"
    }

    if ($FullBuild -and -not (Test-Directory $OnnxRoot)) {
        Write-DevWarn "FullBuild enabled but ONNX Runtime path missing. Set -OnnxRoot or `$env:DILI_ONNX_ROOT."
    } elseif (Test-Directory $OnnxRoot) {
        Write-DevInfo "onnxruntime: $OnnxRoot"
    }

    if (Get-Command go -ErrorAction SilentlyContinue) {
        Write-DevInfo "go: $(go version)"
        Write-DevInfo "GOROOT(session): $(go env GOROOT)"
        Write-DevInfo "GOTOOLCHAIN(session): $(go env GOTOOLCHAIN)"
    } else {
        Write-DevWarn "go not found in PATH"
    }
}

function Initialize-DevEnv {
    $repoRoot = Split-Path -Parent $PSScriptRoot
    Remove-Item Env:GOROOT -ErrorAction SilentlyContinue
    $env:GOTOOLCHAIN = $Toolchain
    $env:CGO_ENABLED = "1"
    $env:DILI_REPO_ROOT = $repoRoot

    $mingwBin = Join-Path $MingwRoot "bin"
    if (Ensure-PathContains $mingwBin) {
        Write-DevInfo "PATH contains mingw64: $mingwBin"
    } else {
        Write-DevWarn "mingw64 bin not found: $mingwBin"
    }

    $tenVadLib = Join-Path $repoRoot "lib\ten-vad\lib\Windows\x64"
    if (Ensure-PathContains $tenVadLib) {
        Write-DevInfo "PATH contains ten_vad: $tenVadLib"
    } else {
        Write-DevWarn "ten_vad runtime not found: $tenVadLib"
    }

    $gcc = Join-Path $mingwBin "gcc.exe"
    $gxx = Join-Path $mingwBin "g++.exe"
    if (Test-Path $gcc) {
        $env:CC = $gcc
        $env:CXX = $gxx
        Write-DevInfo "CC=$gcc"
    }

    $pkgConfigDir = Join-Path $MingwRoot "lib\pkgconfig"
    if (Test-Directory $pkgConfigDir) {
        $env:PKG_CONFIG_PATH = $pkgConfigDir
        Write-DevInfo "PKG_CONFIG_PATH=$pkgConfigDir"
    }

    if (Test-Directory $OnnxRoot) {
        $includeDir = Join-Path $OnnxRoot "include"
        $libDir = Join-Path $OnnxRoot "lib"
        if (Test-Directory $includeDir) {
            $env:C_INCLUDE_PATH = $includeDir
        }
        if (Test-Directory $libDir) {
            $env:LIBRARY_PATH = $libDir
            Add-FrontPath $libDir | Out-Null
        }
        Write-DevInfo "onnxruntime configured: $OnnxRoot"
    }

    if ($FullBuild) {
        Remove-Item Env:DILI_GO_TAGS -ErrorAction SilentlyContinue
        Write-DevInfo "FullBuild mode: no default -tags"
    } else {
        $env:DILI_GO_TAGS = "nolibopusfile"
        Write-DevInfo "dev build tags: $($env:DILI_GO_TAGS)"
    }
}

function dev-go {
    if ($args.Count -eq 0) {
        & go
        return
    }

    $cmd = $args[0]
    $rest = @()
    if ($args.Count -gt 1) {
        $rest = $args[1..($args.Count - 1)]
    }

    if ($env:DILI_GO_TAGS) {
        & go $cmd -tags $env:DILI_GO_TAGS @rest
    } else {
        & go @args
    }
}

if (-not $ForceImport -and $MyInvocation.InvocationName -ne '.') {
    Write-DevWarn 'not dot-sourced; use one of:'
    Write-DevInfo '  PowerShell: . .\scripts\dev-env.ps1'
    Write-DevInfo '  cmd.exe:    scripts\dev-go.cmd run .\cmd\server -c config\config.yaml'
    exit 1
}

if ($Check) {
    Test-DevEnv
    return
}

Initialize-DevEnv
Test-DevEnv

Write-DevInfo 'ready. example:'
Write-DevInfo '  dev-go run .\cmd\server -c .\config\config.yaml'
