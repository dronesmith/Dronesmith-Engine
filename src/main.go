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
	// Cloud Listener
	//
	var cl *cloudlink.CloudLink
	var err error
	if cl, err = cloudlink.NewCloudLink(); err != nil {
		panic(err)
	} else {
		go cl.Serve()
	}

	//
	// MAVLink Listener
	//
	go fmulink.Serve(cl)

	//
	// Status Server
	//
	status := statusServer.NewStatusServer(statusServer.SERVE_ADDRESS, cl)
	go status.Serve()

	log.Println("MAIN | Listening.")

	// Run forever
	for {
		select {
		}
	}
}
