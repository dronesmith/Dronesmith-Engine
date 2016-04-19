package config

import (
  "flag"
  "log"
  "os"
)

func init() {
  flag.Parse()

  Version = "alpha-" + gitHash[len(gitHash)-8:]

  logFile, _ = os.Create(*loggingFile)
  logger = log.New(logFile, "[MON] ", log.LstdFlags)

}

const (
  LOG_DEBUG = iota
  LOG_INFO
  LOG_WARN
  LOG_ERROR
)

var (
    // Config flags
    LinkPath        = flag.String(      "master", "127.0.0.1:14550", 	              "Flight controller address, as either a UDP Address or serial device path.")
    Output          = flag.String(      "output", "", 									            "Create outputs for other apps to connect to the FC.")
    // UseNsh    = flag.Bool(    "shell",  false,  						  "Puts FC in shell mode, allowing access to the debug shell.")
    StatusAddress   = flag.String(      "status", "127.0.0.1:8080",                 "Address which the status server will serve on. Shoild be in <IP>:<Port> format.")
    DSCAddress      = flag.String(      "dsc",    "127.0.0.1:4002",                 "Address to talk to DSC. Should be in <IP>:<Port> format.")
    SetupPath       = flag.String(      "setup",  "/var/lib/edison_config_tools/",  "Path to files for initial setup.") // TODO change this to `/var/lib/lmon-setup`

    // Privates
    loggingFile     = flag.String(      "log",    "dslink.log",                     "Log File path and name, relative to the GOPATH.")
    daemon          = flag.Bool(        "daemon", false,                            "Surpresses console logging if true.")

    // set by the linker
    gitHash   string

    logFile *os.File
    logger *log.Logger

    Version   string
)

func Log(level int, vals... interface{}) {
    switch level {
      // debugs and warnings don't get saved to a file. This is to avoid clutter.
    case LOG_DEBUG:
      if ! *daemon {
        log.SetPrefix("[DEBUG] ")
        log.Println(vals...)
      }
    case LOG_WARN:
      if ! *daemon {
        log.SetPrefix("[WARN] ")
        log.Println(vals...)
      }
    case LOG_ERROR:
      logger.SetPrefix("!! [ERROR] ")
      logger.Println(vals...)
      if ! *daemon {
        log.SetPrefix("!! [ERROR] ")
        log.Println(vals...)
      }
    case LOG_INFO:
      if !*daemon {
        log.SetPrefix("[INFO] ")
        log.Println(vals...)
      }

      logger.SetPrefix("[INFO] ")
      logger.Println(vals...)
    }
}
