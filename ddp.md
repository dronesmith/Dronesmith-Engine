
# Dronesmith Data Protocol

## Stuff from the old readme

#### Header

	byte 0-3: Unique Session Id
	byte 4: Type
	byte 5: Length of payload
	byte 6-N: Payload. See notes below.
	byte N-7,8: two byte little endian CRC-16.

<s>All DroneDP messages are encrypted/decrypted as AES-256 using the following as a key sequence:

	d7 e6 af 0b 14 90 7e a5 0a fd e8 bb 57 4f 3d 99 81 88 d9 f5 1b 90 7d 3d 44 e7 94 e3 30 f0 55 d9
</s>

The crypto is currently in development. AES is used for protection while maintaining low overhead. Actual key authentication is still in development.  

#### Messages

The following messages are currently used:

	0x10: OP_STATUS - Payload is serialized as JSON. Contains the following fields:
		op: <sub operation> <required>

		subop <status> params:
		password: plaintext password being sent to server. 		email: plaintext email being sent to server.
		serialId: Handshake id from flight controller.
		drone: JSON object of drone metadata from server.
		codeStatus: JSON object of code execution related information. Only contains a single subParameter, script, which is either a pid of the currently running code, or null.

		subop <code> params:
		msg: stdout/stderr string from code process.
		status: null if stopped running, otherwise running PID.

		subop <connect> params:
		email: email
		password: password
		serialId: serialId
		codeStatus: codeStatus object

	0xFD: OP_MAVLINK_TEXT - Payload is serialized MAVLink as JSON. Serializing the binary MAVLink as json makes it easier for the server to handle, and doing it on the client side is less load for the server, which is why the binary MAVLink is not simply echoed to the server. The MAVLink messages should look the same as the ones that regular lucimon currently sends out.
	
	0xFE: OP_MAVLINK_BIN - Send binary MAVLink data. 
	
	0x11: OP_CODE - Used for upload code snippets


#### Handshake

Handshake sequence of Lucimon with Cloud:

1. If sessionId is null, chirp following OP_STATUS, with subop 'connect' message with data loaded from config.json, see above. This data is already entered previously by the first time setup of Luci. Send message once every second.
2. Once a sessionId is gotten, send OP_STATUS with subop 'status' message once every second. Be sure to include proper sessionId. This is more or less a request message to the server.
3. On an OP_STATUS reply, Update drone data if different, update session if different. Probably good idea to log if session or drone data changes. This may be inscure, haven't thought this through entirely yet. If you get a codeBuffer and nothign is running, spawn a sub python process with the codebuffer (use exec.py).
4. Always echo mavlink data as OP_MAVLINK_TEXT if you get any from the flight controller, even if there's no connection.
5. If there's no valid reply from server after 5 seconds, reset everything and return to 1.  

## DDP Flight Protocol

And important subset of DDP is the flight protocol, which acts as an intermediate between different protocols. DDP Flight is used when communicating with Dronesmith Cloud, and is the content stored in DSC after a mission is complete. Using a single protocol allows Dronesmith Suite to be cross platforms with all popular, cutting edge UAV technologies, making development faster and easier. 

### Message Format
All messages use the DDP header, and are registered with the opcode `0x20`. See above information about the DDP header.

### Handshake

DDP Missions are encoded by DSLink during whenever one of the following events happen:

1. UAV is placed into a "flight-worthy" mode, such as arming it.
2. The user requests to record a mission from Dronesmith Cloud. Please note this overrides any arming or disarming mechanisms that are used to get a recording. 

Likewise, DDP missions end when one of the following two events happen:

1. The UAV is placed into a disarmed mode. Overruled if the user requested to record the mission via DSC. 
2. The user requests to stop recording from Dronesmith Cloud. 

Missions are saved in a user-defined directory. This directory can be changed, if the user wishes to use their own storage, for instance.

Syncing with the cloud is based on the availility of DSC. When a new recording is requested, a DDP encoder begins translating the data and saving it to a file. The file is the single source of truth. A separate concurrent routine reads the file, and streams the data to the cloud. It will only stream when it has connection to DSC. Each stream begins with a `OP_STREAM_START` and ends with `OP_STREAM_END` to ensure the stream was completed. If no stream data is received after a certain period of time, the stream will time out, and must be restarted. DDP flights frames are recorded with sequence numbers, so DSC can use this to detect if certain messages are lost or not. Keep in mind that DSC assumes lossiness by default, however, the sequence numbers can be used to ensure the full Mission is collected. 

After a stream is successful, DSC will echo back a unique ID for that particular stream, along with a time stamp of its ending. The unique ID also signifies to DSLink that the stream has finished syncing successfully. 

### Message Types

All messages begin with a 2 byte sequence ID, followed by an 8 byte timestamp in milliseconds.

	Status - Displays current information about the drone. 
		Flight Mode
		System Status
		Which subsystems are working are not working
		
	GPS - Global Position Information. Latitudes and Longitudes, altitude.
	
	IMU - Provides 10 axis sensor values. Gyro, accel, mag, and
	
	Attitude - Provides Attitude information in 3 axis of rotation.
	
	VFR - Altitude, Linear and angular velocity, airspeed, heading.
	
	Radio Control - RC Channels, RSSI, status information.
	
	Motors - Motor control and outputs information.
	
	Mission Status - If in guided mode, shows current mission details. 
	
	Battery - Battery percent, if it's charging or not.
	
	Local Position - Local position information, if available.
	

