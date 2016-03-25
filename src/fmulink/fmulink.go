package fmulink

import (
  "log"
  "net"
  "sync"
  "time"

  "mavlink/parser"
)

const (
  FMUSTATUS_UNKNOWN = "unknown"
  FMUSTATUS_DOWN = "down"
  FMUSTATUS_GOOD = "good"
  FMUSTATUS_ERROR = "error"
)

var (
  status         Status
  fmu            Fmu

  Params         map[string]interface{}
  Managers       map[int]*MsgManager
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

  mut           sync.RWMutex
}

func GetStatus() *Status {
  status.mut.RLock()
  defer status.mut.RUnlock()
  return &status
}

func GetData() *Fmu {
  fmu.mut.RLock()
  defer fmu.mut.RUnlock()
  return &fmu
}

type Fmu struct {
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

  mut               sync.RWMutex
}

type MsgManager struct {
  OnDown  func() // Setting this directly is not thread safe!

  timer   *time.Ticker
  mut     sync.RWMutex
  quit    chan bool
  stamp   chan time.Time
}

func NewMsgManager(interval time.Duration) *MsgManager {
  mm := MsgManager{
    quit: make(chan bool),
    stamp: make(chan time.Time),
    OnDown: func() {}, // Does nothing.
  }

  mm.Sched(interval)

  return &mm
}

func (mm *MsgManager) Update() {
  mm.stamp <- time.Now()
}

func (mm *MsgManager) Sched(interval time.Duration) {
  mm.mut.Lock()
  mm.timer = time.NewTicker(interval)
  mm.mut.Unlock()

  lastPrev := time.Now()

  go func() {
    for {
      select {
      case c := <-mm.timer.C:
        dt := mm.getDt(c, lastPrev)
        if dt > uint64(interval / time.Millisecond) {
          mm.OnDown()
        }

      case prev := <- mm.stamp:
        lastPrev = prev

      case <- mm.quit:
        return
      }
    }
  }()
}

func (mm *MsgManager) Stop() {
  mm.quit <- true
}

func (mm *MsgManager) getDt(curr, prev time.Time) uint64 {
  mm.mut.RLock()
  defer mm.mut.RUnlock()
  return uint64((curr.UnixNano() - prev.UnixNano()) / 1000000)
}


