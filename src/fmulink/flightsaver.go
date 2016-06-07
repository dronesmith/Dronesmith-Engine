package fmulink

import (
  "path"
  "os"
  "time"
  "fmt"
  "config"
  "sync"
)

type FlightSaver struct {
  logPath     string
  isLogging   bool
  file        *os.File
  fname       string
  msgSet      map[uint8]bool
  duration    time.Duration
  timer       *time.Timer
  mut         sync.Mutex
  quit        chan bool
}

func NewFlightSaver(fpath string) *FlightSaver {
  dur := time.Duration(*config.SyncThrottle) * time.Nanosecond

  return &FlightSaver{
    fpath,
    false,
    nil,
    "",
    make(map[uint8]bool),
    dur,
    nil,
    sync.Mutex{},
    make(chan bool),
  }
}

// We want to throttle the messages but also ensure we're accessing them fairely.
// My solution is to use a set that contains the packet Id, and flush
// the set at the end of an interval. If it's not in the set, that means it hasn't
// been accessed in the time period, and should be logged. Otherwise, ignore it
// until the timer is up. This ensures a fairness bound on all incoming messages. The GC
// will handle the dangling reference when the set is reallocated.
//
// TODO - there are certain messages like mission data we want to log regardless. Perhaps
// making the bool double as a flag that this message should be logged always.
func (fs *FlightSaver) throttler() {
  for {
    select {
    case <- fs.timer.C:
      // empty the set to allow more messages
      // config.Log(config.LOG_DEBUG, "Timer wake up")
      fs.mut.Lock()
      fs.msgSet = make(map[uint8]bool)
      fs.timer.Reset(fs.duration)
      fs.mut.Unlock()

    case <- fs.quit:
      return
    }
  }
}

func (fs *FlightSaver) Start() error {
  if fs.timer != nil {
    fs.timer.Reset(fs.duration)
  } else {
    fs.timer = time.NewTimer(fs.duration)
  }

  go fs.throttler()

  fs.msgSet = make(map[uint8]bool)
  fs.fname =  "Flight " + time.Now().Format(time.UnixDate) + ".log"
  fpath := path.Join(fs.logPath, fs.fname)
  if f, err := os.Create(fpath); err != nil {
    return err
  } else {
    fs.file = f
    fs.isLogging = true
    return nil
  }
}

func (fs *FlightSaver) End() {
  fs.isLogging  = false
  fs.fname = ""
  fs.file.Close()
  fs.timer.Stop()
  fs.quit <- true
}

func (fs *FlightSaver) Persist(data *[]byte, hdr uint8) error {
  if fs.isLogging && fs.file != nil {
    if chunk, err := time.Now().MarshalBinary(); err != nil {
      return err
    } else {
      fs.mut.Lock()
      if _, found := fs.msgSet[hdr]; !found {
        fs.msgSet[hdr] = true
        fs.mut.Unlock()
        if _, err := fs.file.Write(chunk); err != nil {
          return err
        } else if _, err := fs.file.Write(*data); err != nil {
          return err
        } else {
          return nil
        }
      } else {
        fs.mut.Unlock() // not deferring this because we want to not lock asap
        // Message is in the set, so avoid syncing till the timer empties the set.
        return nil
      }
    }

  } else {
    // Probably won't catch this error, but indicate it anyways.
    return fmt.Errorf("Attempted to log flight data when none are open!")
  }
}

func (fs *FlightSaver) IsLogging() bool {
  return fs.isLogging
}

func (fs *FlightSaver) Name() string {
  return fs.fname
}
