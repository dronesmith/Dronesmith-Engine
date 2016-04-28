package translator

import (
  "config"
  "mavlink/parser"
  "time"
  "math"
)

type MavlinkTranslator struct {
  conn        io.ReadWriter
  dec         *mavlink.Decoder
  enc         *mavlink.Encoder

  output      chan *[]byte

  drone       Drone
  telem       Telemetry
}

func (m *MavlinkTranslator) Parse() {
  if pkt, err := m.dec.Decode(); err != nil {
    config.Log(config.LOG_WARN, "fl: ", "Decode fail:", err)
  } else {

    // Echo raw message to outputs
    m.output <- m.unrollPacket(pkt)

    switch pkt.MsgID {

    case mavlink.MSG_ID_HEARTBEAT:
      var pv mavlink.Heartbeat
      if err := pv.Unpack(pkt); err == nil {
        m.handleHeartBeat(&pv)
      }

    case mavlink.MSG_ID_SYS_STATUS:
      var pv mavlink.SysStatus
      if err := pv.Unpack(pkt); err == nil {
        m.handleSysStatus(&pv)
      }

    case mavlink.MSG_ID_PARAM_VALUE:
      var pv mavlink.ParamValue
      if err := pv.Unpack(pkt); err == nil {
        m.handleParamType(&pv)
      }

    case mavlink.MSG_ID_GPS_RAW_INT:
      var pv mavlink.GpsRawInt
      if err := pv.Unpack(pkt); err == nil {
        updateTimeStamp("Gps")
        m.telem.Gps.Latitude = pv.Lat
        m.telem.Gps.Longitude = pv.Lon
        m.telem.Gps.Satellites = pv.SatellitesVisible
        m.telem.Gps.Altitude = pv.Alt
      }

    case mavlink.MSG_ID_GPS_STATUS:
      var pv mavlink.GpsStatus
      if err := pv.Unpack(pkt); err == nil {
        updateTimeStamp("Gps")
        m.telem.Gps.Satellites = pv.SatellitesVisible
      }

    case mavlink.MSG_ID_SCALED_IMU:
      var pv mavlink.ScaledImu
      if err := pv.Unpack(pkt); err == nil {
        updateTimeStamp("Imu")
        m.telem.Imu.IsRaw = false

        m.telem.Imu.GyroX = m.radToDeg(pv.Xgyro / 1000)
        m.telem.Imu.GyroY = m.radToDeg(pv.Ygyro / 1000)
        m.telem.Imu.GyroZ = m.radToDeg(pv.Zgyro / 1000)

        // NOTE Units are in milliG's I think?
        // See MAVLink documentation. Abreviated as `mg`.
        // Raw accelerometers tend to output this value.
        m.telem.Imu.AccelX = (pv.Xacc / 1000) * 9.807
        m.telem.Imu.AccelY = (pv.Yacc / 1000) * 9.807
        m.telem.Imu.AccelZ = (pv.Zacc / 1000) * 9.807

        m.telem.Imu.MagX = (pv.Xmag / 1000) * 1e4
        m.telem.Imu.MagY = (pv.Ymag / 1000) * 1e4
        m.telem.Imu.MagZ = (pv.Zmag / 1000) * 1e4
      }

    case mavlink.MSG_ID_RAW_IMU:
      var pv mavlink.RawImu
      if err := pv.Unpack(pkt); err == nil {
        updateTimeStamp("Imu")

        m.telem.Imu.IsRaw = true
        m.telem.Imu.GyroX = float32(m.telem.Xgyro)
        m.telem.Imu.GyroY = float32(m.telem.Ygyro)
        m.telem.Imu.GyroZ = float32(m.telem.Zgyro)
        m.telem.Imu.AccelX = float32(m.telem.Xaccel)
        m.telem.Imu.AccelY = float32(m.telem.Yaccel)
        m.telem.Imu.AccelZ = float32(m.telem.Zaccel)
        m.telem.Imu.MagX = float32(m.telem.Xmag)
        m.telem.Imu.MagY = float32(m.telem.Ymag)
        m.telem.Imu.MagZ = float32(m.telem.Zmag)
      }

    case mavlink.MSG_ID_RAW_PRESSURE:
      var pv mavlink.RawPressure
      if err := pv.Unpack(pkt); err == nil {
        updateTimeStamp("Imu")
        m.telem.Imu.IsRawPressure = true

        m.telem.Imu.BaroAbs = pv.PressAbs

        if pv.PressDiff1 != 0 {
          m.telem.Imu.BaroDiff = pv.PressDiff1
        } else if pv.PressDiff2 != 0 {
          m.telem.Imu.BaroDiff = pv.PressDiff2
        }

        m.telem.Imu.Temp = pv.Temperature

      }

    case mavlink.MSG_ID_SCALED_PRESSURE:
      var pv mavlink.ScaledPressure
      if err := pv.Unpack(pkt); err == nil {
        updateTimeStamp("Imu")
        m.telem.Imu.IsRawPressure = false

        m.telem.Imu.BaroAbs = pv.PressAbs
        m.telem.Imu.BaroDiff = pv.PressDiff

        m.telem.Imu.Temp = pv.Temperature * 100.0
      }

    case mavlink.MSG_ID_ATTITUDE:
      var pv mavlink.Attitude
      if err := pv.Unpack(pkt); err == nil {
        updateTimeStamp("Att")

        m.telem.Att.Roll = m.radToDeg(pv.Roll)
        m.telem.Att.Pitch = m.radToDeg(pv.Pitch)
        m.telem.Att.Yaw = m.radToDeg(pv.Yaw)
      }

    case mavlink.MSG_ID_ATTITUDE_QUATERNION:
      var pv mavlink.AttitudeQuaternion
      if err := pv.Unpack(pkt); err == nil {
        // https://en.wikipedia.org/wiki/Conversion_between_quaternions_and_Euler_angles
        m.telem.Att.Roll = math.Atan((2 * (pv.q0 * pv.q1 + pv.q2 * pv.q3)) / (1 - (2 * (pv.q1 * pv.q1 + pv.q2 * pv.q2))))
        m.telem.Att.Pitch = math.Asin(2 * (pv.q0 * pv.q2 - pv.q3 * pv.q1))
        m.telem.Att.Yaw = math.Atan((2 * (pv.q0 * pv.q3 + pv.q1 * pv.q2)) / (1 - (2 * (pv.q2 * pv.q2 + pv.q3 * pv.q3))))
      }

    case mavlink.MSG_ID_LOCAL_POSITION_NED:
      var pv mavlink.LocalPositionNed
      if err := pv.Unpack(pkt); err == nil {
        updateTimeStamp("Flow")
        m.telem.Flow.LocalX = pv.X
        m.telem.Flow.LocalY = pv.Y
        m.telem.Flow.LocalZ = pv.Z
      }

    case mavlink.MSG_ID_GLOBAL_POSITION_INT:
      var pv mavlink.GlobalPosInt
      if err := pv.Unpack(pkt); err == nil {
        updateTimeStamp("Gps")
        m.telem.Gps.Latitude = pv.Lat
        m.telem.Gps.Longitude = pv.Lon
        m.telem.Gps.Altitude = pv.Alt
      }

    case mavlink.MSG_ID_RC_CHANNELS_SCALED:
      var pv mavlink.RcChannelsScaled
      if err := pv.Unpack(pkt); err == nil {
        updateTimeStamp("Ctrl")
        m.telem.Ctrl.IsRaw = false

        port := pv.Port

        // TODO channel mapping

        // Mavlink hardcodes these to 8
        m.telem.Ctrl.Inputs[port][0].Value = ((pv.Chan1Scaled + 10000) / 20000) * 100
        m.telem.Ctrl.Inputs[port][1].Value = ((pv.Chan2Scaled + 10000) / 20000) * 100
        m.telem.Ctrl.Inputs[port][2].Value = ((pv.Chan3Scaled + 10000) / 20000) * 100
        m.telem.Ctrl.Inputs[port][3].Value = ((pv.Chan4Scaled + 10000) / 20000) * 100
        m.telem.Ctrl.Inputs[port][4].Value = ((pv.Chan5Scaled + 10000) / 20000) * 100
        m.telem.Ctrl.Inputs[port][5].Value = ((pv.Chan6Scaled + 10000) / 20000) * 100
        m.telem.Ctrl.Inputs[port][6].Value = ((pv.Chan7Scaled + 10000) / 20000) * 100
        m.telem.Ctrl.Inputs[port][7].Value = ((pv.Chan8Scaled + 10000) / 20000) * 100

        if pv.Rssi != 255 {
          m.telem.Ctrl.Signal = pv.Rssi
        }
      }

    case mavlink.MSG_ID_RC_CHANNELS_RAW:
      var pv mavlink.RcChannelsRaw
      if err := pv.Unpack(pkt); err == nil {
        updateTimeStamp("Ctrl")
        m.telem.Ctrl.IsRaw = true

        port := pv.Port

        // TODO channel mapping

        // MAVLink hardcodes these to 8
        m.telem.Ctrl.Inputs[port][0].Value = pv.Chan1Raw
        m.telem.Ctrl.Inputs[port][1].Value = pv.Chan2Raw
        m.telem.Ctrl.Inputs[port][2].Value = pv.Chan3Raw
        m.telem.Ctrl.Inputs[port][3].Value = pv.Chan4Raw
        m.telem.Ctrl.Inputs[port][4].Value = pv.Chan5Raw
        m.telem.Ctrl.Inputs[port][5].Value = pv.Chan6Raw
        m.telem.Ctrl.Inputs[port][6].Value = pv.Chan7Raw
        m.telem.Ctrl.Inputs[port][7].Value = pv.Chan8Raw

        if pv.Rssi != 255 {
          m.telem.Ctrl.Signal = pv.Rssi
        }
      }

    case mavlink.MSG_ID_SERVO_OUTPUT_RAW:
      var pv mavlink.RcChannelsRaw
      if err := pv.Unpack(pkt); err == nil {
        updateTimeStamp("Act")
        port := pv.Port

        // MAVLink hardcodes these to 8
        m.telem.Act[port][0] = pv.Servo1Raw
        m.telem.Act[port][1] = pv.Servo2Raw
        m.telem.Act[port][2] = pv.Servo3Raw
        m.telem.Act[port][3] = pv.Servo4Raw
        m.telem.Act[port][4] = pv.Servo5Raw
        m.telem.Act[port][5] = pv.Servo6Raw
        m.telem.Act[port][6] = pv.Servo7Raw
        m.telem.Act[port][7] = pv.Servo8Raw
      }

    case mavlink.MSG_ID_MISSION_ITEM:
      var pv mavlink.MissionItem
      if err := pv.Unpack(pkt); err == nil {
        updateTimeStamp("Ctrl")

        switch pv.Frame {
        case mavlink.MAV_FRAME_GLOBAL: fallthrough
        case mavlink.MAV_FRAME_GLOBAL_RELATIVE_ALT: fallthrough
        case mavlink.MAV_FRAME_GLOBAL_INT: fallthrough
        case mavlink.MAV_FRAME_GLOBAL_RELATIVE_ALT_INT: fallthrough
        case mavlink.MAV_FRAME_GLOBAL_TERRAIN_ALT: fallthrough
        case mavlink.MAV_FRAME_GLOBAL_TERRAIN_ALT_INT:
          m.telem.Ctrl.CurrMission.PosLat = pv.X
          m.telem.Ctrl.CurrMission.PosLon = pv.Y
          m.telem.Ctrl.CurrMission.PosZ = pv.Z

        case mavlink.MAV_FRAME_LOCAL_ENU: fallthrough
        case mavlink.MAV_FRAME_LOCAL_OFFSET_NED: fallthrough
        case mavlink.MAV_FRAME_BODY_NED: fallthrough
        case mavlink.MAV_FRAME_BODY_OFFSET_NED:
          m.telem.Ctrl.CurrMission.PosX = pv.X
          m.telem.Ctrl.CurrMission.PosY = pv.Y
          m.telem.Ctrl.CurrMission.PosZ = pv.Z

        default:
          config.Log(config.LOG_WARN, "Invalid coordinate frame for mission: ", pv.Frame)

        }

      }

    case mavlink.MSG_ID_MISSION_CURRENT:
      var pv mavlink.MissionCurrent
      if err := pv.Unpack(pkt); err == nil {
        updateTimeStamp("Ctrl")

        m.telem.Ctrl.MissionSeq = pv.Seq
      }

    case mavlink.MSG_ID_MISSION_COUNT:
      var pv mavlink.MissionCount
      if err := pv.Unpack(pkt); err == nil {
        updateTimeStamp("Ctrl")

        m.telem.Ctrl.MissionCnt = pv.Count
      }

    case mavlink.MSG_ID_GPS_GLOBAL_ORIGIN:
      var pv mavlink.GpsGlobalOrigin
      if err := pv.Unpack(pkt); err == nil {
        updateTimeStamp("Fc")

        m.telem.Fc.HomeLat = pv.Latitude
        m.telem.Fc.HomeLon = pv.Longitude
        m.telem.Fc.HomeAlt = pv.Altitude
      }

    case mavlink.MSG_ID_ATTITUDE_QUATERNION_COV:

    case mavlink.MSG_ID_NAV_CONTROLLER_OUTPUT:

    case mavlink.MSG_ID_GLOBAL_POSITION_INT_COV:

    case mavlink.MSG_ID_LOCAL_POSITION_NED_COV:

    case mavlink.MSG_ID_RC_CHANNELS:

    case mavlink.MSG_ID_MISSION_ITEM_INT:

    case mavlink.MSG_ID_VFR_HUD:
      var pv mavlink.VfrHud
      if err := pv.Unpack(pkt); err == nil {
        updateTimeStamp("Vfr")
        m.telem.Vfr.VelocityAir = pv.Airspeed
        m.telem.Vfr.VelocityGnd = pv.Groundspeed
        m.telem.Vfr.Heading     = pv.Heading
        m.telem.Vfr.Altitude    = pv.Alt
      }

    case mavlink.MSG_ID_ATTITUDE_TARGET:

    case mavlink.POSITION_TARGET_LOCAL_NED:

    case mavlink.MSG_ID_POSITION_TARGET_GLOBAL_INT:

    case mavlink.MSG_ID_CONTROL_SYSTEM_STATE:

    case mavlink.MSG_ID_OPTICAL_FLOW:

    case mavlink.MSG_ID_HIGHRES_IMU:

    case mavlink.MSG_ID_OPTICAL_FLOW_RAD:

    case mavlink.MSG_ID_RADIO_STATUS:

    case mavlink.MSG_ID_BATTERY_STATUS:
      var pv mavlink.BatteryStatus
      if err := pv.Unpack(pkt); err == nil {
        updateTimeStamp("Pow")

        switch pv.BatteryFunction {
        case mavlink.MAV_BATTERY_FUNCTION_UNKNOWN:
          m.telem.Pow.Status = POW_UNKNOWN
        default:
        }

        m.telem.Pow.Percent = pv.BatteryRemaining

        if pv.BatteryRemaining != -1 {
          if pv.BatteryRemaining > 30 {
            m.telem.Pow.Status = POW_NOMINAL
          } else if pv.BatteryRemaining > 15 {
            m.telem.Pow.Status = POW_WARN
          } else {
            m.telem.Pow.Status = POW_CRIT
          }
        } else {
          m.telem.Status = POW_CHARGED
        }

        m.telem.Pow.Temp = pv.Temperature * 100
      }


    case mavlink.MSG_ID_AUTOPILOT_VERSION:
      var pv mavlink.AutopilotVersion
      if err := pv.Unpack(pkt); err == nil {
        m.handleAP(&pv)
      }


    case mavlink.MSG_ID_HOME_POSITION:

    case mavlink.MSG_ID_EXTENDED_SYS_STATE:

    case mavlink.MSG_ID_DISTANCE_SENSOR:

    case mavlink.MSG_ID_ACTUATOR_CONTROL_TARGET:

    case mavlink.MSG_ID_ALTITUDE:

    case mavlink.MSG_ID_CONTROL_SYSTEM_STATE:

    // Not sure what to do with these messages.
    case mavlink.MSG_ID_SCALED_IMU2: fallthrough

    default:
      // unsupported messages
      // TODO add to raw pool
    }
  }
}

func (m *MavlinkTranslator) GetRawOutput() *[]byte {
  return <- m.output
}

func (m *MavlinkTranslator) updateTimeStamp(key string) {
  m.Stamps[key] = time.Now()
}

func (m *MavlinkTranslator) radToDeg(rads float32) float32 {
  return rads * (180 / math.Pi)
}


// Get a raw byte array from a mavlink packet
func (m *MavlinkTranslator) unrollPacket(pkt *mavlink.Packet) *[]byte {
  plen := len(pkt.Payload)
  buf := make([]byte, plen+8)

  buf[0] = 0xFE // header
  buf[1] = byte(plen)
  buf[2] = byte(pkt.SeqID)
  buf[3] = byte(pkt.SysID)
  buf[4] = byte(pkt.CompID)
  buf[5] = byte(pkt.MsgID)

  for i := 0; i < plen; i++ {
    buf[i+6] = pkt.Payload[i]
  }

  buf[plen+6] = byte(pkt.Checksum & 0xFF)
  buf[plen+7] = byte(pkt.Checksum >> 8)

  return &buf
}
