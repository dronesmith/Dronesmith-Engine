#! /bin/bash

set -e

go run -ldflags "-X config.gitHash=`git rev-parse HEAD`" src/main.go --master "/dev/tty.usbmodem1:115200" --setup "old/" --assets "$GOPATH/" --dsc "24.234.109.135:4002"
