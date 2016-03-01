package main

import (
	"fmt"
	"net"
	"mavlink"
)

func main() {
	port := "127.0.0.1:14550"
	udpAddress, err := net.ResolveUDPAddr("udp4", port)

	if err != nil {
		fmt.Println("error resolving UDP address on ", port)
		fmt.Println(err)
		return
	}

	fmt.Printf("[MON] Fetching XML...\n")
	mav := mavlink.NewMavlink(mavlink.DEFAULT_MAVLINK_XML)

	fmt.Print("[MON] Initiating connection...\n")
	conn, err := net.ListenUDP("udp", udpAddress)

	if err != nil {
		fmt.Println("error listening on UDP port ", port)
		fmt.Println(err)
		return
	}

	defer conn.Close()

	buf := make([]byte, 2048)

	fmt.Printf("[MON] Listening.\n")

	for {
		n, address, err := conn.ReadFromUDP(buf)

		if err != nil {
			fmt.Println("error reading data from connection")
			fmt.Println(err)
			return
		}

		if address != nil {
			// fmt.Println("got message from", address, " with n = ", n)
			if n > 0 {
				go mav.Parse(buf)
			}
		}
	}
}
