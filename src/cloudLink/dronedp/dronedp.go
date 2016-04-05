package dronedp

import (
  "bytes"
  "encoding/binary"
  "encoding/json"
  "strconv"
  "errors"

  "cloudlink/crc16"
  "mavlink/parser"
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

type Msg struct {
  Op      OP
  Session uint32
  Data    interface{}
}

// =============================================================================
// GenerateMsg
// =============================================================================
func GenerateMsg(opCode OP, session uint32, data interface{}) ([]byte, error) {
  var err error = nil
  var payload []byte

  switch opCode {
    // Binary encoded messages for storing flight data. TODO.
  case OP_MAVLINK_BIN:
    return nil, errors.New("D2P.Gen: OP_MAVLINK_BIN is unsupported.")

    // Status and MAVLINK messages contain are json encoded
  case OP_MAVLINK_TEXT:
    fallthrough
  case OP_STATUS:
    payload, err = json.Marshal(data)
    if err != nil {
      return nil, err
    }
  default:
    return nil, errors.New("D2P.Gen: Unknown Op code.")
  }

  buf := bytes.NewBuffer(make([]byte, 0))

  // session
  {
    seg := strconv.Itoa(int(session)) // session
    _, err = buf.WriteString(seg)
  }

  // Op
  err = buf.WriteByte(byte(opCode))

  // payload len
  {
    seg := make([]byte, 2)
    binary.BigEndian.PutUint16(seg, uint16(len(payload)))
    _, err = buf.Write(seg)
  }

  // payload data
  _, err = buf.Write(payload)

  // crc
  {
    seg := make([]byte, 2)
    binary.BigEndian.PutUint16(seg, crc16.Crc16(buf.Bytes()))
    _, err = buf.Write(seg)
  }

  return buf.Bytes(), err
}

// =============================================================================
// ParseMsg
// =============================================================================
func ParseMsg(data []byte) (*Msg, error) {
  var err error

  buf := bytes.NewReader(data)
  msg := &Msg{}

  // Check CRC
  dataSize := len(data)
  crcVal := binary.BigEndian.Uint16(data[dataSize-2:dataSize])
  if crc16.Crc16(data[:dataSize-2]) != crcVal {
    return nil, errors.New("D2P.Parse: CRC Error")
  }

  // Session
  slice := make([]byte, 4)
  _, err = buf.Read(slice)
  msg.Session = binary.BigEndian.Uint32(slice)

  // Op code
  var b byte
  b, err = buf.ReadByte()
  msg.Op = OP(b)

  // Length
  slice = make([]byte, 2)
  _, err = buf.ReadAt(slice, 5)
  length := binary.BigEndian.Uint16(slice)

  // Payload
  decoded := make([]byte, length)
  _, err = buf.ReadAt(decoded, 7)

  if err != nil {
    return nil, err
  }

  switch (msg.Op) {
  case OP_MAVLINK_BIN:
    return nil, errors.New("D2P.Parse: OP_MAVLINK_BIN is unsupported.")

  case OP_MAVLINK_TEXT:
    fallthrough
  case OP_STATUS:
    msg.Data = &mavlink.Packet{}
    err = json.Unmarshal(decoded, msg.Data)

  case OP_CODE:
    msg.Data = string(decoded[:])

  default:
    return nil, errors.New("D2P: Unknown Op code.")

  }

  return msg, err
}
