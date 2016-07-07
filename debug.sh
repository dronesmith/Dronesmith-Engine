#! /bin/bash

set -e
# --dsc "24.234.109.135:4002"
# --dsc 24.234.109.135:4002 --dscHttp 24.234.109.135:80
go run -ldflags "-X config.gitHash=`git rev-parse HEAD`" src/main.go --master "0.0.0.0:14550" --remote "0.0.0.0:14556" --sync 1 --dsc cloud.dronesmith.io:4002 --dscHttp cloud.dronesmith.io:80 --setup "old/" --assets "$GOPATH/" --flights "$GOPATH/flights"
