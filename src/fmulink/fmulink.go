package fmulink

import (
  "log"
  "net"
  "regexp"
  "sync"
  "time"
  "io"
  "strconv"
  "strings"

  "mavlink/parser"
  "fmulink/serial"

  "cloudlink"
  "config"
)

const (
  FMUSTATUS_UNKNOWN = "unknown"
  FMUSTATUS_DOWN = "offline"
  FMUSTATUS_GOOD = "online"
  FMUSTATUS_ERROR = "error"

  UDP_REGEX = `^(((\d{1,3}\.){3}\d)|localhost):\d{1,5}$`
  DEFAULT_BAUD = 57600

  MAVLINK_EXEC_STRING = "sh /etc/init.d/rc.usb\r\n"
)

var (
  status         Status
  fmu            Fmu

  Params         map[string]interface{}
  Managers       map[int]*MsgManager
  Outputs        *OutputManager = NewOutputManager()

  // Telem         map[string]mavlink.Message

  AutopilotCaps  *mavlink.AutopilotVersion
)

type Status struct {
  // Link
  Link        string

  // Flight Data
  FlightData  string

  // Attitude Controller
  AttCtrl     string

  // Attitude Estimator
  AttEst      string

  // Power Monitor
  Power       string

  // Sensors
  Sensors     string

  // Radio Control
  RC          string

  // Local Position Estimator
  LocalPosEst string

  // Global Position Estimator
  GlobalPosEst string

  // Position Controller
  GlobalPosCtrl string

  // GPS
  Gps           string

  // Motoros
  Servos        string

  // Motor Target
  Actuators     string

  // Altitude
  Altitude      string

  mut           sync.RWMutex
}

// func GetStatus() *Status {
//   status.mut.RLock()
//   defer status.mut.RUnlock()
//   return &status
// }

func GetData() *Fmu {
  fmu.mut.RLock()
  defer fmu.mut.RUnlock()
  return &fmu
}

type Fmu struct {
  Meta              Status

  Hb                mavlink.Heartbeat
  Sys               mavlink.SysStatus

  AttEst            mavlink.Attitude
  AttCtrl           mavlink.AttitudeTarget
  Vfr               mavlink.VfrHud
  GlobalPos         mavlink.GlobalPositionInt
  LocalPos          mavlink.LocalPositionNed
  GlobalPosTarget   mavlink.PositionTargetGlobalInt
  Gps               mavlink.GpsRawInt
  GpsGlobalOrigin   mavlink.GpsGlobalOrigin
  Imu               mavlink.HighresImu
  Battery           mavlink.BatteryStatus
  RcValues          mavlink.RcChannels
  RcStatus          mavlink.RadioStatus
  Servos            mavlink.ServoOutputRaw
  Actuators         mavlink.ActuatorControlTarget
  Altitude          mavlink.Altitude
  ExSys             mavlink.ExtendedSysState

  mut               sync.RWMutex
}

