package main

import (
	"fmulink"
	"statusServer"
	"cloudlink"
	"config"
)

func main() {

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
	status := statusServer.NewStatusServer(*config.StatusAddress, cl)
	go status.Serve()

	config.Log(config.LOG_INFO, "===============================================================")
	config.Log(config.LOG_INFO, "DRONESMITH LINK ver", config.Version)
	config.Log(config.LOG_INFO, "===============================================================")

	// Run forever
	for {
		select {
		}
	}
}
