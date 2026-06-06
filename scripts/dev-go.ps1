# Run go commands with dev-env applied.
# Usage:
#   powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\dev-go.ps1 run .\cmd\server -c config\config.yaml

$ErrorActionPreference = "Stop"

. "$PSScriptRoot\dev-env.ps1" -ForceImport

if ($args.Count -eq 0) {
    Write-Host "[dev-go] usage:"
    Write-Host "  scripts\dev-go.cmd run .\cmd\server -c config\config.yaml"
    Write-Host "  scripts\dev-go.cmd build -o xiaozhi_server.exe .\cmd\server"
    exit 1
}

dev-go @args
exit $LASTEXITCODE
