
# Dronesmith Link

DS Link is the client application that runs on Lucis, and any other Linux capable companion chip.

## MAVLink Protocol

	byte 0: 0xFE - header byte.
	byte 1: 0xNN - payload length.
	byte 2: 0xNN - packet sequence. Increases by 1 on each MAVLink send.
	byte 3: 0xNN - System Id. Will be 1. Can never be 0xFF or 0xFE. Normally used to differentiate between multiple drones on the same network, but not used in our cases.
	byte 4: 0xNN - Component Id. Usually 1 as well. Can never be 0xFF or 0xFE. Used to differentiate different devices on the same drone.
	byte 5: 0xNN - Message Id. (see https://github.com/mavlink/mavlink/blob/master/message_definitions/v1.0/common.xml)
	byte 6-N: <buffer> - payload buffer, depends on Message Id. See above link.
	byte N-7,8: 0x0CRC - CRC (low byte, then high byte).

Format is in little endian format. For the CRC, it's a little weird. The CRC algorithm is first run over the XML tagged data which produces a seed CRC number. When a message is generated or parsed, first the CRC is done on the message using 0 as the seed. Then, a second CRC is done on the entire packet including the prior CRC, using the message seed. See the comment by Andrew in this post: http://diydrones.com/forum/topics/mavlink-1-0-checksum-protocol

I have the code creating a MAVLink structure from the XML, but it is not validiating the CRCs properly. I added the seed values for each message hardcoded in there, but it'd be nice to dynamically generate them with the MAVLink XML fetched dynamically.

## DroneDP Protocol
See dronedp.js.

	byte 0-3: Unique Session Id
	byte 4: Type (either 0x10, or 0xFD)
	byte 5: Length of payload
	byte 6-N: Payload. See notes below.
	byte N-7,8: two byte little endian CRC-16.

All DroneDP messages are encrypted/decrypted as AES-256 using the following as a key sequence:

	d7 e6 af 0b 14 90 7e a5 0a fd e8 bb 57 4f 3d 99 81 88 d9 f5 1b 90 7d 3d 44 e7 94 e3 30 f0 55 d9

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


Please note DroneDP is subject to a lot of change, as things get developed. But it's simple enough now it should be pretty easy to implement.

Handshake sequence of Lucimon with Cloud:

1. If sessionId is null, chirp following OP_STATUS, with subop 'connect' message with data loaded from config.json, see above. This data is already entered previously by the first time setup of Luci. Send message once every second.
2. Once a sessionId is gotten, send OP_STATUS with subop 'status' message once every second. Be sure to include proper sessionId. This is more or less a request message to the server.
3. On an OP_STATUS reply, Update drone data if different, update session if different. Probably good idea to log if session or drone data changes. This may be inscure, haven't thought this through entirely yet. If you get a codeBuffer and nothign is running, spawn a sub python process with the codebuffer (use exec.py).
4. Always echo mavlink data as OP_MAVLINK_TEXT if you get any from the flight controller, even if there's no connection.
5. If there's no valid reply from server after 5 seconds, reset everything and return to 1.  
