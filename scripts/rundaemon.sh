#! /bin/sh

set -e

cd /opt/dslink

sleep 10

/opt/dslink/scripts/switchNet.sh &


sleep 30

while true
do
   /opt/dslink/scripts/run.sh
   sleep 1
done
