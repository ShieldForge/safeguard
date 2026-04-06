@echo off
REM Stop local Vault dev server

echo.
echo Stopping Vault dev server...

tasklist /FI "IMAGENAME eq vault.exe" 2>NUL | find /I /N "vault.exe">NUL
if "%ERRORLEVEL%"=="0" (
    taskkill /F /IM vault.exe >nul 2>&1
    echo [OK] Vault server stopped
) else (
    echo [INFO] No Vault processes found
)

echo.
