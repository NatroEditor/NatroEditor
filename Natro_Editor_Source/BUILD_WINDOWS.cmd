@echo off
setlocal
cd /d "%~dp0"
where go >nul 2>&1 || (echo Please install Go 1.23+ from go.dev and try again.& exit /b 1)
where python >nul 2>&1 || (echo Please install Python 3 and try again.& exit /b 1)
if not exist "build" mkdir "build"
cd launcher
go test ./... || exit /b 1
go build -trimpath -ldflags="-H=windowsgui -s -w" -o "..\build\Natro_Editor_without_icon.exe" . || exit /b 1
cd ..
python tools\attach_icon.py build\Natro_Editor_without_icon.exe build\Natro_Editor.exe resources\original_app_icon.rsrc || exit /b 1
echo.
echo Finished: build\Natro_Editor.exe
echo NOTE: This rebuild is unsigned. Windows reputation checks may warn.
