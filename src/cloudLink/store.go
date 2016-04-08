package cloudlink

import (
  "encoding/pem"
  "path"
  "os"
  "strings"
)

const (
  FILE_KEY = ".lmon"
)

type Store struct {
  path    string

  // store data
  email   string
  pass    string
}

func NewStore(apath string) (*Store, error) {
  fpath := path.Join(apath, FILE_KEY)
  s := &Store{
    path: fpath,
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
    s.Get()
  }

  return s, nil
}

func (s *Store) Set(email, pass string) error {
  if file, err := os.OpenFile(s.path, os.O_WRONLY, 0600); err != nil {
    return err
  } else {
    defer file.Close()
    // hash



    hash := &pem.Block{Type: "LMON", Bytes: []byte(strings.Join([]string{email,pass}, ";"))}

    // write
    if err := pem.Encode(file, hash); err != nil {
      return err
    } else {
      s.email = email
      s.pass = pass
    }
  }
  return nil
}

func (s *Store) Load() error {
  if file, err := os.Open(s.path); err != nil {
    return err
  } else {
    defer file.Close()
    buf := make([]byte, 256)
    if _, err := file.Read(buf); err != nil {
      return err
    } else {
      blk, _ := pem.Decode(buf)

      vals := strings.Split(string(blk.Bytes), ";")

      // Update
      s.email = vals[0]
      s.pass = vals[1]

      return nil
    }
  }
}

func (s *Store) Get() (string, string) {
  return s.email, s.pass
}
