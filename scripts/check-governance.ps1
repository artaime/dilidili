# Optional governance checks (full tier)
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$failed = $false

function Warn([string]$msg) {
    Write-Host "WARN: $msg" -ForegroundColor Yellow
    $script:failed = $true
}

$staged = git diff --cached --name-only 2>$null
foreach ($pattern in @('\.env$', '\.pro\.yaml$', '\.pro\.json$', '\.dev\.yaml$', '\.dev\.json$', '\.local\.', '\.pem$', '\.key$')) {
    if ($staged -match $pattern) {
        Warn "staged files may include secrets ($pattern)"
        break
    }
}

$required = @(
    "AGENTS.md",
    "docs/PROJECT_MAP.md",
    "docs/dev/CHANGELOG.md",
    "docs/dev/HOW_TO_USE.md",
    ".cursor/rules/00-core.mdc"
)
foreach ($path in $required) {
    if (-not (Test-Path (Join-Path $root $path))) {
        Warn "missing governance file: $path"
    }
}

$changelogPath = Join-Path $root "docs/dev/CHANGELOG.md"
if (-not (Test-Path $changelogPath)) {
    Warn "missing docs/dev/CHANGELOG.md"
} elseif ((Get-Content $changelogPath -Raw) -notmatch '\[Unreleased\]') {
    Warn "CHANGELOG missing [Unreleased]"
}

if ($failed) {
    Write-Host "Governance check finished with warnings." -ForegroundColor Yellow
    exit 1
}

Write-Host "Governance check passed." -ForegroundColor Green
exit 0
