package cloudlink

import (
  "bytes"
  "config"
  "net"
  "time"
  "strings"
  "strconv"
  "math/rand"
  "sync"
  "net/http"
  "encoding/json"
  "fmt"

  "mavlink/parser"

  "cloudlink/dronedp"
)

const (
	// Default address, if otherwise not specified by cli args. IP is the cloud,
	// 4002 is the dronedp listening port.
	// DEFAULT_DSC_ADDRESS = "24.234.109.135:4002"
  // DEFAULT_DSC_ADDRESS = "127.0.0.1:4002"

  TIME_OUT_CNT = 15
)

type CloudLink struct {
  addr        *net.UDPAddr
  conn        *net.UDPConn
  rx          []byte
  sessionId   uint32
  messageCnt  int
  codeStatus  int
  terminalOnline bool
  timer       *time.Timer

  uid         string

  termRunner  *TermLauncher
  codeRunner  *CodeLauncher
  syncer      *FlightSyncer
  store       *Store

  rawFmuCmd   []byte
  packmut     sync.RWMutex

  msgs        map[byte][]byte
  syncTimer   *time.Timer
  msgMut      sync.RWMutex

  noThrottleMsg chan []byte
}

func NewCloudLink() (*CloudLink, error) {
  var err error
  cl := &CloudLink{}

  cl.packmut = sync.RWMutex{}
  cl.msgMut = sync.RWMutex{}
  cl.noThrottleMsg = make(chan []byte)

  cl.msgs = make(map[byte][]byte)

  // cl.codeRunner, err = NewCodeLauncher(*config.AssetsPath + "assets/exec.py")
  // if err != nil {
  //   return nil, fmt.Errorf("Asset Manager: %s could not be opened.\n", *config.AssetsPath + "assets/exec.py")
  // }

  cl.termRunner, err = NewTermLauncher(*config.AssetsPath + "assets/ngrok")
  if err != nil {
    return nil, fmt.Errorf("Asset Manager: %s could not be opened.\n", *config.AssetsPath + "assets/ngrok")
  }

  cl.syncer = NewFlightSyncer(*config.FlightLogPath)

  // Use cwd
  cl.store, err = NewStore(*config.AssetsPath + ".")
  if err != nil {
    return nil, err
  }

  cl.store.Load()

  return cl, nil
}

func (cl *CloudLink) GetStore() *Store {
  return cl.store
}

func (cl *CloudLink) SendSensor(name string, val map[string]interface{}) (*http.Response, error) {
  // JSON req
  if buf, err := json.Marshal(val); err != nil {
    return nil, err
  } else {

    return http.Post(*config.DSCHttp + "/rt/drone/" + cl.syncer.DroneId + "/sensor/" + name, "application/json", bytes.NewBuffer(buf))
  }
}

func (cl *CloudLink) Logout() error {
  cl.sessionId = 0
  cl.messageCnt = TIME_OUT_CNT
  cl.codeStatus = 0
  cl.terminalOnline = false

  if err := cl.store.Del(); err != nil {
    return err
  } else {
    // skip wifi setup.
    cl.store.Set("step", "setupWifi")
    return nil
  }
}

func (cl *CloudLink) IsOnline() bool {
  attempts := 0

  for {
    time.Sleep(time.Second)
    if cl.sessionId == 0 {
      attempts++
    } else {
      return true
    }

    if attempts >= 5 {
      return false
    }
  }
}

func (cl *CloudLink) IsOnlineNonBlock() bool {
  return cl.sessionId != 0
}

func (cl *CloudLink) SendSyncLock(name string) {
  cl.syncer.Lock(name)
}

func (cl *CloudLink) SendSyncUnlock() {
  cl.syncer.Unlock()
}

func (cl *CloudLink) Monitor() {
  for {
    if err := cl.Serve(); err != nil {
      config.Log(config.LOG_ERROR, "cl: ", "CL Link down!")
    } else {
      config.Log(config.LOG_ERROR, "cl: ", "CL Stopped serving!")
    }
    time.Sleep(15 * time.Second)
  }
}