### Drone Meta Data

Each mission begins with a drone metadata list, a set of data about the drone during the time of the Mission. This data does not change during the mission.

	UAV Type (quadrotor, hexrotor, etc...)
	Mission start timestamp
	Mission end timestamp
	Mission unique Id
	Flight Protocol Version 
	Settings (this will be param data on MAVLink drones, varies per protocol)
	Onboard Peripherals
		GPS
		Optical Flow
		Lidar
		etc..
	


## Third Party Flight Protocol Comparisons

### DJI Protocol(s)
**Primary users** 

DJI (Phantom, Matrice, …)

DJI’s protocol is used for all DJI drones. Given the popularity of DJI drones, it makes sense that we should support it as one of our interfaces. It is an overall secure protocol, and uses a more sophisticated message classification system. It is not as intricate with data as MAVLink is however.

#### DJI Packet structure

Header

	0x00 - Header byte. Always 0xAA
	0x0000 - First 10 bits are the length. Next 6 bits are the version.
	0x00 - First 5 bits are the session, next bit is the frame type (either CMD or ACK), final 2 are reserved with 0s.
	0x00 - First 5 bits are the padding. Next 3 are whether or not encryption is used.
	0x000000 - Reserved. Zero’d.
	0x0000 - Frame sequence. Increments after every transmission. 
	0x0000 - Header checksum.
	0x… - Data, defined by the 10 bit length.
	0x00000000 - 32 bit checksum for the entire frame.

DJI’s protocol uses a distinctive structure for either commands or “normal” messages which it calls, ACK messages. 

The CMD messages all use the following format:

	0xXX - CMD Set. Used to specify the type of command.
	0xXX - CMD ID. Opcode. 
	0x… - CMD Val. Arguments for the command.

ACK frames are unique to each ACK from a particular command.

DJI’s protocol is not a time based protocol, but a command<=>ack protocol. The command set roughly determines how the operation is handled.

**Activation Commands**: Essentially used to get protocol meta data. Also used to set the frequency of “push data” commands. 

**Control Commands**: Used to control the vehicle. 

**Push Data Commands**: Configured by activation commands. They have no ACKs. These are sent periodically. 

**Ground control commands**: Used to do waypoint based missions.

**Sync Command**: Used to configure a heartbeat.

**Virtual RC Command**: used to override motor control with custom RC values. 

### Parrot AR Protocol
**Primary users** 

Parrot AR Drone, Parrot Bebop, …

Parrot's protocol is by far the simplest and most straight forward. This is probably why it is quite popular among researchers for testing control theory applications. 

Parrot uses an ASCII encoded protocol, which is unusual and inefficient. Most flight protocol are either little or big endian binary encoded. All commands begin with `AT*` as the header, followed the `command name`, then `=`, then a sequence number as an ascii encoded string, which is incremented after each frame is sent. Then a sequence of ASCII arguments each `,` separated, and finally a `\n` (linefeed, `0x10`) as the termination command. 

Parrot supports the following types. Use traditional ASCII parsing to determine them:

	2's complement signed ints
	null termined strings
	IEEE-754 floats

There is no checksum. Parrot's UAVs all rely on UDP as their transport protocol, which contains its own checksum, so while it is not necessarily needed, it is limiting. Overall, this is just not a well designed protocol. 

### MAVLink
**Primary users**

3D Robotics (APM, Solo)

PX4 Project (Pixhawk, PixRacer)

…Number of other open source Drones and Flight stacks

MAVLink is an open protocol. There have been two major revisions. This document will focus version 2. There is also a third version in development. It features improvements such as encryption and additional security. Currently, it is not in production use, but there exist development builds of it. 

MAVLink is the by far the ‘largest’ protocol in terms of number of messages and users. This is mostly due to its ubiquitous nature of being the de facto open communications protocol. Its main issues are its lack of security measures, making it easy to hack, and its openness means the protocol is subject to rapid change and many variation of it exist. 

#### MAVLink Structure

Uniform header. All messages contain the following header information:

	0xFE - Starting byte (0x55 on older version)
	0xXX - Payload length (max 255)
	0xXX - Packet sequence (increments by 1 on every message, rolls over at 255.)
	0xXX - System ID. Used to identify the drone. Will be largely ignored by DSS.
	0xXX - Component ID. Used to identify components connected to the drone. Will be largely ignored by DSS.
	0xXX - Message ID. Op code.
	0x… - Data, the size of the payload defined by the length above. 
	0xXXXX - Checksum. 

MAVLink uses two checksums. The first is derived from the message. The second is derived from a magic number created during the inception of the protocol message. Each message field is sorted by the size of the base type. The Endianness is little. The following are the supported payload types:

	char 
	uint8_t
	int8_t
	uint16_t
	int16_t
	uint32_t
	int32_t
	uint64_t
	int64_t
	float
	double
	uint8_t_mavlink_version <special type>

Any of these types may also be declared as a C-style array. Ex: `uint8_t[3]` - triplet of unsigned bytes.

MAVLink is a time based protocol that sends out messages in periodic intervals. It supports two kinds of command messages, command ints which request a single chunk of data, and command longs that request a slew of data and contain special ACKs. The only message actually required for MAVLink to work is the heartbeat message, which should be sent out periodically every `1` second. 

