#! /bin/sh

set -e

#ifconfig eth0 | perl -n -e 'if (m/inet addr:([\d\.]+)/g) { print $1 }'

b=$(ifconfig wlan0 | perl -n -e 'if (m/inet addr:([\d\.]+)/g) { print $1 }')
#echo $b

/opt/dslink/dslink --master /dev/ttyMFD1:921600 --dsc 24.234.109.135:4002 --dscHttp 24.234.109.135:80 --status $b:80 --log /opt/dslink/log/dslink_`date +%s`.log --output "0.0.0.0:14551" --flights "/opt/dslink/flights" --daemon
