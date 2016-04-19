package statusServer

import (
  "fmt"
  "encoding/json"
  "os/exec"
  "regexp"
)

const (
  EDISON_EXEC = "/usr/bin/configure_edison" // change this to /bin/edison_configure
)

func checkIP() (bool, string, error) {
  out, err := runEdisonCmd("--showWiFiIP")
  if err != nil {
    return false, "", err
  }

  re := regexp.MustCompile(`\d{3}\.\d{3}\.\d{3}\.\d{3}`).Find([]byte(out))

  if re != nil {
    return true, string(re), nil
  } else {
    return false, "", nil
  }
}

func enableAP(enable bool) error {
  var err error
  if enable {
    _, err = runEdisonCmd("--enableOneTimeSetup")
  } else {
    _, err = runEdisonCmd("--disableOneTimeSetup")
  }

  return err
}

func getNames() (map[string]string, error) {
  obj := make(map[string]string)
  out, err := runEdisonCmd("--showNames")
  if err != nil {
    return nil, err
  }

  if err = json.Unmarshal([]byte(out), obj); err != nil {
    return nil, err
  } else {
    return obj, nil
  }
}

func setName(name string) error {
  if (len(name) < 4) {
    return fmt.Errorf("Name must be at least 4 characters.")
  }
  if _, err := runEdisonCmd("--changeName", name); err != nil {
    return err
  } else {
    return nil
  }
}

func runEdisonCmd(args ... string) (string, error) {
  out, err := exec.Command(EDISON_EXEC, args ...).Output()
  return string(out), err
}
