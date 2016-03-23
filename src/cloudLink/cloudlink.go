package cloudlink

import (
  "net"
)

const (
	// Default address, if otherwise not specified by cli args. IP is the cloud,
	// 4002 is the dronedp listening port.
	DEFAULT_DSC_ADDRESS = "24.234.109.135:4002"
)

type CloudLink struct {
  addr     *net.UDPAddr
  conn     *net.UDPConn
  rx       *[]byte
}

func NewCloudLink() (*CloudLink, error) {
  cl := &CloudLink{}
  if cl.addr, err := net.ResolveUDPAddr("udp4", DEFAULT_DSC_ADDRESS); err != nil {
		return nil, err
	} else {
		if cl.conn, err := net.ListenUDP("udp", cl.addr); err != nil {
			return nil, err
		} else {
      return cl, nil
    }
	}
}

func (cl *CloudLink) Run() error {
  cl.rx = &make([]byte, 1024)

  // Set up poll tasks



  for {
    n, address, err := cl.conn.ReadFromUDP(*cl.rx)

    if err != nil {
      fmt.Println("error reading data from connection")
      fmt.Println(err)
    } else if n > 0 {
      // TODO decode
    }
  }
}

func onSendMessage() {
  // get userData


}

type clTask interface {
  Run(uint32) error
  Pause()
  Continue()
  Log(...interface{})
}

type SendTask struct {
  clTask
  opCode    uint8
  logName   string
  job       func
  pause     chan bool
  cont      chan bool
}

func NewSendTask(op uint8, name string) SendTask {
  return &SendTask{
    opCode: op,
    logName: name
  }
}

func (st *SendTask) Log(args ...interface{}) {
  log.Println("[" + st.logName + "] ", args...)
}

func (st *SendTask) Run(intervalMs uint32) error {
  for {
    select {
    case <-st.pause:
      <-st.cont
    default:
      if err := st.job(); err != nil {
        return error(err)
      }
      time.Sleep(intervalMs)
    }
  }

  return nil
}

func (st *SendTask) Pause() {
  st.pause <-true
}

func (st *PollTask) Continue() {
  st.cont <-true
}
