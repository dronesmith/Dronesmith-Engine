#! /bin/bash

set -e

# Note the build process assumes a relative GOPATH

# Set for OSX
#export GOOS=darwin
#export GOARCH=amd64

# Generate MAVLink
# Due to bug on latest version, we'll only generate this on demand
#go build -o bin/generator mavlink/generator
#./bin/generator -f api/mavlink/message_definitions/v1.0/common.xml -o src/mavlink/parser/common.go

# Set for edison
export GOARCH=amd64
export GOOS=linux

# Build
go build -ldflags "-X config.gitHash=`git rev-parse HEAD`" -o bin/dslink src/main.go

d=`date +%s`
ds_name="dslink_$d"
path="release/$ds_name/ipk-build/opt/dslink"

mkdir -p $path
mkdir -p "release/$ds_name/ipk-build/etc/init.d/"
mkdir -p "release/$ds_name/ipk-build/usr/bin/"
mkdir -p "release/$ds_name/ipk-build/CONTROL"
mkdir -p "$path/flights"
cp bin/dslink "$path/dslink"
mkdir "$path/assets"
mkdir "$path/log"
cp -r assets/ "$path/assets/"
cp -r scripts/ "$path/scripts/"
cp scripts/config.json "$path/config.json"
cp scripts/control "release/$ds_name/ipk-build/CONTROL/control"
cp scripts/startdslink.sh "release/$ds_name/ipk-build/etc/init.d/startdslink.sh"
cp scripts/runvanilla.sh "release/$ds_name/ipk-build/usr/bin/dslink"
mv "$path/scripts/rundaemon.sh" "$path/rundaemon.sh"
rm "$path/scripts/build.sh"

# Not needed on i386 builds
rm -f "$path/assets/ngrok/ngrok_osx"

# tar -cvf "release/$ds_name.tar" $path
# rm -rf $path


# reset for OSX
export GOOS=darwin
export GOARCH=amd64
