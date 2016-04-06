package cloudlink

import (
  "log"
  "net"
  "time"

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
}

func NewCloudLink() (*CloudLink, error) {
  var err error
  cl := &CloudLink{}

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
          log.Println(decoded.Data)

          cl.handleMessage(decoded)
        }
      }
    }()

    select {
    case <-time.Tick(1 * time.Second):
      cl.sendStatus()
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

func (cl *CloudLink) sendStatus() {
  var sm dronedp.StatusMsg
  if cl.sessionId == 0 {
    sm = dronedp.StatusMsg{Op: "connect",
      Serial: "1-golang", Email: "geoff@skyworksas.com", Password: "test12345",}
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

  // decoded.Data.(StatusMsg)
  log.Println("JSON: ", decoded.Data)
}

func (cl *CloudLink) checkOnline() {
  cl.messageCnt--
  if cl.messageCnt == 0 {
    cl.sessionId = 0
    cl.messageCnt = TIME_OUT_CNT
    log.Println("WARN: No response from server.")
  }
}
