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
		// It's possible there may be no connection due to network being down,
		// so keep trying every few seconds to serve.
		go cl.Monitor()
	}

	config.Log(config.LOG_INFO, "===============================================================")
	config.Log(config.LOG_INFO, "DRONESMITH LINK ver", config.Version)
	config.Log(config.LOG_INFO, "===============================================================")

	//
	// MAVLink Listener
	//
	go fmulink.Serve(cl)

	//
	// Status Server
	//
	status := statusServer.NewStatusServer(*config.StatusAddress, cl)
	status.Serve()
}