func Serve(cl *cloudlink.CloudLink) {
  defer func() {
    if r := recover(); r != nil {
      log.Println("!! CRITICAL !! Is the master link alive?")

      <- time.After(5 * time.Second)
      Serve(cl)
    }
  }()

  var mavConn io.ReadWriter

  addr := config.LinkPath
  out := config.Output

  if matched, err := regexp.MatchString(UDP_REGEX, *addr); err != nil {
    panic(err)
  } else if matched {
    udpAddr, err := net.ResolveUDPAddr("udp", *addr)
    if err != nil {
      panic(err)
    }

    conn, listenerr := net.ListenUDP("udp", udpAddr)
    if listenerr != nil {
      panic(listenerr)
    }

    mavConn = conn
    log.Println("Listening on", udpAddr)

  } else {

    /*
    Example formats

    Windows:
      COM43:115200

    Linux:
      /dev/ttyMFD1:115200

    OSX:
      /dev/tty.usbserial:115200
    */

    cfg := regexp.MustCompile(`:`).Split(*addr, 2)
    var baud int

    // assume a baudrate if none provided
    if len(cfg) < 2 {
      baud = DEFAULT_BAUD
    } else {
      var err error
      baud, err = strconv.Atoi(cfg[1])
      if err != nil {
        baud = DEFAULT_BAUD
      }
    }

    if conn, err := serial.OpenPort(&serial.Config{Name: cfg[0], Baud: baud}); err != nil {
      panic(err)
    } else {
      mavConn = conn
      log.Println("Listening on", *addr)
    }
  }

  // See if our link sending MAVLink or in the shell.
  checkShell(mavConn)

  // create outputs from command line. Max of 20 may be init at once.
  outs := regexp.MustCompile(`,`).Split(*out, 20)

  for i := range outs {
    if outs[i] != "" {
      if err := Outputs.Add(outs[i]); err != nil {
        log.Println(err)
      }
    }
  }

  enc := mavlink.NewEncoder(mavConn)
	dec := mavlink.NewDecoder(mavConn)

  gotCaps := false

  status = Status{
    Link: FMUSTATUS_UNKNOWN,
  }

  fmu = Fmu{
  }

  Params :=     make(map[string]interface{})
  Managers :=   make(map[int]MsgManager)
  // Telem :=      make(map[string]mavlink.Message)

  {
    hbmm := NewMsgManager(time.Second * 2)
    hbmm.OnDown = func() {
      log.Println("Link Down")
      fmu.Meta.Link = FMUSTATUS_DOWN
    }
    Managers[mavlink.MSG_ID_HEARTBEAT] = *hbmm

    vfrmm := NewMsgManager(time.Second)
    vfrmm.OnDown = func() { fmu.Meta.FlightData = FMUSTATUS_DOWN }
    Managers[mavlink.MSG_ID_VFR_HUD] = *vfrmm

    attCtrlmm := NewMsgManager(time.Second)
    attCtrlmm.OnDown = func() { fmu.Meta.AttCtrl = FMUSTATUS_DOWN }
    Managers[mavlink.MSG_ID_ATTITUDE_TARGET] = *attCtrlmm

    attEstmm := NewMsgManager(time.Second)
    attEstmm.OnDown = func() { fmu.Meta.AttEst = FMUSTATUS_DOWN }
    Managers[mavlink.MSG_ID_ATTITUDE] = *attEstmm

    batStatusmm := NewMsgManager(time.Second * 4)
    batStatusmm.OnDown = func() { fmu.Meta.Power = FMUSTATUS_DOWN }
    Managers[mavlink.MSG_ID_BATTERY_STATUS] = *batStatusmm

    imumm := NewMsgManager(time.Second)
    imumm.OnDown = func() { fmu.Meta.Sensors = FMUSTATUS_DOWN }
    Managers[mavlink.MSG_ID_HIGHRES_IMU] = *imumm

    rcmm := NewMsgManager(time.Second)
    rcmm.OnDown = func() { fmu.Meta.RC = FMUSTATUS_DOWN }
    Managers[mavlink.MSG_ID_RC_CHANNELS] = *rcmm

    localmm := NewMsgManager(time.Second)
    localmm.OnDown = func() { fmu.Meta.LocalPosEst = FMUSTATUS_DOWN }
    Managers[mavlink.MSG_ID_LOCAL_POSITION_NED] = *localmm

    globalEstmm := NewMsgManager(time.Second)
    globalEstmm.OnDown = func() { fmu.Meta.GlobalPosEst = FMUSTATUS_DOWN }
    Managers[mavlink.MSG_ID_GLOBAL_POSITION_INT] = *globalEstmm

    globalPosmm := NewMsgManager(time.Second)
    globalPosmm.OnDown = func() { fmu.Meta.GlobalPosCtrl = FMUSTATUS_DOWN }
    Managers[mavlink.MSG_ID_POSITION_TARGET_GLOBAL_INT] = *globalPosmm

    gpsmm := NewMsgManager(time.Second * 2)
    gpsmm.OnDown = func() { fmu.Meta.Gps = FMUSTATUS_DOWN }
    Managers[mavlink.MSG_ID_GPS_RAW_INT] = *gpsmm

    altmm := NewMsgManager(time.Second * 2)
    altmm.OnDown = func() { fmu.Meta.Altitude = FMUSTATUS_DOWN }
    Managers[mavlink.MSG_ID_ALTITUDE] = *altmm

    servomm := NewMsgManager(time.Second)
    servomm.OnDown = func() { fmu.Meta.Servos = FMUSTATUS_DOWN }
    Managers[mavlink.MSG_ID_SERVO_OUTPUT_RAW] = *servomm

    actmm := NewMsgManager(time.Second)
    actmm.OnDown = func() { fmu.Meta.Actuators = FMUSTATUS_DOWN }
    Managers[mavlink.MSG_ID_ACTUATOR_CONTROL_TARGET] = *actmm
  }

  // listen for inputs
  go func() {
    for {
      b := <- Outputs.Input
      mavConn.Write(b)
    }
  }()

  // handle outputs
  go func() {
    for {
      // inBuf, num := getPacket(mavConn)
      // num, _ := mavConn.Read(inBuf)
      // if num > 0 {

        // Add a small delay to give the handler some down time.
        // XXX if this causes latency issue, consider going to a higher resolution.
        // Probably not a good idea to remove all together, as it can cause mavlink read issues.
        time.Sleep(50 * time.Microsecond)

        // log.Println(inBuf[:num])
    		if pkt, err := dec.Decode(); err != nil {
    			log.Println("Decode fail:", err)
    		} else {

          // get byte array
          bin := unrollPacket(pkt)
          // Echo to outputs
          Outputs.Send(bin)

          // Update cloud
          go cl.UpdateFromFMU(*bin)

          // {
          //   var pv mavlink.Message
          //   if err := pv.Unpack(pkt); err == nil {
          //     Telem[pv.MsgName()] = pv
          //   }
          //
          //   log.Println(Telem)
          // }

          // Update FMU struct
          fmu.Meta.mut.Lock()
          fmu.mut.Lock()
          switch pkt.MsgID {

            // Params
          case mavlink.MSG_ID_PARAM_VALUE:
            var pv mavlink.ParamValue
            if err := pv.Unpack(pkt); err == nil {
              Params[string(pv.ParamId[:len(pv.ParamId)])] = pv.ParamValue
            }

          case mavlink.MSG_ID_AUTOPILOT_VERSION:
            var pv mavlink.AutopilotVersion
            if err := pv.Unpack(pkt); err == nil {
              AutopilotCaps = &pv
              gotCaps = true
              cl.UpdateSerialId(pv.Uid)
            }

            // Status Text
          case mavlink.MSG_ID_STATUSTEXT:
            var pv mavlink.Statustext
            if err := pv.Unpack(pkt); err == nil {
              handleStatusText(&pv)
            }

            // VFR
          case mavlink.MSG_ID_VFR_HUD:
            var pv mavlink.VfrHud
            if err := pv.Unpack(pkt); err == nil {
              fmu.Vfr = pv
              mm := Managers[int(pkt.MsgID)]
              fmu.Meta.FlightData = FMUSTATUS_GOOD
              mm.Update()
            }

            // Attitude Controller
          case mavlink.MSG_ID_ATTITUDE_TARGET:
            var pv mavlink.AttitudeTarget
            if err := pv.Unpack(pkt); err == nil {
              fmu.AttCtrl = pv
              mm := Managers[int(pkt.MsgID)]
              fmu.Meta.AttCtrl = FMUSTATUS_GOOD
              mm.Update()
            }

            // Attitude Estimator
          case mavlink.MSG_ID_ATTITUDE:
            var pv mavlink.Attitude
            if err := pv.Unpack(pkt); err == nil {
              fmu.AttEst = pv
              mm := Managers[int(pkt.MsgID)]
              fmu.Meta.AttEst = FMUSTATUS_GOOD
              mm.Update()
            }

            // Global Position
          case mavlink.MSG_ID_GLOBAL_POSITION_INT:
            var pv mavlink.GlobalPositionInt
            if err := pv.Unpack(pkt); err == nil {
              fmu.GlobalPos = pv
              mm := Managers[int(pkt.MsgID)]
              fmu.Meta.GlobalPosEst = FMUSTATUS_GOOD
              mm.Update()
            }

            // Local Position
          case mavlink.MSG_ID_LOCAL_POSITION_NED:
            var pv mavlink.LocalPositionNed
            if err := pv.Unpack(pkt); err == nil {
              fmu.LocalPos = pv
              mm := Managers[int(pkt.MsgID)]
              fmu.Meta.LocalPosEst = FMUSTATUS_GOOD
              mm.Update()
            }

            // Global Position Target
          case mavlink.MSG_ID_POSITION_TARGET_GLOBAL_INT:
            var pv mavlink.PositionTargetGlobalInt
            if err := pv.Unpack(pkt); err == nil {
              fmu.GlobalPosTarget = pv
              mm := Managers[int(pkt.MsgID)]
              fmu.Meta.GlobalPosCtrl = FMUSTATUS_GOOD
              mm.Update()
            }

            // Gps data
          case mavlink.MSG_ID_GPS_RAW_INT:
            var pv mavlink.GpsRawInt
            if err := pv.Unpack(pkt); err == nil {
              fmu.Gps = pv
              mm := Managers[int(pkt.MsgID)]
              fmu.Meta.Gps = FMUSTATUS_GOOD
              mm.Update()
            }

            // Gps home
          case mavlink.MSG_ID_GPS_GLOBAL_ORIGIN:
            var pv mavlink.GpsGlobalOrigin
            if err := pv.Unpack(pkt); err == nil {
              fmu.GpsGlobalOrigin = pv
            }

            // Sensors
          case mavlink.MSG_ID_HIGHRES_IMU:
            var pv mavlink.HighresImu
            if err := pv.Unpack(pkt); err == nil {
              fmu.Imu = pv
              mm := Managers[int(pkt.MsgID)]
              fmu.Meta.Sensors = FMUSTATUS_GOOD
              mm.Update()
            }

            // Battery
          case mavlink.MSG_ID_BATTERY_STATUS:
            var pv mavlink.BatteryStatus
            if err := pv.Unpack(pkt); err == nil {
              fmu.Battery = pv
              mm := Managers[int(pkt.MsgID)]
              fmu.Meta.Power = FMUSTATUS_GOOD
              mm.Update()
            }

            // RC Values
          case mavlink.MSG_ID_RC_CHANNELS:
            var pv mavlink.RcChannels
            if err := pv.Unpack(pkt); err == nil {
              fmu.RcValues = pv
              mm := Managers[int(pkt.MsgID)]
              fmu.Meta.RC = FMUSTATUS_GOOD
              mm.Update()
            }

            // RC Status
          case mavlink.MSG_ID_RADIO_STATUS:
            var pv mavlink.RadioStatus
            if err := pv.Unpack(pkt); err == nil {
              fmu.RcStatus = pv
            }

            // Basic Connectivity
          case mavlink.MSG_ID_HEARTBEAT:
            var pv mavlink.Heartbeat
            if err := pv.Unpack(pkt); err == nil {
              fmu.Hb = pv

              if !gotCaps {
                getCaps(enc)
              }

              mm := Managers[int(pkt.MsgID)]

              if fmu.Meta.Link == FMUSTATUS_DOWN || fmu.Meta.Link == FMUSTATUS_UNKNOWN {
                log.Println("Link Established.")
                log.Println("\tType:", pv.Type)
                log.Println("\tAutopilot:", pv.Autopilot)
                log.Println("\tPrimary Mode:", pv.BaseMode)
                log.Println("\tSecondary Mode:", pv.CustomMode)
                log.Println("\tSystem Status:", pv.SystemStatus)
                log.Println("\tVersion:", pv.MavlinkVersion)
              }

              fmu.Meta.Link = FMUSTATUS_GOOD

              mm.Update()
            }

            // System Status
          case mavlink.MSG_ID_SYS_STATUS:
            var pv mavlink.SysStatus
            if err := pv.Unpack(pkt); err == nil {
              fmu.Sys = pv
            }

          case mavlink.MSG_ID_SERVO_OUTPUT_RAW:
            var pv mavlink.ServoOutputRaw
            if err := pv.Unpack(pkt); err == nil {
              fmu.Servos = pv
              mm := Managers[int(pkt.MsgID)]
              fmu.Meta.Servos = FMUSTATUS_GOOD
              mm.Update()
            }

          case mavlink.MSG_ID_ACTUATOR_CONTROL_TARGET:
            var pv mavlink.ActuatorControlTarget
            if err := pv.Unpack(pkt); err == nil {
              fmu.Actuators = pv
              mm := Managers[int(pkt.MsgID)]
              fmu.Meta.Actuators = FMUSTATUS_GOOD
              mm.Update()
            }

          case mavlink.MSG_ID_ALTITUDE:
            var pv mavlink.Altitude
            if err := pv.Unpack(pkt); err == nil {
              fmu.Altitude = pv
              mm := Managers[int(pkt.MsgID)]
              fmu.Meta.Altitude = FMUSTATUS_GOOD
              mm.Update()
            }

          case mavlink.MSG_ID_EXTENDED_SYS_STATE:
            var pv mavlink.ExtendedSysState
            if err := pv.Unpack(pkt); err == nil {
              fmu.ExSys = pv
            }


          default:
            log.Println("Unknown MSG:", pkt.MsgID)
          }
          fmu.Meta.mut.Unlock()
          fmu.mut.Unlock()
        }
      // }
    }
  }()
}

