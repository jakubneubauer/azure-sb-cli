#!/bin/bash

VERSION="0.4"
DATE=`date -u '+%Y-%m-%d_%H:%M:%S'`

build() {
  go build -o azure-sb-cli$EXT -ldflags "-X main.buildVersion=$VERSION -X main.buildDate=$DATE" main.go
}

# I use linux => this is the Linux build ;-)
GOOS=linux GOARCH=amd64 EXT="" build

# Windows binary build
GOOS=windows GOARCH=386 EXT=.exe build
