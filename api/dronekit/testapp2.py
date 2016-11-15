
#
# Simple script to change the color of the RGBLED depending on the orientation of the
# drone.
#

import sys

import random
import math
from time import *

from pymavlink import *
from dronekit import *

__DRONE__ = '0.0.0.0:14551'

def translate(value, leftMin, leftMax, rightMin, rightMax):
    # Figure out how 'wide' each range is
    leftSpan = leftMax - leftMin
    rightSpan = rightMax - rightMin

    # Convert the left range into a 0-1 range (float)
    valueScaled = float(value - leftMin) / float(leftSpan)

    # Convert the 0-1 range into a value in the right range.
    return rightMin + (valueScaled * rightSpan)

def getProperty(drone, param):
    return drone.parameters[param]

def up(drone, rate = 1.0):
    channel = int(math.floor(getProperty(drone, 'RC_MAP_THROTTLE')))

    if channel <= 0:
        print('>>> No RC channels detected.')
        return

    channelMin = getProperty(drone, 'RC'+str(channel)+'_MIN')
    channelMax = getProperty(drone, 'RC'+str(channel)+'_MAX')
    channelMid = getProperty(drone, 'RC'+str(channel)+'_TRIM')

    maxRange = channelMax - channelMid
    minRange = channelMin - channelMid

    if rate > 10:
        rate = 10

    if rate < 0:
        rate = 0

    # ensure the UAV does not move when rate is 0
    if rate == 0.0:
        drone.vehicle.channels.overrides[channel] = channelMid
        return

    drone.vehicle.channels.overrides[channel] = translate(rate, 0, 10, channelMid, channelMax)

def sendAttitudeTarget(drone, thrust):
    # print(drone.message_factory)
    # for attr in dir(drone.message_factory):
    #     print(attr)
    drone.message_factory.set_attitude_target_send(
                0, # timestamp
                0, 1,    # target system, target component
                0,  # type mask
                (1.0, 0.0, 0.0, 0.0), # w, x, y, z
                0, # roll rate
                0, # pitch rate
                10, # yaw rate
                thrust, # thrust
                )
    # drone.message_factory.set_attitude_target_send(msg)

drone = connect(__DRONE__, wait_ready=True)

# drone.mode = 'STABILIZE'
# drone.armed = True

thr = 0
for i in range(30):
    print('setting attitude target')
    sendAttitudeTarget(drone, thr)
    thr += 10
    if thr >= 100:
        thr = 0
    sleep(5)

sleep(5)


drone.armed = False
drone.close()
