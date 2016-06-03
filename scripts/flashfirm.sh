#!/bin/sh


echo "Power off the FMU"

wget http://stage.dronesmith.io/cdn/luci.px4 /opt/dslink/luci.px4
python /opt/dslink/scripts/px_uploader.py --port /dev/ttyMFD1 --baud 115200 /opt/dslink/luci.px4