func Serve(addr string) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		panic(err)
	}

	conn, listenerr := net.ListenUDP("udp", udpAddr)
	if listenerr != nil {
		panic(listenerr)
	}

	log.Println("Listening on", udpAddr)

	dec := mavlink.NewDecoder(conn)

  status = Status{
    Link: FMUSTATUS_UNKNOWN,
  }

  fmu = Fmu{
  }

  Params :=     make(map[string]interface{})
  Managers :=   make(map[int]MsgManager)

  {
    hbmm := NewMsgManager(time.Second * 2)
    hbmm.OnDown = func() {
      log.Println("Link Down")
      status.Link = FMUSTATUS_DOWN
    }
    Managers[mavlink.MSG_ID_HEARTBEAT] = *hbmm

    vfrmm := NewMsgManager(time.Second)
    vfrmm.OnDown = func() { status.FlightData = FMUSTATUS_DOWN }
    Managers[mavlink.MSG_ID_VFR_HUD] = *vfrmm

    attCtrlmm := NewMsgManager(time.Second)
    attCtrlmm.OnDown = func() { status.AttCtrl = FMUSTATUS_DOWN }
    Managers[mavlink.MSG_ID_ATTITUDE_TARGET] = *attCtrlmm

    attEstmm := NewMsgManager(time.Second)
    attEstmm.OnDown = func() { status.AttEst = FMUSTATUS_DOWN }
    Managers[mavlink.MSG_ID_ATTITUDE] = *attEstmm

    batStatusmm := NewMsgManager(time.Second)
    batStatusmm.OnDown = func() { status.Power = FMUSTATUS_DOWN }
    Managers[mavlink.MSG_ID_BATTERY_STATUS] = *batStatusmm

    imumm := NewMsgManager(time.Second)
    imumm.OnDown = func() { status.Sensors = FMUSTATUS_DOWN }
    Managers[mavlink.MSG_ID_HIGHRES_IMU] = *imumm

    rcmm := NewMsgManager(time.Second)
    rcmm.OnDown = func() { status.RC = FMUSTATUS_DOWN }
    Managers[mavlink.MSG_ID_RC_CHANNELS] = *rcmm

    localmm := NewMsgManager(time.Second)
    localmm.OnDown = func() { status.LocalPosEst = FMUSTATUS_DOWN }
    Managers[mavlink.MSG_ID_LOCAL_POSITION_NED] = *localmm

    globalEstmm := NewMsgManager(time.Second)
    localmm.OnDown = func() { status.GlobalPosEst = FMUSTATUS_DOWN }
    Managers[mavlink.MSG_ID_GLOBAL_POSITION_INT] = *globalEstmm

    globalPosmm := NewMsgManager(time.Second)
    localmm.OnDown = func() { status.GlobalPosCtrl = FMUSTATUS_DOWN }
    Managers[mavlink.MSG_ID_POSITION_TARGET_GLOBAL_INT] = *globalPosmm
  }

  go func() {
    for {
  		if pkt, err := dec.Decode(); err != nil {
  			log.Println("Decode fail:", err)
  		} else {
        status.mut.Lock()
        fmu.mut.Lock()
        switch pkt.MsgID {

          // Params
        case mavlink.MSG_ID_PARAM_VALUE:
          var pv mavlink.ParamValue
          if err := pv.Unpack(pkt); err == nil {
            Params[string(pv.ParamId[:len(pv.ParamId)])] = pv.ParamValue
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
            status.FlightData = FMUSTATUS_GOOD
            mm.Update()
          }

          // Attitude Controller
        case mavlink.MSG_ID_ATTITUDE_TARGET:
          var pv mavlink.AttitudeTarget
          if err := pv.Unpack(pkt); err == nil {
            fmu.AttCtrl = pv
            mm := Managers[int(pkt.MsgID)]
            status.AttCtrl = FMUSTATUS_GOOD
            mm.Update()
          }

          // Attitude Estimator
        case mavlink.MSG_ID_ATTITUDE:
          var pv mavlink.Attitude
          if err := pv.Unpack(pkt); err == nil {
            fmu.AttEst = pv
            mm := Managers[int(pkt.MsgID)]
            status.AttEst = FMUSTATUS_GOOD
            mm.Update()
          }

          // Global Position
        case mavlink.MSG_ID_GLOBAL_POSITION_INT:
          var pv mavlink.GlobalPositionInt
          if err := pv.Unpack(pkt); err == nil {
            fmu.GlobalPos = pv
            mm := Managers[int(pkt.MsgID)]
            status.GlobalPosEst = FMUSTATUS_GOOD
            mm.Update()
          }

          // Local Position
        case mavlink.MSG_ID_LOCAL_POSITION_NED:
          var pv mavlink.LocalPositionNed
          if err := pv.Unpack(pkt); err == nil {
            fmu.LocalPos = pv
            mm := Managers[int(pkt.MsgID)]
            status.LocalPosEst = FMUSTATUS_GOOD
            mm.Update()
          }

          // Global Position Target
        case mavlink.MSG_ID_POSITION_TARGET_GLOBAL_INT:
          var pv mavlink.PositionTargetGlobalInt
          if err := pv.Unpack(pkt); err == nil {
            fmu.GlobalPosTarget = pv
            mm := Managers[int(pkt.MsgID)]
            status.GlobalPosCtrl = FMUSTATUS_GOOD
            mm.Update()
          }

          // Gps data
        case mavlink.MSG_ID_GPS_RAW_INT:
          var pv mavlink.GpsRawInt
          if err := pv.Unpack(pkt); err == nil {
            fmu.Gps = pv
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
            status.Sensors = FMUSTATUS_GOOD
            mm.Update()
          }

          // Battery
        case mavlink.MSG_ID_BATTERY_STATUS:
          var pv mavlink.BatteryStatus
          if err := pv.Unpack(pkt); err == nil {
            fmu.Battery = pv
            mm := Managers[int(pkt.MsgID)]
            status.Power = FMUSTATUS_GOOD
            mm.Update()
          }

          // RC Values
        case mavlink.MSG_ID_RC_CHANNELS:
          var pv mavlink.RcChannels
          if err := pv.Unpack(pkt); err == nil {
            fmu.RcValues = pv
            mm := Managers[int(pkt.MsgID)]
            status.RC = FMUSTATUS_GOOD
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
            mm := Managers[int(pkt.MsgID)]

            if status.Link == FMUSTATUS_DOWN || status.Link == FMUSTATUS_UNKNOWN {
              log.Println("Link Established.")
              log.Println("\tType:", pv.Type)
              log.Println("\tAutopilot:", pv.Autopilot)
              log.Println("\tPrimary Mode:", pv.BaseMode)
              log.Println("\tSecondary Mode:", pv.CustomMode)
              log.Println("\tSystem Status:", pv.SystemStatus)
              log.Println("\tVersion:", pv.MavlinkVersion)
            }

            status.Link = FMUSTATUS_GOOD

            mm.Update()
          }

          // System Status
        case mavlink.MSG_ID_SYS_STATUS:
          var pv mavlink.SysStatus
          if err := pv.Unpack(pkt); err == nil {
            fmu.Sys = pv
          }

        default:
          log.Println("Unknown MSG:", pkt.MsgID)
        }
        status.mut.Unlock()
        fmu.mut.Unlock()
      }
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