func (cl *CloudLink) Serve() error {
  {
    var err error
    var localAddr *net.UDPAddr

    // Attempt connection
    if cl.addr, err = net.ResolveUDPAddr("udp4", *config.DSCAddress); err != nil {
  		return err
  	} else {
      if localAddr, err = net.ResolveUDPAddr("udp4", "0.0.0.0:0"); err != nil {
        return err
      } else {
        if cl.conn, err = net.DialUDP("udp", localAddr, cl.addr); err != nil {
    			return err
    		}
        config.Log(config.LOG_INFO, "cl:  Listening on", *config.DSCAddress)
      }
  	}
  }

  cl.rx = make([]byte, 2048)
  cl.sessionId = 0
  cl.messageCnt = TIME_OUT_CNT
  cl.codeStatus = 0
  cl.terminalOnline = false
  cl.timer = time.NewTimer(1 * time.Second)
  cl.syncTimer = time.NewTimer((time.Duration)(*config.SyncThrottle) * time.Millisecond)


  // read thread
  go func() {
    for {
      n, _, err := cl.conn.ReadFromUDP(cl.rx)

      if err != nil {
        // log.Println(err)
        cl.sessionId = 0
      } else if n > 0 {
        // parse message
        if decoded, err := dronedp.ParseMsg(cl.rx[:n]); err != nil {
          config.Log(config.LOG_WARN, err)
        } else {
          cl.handleMessage(decoded)
        }
      }
      // Wait a little before reading again
      time.Sleep(1 * time.Millisecond)
    }
  }()

  for {

    select {
      // "Untrottled" messages. We want to send these back as fast as possible.
    case packet := <-cl.noThrottleMsg:
      if send, err := dronedp.GenerateMsg(dronedp.OP_MAVLINK_BIN, cl.sessionId, packet); err != nil {
        config.Log(config.LOG_WARN, "cl: ", err)
      } else {
        // could be no connection
        if cl.conn != nil {
          cl.conn.Write(send)
        }
      }

    case <-cl.syncTimer.C:
      if cl.IsOnlineNonBlock() {
        cl.msgMut.RLock()
        for _, packet := range cl.msgs {
          if send, err := dronedp.GenerateMsg(dronedp.OP_MAVLINK_BIN, cl.sessionId, packet); err != nil {
            config.Log(config.LOG_WARN, "cl: ", err)
          } else {
            // could be no connection
            if cl.conn != nil {
              cl.conn.Write(send)
              op := packet[0x05]
              delete(cl.msgs, op)
            }
          }
        }
        cl.msgMut.RUnlock()
      }

      cl.syncTimer.Reset((time.Duration)(*config.SyncThrottle) * time.Millisecond)

    case <-cl.timer.C:
      cl.timer.Reset(1 * time.Second)
      cl.uid = cl.store.Get("ruid")
      if cl.uid != "" {
        cl.sendStatus()
      } else {
        cl.genRandomId()
        cl.uid = cl.store.Get("ruid");
      }

    // case cl.codeStatus = <-cl.codeRunner.Pid:
      // just need the figure, no update

    // case str := <-cl.codeRunner.Update:
    //   config.Log(config.LOG_INFO, "cl: ", str)
    //   // send code updates
    //   cop := dronedp.CodeMsg{Op: "code", Msg: str, Status: cl.codeStatus}
    //   if send, err := dronedp.GenerateMsg(dronedp.OP_STATUS, cl.sessionId, cop); err != nil {
    //     config.Log(config.LOG_WARN, "cl: ", err)
    //   } else {
    //     cl.conn.Write(send)
    //   }

    case publicTunnel := <-cl.termRunner.Update:
      // send terminal update
      // log.Println(publicTunnel)
      urls := strings.Split(publicTunnel, "tcp://")
      urls = strings.Split(urls[1], ":")
      if ival, err := strconv.Atoi(urls[1]); err != nil {
        config.Log(config.LOG_WARN, "cl: ", err)
      } else {
        top := dronedp.TerminalMsg{
            Op: "terminal",
            Status: cl.terminalOnline,
            Msg: dronedp.TerminalInfo{
              Url: urls[0], Port: ival, User: "", Pass: "",
            },
        }

        if send, err := dronedp.GenerateMsg(dronedp.OP_TERMINAL, cl.sessionId, top); err != nil {
          config.Log(config.LOG_WARN, "cl: ", err)
        } else {
          cl.conn.Write(send)
        }
      }
    }
  }

  // dealloc timer
  cl.timer.Stop()
  return nil
}

func (cl *CloudLink) SetRawFmuCmd(chunk []byte) {
  cl.packmut.Lock()
  cl.rawFmuCmd = chunk
  cl.packmut.Unlock()
}

func (cl *CloudLink) GetRawFmuCmd() []byte {
  cl.packmut.RLock()
  defer cl.packmut.RUnlock()
  return cl.rawFmuCmd
}

func (cl *CloudLink) NullRawFmuCmd() {
  cl.packmut.Lock()
  cl.rawFmuCmd = nil
  cl.packmut.Unlock()
}

