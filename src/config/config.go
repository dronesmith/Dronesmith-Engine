package config

import (
  "flag"
  "log"
  "os"
  "encoding/json"
  "io/ioutil"
)

func init() {
  flag.Parse()

  Version = "alpha-" + gitHash[len(gitHash)-8:]

  logFile, _ = os.Create(*loggingFile)
  logger = log.New(logFile, "[MON] ", log.LstdFlags)

  file, e := ioutil.ReadFile("./config.json")
   if e != nil {
       logger.Printf("Error: %v\n", e)
   } else {
     var jsontype map[string]interface{}
     json.Unmarshal(file, &jsontype)

     // update config with each type
     if jsontype["flights"] != nil {
       flights := jsontype["flights"].(string)
       FlightLogPath = &flights
     }

     if jsontype["master"] != nil {
       master := jsontype["master"].(string)
       LinkPath = &master
     }

     if jsontype["output"] != nil {
       output := jsontype["output"].(string)
       Output = &output
     }

     if jsontype["status"] != nil {
       status := jsontype["status"].(string)
       StatusAddress = &status
     }

     if jsontype["dsc"] != nil {
       dsc := jsontype["dsc"].(string)
       DSCAddress = &dsc
     }

     if jsontype["dscHttp"] != nil {
       dscHttp := jsontype["dscHttp"].(string)
       DSCHttp = &dscHttp
     }

     if jsontype["setup"] != nil {
       setup := jsontype["setup"].(string)
       SetupPath = &setup
     }

     if jsontype["assets"] != nil {
       assets := jsontype["assets"].(string)
       AssetsPath = &assets
     }

     if jsontype["sync"] != nil {
       syncf := jsontype["sync"].(float64)
       synci := int(syncf)
       SyncThrottle = &synci
     }

     if jsontype["noflights"] != nil {
       noflights := jsontype["noflights"].(bool)
       DisableFlights = &noflights
     }

     if jsontype["remote"] != nil {
       remote := jsontype["remote"].(string)
       Remote = &remote
     }

     if jsontype["log"] != nil {
       log := jsontype["log"].(string)
       loggingFile = &log
     }

     if jsontype["daemon"] != nil {
       dae := jsontype["daemon"].(bool)
       daemon = &dae
     }
   }
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
    StatusAddress   = flag.String(      "status", "127.0.0.1:8080",                 "Address which the status server will serve on. Should be in <IP>:<Port> format.")
    DSCAddress      = flag.String(      "dsc",    "127.0.0.1:4002",                 "Address to talk to DSC. Should be in <IP>:<Port> format.")
    DSCHttp         = flag.String(      "dscHttp", "127.0.0.1:4000",                "HTTP Address to talk to DSC. Should be in <IP>:<Port> format.")
    SetupPath       = flag.String(      "setup",  "/var/lib/edison_config_tools/",  "Path to files for initial setup.") // TODO change this to `/var/lib/lmon-setup`
    AssetsPath      = flag.String(      "assets", "/opt/dslink/",                   "Path to assets")

    FlightLogPath   = flag.String(      "flights", "/opt/dslink/flightdata",        "Path to store flight log data")

    SyncThrottle    = flag.Int(         "sync",    1000,                             "Update time period to sync flight data")

    DisableFlights  = flag.Bool(        "noflights", false,                        "Disables flight logging")

    Remote          = flag.String(      "remote",  "",                              "Specify a remote UDP address. Required for certain datalinks, such as SITL mode.")

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
      if !*daemon {
        log.SetPrefix("[DEBUG] ")
        log.Println(vals...)
      }
    case LOG_WARN:
      if !*daemon {
        log.SetPrefix("[WARN] ")
        log.Println(vals...)
      }
    case LOG_ERROR:
      logger.SetPrefix("[ERROR] ")
      logger.Println(vals...)
      if !*daemon {
        log.SetPrefix("[ERROR] ")
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
