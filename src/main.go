package main

import (
	"fmt"
	"net"
	"io/ioutil"
	"encoding/xml"
	"encoding/binary"
	"time"
	"bytes"
	"strings"
	"strconv"
)

var MAVLINK_CRCS []byte

const (
	MAVLINK_XML = "assets/common.xml"
	MAVLINK_VERSION = 688383
)


// The raw, parsed Mavlink protocol
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

// An easier to read version of the Mavlink Protocol
type Mavlink struct {
	Constants			map[string]constant
	Messages 			map[string]messageMap

	decoded				rawMavlink
}

type messageMap struct {
	Id 						int
	Fields 				map[string]field
	Name					string
	Description		string
}

func NewMavlink(url string) *Mavlink {
	mav := &Mavlink{
		Constants: 	make(map[string]constant),
		Messages:		make(map[string]messageMap),
		decoded:		rawMavlink{},
	}

	if data, err := ioutil.ReadFile(MAVLINK_XML); err != nil {
		panic(err)
	} else {
		if err := xml.Unmarshal(data, &mav.decoded); err != nil {
			panic(err)
		} else {
			if mav.decoded.Version != MAVLINK_VERSION {
				panic("Failed to initialize. Invalid MAVLink Version.")
			}

			split := strings.Split(mav.decoded.MsgCrcs, ", ")
			MAVLINK_CRCS = make([]byte, len(split))

			// Populate crcs
			for i, v := range split {
				if conv, err := strconv.Atoi(v); err != nil {
					panic(err)
				} else {
					MAVLINK_CRCS[i] = byte(conv)
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

type MavlinkMessageHeader struct {
	Header					uint8
	PayloadSize			uint8
	PacketSequence 	uint8
	SystemId				uint8
	ComponentId 		uint8
	MessageId 			uint8
}

type MavlinkMessage struct {
	Header 					MavlinkMessageHeader
	Payload 				[]byte // TODO make this a generic payload interface
	Checksum				uint16
}

// TODO
type MavlinkData interface {

}

func (mav *Mavlink) Parse(data []byte) *MavlinkMessage {
	msg := &MavlinkMessage{}
	hdr := bytes.NewBuffer(data[0:6])

	if err := binary.Read(hdr, binary.LittleEndian, &msg.Header); err != nil {
		panic(err)
	} else {
		msg.Payload = data[7:msg.Header.PayloadSize+7]
		msg.Checksum = binary.LittleEndian.Uint16(data[6 + msg.Header.PayloadSize:])
		// fmt.Printf("%v\n", msg)

		if mav.crc(data[1:msg.Header.PayloadSize+6],msg.Header.MessageId) != msg.Checksum {
			fmt.Printf("Invalid CRC from %d\n", msg.Header.MessageId)
		}

	}

	return msg
}

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
	mav.crcAccum(MAVLINK_CRCS[id], &crcAccum)
	return crcAccum
}

func (mav *Mavlink) crcAccum(b uint8, t *uint16) {
	var tmp uint8

	tmp = b ^ (uint8)(*t & 0xff)
	tmp ^= (tmp << 4)
	*t = (uint16(*t) >> 8) ^ (uint16(tmp) << 8) ^ (uint16(tmp) << 3) ^ (uint16(tmp) >> 4)
}

// func (mav *Mavlink) crcAccumBuffer()

// func (parsed *MavlinkDecode) mapify() {
// 	// mav = &Mavlink{
// 	// 	Constants: make(map[string]constant),
// 	// 	Messages:	 make(map[string]messageMap),
// 	// }
//
// 	// Shallow copy to maps to make it easier to work with
// 	for _, elem := range parsed.Constants {
// 		mav.Constants[elem.Name] = elem
// 	}
//
// 	for _, elem := range parsed.Messages {
// 		mav.Messages[elem.Name] = messageMap{
// 			Id: 					elem.Id,
// 			Fields: 			make(map[string]field),
// 			Name: 				elem.Name,
// 			Description:	elem.Description,
// 		}
//
// 		for _, field := range elem.Fields {
// 			mav.Messages[elem.Name].Fields[field.Name] = field
// 		}
// 	}
//
// 	return
// }

func main() {
    // laddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:14550")
    // conn, _ := net.ListenUDP("udp", laddr) // check errors
		//
    // parser := mav.GetMavParser()
    // reader := bufio.NewReader(conn)
		//
    // for {
    //     c, _ := reader.ReadByte() // check errors
		//
    //     packet, err := parser(c)
    //     if err != nil {
    //         log.Fatalf("Parser error: ", err)
    //     } else if packet != nil {
    //         fmt.Println("Packet received :", packet.Bytes())
    //     }
    // }

	// parsed := &MavlinkDecode{}
	//
	// fmt.Printf("[MON] Fetching XML...\n")
	//
	// if res, err := http.Get(MAVLINK_XML); err != nil {
	// 	panic(err)
	// } else {
	// 	defer res.Body.Close()
	// 	if contents, err := ioutil.ReadAll(res.Body); err != nil {
	// 		panic(err)
	// 	} else {
	// 		if err := xml.Unmarshal(contents, &parsed); err != nil {
	// 			panic(err)
	// 		} else {
	// 			mavlink := parsed.mapify()
	// 		}
	// 	}
	// }

	port := "127.0.0.1:14550"
	udpAddress, err := net.ResolveUDPAddr("udp4", port)

	if err != nil {
		fmt.Println("error resolving UDP address on ", port)
		fmt.Println(err)
		return
	}

	fmt.Printf("[MON] Fetching XML...\n")
	mav := NewMavlink(MAVLINK_XML)

	fmt.Print("[MON] Initiating connection...\n")
	conn, err := net.ListenUDP("udp", udpAddress)

	if err != nil {
		fmt.Println("error listening on UDP port ", port)
		fmt.Println(err)
		return
	}

	defer conn.Close()

	buf := make([]byte, 2048)

	fmt.Printf("[MON] Listening.\n")

	for {
		time.Sleep(100 * time.Millisecond)

		n, address, err := conn.ReadFromUDP(buf)

		if err != nil {
			fmt.Println("error reading data from connection")
			fmt.Println(err)
			return
		}

		if address != nil {
			// fmt.Println("got message from", address, " with n = ", n)
			if n > 0 {
				mav.Parse(buf)
			}
		}
	}
}
