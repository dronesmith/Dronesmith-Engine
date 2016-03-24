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
  status            Status
  fmu               Fmu

  Params            map[string]interface{}
  Managers          map[int]*MsgManager
)


type Status struct {
  // Packet Updates
  HbStatus          string
  mut               sync.RWMutex
}

type Fmu struct {
  Hb                mavlink.Heartbeat
  Sys               mavlink.SysStatus

  AttEst            mavlink.Attitude
  AttCtrl           mavlink.AttitudeTarget
  Vfr               mavlink.VfrHud

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

  log.SetPrefix("[FMULINK] ")
	log.Println("Listening on", udpAddr)

	dec := mavlink.NewDecoder(conn)

  status = Status{
    HbStatus: FMUSTATUS_UNKNOWN,
  }

  fmu = Fmu{
  }

  Params :=     make(map[string]interface{})
  Managers :=   make(map[int]MsgManager)

  hbmm := NewMsgManager(time.Second * 2)


  hbmm.OnDown = func() {
    log.Println("Link Down")
    status.HbStatus = FMUSTATUS_DOWN
  }
  Managers[mavlink.MSG_ID_HEARTBEAT] = *hbmm

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

          // Attitude Controller
        case mavlink.MSG_ID_ATTITUDE_TARGET:

          // Attitude Estimator
        case mavlink.MSG_ID_ATTITUDE:

        case mavlink.MSG_ID_GLOBAL_POSITION_INT:
        case mavlink.MSG_ID_LOCAL_POSITION_NED:
        case mavlink.MSG_ID_POSITION_TARGET_GLOBAL_INT:
        case mavlink.MSG_ID_GPS_RAW_INT:
        case mavlink.MSG_ID_GPS_GLOBAL_ORIGIN:
        case mavlink.MSG_ID_HIGHRES_IMU:
        case mavlink.MSG_ID_BATTERY_STATUS:
        case mavlink.MSG_ID_RC_CHANNELS:
        case mavlink.MSG_ID_RADIO_STATUS:

          // Basic Connectivity
        case mavlink.MSG_ID_HEARTBEAT:
          var pv mavlink.Heartbeat
          if err := pv.Unpack(pkt); err == nil {
            fmu.Hb = pv
            mm := Managers[int(pkt.MsgID)]

            if status.HbStatus == FMUSTATUS_DOWN || status.HbStatus == FMUSTATUS_UNKNOWN {
              log.Println("Link Established.")
              log.Println("\tType:", pv.Type)
              log.Println("\tAutopilot:", pv.Autopilot)
              log.Println("\tPrimary Mode:", pv.BaseMode)
              log.Println("\tSecondary Mode:", pv.CustomMode)
              log.Println("\tSystem Status:", pv.SystemStatus)
              log.Println("\tVersion:", pv.MavlinkVersion)
            }

            status.HbStatus = FMUSTATUS_GOOD

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

  // Run forever
  for {
    select {
    }
  }
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
