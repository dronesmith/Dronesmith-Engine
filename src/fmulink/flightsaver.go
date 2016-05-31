package fmulink

import (
  "path"
  "os"
  "time"
  "fmt"
)

type FlightSaver struct {
  logPath string
  isLogging bool
  file *os.File
  fname string
}

func NewFlightSaver(fpath string) *FlightSaver {
  return &FlightSaver{
    fpath,
    false,
    nil,
    "",
  }
}

func (fs *FlightSaver) Start() error {
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
}

func (fs *FlightSaver) Persist(data *[]byte) error {
  if fs.isLogging && fs.file != nil {
    if chunk, err := time.Now().MarshalBinary(); err != nil {
      return err
    } else {
      if _, err := fs.file.Write(chunk); err != nil {
        return err
      } else if _, err := fs.file.Write(*data); err != nil {
        return err
      } else {
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
