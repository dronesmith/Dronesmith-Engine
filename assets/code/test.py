
import sys

from lucikit import *
from pymavlink import *

import random
from time import *


# Connect to the Vehicle
vehicle = connect('0.0.0.0:14551', wait_ready=True)

print "Sending RGB Command"

# vehicle._rgbled.color((1.0, 0.0, 1.0))
#
# vehicle._rgbled.pulse()
# vehicle._rgbled.color((random.random(), random.random(), random.random()))
# sleep(2)
# vehicle._rgbled.pulse()
# sleep(2)
# vehicle._rgbled.color((random.random(), random.random(), random.random()))
# sleep(2)
# vehicle._rgbled.setDefault()
#
# vehicle._rgbled.color((1.0, 0.0, 1.0))

vehicle.mode    = VehicleMode("GUIDED")
vehicle.armed   = True
sleep(5)

print "Taking off!"
vehicle.simple_takeoff(2)
sleep(5)

#Close vehicle object before exiting script
print "Close vehicle object"
sleep(5)
vehicle.close()
