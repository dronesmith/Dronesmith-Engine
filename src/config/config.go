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
    LinkPath        = flag.String(      "master", "127.0.0.1:14550", 	  "Flight controller address, as either a UDP Address or serial device path.")
    Output          = flag.String(      "output", "", 									"Create outputs for other apps to connect to the FC.")
    // UseNsh    = flag.Bool(    "shell",  false,  						  "Puts FC in shell mode, allowing access to the debug shell.")
    StatusAddress   = flag.String(      "status", "127.0.0.1:8080",     "Address which the status server will serve on. Shoild be in <IP>:<Port> format.")
    DSCAddress      = flag.String(      "dsc",    "127.0.0.1:4002",     "Address to talk to DSC. Should be in <IP>:<Port> format.")
    loggingFile     = flag.String(      "log",    "dslink.log",         "Log File path and name, relative to the GOPATH.")

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
      log.SetPrefix("[DEBUG] ")
      log.Println(vals...)
    case LOG_WARN:
      log.SetPrefix("[WARN] ")
      log.Println(vals...)
    case LOG_ERROR:
      log.SetPrefix("!! [ERROR] ")
      logger.SetPrefix("!! [ERROR] ")
      log.Println(vals...)
      logger.Println(vals...)
    case LOG_INFO:
      log.SetPrefix("[INFO] ")
      logger.SetPrefix("[INFO] ")
      log.Println(vals...)
      logger.Println(vals...)
    }
}
