package main

import (
	"fmt"
	"fmulink"
	"statusServer"
)

const (
	// Default address, if otherwise not specified by cli args. IP is the cloud,
	// 4002 is the dronedp listening port.
	DEFAULT_DSC_ADDRESS = "24.234.109.135:4002"
)

func main() {

	//
	// Status Server
	//
	status := statusServer.NewStatusServer(statusServer.SERVE_ADDRESS)
	go status.Serve()

	//
	// MAVLink UDP Listener
	//
	port := "127.0.0.1:14550"
	fmt.Printf("[MON] Listening.\n")
	fmulink.Serve(port)
}