func printStatus(pvp *mavlink.SysStatus) {
  pv := *pvp

  log.Println("Status.")
  log.Println("\tSensors Present:", pv.OnboardControlSensorsPresent)
  log.Println("\tSensors Enabled:", pv.OnboardControlSensorsEnabled)
  log.Println("\tSensors Health:", pv.OnboardControlSensorsHealth)
  log.Println("\tLoad:", pv.Load)
  log.Println("\tVolt Bat:", pv.VoltageBattery)
  log.Println("\tCurr Bat:", pv.CurrentBattery)
  log.Println("\tDropRateComm:", pv.DropRateComm)
  log.Println("\tBattery Remaining:", pv.BatteryRemaining)
  log.Println("\tErrorsComm:", pv.ErrorsComm)
  log.Println("\tErrorsCount1", pv.ErrorsCount1)
  log.Println("\tErrorsCount2", pv.ErrorsCount2)
  log.Println("\tErrorsCount3", pv.ErrorsCount3)
  log.Println("\tErrorsCount4", pv.ErrorsCount4)
}

func handleStatusText(pvp *mavlink.Statustext) {
  pv := *pvp
  text := string(pv.Text[:len(pv.Text)])

  switch pv.Severity {
  case mavlink.MAV_SEVERITY_EMERGENCY:
    log.Println("!! SEVERE !! EMERGENCY !! SEVERE !!")
    log.Println(text)
  case mavlink.MAV_SEVERITY_ALERT:
    log.Println("WARNING | Noncritical Systems Failure")
    log.Println(text)
  case mavlink.MAV_SEVERITY_CRITICAL:
    log.Println("IMPORTANT |", text)
  case mavlink.MAV_SEVERITY_ERROR:
    log.Println("WARNING | Systems Failure")
    log.Println(text)
  case mavlink.MAV_SEVERITY_WARNING:
    log.Println("WARNING |", text)
  case mavlink.MAV_SEVERITY_NOTICE:
    log.Println("Huh? |", text)
  case mavlink.MAV_SEVERITY_INFO:
    log.Println("FMU:", text)
  case mavlink.MAV_SEVERITY_DEBUG:
    log.Println("FMU (DEVELOPMENT):", text)
  }
}

