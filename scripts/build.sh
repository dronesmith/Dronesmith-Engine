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

d=`date +%s`
ds_name="dslink_$d"
path="release/$ds_name"

mkdir $path
cp bin/dslink "$path/dslink"
mkdir "$path/assets"
cp -r assets/ "$path/assets/"
cp -r scripts/ "$path/scripts/"
cp -r lucikit/ "$path/lucikit/"

rm -rf "$path/assets/ngrok/ngrok_osx"

tar -cvf "release/$ds_name.tar" $path
rm -rf $path
