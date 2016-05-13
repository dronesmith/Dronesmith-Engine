package cloudlink

import (
  "encoding/pem"
  "path"
  "os"
  "strings"
  "sync"
)

const (
  FILE_KEY = ".lmon"
)

type Store struct {
  path    string

  // store data
  data    map[string]string

  mut     sync.RWMutex
}

func NewStore(apath string) (*Store, error) {
  fpath := path.Join(apath, FILE_KEY)
  s := &Store{
    path: fpath,
    data: make(map[string]string),
  }
  if _, err := os.Stat(fpath); os.IsNotExist(err) {
    // Make a new file
    if f, err := os.Create(fpath); err != nil {
      return nil, err
    } else {
      f.Close()
    }
  } else if err != nil {
    // Some other error that we cant to catch
    return nil, err
  } else {
    // Load old file
    s.Load()
  }

  return s, nil
}

func (s *Store) Set(name, value string) error {
  s.mut.Lock()
  defer s.mut.Unlock()

  file, err := os.OpenFile(s.path, os.O_WRONLY, 0600)

  if err != nil {
    if file, err = os.Create(s.path); err != nil {
      return err
    }
  }
    defer file.Close()

    s.data[name] = value

    // rehash data
    strArray := make([]string, 0, len(s.data))
    for k, v := range(s.data) {
      strArray = append(strArray, k + ":" + v)
    }

    hash := &pem.Block{Type: "LMON", Bytes: []byte(strings.Join(strArray, ";"))}

    // write
    if err := pem.Encode(file, hash); err != nil {
      delete(s.data, name) // remove from memory
      return err
    }

  return nil
}

func (s *Store) Del() error {
  s.mut.Lock()
  defer s.mut.Unlock()

  s.data = nil

  if err := os.Remove(FILE_KEY); err != nil {
    return err
  } else {
    return nil
  }
}

func (s *Store) Load() error {
  s.mut.Lock()
  defer s.mut.Unlock()

  if file, err := os.Open(s.path); err != nil {
    return err
  } else {
    defer file.Close()
    buf := make([]byte, 512)
    if _, err := file.Read(buf); err != nil {
      return err
    } else {
      blk, _ := pem.Decode(buf)

      vals := strings.Split(string(blk.Bytes), ";")

      // reinit map since the file is our single source of truth.
      s.data = make(map[string]string)

      for _, v := range(vals) {
        arr := strings.Split(v, ":")
        if len(arr) > 1 {
          key, value := arr[0], arr[1]
          s.data[key] = value  
        }
      }

      return nil
    }
  }
}

func (s *Store) Get(name string) string {
  s.mut.RLock()
  defer s.mut.RUnlock()

  val, ok := s.data[name]

  if !ok {
    return ""
  } else {
    return val
  }
}
