package fmulink

import (
  "time"
  "sync"
)

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
