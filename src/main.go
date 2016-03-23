package main

import (
	"fmt"
	// "net"

	"fmulink"

	// "mavlink"
	"setupServer"
)

const (
	// Default address, if otherwise not specified by cli args. IP is the cloud,
	// 4002 is the dronedp listening port.
	DEFAULT_DSC_ADDRESS = "24.234.109.135:4002"
)

func main() {

	// Launch the http server
	go setupServer.Init()

	//
	// MAVLink UDP Listener
	//
	port := "127.0.0.1:14550"
	// udpAddress, err := net.ResolveUDPAddr("udp4", port)

	// if err != nil {
	// 	fmt.Println("error resolving UDP address on ", port)
	// 	fmt.Println(err)
	// 	return
	// }
	//
	// fmt.Printf("[MON] Fetching XML...\n")
	// // mav := mavlink.NewMavlink(mavlink.DEFAULT_MAVLINK_XML)
	//
	// fmt.Print("[MON] Initiating connection...\n")
	// conn, err := net.ListenUDP("udp", udpAddress)
	//
	// if err != nil {
	// 	fmt.Println("error listening on UDP port ", port)
	// 	fmt.Println(err)
	// 	return
	// }
	//
	// defer conn.Close()

	// buf := make([]byte, 2048)


	//
	// DroneDP UDP
	//
	// if cloudUdpAddr, err := net.ResolveUDPAddr("udp4", DEFAULT_DSC_ADDRESS); err != nil {
	// 	panic(err)
	// } else {
	// 	if cloudConn, err := net.ListenUDP("udp", cloudUdpAddr); err != nil {
	// 		panic(err)
	// 	}
	// }

	fmt.Printf("[MON] Listening.\n")

	fmulink.ListenAndServe(port)

	// for {
	// 	n, address, err := conn.ReadFromUDP(buf)
	//
	// 	if err != nil {
	// 		fmt.Println("error reading data from connection")
	// 		fmt.Println(err)
	// 		return
	// 	}
	//
	// 	if address != nil {
	// 		// fmt.Println("got message from", address, " with n = ", n)
	// 		if n > 0 {
	// 			go mav.Parse(buf)
	// 		}
	// 	}
	//
	// 	// n, address, err := cloudConn.ReadFromUDP(buf)
	// 	//
	// 	// if err != nil {
	// 	// 	fmt.Println("error reading data from connection")
	// 	// 	fmt.Println(err)
	// 	// 	return
	// 	// }
	//
	//
	// }
}
