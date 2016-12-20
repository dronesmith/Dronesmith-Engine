#!/bin/sh

set -e

echo "Adding startup script"
mkdir -p /etc/init.d/
mkdir -p /var/log/dsengine/
rm /usr/lib/edison_config_tools/edison-config-server.js
cp /opt/dsengine/scripts/startdsengine.sh /etc/init.d/startdsengine.sh
update-rc.d startdsengine.sh defaults
echo "Please reboot for changes to take effect."
