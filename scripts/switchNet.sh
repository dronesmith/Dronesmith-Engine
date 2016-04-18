#!/bin/sh

if [ -z $LUCI_NET_STATUS ]
then
    echo "Could not find env var. Setting."
    export LUCI_NET_STATUS=up
fi

while true
do
ping -c 5 8.8.8.8>>/dev/null

if [ $? -eq  0 ]
    then
    if [ "$LUCI_NET_STATUS" == "down" ]
    then
       echo "Able to reach internet"
       configure_edison --disableOneTimeSetup
       export LUCI_NET_STATUS=up
    fi
else
    if [ "$LUCI_NET_STATUS" == "up" ]
    then
       echo "Unable to reach internet"
       configure_edison --enableOneTimeSetup
       export LUCI_NET_STATUS=down
    fi
fi
sleep 5
done