// Get a raw byte array from a mavlink packet
func unrollPacket(pkt *mavlink.Packet) *[]byte {
  plen := len(pkt.Payload)
  buf := make([]byte, plen+8)

  buf[0] = 0xFE // header
  buf[1] = byte(plen)
  buf[2] = byte(pkt.SeqID)
  buf[3] = byte(pkt.SysID)
  buf[4] = byte(pkt.CompID)
  buf[5] = byte(pkt.MsgID)

  for i := 0; i < plen; i++ {
    buf[i+6] = pkt.Payload[i]
  }

  buf[plen+6] = byte(pkt.Checksum & 0xFF)
  buf[plen+7] = byte(pkt.Checksum >> 8)

  return &buf
}

func getCaps(conn *mavlink.Encoder) {
  capCmd := &mavlink.CommandInt{
    TargetSystem: 1, TargetComponent: 1,
    Command: 520,
  }

  log.Println("Getting capabilities")
  conn.Encode(1, 1, capCmd)
}

// func decodeMsg(pkt *mavlink.Packet) *mavlink.Message {
//   var pv mavlink.Message
//
//   switch pkt.MsgID {
//   case mavlink.MSG_ID_HEARTBEAT:            pv = new(mavlink.Heartbeat)       // Basic UAV info.
//   case mavlink.MSG_ID_SYS_STATUS:           pv = new(mavlink.SysStatus)       // FMU attached peripherals.
//   case mavlink.MSG_ID_HIGHRES_IMU:          pv = new(mavlink.HighresImu)      // Sensor information
//   case mavlink.MSG_ID_ATTITUDE:             pv = new(mavlink.Attitude)        // Attitude estimator
//   case mavlink.MSG_ID_ATTITUDE_TARGET:      pv = new(mavlink.AttitudeTarget)  // Attitude controller
//   case mavlink.MSG_ID_VFR_HUD:              pv = new(mavlink.VfrHud)          // General flight info (mainly used for GCS display)
//   case mavlink.MSG_ID_GPS_RAW_INT:          pv = new(mavlink.GpsRawInt)       // GPS Sensor
//   case mavlink.MSG_ID_GLOBAL_POSITION_INT:
//
//     // Position Estimator (Local)
//   case mavlink.MSG_ID_LOCAL_POSITION_NED:
//
//     // Position Controller
//   case mavlink.MSG_ID_POSITION_TARGET_GLOBAL_INT:
//
//     // Position Estimator (Altitude)
//   case mavlink.MSG_ID_ALTITUDE:
//
//     // Distance Sensor
//   case mavlink.MSG_ID_DISTANCE_SENSOR:
//
//     // Optical Flow
//   case mavlink.MSG_ID_OPTICAL_FLOW_RAD:
//
//     // Home location
//   case mavlink.MSG_ID_HOME_POSITION:
//
//     // System state for vtol
//   case mavlink.MSG_ID_EXTENDED_SYS_STATE:
//
//     // Vision NED
//   case mavlink.MSG_ID_VISION_POSITION_ESTIMATE:
//
//     // Motor control
//   case mavlink.MSG_ID_ACTUATOR_CONTROL_TARGET:
//
//     // Motor output
//   case mavlink.MSG_ID_SERVO_OUTPUT_RAW:
//
//     // Radio Control
//   case mavlink.MSG_ID_RC_CHANNELS:
//
//     // Radio Status
//   case mavlink.MSG_ID_RADIO_STATUS:
//
//     // Battery Info
//   case mavlink.MSG_ID_BATTERY_STATUS:
//
//
//
//     // TODO mission messages.
//
//     // FTP stream. Special case when requesting data from file system.
//   // case mavlink.MSG_ID_FILE_TRANSFER_PROTOCOL
//   //
//   //   // Params. Special case when loading params from FMU.
//   // case mavlink.MSG_ID_PARAM_VALUE:
//   //
//   //   // Special case. Streamed when FMU is configured.
//   // case mavlink.MSG_ID_COMMAND_LONG:
//   //
//   //   // Special case. Should be added to a log buffer
//   // case mavlink.MSG_ID_STATUSTEXT:
//   }
// }


func checkShell(conn io.ReadWriter) {
  b := make([]byte, 263)
  gotReply := false

  go func() {
    for {
      <-time.After(5 * time.Second)
      if !gotReply {
        log.Println("Got no response after 5 seconds. Link is probably in SHELL mode.")
        conn.Write([]byte("reboot\r\n"))
      }
    }
  }()

  for {
    if n, _ := conn.Read(b); n > 0 {
      gotReply = true
      if strings.Contains(string(b[:n]), "\r\nnsh>") {
        log.Println("Link is in SHELL Mode")
        conn.Write([]byte(MAVLINK_EXEC_STRING))
      } else if strings.Contains(string(b[:n]), "\xFE") {
        log.Println("Link is in MAVLINK Mode")
      }

      return
    }
  }
}
