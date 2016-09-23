package fmulink

import (
  "math"
  "net"
  "regexp"
  "sync"
  "time"
  "io"
  // "os"
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

  // New versions of PX4 dropped the `rc.usb` script
  MAVLINK_EXEC_STRING = "mavlink start -r 800000 -d /dev/ttyACM0\r\n"
)

var (
  status         Status
  fmu            Fmu

  Params         map[string]interface{}
  Managers       map[int]*MsgManager
  Outputs        *OutputManager = NewOutputManager()
  Saver          *FlightSaver

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

  // Motors
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
  Generic           map[string]*mavlink.Packet
  CloudOnline       string

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
      config.Log(config.LOG_ERROR, "fl: ", "LINK LOST. Is the master link alive?")

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

    if *config.Remote != "" {
      udpAddr, err := net.ResolveUDPAddr("udp", *addr)
      if err != nil {
        config.Log(config.LOG_ERROR, err)
        panic(err)
      }
      sudpAddr, err2 := net.ResolveUDPAddr("udp", *config.Remote)
      if err != nil {
        config.Log(config.LOG_ERROR, err2)
        panic(err)
      }

      conn, listenerr := net.DialUDP("udp", udpAddr, sudpAddr)
      if listenerr != nil {
        config.Log(config.LOG_ERROR, listenerr)
        panic(listenerr)
      }

      mavConn = conn
      config.Log(config.LOG_INFO, "[REMOTE] ", "Listening on", udpAddr)
    } else {
      udpAddr, err := net.ResolveUDPAddr("udp", *addr)
      if err != nil {
        panic(err)
      }

      conn, listenerr := net.ListenUDP("udp", udpAddr)
      if listenerr != nil {
        panic(listenerr)
      }

      mavConn = conn
      config.Log(config.LOG_INFO, "fl: ", "Listening on", udpAddr)
    }


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
      config.Log(config.LOG_INFO, "fl: ", "Listening on", *addr)
    }
  }

  // create outputs from command line. Max of 20 may be init at once.
  outs := regexp.MustCompile(`,`).Split(*out, 20)

  for i := range outs {
    if outs[i] != "" {
      if err := Outputs.Add(outs[i]); err != nil {
        config.Log(config.LOG_ERROR, "fl: ", err)
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
    Generic: make(map[string]*mavlink.Packet),
    CloudOnline: FMUSTATUS_DOWN,
  }

  Params :=     make(map[string]interface{})
  Managers :=   make(map[int]MsgManager)
  // Telem :=      make(map[string]mavlink.Message)
  Saver = NewFlightSaver(*config.FlightLogPath)

  {
    hbmm := NewMsgManager(time.Second * 2)
    hbmm.OnDown = func() {
      config.Log(config.LOG_ERROR, "fl: ", "Link Down")
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
          // if _, ok := err.(io.Reader); !ok {
          //   panic(err)
          // }
    			config.Log(config.LOG_DEBUG, "fl: ", "Decode fail:", err)
    		} else {

          // get byte array
          bin := unrollPacket(pkt)
          // Echo to outputs
          Outputs.Send(bin)

          // Log Data (if in log mode)
          if !*config.DisableFlights && Saver.IsLogging() {
            if err := Saver.Persist(bin, pkt.MsgID); err != nil {
              config.Log(config.LOG_ERROR, "fmu: ", err)
            }
          }

          // Update cloud
          go cl.UpdateFromFMU(*bin)

          // Get update from cloud (if any)
          go func() {
            packet := cl.GetFmuCmd()
            if packet.Command != 0 {
              config.Log(config.LOG_INFO, "fl:  Sending command to FMU...")

              enc.Encode(255, 1, &packet)
              // sendArmed(enc)
              cl.NullFmuCmd()
            }
          }()

          // Update FMU struct
          fmu.Meta.mut.Lock()
          fmu.mut.Lock()

          // XXX we'll consider this an `update` and check to see if the cloud is up.
          // Probably better to decouple this, but for now it's ok.

          {
            b := cl.IsOnlineNonBlock()
            if b {
              fmu.CloudOnline = FMUSTATUS_GOOD
            } else {
              fmu.CloudOnline = FMUSTATUS_DOWN
            }
          }

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

              // if !gotCaps {
              //   getCaps(enc)
              // }

              mm := Managers[int(pkt.MsgID)]

              if fmu.Meta.Link == FMUSTATUS_DOWN || fmu.Meta.Link == FMUSTATUS_UNKNOWN {
                config.Log(config.LOG_INFO, "fl: ", "Link Established.")
                config.Log(config.LOG_INFO, "fl: ", "\tType:", pv.Type)
                config.Log(config.LOG_INFO, "fl: ", "\tAutopilot:", pv.Autopilot)
                config.Log(config.LOG_INFO, "fl: ", "\tPrimary Mode:", pv.BaseMode)
                config.Log(config.LOG_INFO, "fl: ", "\tSecondary Mode:", pv.CustomMode)
                config.Log(config.LOG_INFO, "fl: ", "\tSystem Status:", pv.SystemStatus)
                config.Log(config.LOG_INFO, "fl: ", "\tVersion:", pv.MavlinkVersion)
              }

              // Use Acro/Manual as our trigger for testing.
              // if pv.BaseMode & 16 == 16 && !Saver.IsLogging() {
              //   config.Log(config.LOG_INFO, "fl: Event Trigger: Start logging.")
              //   Saver.Start()
              //   cl.SendSyncLock(Saver.Name())
              // } else if pv.BaseMode & 16 == 0 && Saver.IsLogging() {
              //   config.Log(config.LOG_INFO, "fl: Event Trigger: Stop logging.")
              //   Saver.End()
              //   cl.SendSyncUnlock()
              // }

              if !*config.DisableFlights {
                if pv.BaseMode & 128 == 128 && !Saver.IsLogging() {
                  config.Log(config.LOG_INFO, "fl: Event Trigger: Start logging.")
                  Saver.Start()
                  cl.SendSyncLock(Saver.Name())
                } else if pv.BaseMode & 128 == 0 && Saver.IsLogging() {
                  config.Log(config.LOG_INFO, "fl: Event Trigger: Stop logging.")
                  Saver.End()
                  cl.SendSyncUnlock()
                }
              }

              fmu.Meta.Link = FMUSTATUS_GOOD

              mm.Update()
            }

          // case mavlink.MSG_ID_MISSION_CURRENT:
            // got a mission current message
            // TODO

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

              // Golang JSON cannot parse NaNs, so we'll make these 0.
              // So far this issue is only encountered in these altitude messages,
              // but if it persists, we'll add a formal prune method.
              if math.IsNaN(float64(pv.AltitudeAmsl)) {
                pv.AltitudeAmsl = 0.0
              }

              if math.IsNaN(float64(pv.AltitudeMonotonic)) {
                pv.AltitudeMonotonic = 0.0
              }

              if math.IsNaN(float64(pv.AltitudeLocal)) {
                pv.AltitudeLocal = 0.0
              }

              if math.IsNaN(float64(pv.AltitudeRelative)) {
                pv.AltitudeRelative = 0.0
              }

              if math.IsNaN(float64(pv.AltitudeTerrain)) {
                pv.AltitudeTerrain = 0.0
              }

              if math.IsNaN(float64(pv.BottomClearance)) {
                pv.BottomClearance = 0.0
              }

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

            // SITL mode TODO
            if pkt.MsgID != 31 && pkt.MsgID != 85 && pkt.MsgID != 231 && pkt.MsgID != 242 && pkt.MsgID != 241 {
              // config.Log(config.LOG_DEBUG, "fl: ", "Unknown MSG:", pkt.MsgID)
            }
            fmu.Generic[strconv.Itoa(int(pkt.MsgID))] = pkt
          }
          fmu.Meta.mut.Unlock()
          fmu.mut.Unlock()
        }
      // }
    }
  }()

  // See if our link sending MAVLink or in the shell.
  checkShell(mavConn)
}

