package config

import (
  "flag"
)

func init() {
  flag.Parse()

  Version = "alpha-" + gitHash[len(gitHash)-8:]
}

var (
    LinkPath  = flag.String(  "master", "127.0.0.1:14550", 	  "Flight controller address, as either a UDP Address or serial device path.")
    Output    = flag.String(  "output", "", 									"Create outputs for other apps to connect to the FC.")
    UseNsh    = flag.Bool(    "shell",  false,  						  "Puts FC in shell mode, allowing access to the debug shell.")

    // set by the linker
    gitHash   string

    Version   string
)
