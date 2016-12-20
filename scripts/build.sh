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
export GOARCH=386
export GOOS=linux

# Build
go build -ldflags "-X config.gitHash=`git rev-parse HEAD`" -o bin/dsengine src/main.go

d=`date +%s`
ds_name="dsengine_$d"
path="release/$ds_name/ipk-build/opt/dsengine"

mkdir -p $path
mkdir -p "release/$ds_name/ipk-build/etc/init.d/"
mkdir -p "release/$ds_name/ipk-build/usr/bin/"
mkdir -p "release/$ds_name/ipk-build/CONTROL"
mkdir -p "$path/flights"
cp bin/dsengine "$path/dsengine"
mkdir "$path/assets"
mkdir "$path/log"
cp -r assets/ "$path/assets/"
cp -r scripts/ "$path/scripts/"
cp scripts/config.json "$path/config.json"
cp scripts/control "release/$ds_name/ipk-build/CONTROL/control"
cp scripts/startdsengine.sh "release/$ds_name/ipk-build/etc/init.d/startdsengine.sh"
cp scripts/runvanilla.sh "release/$ds_name/ipk-build/usr/bin/dsengine"
mv "$path/scripts/rundaemon.sh" "$path/rundaemon.sh"
rm "$path/scripts/build.sh"

# Not needed on i386 builds
rm -f "$path/assets/ngrok/ngrok_osx"

# tar -cvf "release/$ds_name.tar" $path
# rm -rf $path


# reset for OSX
export GOOS=darwin
export GOARCH=amd64