func (cl *CloudLink) UpdateFromFMU(packet []byte) {
  // config.Log(config.LOG_INFO, "packet: ", packet)
  // update message map
  if len(packet) > 0x05 {
    cl.msgMut.Lock()
    op := packet[0x05]
    cl.msgs[op] = packet
    cl.msgMut.Unlock()
  }

  // Check to see if it's an unthrottled message
  switch packet[0x05] {
  case mavlink.MSG_ID_PARAM_VALUE: fallthrough
  case mavlink.MSG_ID_MISSION_COUNT: fallthrough
  case mavlink.MSG_ID_MISSION_ACK: fallthrough
  case mavlink.MSG_ID_COMMAND_ACK: fallthrough
  case mavlink.MSG_ID_AUTOPILOT_VERSION:
    cl.SetUnthrottledMsg(packet)
  }
}

func (cl *CloudLink) SetUnthrottledMsg(p []byte) {
  cl.noThrottleMsg <- p
}

func (cl *CloudLink) UpdateSerialId(uid uint64) {
  // XXX
  if p := cl.store.Get("ruid"); p == "" {
    println("Is nil generating random id")
    cl.genRandomId()
  }

  ruid := cl.store.Get("ruid");
  s := strconv.Itoa(int(uid))

  cl.uid = s + ruid
}

func (cl *CloudLink) sendStatus() {
  var sm dronedp.StatusMsg
  if cl.sessionId == 0 {
    if cl.syncer.IsRunning() {
      config.Log(config.LOG_INFO, "cl: ", "Turning off Syncer")
      cl.syncer.Stop()
    }

     em := cl.store.Get("email")
     ps := cl.store.Get("pass")
    sm = dronedp.StatusMsg{Op: "connect",
      Serial: string(cl.uid), SimId: *config.SimId, Email: em, Password: ps,}
  } else {
    sm = dronedp.StatusMsg{Op: "status",}
  }

  // config.Log(config.LOG_INFO, sm)
  if ddpdata, err := dronedp.GenerateMsg(dronedp.OP_STATUS, cl.sessionId, sm); err != nil {
    config.Log(config.LOG_WARN, "cl: ", err)
  } else {
    cl.conn.Write(ddpdata)
  }

  cl.checkOnline()
}

func (cl *CloudLink) handleMessage(decoded *dronedp.Msg) {
  cl.messageCnt = TIME_OUT_CNT
  if decoded.Session != cl.sessionId {
    config.Log(config.LOG_INFO, "cl: ", "Session changed:", decoded.Session)
    cl.sessionId = decoded.Session
  }

  switch decoded.Op {
  case dronedp.OP_MAVLINK_BIN:
    // Send message to FMU
    chunk := decoded.Data.([]byte)
    cl.SetRawFmuCmd(chunk)
  case dronedp.OP_STATUS:
    statusMsg, _ := decoded.Data.(*dronedp.StatusMsg)
    droneId := statusMsg.Drone["_id"].(string)

    // avoid sending to the wrong person
    if cl.syncer.IsRunning() && (cl.syncer.UserId != statusMsg.User || cl.syncer.DroneId != droneId) {
      config.Log(config.LOG_INFO, "cl: ", "Turning off Syncer")
      cl.syncer.Stop()
    }

    // Get this party started
    if !cl.syncer.IsRunning() {
      config.Log(config.LOG_INFO, "cl: ", "Turning on Syncer")
      cl.syncer.Start(statusMsg.User, droneId)
    }

    // Code launching currently not supported.
    // if statusMsg.Code != "" && cl.codeStatus == 0 {
    //   config.Log(config.LOG_INFO, "cl: ", "Got CODE, running job.")
    //
    //   go func() {
    //     if err := cl.codeRunner.execScript(statusMsg.Code); err != nil {
    //       config.Log(config.LOG_ERROR, "cl: ", err)
    //     }
    //   }()
    // }

    if statusMsg.Terminal {
      if !cl.terminalOnline {
        config.Log(config.LOG_INFO, "cl: ", "Got TERMINAL, opening tunnel")
        cl.terminalOnline = true

        go func() {
          if err := cl.termRunner.Open(); err != nil {
            config.Log(config.LOG_ERROR, "cl: ", err)
          }
        }()
      }
    } else {
      if cl.terminalOnline {
        config.Log(config.LOG_INFO, "cl: ", "Got TERMINAL, shutting down tunnel")

        go func() {
          if err := cl.termRunner.Close(); err != nil {
            config.Log(config.LOG_ERROR, "cl: ", err)
          } else {
            cl.terminalOnline = false
          }
        }()
      }
    }
  }
}

func (cl *CloudLink) checkOnline() {
  cl.messageCnt--
  if cl.messageCnt == 0 {
    cl.sessionId = 0
    cl.messageCnt = TIME_OUT_CNT
    config.Log(config.LOG_WARN, "cl: ", "No response from server.")
  }
}

func (cl *CloudLink) genRandomId() {
  // set seed
  rand.Seed(time.Now().UTC().UnixNano())
  s := strconv.Itoa(rand.Int())
  cl.store.Set("ruid", s)
}
