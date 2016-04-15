package fmulink

import (
  "fmt"
  "config"
  "net"
  "sync"

  "time"

)

type OutputManager struct {
  links       map[string]*outputLink

  mavMessage  chan *[]byte
  quit        chan bool

  Input       chan []byte

  mut         sync.RWMutex
}

type outputLink struct {
  Conn        *net.UDPConn
  Quit        chan bool
}

func NewOutputManager() *OutputManager {
  o := &OutputManager{
    make(map[string]*outputLink),
    make(chan *[]byte),
    make(chan bool),
    make(chan []byte),
    sync.RWMutex{},
  }

  go o.listen()

  return o
}

func (o *OutputManager) listen() {
  for {
    select {

    case pkt := <-o.mavMessage:
      o.mut.RLock()
      for _, e := range o.links {
        // var buf bytes.Buffer
        // binary.Write(&buf, binary.BigEndian, pkt)
        if _, err := e.Conn.Write(*pkt); err != nil {
          config.Log(config.LOG_WARN, "out: ", err)
        }
      }
      o.mut.RUnlock()

    case <-o.quit:
      return
    }
  }
}

func (o *OutputManager) Send(data *[]byte) {
  o.mavMessage <-data
}

func (o *OutputManager) Add(addr string) error {

  udpAddr, err := net.ResolveUDPAddr("udp", addr)
  if err != nil {
    return err
  }

  localAddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:0")
  if err != nil {
    return err
  }

  conn, err := net.DialUDP("udp", localAddr, udpAddr)
  if err != nil {
    return err
  }

  o.mut.Lock()
  defer o.mut.Unlock()
  o.links[addr] = &outputLink{conn,make(chan bool),}

  // set up input listener
  go func() {
    b := make([]byte, 263)
    timer := time.NewTicker(100 * time.Millisecond)
    for {
      select {
      case <- timer.C:
        if size, err := conn.Read(b); err != nil {
          config.Log(config.LOG_WARN, "in: ", err)
        } else if size > 0 {
          o.Input <- b
        }

      case <- o.links[addr].Quit:
        return
      }
    }
  }()

  return nil
}

func (o* OutputManager) Remove(addr string) error {
  o.mut.Lock()
  defer o.mut.Unlock()
  item, found := o.links[addr]
  if found {
    item.Conn.Close()
    item.Quit <- true
    delete(o.links, addr)
  } else {
    return fmt.Errorf("No key %s exists.\n", addr)
  }
  return nil
}

func (o *OutputManager) Kill() {
  o.quit <-true
}
