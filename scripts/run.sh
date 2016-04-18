#! /bin/sh

set -e

# Note the build process assumes a relative GOPATH

export GOPATH = /opt/dslink/

#ifconfig eth0 | perl -n -e 'if (m/inet addr:([\d\.]+)/g) { print $1 }'

b=$(ifconfig wlan0 | perl -n -e 'if (m/inet addr:([\d\.]+)/g) { print $1 }')
#echo $b

/opt/dslink --master /dev/ttyMFD1:921600 --dsc 24.234.109.135:4002 --status $b:80 --log /var/log/dslink/dslink_`date +%s`.log --daemon
