#!/bin/bash
# build.sh - build multiplataforma de la app P2PFS

APP_NAME="p2pfs"

echo "📦 Compilando para Linux AMD64..."
GOOS=linux GOARCH=amd64 go build -o build/$APP_NAME-linux cmd/main.go

echo "📦 Compilando para Windows AMD64..."
GOOS=windows GOARCH=amd64 go build -o build/$APP_NAME.exe cmd/main.go

echo "📦 Compilando para macOS AMD64..."
GOOS=darwin GOARCH=amd64 go build -o build/$APP_NAME-macos cmd/main.go

echo "✅ Compilación completada. Binarios disponibles en ./build/"

