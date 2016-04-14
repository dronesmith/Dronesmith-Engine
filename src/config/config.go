package config

import (
  "flag"
  "log"
  "os"
)

func init() {
  flag.Parse()

  Version = "alpha-" + gitHash[len(gitHash)-8:]

  logFile, _ = os.Create("dslink.log")
  logger = log.New(logFile, "[MON] ", log.LstdFlags)

}

const (
  LOG_DEBUG = iota
  LOG_INFO
  LOG_WARN
  LOG_ERROR
)

var (
    LinkPath  = flag.String(  "master", "127.0.0.1:14550", 	  "Flight controller address, as either a UDP Address or serial device path.")
    Output    = flag.String(  "output", "", 									"Create outputs for other apps to connect to the FC.")
    UseNsh    = flag.Bool(    "shell",  false,  						  "Puts FC in shell mode, allowing access to the debug shell.")

    // set by the linker
    gitHash   string

    logFile *os.File
    logger *log.Logger

    Version   string
)

func Log(level int, vals... string) {
    switch level {
      // debugs and warnings don't get saved to a file. This is to avoid clutter.
    case LOG_DEBUG:
      fallthrough
    case LOG_WARN:
      log.Println(vals)
    case LOG_ERROR:
      log.Println(vals)
      logger.Println(vals)
    case LOG_INFO:
      log.Println(vals)
      logger.Println(vals)
    }
}
