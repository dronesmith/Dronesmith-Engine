package fmulink

import (
  "fmt"
  "log"
  "net"

  "mavlink/parser"
)

type FMUStatus struct {
  // ATTITUDE
  AttitudeEst       string

  // RC_CHANNELS
  RadioControl      string

  // HIGHRES_IMU
  Sensors           string

  // SYS_STATUS
  Status            string

  // BATTERY_STATUS
  Battery           string
}

func ListenAndServe(addr string) {

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.Fatal(err)
	}

	conn, listenerr := net.ListenUDP("udp", udpAddr)
	if listenerr != nil {
		log.Fatal(listenerr)
	}

	log.Println("listening on", udpAddr)

	dec := mavlink.NewDecoder(conn)

	for {
		pkt, err := dec.Decode()
		if err != nil {
			log.Println("Decode fail:", err)
			// if pkt != nil {
			// 	log.Println(*pkt)
			// }
			// continue
		}

    switch pkt.MsgID {
    case mavlink.MSG_ID_PARAM_VALUE:
        var pv mavlink.ParamValue
        if err := pv.Unpack(pkt); err == nil {
            // handle param value
            fmt.Println(string(pv.ParamId[:len(pv.ParamId)]), pv.ParamType, pv.ParamValue)
        }
    }

		// log.Println("msg rx:", pkt.MsgID, pkt.Payload)
	}
}
