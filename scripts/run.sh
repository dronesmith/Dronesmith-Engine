#! /bin/sh

set -e

#ifconfig eth0 | perl -n -e 'if (m/inet addr:([\d\.]+)/g) { print $1 }'

b=$(ifconfig wlan0 | perl -n -e 'if (m/inet addr:([\d\.]+)/g) { print $1 }')
#echo $b

/opt/dslink/dslink --log /opt/dslink/log/dslink_`date +%s`.log
