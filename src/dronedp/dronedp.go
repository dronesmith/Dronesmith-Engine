
import (
  "encoding/json"
  "fmt"
)

const (
  SECURE_KEY string = "d7 e6 af 0b 14 90 7e a5 0a fd e8 bb 57 4f 3d 99 81 88 d9 f5 1b 90 7d 3d 44 e7 94 e3 30 f0 55 d9"

  // Ops
  OP_STATUS = 0x10
  OP_CODE = 0x11
  OP_MAVLINK_TEXT = 0xFD
  OP_MAVLINK_BIN = 0xFE
)

var (
  UseEncryption = true
)



// func GenerateMsg(opCode, session, data []byte) error {
//   switch opCode {
//   case OP_MAVLINK_BIN:
//     return fmt.Errorf("DDP: OP_MAVLINK_BIN is unsupported.")
//   case OP_MAVLINK_TEXT:
//     fallthrough
//   case OP_STATUS:
//     json.Marshal(data)
//   }
// }
