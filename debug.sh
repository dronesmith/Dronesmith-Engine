#! /bin/bash

set -e
# --dsc "24.234.109.135:4002"
# --dsc 24.234.109.135:4002 --dscHttp 24.234.109.135:80
go run -ldflags "-X config.gitHash=`git rev-parse HEAD`" src/main.go --master "/dev/tty.usbmodem1:115200" --dsc 24.234.109.135:4002 --dscHttp 24.234.109.135:80 --setup "old/" --assets "$GOPATH/" --flights "$GOPATH/flights"
