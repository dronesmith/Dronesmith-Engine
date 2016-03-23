package fmulink

import (

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
