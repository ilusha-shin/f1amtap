@echo off
setlocal
cd /d "%~dp0"

echo === f1amtap build (Windows) ===
where go >nul 2>nul
if errorlevel 1 (
    echo [ERROR] Go not found in PATH. Install from https://go.dev/dl/
    exit /b 1
)

go version
echo.
echo go mod tidy...
go mod tidy
if errorlevel 1 (
    echo [ERROR] go mod tidy failed
    exit /b 1
)

echo.
echo go build -o f1amtap.exe ./cmd/f1amtap
go build -ldflags "-H windowsgui" -o f1amtap.exe ./cmd/f1amtap
if errorlevel 1 (
    echo [ERROR] build failed
    exit /b 1
)

echo.
echo OK: %CD%\f1amtap.exe
echo   CLI: f1amtap.exe input.mid
echo   GUI: f1amtap.exe --gui
exit /b 0
