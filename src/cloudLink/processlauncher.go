package cloudlink

import (
  "bufio"
  "os"
  "os/exec"
  "path/filepath"
  "strconv"
  "runtime"
  "log"
  "fmt"
  "net/http"
  "time"
  "io/ioutil"
  "encoding/json"
)

const (
  EXEC_NAME string = "exec.py"

  NGROK_AUTH_STR string = "7kmpYXuGZaRGXMSADoMus_6fe8N2q9wtFaCdJE5BeSt"
  NGROK_AUTH_TUNNEL string = "http://localhost:4040/api/tunnels"
)

type CodeLauncher struct {
  cmd     *exec.Cmd
  path    string
  Update  chan string
  Pid     chan int
}

func NewCodeLauncher(path string) (*CodeLauncher, error) {
  if absPath, err := filepath.Abs(path); err != nil {
    return nil, err
  } else {
    if _, err := os.Stat(absPath); err != nil {
      return nil, err
    } else {
      return &CodeLauncher{
        nil, absPath, make(chan string), make(chan int),
      }, nil
    }
  }
}

// NOTE blocking
func (cl *CodeLauncher) execScript(code string) error {

  cl.cmd = exec.Command("python", cl.path, "--code", code)

  stdout, err := cl.cmd.StdoutPipe()
  if err != nil {
    return err
  }

  // get errors too
  stderr, err := cl.cmd.StderrPipe()
  if err != nil {
    return err
  }

  // closures to get output/error piped data
  scanner := bufio.NewScanner(stdout)
	go func() {
		for scanner.Scan() {
      // send to cloudlink for update
      cl.Update <- scanner.Text()
		}
	}()

  scannerErr := bufio.NewScanner(stderr)
  go func() {
    for scannerErr.Scan() {
      // send to cloudlink for update
      cl.Update <- scannerErr.Text()
    }
  }()

  if err := cl.cmd.Start(); err != nil {
    return err
  }

  cl.Update <- "Running job " + strconv.Itoa(cl.cmd.Process.Pid) + "..."

  cl.Pid <- cl.cmd.Process.Pid

  err = cl.cmd.Wait()
  cl.cmd = nil
  cl.Pid <- 0
	if err != nil {
    cl.Update <- "Process exited abnormally: " + err.Error()
    return err
  } else {
    cl.Update <- "Process exited successfully."
    return nil
  }

  return nil
}

type TermLauncher struct {
  cmd     *exec.Cmd
  path    string
  Update  chan string
}

func NewTermLauncher(path string) (*TermLauncher, error) {
  tl := &TermLauncher{}
  if absPath, err := filepath.Abs(path); err != nil {
    return nil, err
  } else {
    switch runtime.GOOS {
    case "darwin":
      tl.path = filepath.Join(absPath, "ngrok_osx")
    case "linux":
      tl.path = filepath.Join(absPath, "ngrok_edison")
    default:
      return nil, fmt.Errorf("Unsupported operating system: %v\n", runtime.GOOS)
    }

    tl.Update = make(chan string)

    return tl, nil
  }
}

// NOTE blocking
func (tl *TermLauncher) Open() error {
  tl.cmd = exec.Command(tl.path, "authtoken", NGROK_AUTH_STR)
  if err := tl.cmd.Start(); err != nil {
    return err
  }

  err := tl.cmd.Wait()
  if err != nil {
    return err
  } else {
    log.Println("Auth successful.")

    // spawn the task, make the tunnel
    tl.cmd = exec.Command(tl.path, "tcp", "22")

    if err := tl.cmd.Start(); err != nil {
      return err
    }

    checkTimer := time.NewTimer(1 * time.Second)
    for {
      <- checkTimer.C
      if err := tl.getInfo(); err != nil {
        log.Println(err)
        checkTimer.Reset(1 * time.Second)
      } else {
        checkTimer.Stop()
        break
      }
    }

    err := tl.cmd.Wait()
    if err != nil {
      return err
    } else {
      return nil
    }
  }

  return nil
}

func (tl *TermLauncher) Close() error {
  if tl.cmd != nil && !tl.cmd.ProcessState.Exited() {
    if err := tl.cmd.Process.Kill(); err != nil {
      return err
    } else {
      return nil
    }
  }
  return nil
}

func (tl *TermLauncher) getInfo() error {
  if res, err := http.Get(NGROK_AUTH_TUNNEL); err != nil {
    return err
  } else {
    defer res.Body.Close()

    if body, err := ioutil.ReadAll(res.Body); err != nil {
      return err
    } else {
      var t map[string]interface{}

      json.Unmarshal(body, &t)
      if tunnels, f := t["tunnels"]; !f {
        return fmt.Errorf("No tunnel in response object")
      } else {
        tunnelSlice := tunnels.([]interface{})
        if len(tunnelSlice) == 0 {
          return fmt.Errorf("No open tunnels in reponse object")
        } else {
          tunnel := tunnelSlice[0].(map[string]interface{})
          tl.Update <- tunnel["public_url"].(string)
          return nil
        }
      }
    }
  }
}
