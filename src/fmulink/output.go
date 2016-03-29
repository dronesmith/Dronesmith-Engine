package fmulink

import (
  // "bytes"
  // "encoding/binary"
  "fmt"
  "log"
  "net"

  // "mavlink/parser"
)

type OutputManager struct {
  links       map[string]*net.UDPConn

  mavMessage  chan *[]byte
  quit        chan bool
}

func NewOutputManager() *OutputManager {
  o := &OutputManager{
    make(map[string]*net.UDPConn),
    make(chan *[]byte),
    make(chan bool),
  }

  go o.listen()

  return o
}

func (o *OutputManager) listen() {
  for {
    select {

    case pkt := <-o.mavMessage:
      for _, e := range o.links {
        // var buf bytes.Buffer
        // binary.Write(&buf, binary.BigEndian, pkt)
        if _, err := e.Write(*pkt); err != nil {
          log.Printf("OUTPUT: %v\n", err)
        }
      }

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

  localAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
  if err != nil {
    return err
  }

  conn, err := net.DialUDP("udp", localAddr, udpAddr)
  if err != nil {
    return err
  }

  o.links[addr] = conn
  return nil
}

func (o* OutputManager) Remove(addr string) error {
  item, found := o.links[addr]
  if found {
    item.Close()
    delete(o.links, addr)
  } else {
    return fmt.Errorf("No key %s exists.\n", addr)
  }
  return nil
}

func (o *OutputManager) Kill() {
  o.quit <-true
}
