#!/bin/bash

VERSION="0.4-preview"
DATE=`date -u '+%Y-%m-%d_%H:%M:%S'`

build() {
  go build -o azure-sb-cli$EXT -ldflags "-X main.buildVersion=$VERSION -X main.buildDate=$DATE" main.go
}
build
GOOS=windows GOARCH=386 EXT=.exe build