func printStatus(pvp *mavlink.SysStatus) {
  pv := *pvp

  config.Log(config.LOG_INFO, "fl: ", "Status.")
  config.Log(config.LOG_INFO, "fl: ", "\tSensors Present:", pv.OnboardControlSensorsPresent)
  config.Log(config.LOG_INFO, "fl: ", "\tSensors Enabled:", pv.OnboardControlSensorsEnabled)
  config.Log(config.LOG_INFO, "fl: ", "\tSensors Health:", pv.OnboardControlSensorsHealth)
  config.Log(config.LOG_INFO, "fl: ", "\tLoad:", pv.Load)
  config.Log(config.LOG_INFO, "fl: ", "\tVolt Bat:", pv.VoltageBattery)
  config.Log(config.LOG_INFO, "fl: ", "\tCurr Bat:", pv.CurrentBattery)
  config.Log(config.LOG_INFO, "fl: ", "\tDropRateComm:", pv.DropRateComm)
  config.Log(config.LOG_INFO, "fl: ", "\tBattery Remaining:", pv.BatteryRemaining)
  config.Log(config.LOG_INFO, "fl: ", "\tErrorsComm:", pv.ErrorsComm)
  config.Log(config.LOG_INFO, "fl: ", "\tErrorsCount1", pv.ErrorsCount1)
  config.Log(config.LOG_INFO, "fl: ", "\tErrorsCount2", pv.ErrorsCount2)
  config.Log(config.LOG_INFO, "fl: ", "\tErrorsCount3", pv.ErrorsCount3)
  config.Log(config.LOG_INFO, "fl: ", "\tErrorsCount4", pv.ErrorsCount4)
}

