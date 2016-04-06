package main

import (
	"log"
	"flag"

	"fmulink"
	"statusServer"
	"cloudlink"
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

func main() {
	flag.Parse()
	log.SetPrefix("[MON] ")

	//
	// Cloud Listeners
	//
	var cl *cloudlink.CloudLink
	var err error
	if cl, err = cloudlink.NewCloudLink(); err != nil {
		panic(err)
	} else {
		go cl.Serve()
	}

	//
	// Status Server
	//
	status := statusServer.NewStatusServer(statusServer.SERVE_ADDRESS)
	go status.Serve()

	//
	// MAVLink UDP Listener
	//
	go fmulink.Serve(linkPath, outputs, cl)

	log.Println("Listening.")

	// Run forever
	for {
		select {
		}
	}
}
