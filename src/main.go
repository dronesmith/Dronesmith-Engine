package main

import (
	"log"

	"fmulink"
	"statusServer"
	"cloudlink"
)

const (
	// Default address, if otherwise not specified by cli args. IP is the cloud,
	// 4002 is the dronedp listening port.
	DEFAULT_DSC_ADDRESS string = "24.234.109.135:4002"
)

func main() {
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
	go fmulink.Serve(cl)

	log.Println("MAIN | Listening.")

	// Run forever
	for {
		select {
		}
	}
}