func handleStatusText(pvp *mavlink.Statustext) {
  pv := *pvp
  text := string(pv.Text[:len(pv.Text)])

  switch pv.Severity {
  case mavlink.MAV_SEVERITY_EMERGENCY:
    config.Log(config.LOG_INFO, "fl: ", "!! SEVERE !! EMERGENCY !! SEVERE !!")
    config.Log(config.LOG_INFO, "fl: ", text)
  case mavlink.MAV_SEVERITY_ALERT:
    config.Log(config.LOG_INFO, "fl: ", "WARNING | Noncritical Systems Failure")
    config.Log(config.LOG_INFO, "fl: ", text)
  case mavlink.MAV_SEVERITY_CRITICAL:
    config.Log(config.LOG_INFO, "fl: ", "IMPORTANT |", text)
  case mavlink.MAV_SEVERITY_ERROR:
    config.Log(config.LOG_INFO, "fl: ", "WARNING | Systems Failure")
    config.Log(config.LOG_INFO, "fl: ", text)
  case mavlink.MAV_SEVERITY_WARNING:
    config.Log(config.LOG_INFO, "fl: ", "WARNING |", text)
  case mavlink.MAV_SEVERITY_NOTICE:
    config.Log(config.LOG_INFO, "fl: ", "Huh? |", text)
  case mavlink.MAV_SEVERITY_INFO:
    config.Log(config.LOG_INFO, "fl: ", "FMU:", text)
  case mavlink.MAV_SEVERITY_DEBUG:
    config.Log(config.LOG_INFO, "fl: ", "FMU (DEVELOPMENT):", text)
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

  config.Log(config.LOG_DEBUG, "fl: ", "Getting capabilities")
  conn.Encode(1, 1, capCmd)
}

func sendArmed(conn *mavlink.Encoder) {
  capCmd := &mavlink.CommandLong{
    TargetSystem: 1, TargetComponent: 1,
    Command: 400, Param1: 1.0,
  }

  config.Log(config.LOG_DEBUG, "fl: ", "Arming...")
  conn.Encode(255, 1, capCmd)
}

func sendDisarmed(conn *mavlink.Encoder) {
  capCmd := &mavlink.CommandLong{
    TargetSystem: 1, TargetComponent: 1,
    Command: 400, Param1: 0,
  }

  config.Log(config.LOG_DEBUG, "fl: ", "Disarming...")
  conn.Encode(255, 1, capCmd)
}

func checkShell(conn io.ReadWriter) {
  b := make([]byte, 263)
  // gotReply := false

  // go func() {
  //   for {
  //     <-time.After(5 * time.Second)
  //     if !gotReply {
  //       config.Log(config.LOG_ERROR, "fl: ", "Got no response after 5 seconds. Link is probably in SHELL mode.")
  //       conn.Write([]byte("reboot\r\n"))
  //       os.Exit(0)
  //     } else {
  //       return
  //     }
  //   }
  // }()

  for {
    if n, _ := conn.Read(b); n > 0 {
      // gotReply = true
      if strings.Contains(string(b[:n]), "\r\nnsh>") {
        config.Log(config.LOG_INFO, "fl: ", "Link is in SHELL Mode")
        conn.Write([]byte(MAVLINK_EXEC_STRING))
      } else if strings.Contains(string(b[:n]), "\xFE") {
        config.Log(config.LOG_INFO, "fl: ", "Link is in MAVLINK Mode")
      }

      return
    }
  }
}
