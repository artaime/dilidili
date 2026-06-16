@echo off
cd /d "%~dp0"

dili_server.exe -c main_config.yaml -asr-enable --asr-config asr_server.json --manager-enable --manager-config manager.json

pause
