#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
mkdir -p build
(cd launcher && go test ./... && GOOS=windows GOARCH=amd64 go build -trimpath -ldflags='-H=windowsgui -s -w' -o ../build/Natro_Editor_without_icon.exe .)
python3 tools/attach_icon.py build/Natro_Editor_without_icon.exe build/Natro_Editor.exe resources/original_app_icon.rsrc
printf 'Built: build/Natro_Editor.exe (unsigned)\n'
