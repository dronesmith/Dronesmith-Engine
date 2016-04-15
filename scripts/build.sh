#! /bin/bash

set -e

# Note the build process assumes a relative GOPATH

# Generate MAVLink
go build -o bin/generator mavlink/generator
./bin/generator -f lucikit/message_definitions/v1.0/common.xml -o src/mavlink/parser/common.go

export GOARCH=386
export GOOS=linux

# Build
go build -ldflags "-X config.gitHash=`git rev-parse HEAD`" -o bin/dslink src/main.go
