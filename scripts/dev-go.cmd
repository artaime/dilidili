@echo off
setlocal
cd /d "%~dp0.."
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0dev-go.ps1" %*
exit /b %ERRORLEVEL%
