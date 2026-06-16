# 为本地 Dili UDP 语音端口放行入站（默认 8990）
param(
    [int]$Port = 8990,
    [string]$RuleName = "Dili-UDP-Voice"
)

$ErrorActionPreference = "Stop"
$existing = Get-NetFirewallRule -DisplayName $RuleName -ErrorAction SilentlyContinue
if ($existing) {
    Write-Host "[firewall] 规则已存在: $RuleName"
} else {
    New-NetFirewallRule -DisplayName $RuleName -Direction Inbound -Action Allow -Protocol UDP -LocalPort $Port | Out-Null
    Write-Host "[firewall] 已添加 UDP $Port 入站允许: $RuleName"
}
