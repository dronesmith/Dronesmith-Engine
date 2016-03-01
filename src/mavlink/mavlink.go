package mavlink

import (
  "bytes"
  "encoding/xml"
  "encoding/binary"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

const (
	DEFAULT_MAVLINK_XML = "assets/common.xml"
	MAVLINK_VERSION = 688383
)

/**
 * MagicNum CRC for each message used for validation
 */
var mavlinkCRCs []byte

/**
 * Mavlink Protocol
 * Constants - Definitions of fields and command and the like
 * Messages - Message information
 */
type Mavlink struct {
	Constants			map[string]constant
	Messages 			map[string]messageMap

	decoded				rawMavlink
}

/**
 * Mavlink Message Header
 */
type MavlinkMessageHeader struct {
	Header					uint8
	PayloadSize			uint8
	PacketSequence 	uint8
	SystemId				uint8
	ComponentId 		uint8
	MessageId 			uint8
}

/**
 * Mavlink Message Format
 */
type MavlinkMessage struct {
	Header 					MavlinkMessageHeader
	Id							uint8
	Name						string
	Payload 				map[string]interface{}
	Checksum				uint16
}

/**
 * Struct represents information about a field
 */
type messageMap struct {
	Id 						int
	Fields 				map[string]field
	Name					string
	Description		string
}

/**
 * XML structs for parsing the MAVLink document.
 */
type rawMavlink struct {
  XMLName				xml.Name				`xml:"mavlink"`
 	Version				int 						`xml:"version"`
 	MsgCrcs				string					`xml:"messagecrcs"`
 	Constants 		[]constant			`xml:"enums>enum,omitempty"`
 	Messages 			[]message 			`xml:"messages>message,omitempty"`
}

type constant struct {
 	Description		string 					`xml:"description,omitempty"`
 	Name					string 					`xml:"name,attr"`
 	Entries				[]entry					`xml:"entry,omitempty"`
}

type entry struct {
 	Description		string					`xml:"description"`
 	Value					int							`xml:"value,attr"`
 	Name					string 					`xml:"name,attr"`
 	Params				[]param					`xml:"param,omitempty"`
}

type param struct {
 	Index 				int							`xml:"index,attr"`
 	Description		string 					`xml:",chardata"`
}

type message struct {
 	Id 						int							`xml:"id,attr"`
 	Name					string 					`xml:"name,attr"`
 	Description		string 					`xml:"description,omitempty"`
 	Fields				[]field					`xml:"field,omitempty"`
}

type field struct {
 	Description		string 					`xml:",chardata"`
 	Type					string					`xml:"type,attr"`
 	Name					string					`xml:"name,attr"`
}

/**
 * Creates a new MAVLink struct by parsing a valid XML file. Should before
 * called for anything else.
 */
func NewMavlink(url string) *Mavlink {
	mav := &Mavlink{
		Constants: 	make(map[string]constant),
		Messages:		make(map[string]messageMap),
		decoded:		rawMavlink{},
	}

	if data, err := ioutil.ReadFile(DEFAULT_MAVLINK_XML); err != nil {
		panic(err)
	} else {
		if err := xml.Unmarshal(data, &mav.decoded); err != nil {
			panic(err)
		} else {
			if mav.decoded.Version != MAVLINK_VERSION {
				panic("Failed to initialize. Invalid MAVLink Version.")
			}

			// Populate crcs
			split := strings.Split(mav.decoded.MsgCrcs, ", ")
			mavlinkCRCs = make([]byte, len(split))

			for i, v := range split {
				if conv, err := strconv.Atoi(v); err != nil {
					panic(err)
				} else {
					mavlinkCRCs[i] = byte(conv)
				}
			}

			// Shallow copy to maps to make it easier to work with
			for _, elem := range mav.decoded.Constants {
				mav.Constants[elem.Name] = elem
			}

			for _, elem := range mav.decoded.Messages {
				mav.Messages[elem.Name] = messageMap{
					Id: 					elem.Id,
					Fields: 			make(map[string]field),
					Name: 				elem.Name,
					Description:	elem.Description,
				}

				for _, field := range elem.Fields {
					mav.Messages[elem.Name].Fields[field.Name] = field
				}
			}

			return mav
		}
	}
}

/**
 * Parses a byte slice and returns a new MAVLink Message if valid.
 */
func (mav *Mavlink) Parse(data []byte) *MavlinkMessage {
	msg := &MavlinkMessage{}
	hdr := bytes.NewBuffer(data[0:6])

	if err := binary.Read(hdr, binary.LittleEndian, &msg.Header); err != nil {
		panic(err)
	} else {
		msg.Id = msg.Header.MessageId
		msg.Payload = *mav.parsePayload(msg.Header.MessageId, data[7:msg.Header.PayloadSize+7], &msg.Name)
		msg.Checksum = binary.LittleEndian.Uint16(data[6 + msg.Header.PayloadSize:])

		if mav.crc(data[1:msg.Header.PayloadSize+6],msg.Header.MessageId) != msg.Checksum {
			fmt.Printf("Invalid CRC from %d\n", msg.Header.MessageId)
		}

		fmt.Printf("Message: %v\n\n\n", msg)
	}

	return msg
}

/**
 * Used to update the message with the proper type. It's about the best I can do
 * given Golang's static typing.
 */
func (mav *Mavlink) evalType(T string, data []byte, index *int) interface{} {
	var val int = *index

	inc := func (v *int, a int) {
		*v = *v + a
	}

	switch T {
	case "uint8_t_mavlink_version":
		fallthrough
	case "uint8_t": 	defer inc(index, 1); return uint8(data[val])
	case "int8_t": 		defer inc(index, 1); return int8(data[val])
	case "uint16_t": 	defer inc(index, 2); return binary.LittleEndian.Uint16(data[val : val + 2])
	case "int16_t": 	defer inc(index, 2); return int16(binary.LittleEndian.Uint16(data[val : val + 2]))
	case "uint32_t": 	defer inc(index, 4); return binary.LittleEndian.Uint32(data[val : val + 4])
	case "int32_t": 	defer inc(index, 4); return int32(binary.LittleEndian.Uint32(data[val : val + 4]))
	case "uint64_t": 	defer inc(index, 8); return binary.LittleEndian.Uint32(data[val : val + 8])
	case "int64_t": 	defer inc(index, 8); return int64(binary.LittleEndian.Uint64(data[val : val + 8]))
	case "float": 		defer inc(index, 4); return float32(binary.LittleEndian.Uint32(data[val : val + 4]))
	default:
		fmt.Println("Unknown type: %s", T)
		inc(index, 1)
	}
	return nil
}

/**
 * Uses the Mavlink struct to determine how to parse the payload field, and parses
 * it accordingly.
 */
func (mav *Mavlink) parsePayload(id uint8, data []byte, name *string) *map[string]interface{} {

	for _, message := range mav.decoded.Messages {
		if message.Id == int(id) {
			cnt := 0
			parsedPayload := make(map[string]interface{})

			// fmt.Printf("\n\nMessage info: %v\n", message.Name)
			// fmt.Printf("Message Payload: %v\n", data)
			*name = message.Name

			for _, field := range message.Fields {
				mult := 1
				subType := field.Type

				// handle [ ] types.
				if strings.ContainsRune(field.Type, '[') {
					splits := strings.Split(field.Type, "[")

					subType = splits[0]
					fmt.Sscanf(splits[1], "%d]", &mult)
				}

				// Decode binary data from field types
				if subType == "char" {
					var val bytes.Buffer
					val.Write(data[cnt-1:mult])
					cnt += mult
					parsedPayload[field.Name] = val.String()
				} else {
					for i := 0; i < mult; i++ {
						var fieldName string
						if mult > 1 {
							fieldName = field.Name + strconv.Itoa(i)
						} else {
							fieldName = field.Name
						}
						parsedPayload[fieldName] = mav.evalType(subType, data, &cnt)
					}
				}
			}

			return &parsedPayload
		}
	}

	return nil
}

/**
 * Evaluates the CRC of a MAVLink message.
 */
func (mav *Mavlink) crc(buff []byte, id uint8) uint16 {
	// keeping these here in case changes are made.
	const (
		X25_INIT_CRC = 0xffff
		X25_VALIDATE_CRC = 0xf0b8
	)

	var crcAccum uint16 = X25_INIT_CRC
	for i := range buff {
		mav.crcAccum(buff[i], &crcAccum)
	}

	// Add the seed
	mav.crcAccum(mavlinkCRCs[id], &crcAccum)
	return crcAccum
}

/**
 * Helper function for CRC
 */
func (mav *Mavlink) crcAccum(b uint8, t *uint16) {
	var tmp uint8

	tmp = b ^ uint8(*t & 0xff)
	tmp ^= (tmp << 4)
	*t = (*t >> 8) ^ (uint16(tmp) << 8) ^ (uint16(tmp) << 3) ^ (uint16(tmp) >> 4)
}
