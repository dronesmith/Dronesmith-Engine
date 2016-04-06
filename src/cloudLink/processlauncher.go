package cloudlink

import (
  "bufio"
  "os"
  "os/exec"
  "path/filepath"
  "strconv"
)

const (
  EXEC_NAME string = "exec.py"
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
