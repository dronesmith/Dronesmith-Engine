#! /bin/sh

set -e

cd /opt/dsengine

/opt/dsengine/scripts/getsim.sh

while true
do
   /opt/dsengine/scripts/run.sh
   sleep 1
done
