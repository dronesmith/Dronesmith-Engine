package dronedp

import (
  "bytes"
  "encoding/binary"
  "encoding/json"
  "strconv"
  "errors"
  "log"

  "cloudlink/crc16"
)

type OP uint8

const (
  // TODO - actual TLS here. Keep this out of code. Not actually 'secure.'
  SECURE_KEY string = "d7 e6 af 0b 14 90 7e a5 0a fd e8 bb 57 4f 3d 99 81 88 d9 f5 1b 90 7d 3d 44 e7 94 e3 30 f0 55 d9"

  // Ops
  OP_STATUS OP = 0x10
  OP_CODE OP = 0x11
  OP_MAVLINK_TEXT OP = 0xFD
  OP_MAVLINK_BIN OP = 0xFE
)

var (
  UseEncryption = true
)



func GenerateMsg(opCode OP, session uint32, data interface{}) (*bytes.Buffer, error) {
  var err error = nil
  var payload []byte

  switch opCode {
    // Binary encoded messages for storing flight data. TODO.
  case OP_MAVLINK_BIN:
    return nil, errors.New("D2P: OP_MAVLINK_BIN is unsupported.")

    // Status and MAVLINK messages contain are json encoded
  case OP_MAVLINK_TEXT:
    fallthrough
  case OP_STATUS:
    payload, err = json.Marshal(data)
    if err != nil {
      return nil, err
    }
  }

  buf := bytes.NewBuffer(make([]byte, 0))

  // session
  {
    seg := strconv.Itoa(int(session)) // session
    _, err = buf.WriteString(seg)
  }

  log.Println("1:", buf.Bytes())

  err = buf.WriteByte(byte(opCode))

  log.Println("2:", buf.Bytes())

  // payload len
  {
    seg := make([]byte, 2)
    binary.LittleEndian.PutUint16(seg, uint16(len(payload)))
    _, err = buf.Write(seg)
  }

  log.Println("3:", buf.Bytes())

  // payload data
  _, err = buf.Write(payload)

  log.Println("4:", buf.Bytes())

  // crc
  {
    seg := make([]byte, 2)
    binary.LittleEndian.PutUint16(seg, crc16.Crc16(buf.Bytes()))
    _, err = buf.Write(seg)
  }

  log.Println("5:", buf.Bytes())

  return buf, err
}
