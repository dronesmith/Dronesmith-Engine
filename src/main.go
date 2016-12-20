package main

import (
	"fmulink"
	"statusServer"
	"cloudlink"
	"config"
	"log"
)

func main() {

	//
	// Cloud Listener
	//
	var cl *cloudlink.CloudLink
	var err error
	if cl, err = cloudlink.NewCloudLink(); err != nil {
		log.Fatal(err.Error() + "\nIs your assets path configured?")
	} else {
		// It's possible there may be no connection due to network being down,
		// so keep trying every few seconds to serve.
		go cl.Monitor()
	}

	config.Log(config.LOG_INFO, "===============================================================")
	config.Log(config.LOG_INFO, "DRONESMITH ENGINE ver", config.Version)
	config.Log(config.LOG_INFO, "===============================================================")

	// Needs to be initialized here, since we can't rely on fmulink completing in time.
	fmulink.ConnReady = make(chan bool)

	//
	// MAVLink Listener
	//
	go fmulink.Serve(cl)

	//
	// Status Server
	//
	status := statusServer.NewStatusServer(*config.StatusPort, cl)

	// We got to wait for the FMULink to give us the thumbs up.
	<- fmulink.ConnReady
	status.Serve()
}
