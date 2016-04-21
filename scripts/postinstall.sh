#!/bin/sh

set -e

echo "Adding startup script"
mkdir -p /etc/init.d/
mkdir -p /var/log/dslink/
rm /usr/lib/edison_config_tools/edison-config-server.js
cp /opt/dslink/scripts/startdslink.sh /etc/init.d/startdslink.sh
update-rc.d startdslink.sh defaults
echo "Please reboot for changes to take effect."
