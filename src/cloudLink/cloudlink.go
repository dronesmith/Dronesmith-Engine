package cloudlink

import (
  "log"
  "net"
  "time"
  "strings"
  "strconv"

  "cloudlink/dronedp"
)

const (
	// Default address, if otherwise not specified by cli args. IP is the cloud,
	// 4002 is the dronedp listening port.
	// DEFAULT_DSC_ADDRESS = "24.234.109.135:4002"
  DEFAULT_DSC_ADDRESS = "127.0.0.1:4002"

  TIME_OUT_CNT = 5
)

type CloudLink struct {
  addr        *net.UDPAddr
  conn        *net.UDPConn
  rx          []byte
  sessionId   uint32
  messageCnt  int
  codeStatus  int
  terminalOnline bool

  uid         string

  termRunner  *TermLauncher
  codeRunner  *CodeLauncher
  store       *Store
}

func NewCloudLink() (*CloudLink, error) {
  var err error
  cl := &CloudLink{}

  cl.codeRunner, err = NewCodeLauncher("lucikit/devkit/exec.py")
  if err != nil {
    return nil, err
  }

  cl.termRunner, err = NewTermLauncher("assets/ngrok")
  if err != nil {
    return nil, err
  }

  // Use cwd
  cl.store, err = NewStore(".")
  if err != nil {
    return nil, err
  }

  cl.store.Load()

  if cl.addr, err = net.ResolveUDPAddr("udp4", DEFAULT_DSC_ADDRESS); err != nil {
		return nil, err
	} else {
    if localAddr, err := net.ResolveUDPAddr("udp4", "localhost:0"); err != nil {
      return nil, err
    } else {
      if cl.conn, err = net.DialUDP("udp", localAddr, cl.addr); err != nil {
  			return nil, err
  		} else {
        return cl, nil
      }
    }
	}
}

func (cl *CloudLink) Serve() {
  cl.rx = make([]byte, 1024)
  cl.sessionId = 0
  cl.messageCnt = TIME_OUT_CNT
  cl.codeStatus = 0
  cl.terminalOnline = false

  for {
    go func() {
      n, _, err := cl.conn.ReadFromUDP(cl.rx)

      if err != nil {
        // log.Println(err)
        cl.sessionId = 0
      } else if n > 0 {
        // parse message
        if decoded, err := dronedp.ParseMsg(cl.rx[:n]); err != nil {
          log.Println(err)
        } else {
          cl.handleMessage(decoded)
        }
      }
    }()

    select {
    case <-time.Tick(1 * time.Second):
      if cl.uid != "" {
        cl.sendStatus()
      }

    case cl.codeStatus = <-cl.codeRunner.Pid:
      // just need the figure, no update

    case str := <-cl.codeRunner.Update:
      log.Println(str)
      // send code updates
      cop := dronedp.CodeMsg{Op: "code", Msg: str, Status: cl.codeStatus}
      if send, err := dronedp.GenerateMsg(dronedp.OP_STATUS, cl.sessionId, cop); err != nil {
        log.Println(err)
      } else {
        cl.conn.Write(send)
      }

    case publicTunnel := <-cl.termRunner.Update:
      // send terminal update
      log.Println(publicTunnel)
      urls := strings.Split(publicTunnel, "tcp://")
      urls = strings.Split(urls[1], ":")
      if ival, err := strconv.Atoi(urls[1]); err != nil {
        log.Println(err)
      } else {
        top := dronedp.TerminalMsg{
            Op: "terminal",
            Status: cl.terminalOnline,
            Msg: dronedp.TerminalInfo{
              Url: urls[0], Port: ival, User: "root", Pass: "doingitlive",
            },
        }

        if send, err := dronedp.GenerateMsg(dronedp.OP_STATUS, cl.sessionId, top); err != nil {
          log.Println(err)
        } else {
          cl.conn.Write(send)
        }
      }
    }
  }
}

func (cl *CloudLink) UpdateFromFMU(packet []byte) {
  if send, err := dronedp.GenerateMsg(dronedp.OP_MAVLINK_BIN, cl.sessionId, packet); err != nil {
    log.Println(err)
  } else {
    cl.conn.Write(send)
  }
}

func (cl *CloudLink) UpdateSerialId(uid uint64) {
  s := strconv.Itoa(int(uid))
  cl.uid = s
}

func (cl *CloudLink) sendStatus() {
  var sm dronedp.StatusMsg
  if cl.sessionId == 0 {
     em, ps := cl.store.Get()
    sm = dronedp.StatusMsg{Op: "connect",
      Serial: string(cl.uid), Email: em, Password: ps,}
  } else {
    sm = dronedp.StatusMsg{Op: "status"}
  }

  if ddpdata, err := dronedp.GenerateMsg(dronedp.OP_STATUS, cl.sessionId, sm); err != nil {
    log.Println(err)
  } else {
    cl.conn.Write(ddpdata)
  }

  cl.checkOnline()
}

func (cl *CloudLink) handleMessage(decoded *dronedp.Msg) {
  cl.messageCnt = TIME_OUT_CNT
  if decoded.Session != cl.sessionId {
    log.Println("WARN: Session changed:", decoded.Session)
    cl.sessionId = decoded.Session
  }

  switch decoded.Op {
  case dronedp.OP_STATUS:
    statusMsg, _ := decoded.Data.(*dronedp.StatusMsg)

    if statusMsg.Code != "" && cl.codeStatus == 0 {
      log.Println("Got CODE, running job.")

      go func() {
        if err := cl.codeRunner.execScript(statusMsg.Code); err != nil {
          log.Println(err)
        }
      }()
    }

    if statusMsg.Terminal {
      if !cl.terminalOnline {
        log.Println("Got TERMINAL, opening tunnel")
        cl.terminalOnline = true

        go func() {
          if err := cl.termRunner.Open(); err != nil {
            log.Println(err)
          }
        }()
      }
    } else {
      if cl.terminalOnline {
        log.Println("Got TERMINAL, shutting down tunnel")

        go func() {
          if err := cl.termRunner.Close(); err != nil {
            log.Println(err)
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
    log.Println("WARN: No response from server.")
  }
}
