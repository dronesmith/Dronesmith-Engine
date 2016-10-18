#! /bin/sh

set -e

cd /opt/dslink

/opt/dslink/scripts/getsim.sh

while true
do
   /opt/dslink/scripts/run.sh
   sleep 1
done
