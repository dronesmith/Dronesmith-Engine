import sys

import random
from time import *

from pymavlink.quaternion import *

from pymavlink import *

from dronekit import *

__DRONE__ = '0.0.0.0:14552'


def sendAttitudeTarget(drone, roll, pitch, yaw, thrust):
    mat = Matrix3()
    mat.from_euler(roll * (180.0/3.14159), pitch * (180.0/3.14159), yaw * (180.0/3.14159))
    quat = Quaternion(mat)
    drone.message_factory.set_attitude_target_send(
                0, # timestamp
                0, 1,    # target system, target component
                0,  # type mask
                (quat[0], quat[1], quat[2], quat[3]), # w, x, y, z
                0, # roll rate
                0, # pitch rate
                0, # yaw rate
                thrust, # thrust
                )

def sendNEDPos(drone, pos, vel, rot=0.0):
    drone.message_factory.set_position_target_local_ned_send(
                0, # timestamp
                0, 1, # target system, target component
                7, # local offset NED coordinate frame
                0,
                pos[0], pos[1], -pos[2], # local position
                vel[0], vel[1], vel[2], # local vel
                2.0, 2.0, 2.0,
                rot, .5,
                )

# Connect to the Vehicle
drone = connect(__DRONE__, wait_ready=True)

# drone.armed = True

drone.message_factory.set_mode_send(
   1, # target system
   128 & 4 & 1, # ARMED & AUTO & Custom mode enabled
   7, # offboard control
)

# if drone.armed is not True:
drone.armed = True

print("Attitude test")

x = 0.0
t = 0.0
swap = False
#for i in range(10):
        #print('pos control...')
        #sendNEDPos(drone, (30.0, 20.0, 30.0), (2.0, 2.0, 2.0), 0.0)
        #print(x)
        #sendAttitudeTarget(drone, 0.0, 0.0, x,  .05)
        #x += 5.0
        #sleep(5)
# print("taking off")

a_location = LocationLocal(-34.364114, 149.166022, 30)
drone.armed = True
# drone.simple_takeoff(50)

#drone._master.mav.command_long_send(1, 0, mavutil.mavlink.MAV_CMD_NAV_TAKEOFF,
#                                      0, 0, 0, 0, 0, 0, 0, 50)

#sleep(20)
#print("sending att target")
#sendAttitudeTarget(drone, 5)
# drone.simple_goto(a_location, 5)

#sendNEDPos(drone, (1.0, 2.0, 5.0), (1.0, 1.0, 1.0), 3.0)

cnt = 0
while cnt < 6000:
        #sendNEDPos(drone, (1.0, 1.0, 1.0), (0.0, 0.0, 0.0), 0.0)
        sendAttitudeTarget(drone, 0.0, 0.0, 5.0,  .3)
        cnt += 1
        sleep(.1)


sleep(2)

#drone.armed = False

# for i in range(20):
#     drone.sensor('rads', genRad())
#     sleep(1)

# Close drone object
print("Done.")
drone.close()
