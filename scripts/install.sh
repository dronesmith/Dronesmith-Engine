#!/bin/sh

# Guide for getting simultanous wifi ap and station on Edison
# opkg install iw
# iw dev wlan0 interface add wlan0_ap  type __ap
# ip link set dev wlan0_ap  address 12:34:56:78:ab:ce (or whatever mac you want)
# vi /etc/hostapd/hostapd.conf -> change ssid to something unique (not same as edison name)
# and change interface to wlan0_ap
# hostapd -B /etc/hostapd/hostapd.conf

set -e

function finish {
  echo "!! Error occured. Please try again."
}

trap finish EXIT

echo "[*----] Configuring opkg..."
{
  rm /etc/opkg/base-feeds.conf
  touch /etc/opkg/base-feeds.conf
  echo "src/gz all http://repo.opkg.net/edison/repo/all" >> /etc/opkg/base-feeds.conf
  echo "src/gz edison http://repo.opkg.net/edison/repo/edison" >> /etc/opkg/base-feeds.conf
  echo "src/gz core2-32 http://repo.opkg.net/edison/repo/core2-32" >> /etc/opkg/base-feeds.conf
  opkg update
} &> /dev/null

echo "[**---] Installing dependencies..."
{
  opkg install python-pip
  opkg install git
  pip install "pySerial>=2.0,<=2.9999"
  pip install pymavlink
  pip install mavproxy
} &> /dev/null

echo "[***--] Installing DSLink..."
{
  opkg install http://stage.dronesmith.io/cdn/dslink_alpha_2_luci_x86.ipk
  rm /usr/lib/edison_config_tools/edison-config-server.js
  update-rc.d startdslink.sh defaults
} &> /dev/null

echo "[****-] Flashing the FMU..."
echo "Please note that the FMU must be powered and may need to be rebooted."
{
  cd /opt/dslink
  wget http://stage.dronesmith.io/cdn/luci.px4
} &> /dev/null

/opt/dslink/scripts/flashfirm.sh

echo "[*****] Done. Rebooting in 5..."
sleep 1
echo "4..."
sleep 1
echo "3..."
sleep 1
echo "2..."
sleep 1
echo "1..."
sleep 1
reboot
