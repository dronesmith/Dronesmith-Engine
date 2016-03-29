package main

import (
	"log"
	"flag"

	"fmulink"
	"statusServer"
)

const (
	// Default address, if otherwise not specified by cli args. IP is the cloud,
	// 4002 is the dronedp listening port.
	DEFAULT_DSC_ADDRESS = "24.234.109.135:4002"
)

var (
	linkPath = flag.String("master", "127.0.0.1:14550", "Path to connect lucimon to.")
	outputs = flag.String("output", "", "Additional outputs.")
)

type LinkManager interface {
	Serve()
}

func main() {
	flag.Parse()
	log.SetPrefix("[MON] ")

	//
	// Status Server
	//
	status := statusServer.NewStatusServer(statusServer.SERVE_ADDRESS)
	go status.Serve()

	//
	// MAVLink UDP Listener
	//
	// port := "127.0.0.1:14550"
	go fmulink.Serve(linkPath, outputs)

	log.Println("Listening.")

	// Run forever
	for {
		select {
		}
	}
}
