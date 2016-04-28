package translator

import (
  "time"
)

const (
  // PROTO_VER = C`ldflags="git --HEAD"`
)

type Translator interface {
 /* vi-select !nil */
}

//
// Drone Meta Data. This information remains constant during flights.
// Contains general information about the connected vehicle.
//

// UAV Types
type UAVType int
const (
  TYPE_UNKNOWN UAVType = iota
  TYPE_QUADROTOR
  TYPE_HEXROTOR
  TYPE_OCTROTOR
  TYPE_FIXEDWING
  TYPE_VTOL
  TYPE_ROVER
  TYPE_CUSTOM
)

// Protocol Types
type ProtocolType int
const (
  PROTO_UNKNOWN ProtocolType = iota
  PROTO_MAVLINK
  PROTO_PARROT
  PROTO_DJI
)

// Peripheral Types
type PeripheralType int
const (
  PERIPH_UNKNOWN PeripheralType = iota
  PERIPH_IMU
  PERIPH_GPS
  PERIPH_OPTICALFLOW
  PERIPH_RANGEFINDER
  PERIPH_COMPUTERVISION
  PERIPH_CUSTOM
)

// Represents an external peripheral connected to the FMU.
type Peripheral int {
  Kind    PeripheralType
  Values  map[string]interface{}
}

// Contains general metadata about the connected FMU.
// Doubles as the header for saved/synced flight missions as files and on DSC.
type Drone struct {
  Kind            UAVType        // Type of vehicle

  // Mission related information
  InMission       bool            `json:"-"`         // Sets to true when the user requests to sync data with DSC, and/or drone is armed.
  MissionStart    time.Time                          // Start mission time stamp. Indicates mission is taking place.
  MissionEnd      time.Time       `json:"omitempty"` // End mission time stamp. Indicates mission is completed.
  MissionPaylist  []FlightControl `json:"omitempty"` // Mission flight control points, if any
  MissionId       string          `json:"omitempty"` // Unique Id that is sent back from DSC when a mission sync is completed.

  Protocol        ProtocolType  // Type of protocol

  // In MAVLink's context this includes parameters, System Id, Comp Id...
  ProtocolInfo    map[string]interface{} `json:"omitempty"`
}

//
// Telemetry messages. These are types of telemetry that can be relayed to DSC.
// Each of these messages is associated with a timestamp. Each time one is updated,
// the timestamp is updated. The telemetry data is sent in full to DSC on a certain period
// with the messages whose timestamps are recent enough.
//

// Represents a telemetry message to be relayed to DSC or synced to a file.
type Telemetry struct {
  stamp     time.Time

  Vfr       VFR           `json:"omitempty"`
  Att       Attitude      `json:"omitempty"`
  Imu       IMU           `json:"omitempty"`
  Fc        FlightControl `json:"omitempty"`
  Pow       Power         `json:"omitempty"`
  Gps       GPS           `json:"omitempty"`
  Flow      RangeFinder   `json:"omitempty"`
  Ctrl      Control       `json:"omitempty"`
  Act       Motors        `json:"omitempty"`

  // List of external peripherals connected to the FMU and their values.
  Peripherals     []Peripheral    `json:"omitempty"`
}

// Visual Flight Rules, relate to where the vehicle is going.
type VFR struct {
  Altitude      float32 // Altitude, MSL, Meters
  VelocityAir   float32 // m/s
  VelocityGnd   float32 // m/s
  Heading       float32 // degrees, relative to North (North = 0)
}

// Attitude information
type Attitude struct {
  Roll    float32 // degrees
  Pitch   float32 // degrees
  Yaw     float32 // degrees
}

// IMU Sensors.
type IMU struct {
  GyroX   float32 // degrees / sec
  GyroY   float32
  GyroZ   float32

  AccelX  float32 // m/s^2
  AccelY  float32
  AccelZ  float32

  MagX    float32 // Gauss
  MagY    float32
  MagZ    float32

  BaroAbs   float32 // mbar
  BaroDiff  float32

  Temp    int16 // C

  IsRaw   bool
  IsRawPressure bool
}

// Contains control targets for operating the vehicle.
// These are used to create mission waypoint during guided flight, when seen as
// a telemetry message, it contains the target setpoint information of the FMU.
type FlightControl  struct {

  // Orientation Control
  RollAmt   float32
  PitchAmt  float32
  YawAmt    float32

  // Global Position Control
  PosLat    int32
  PosLon    int32

  // Home Location
  HomeLat   int32
  HomeLon   int32
  HomeAlt   int32

  // Local Position Control
  PosX      float32
  PosY      float32
  PosZ      float32

  // Motor Control
  MotorAmt [][]uint16
}

// Power/Battery Information
type PowerStatus int
const (
  POW_UNKNOWN PowerStatus = iota
  POW_CHARGING
  POW_CHARGED
  POW_NOMINAL
  POW_WARN
  POW_CRIT
)

type Power struct {
  Status    PowerStatus
  Percent   float32
  Temp      int16 // C
}

// Global Position information, generally taken from GPS unit.
type GPS struct {
  Latitude    int32 // degrees * 1e7
  Longitude   int32 // degrees * 1e7
  Altitude    int32 // kilometers
  Satellites  int8
}

// Local position information, generally taken from Optical flow or a range finding device.
type RangeFinder struct {
  LocalX  float32
  LocalY  float32
  LocalZ  float32
}

//
// Control information.
// This message contains the control mode, control inputs, and mission information
//
type ControlMode int
const (
  CTRL_ACRO ControlMode = iota
  CTRL_MANUAL
  CTRL_ALTHOLD
  CTRL_POSHOLD
  CTRL_GUIDED
  CTRL_AUTO
  CTRL_CUSTOM
)

type ControlType int
const (
  CTRL_NONE ControlType = iota
  CTRL_THROTTLE
  CTRL_YAW
  CTRL_PITCH
  CTRL_ROLL
  CTRL_TRIGGER
  CTRL_CUSTOM
)

type ControlInput struct {
  Kind    ControlType
  Value   float32 // percent
}

type Control struct {
  Mode        ControlMode
  Armed       bool

  // First index: Port, second index: channel
  Inputs      [][]ControlInput
  IsRaw       bool

  CurrMission FlightControl
  MissionSeq  int // current mission
  MissionCnt  int // number of missions

  Signal      float32 // percent
}

//
// Motor Feedback information, organized as motor groups
//
type Motors struct {
  Values    [][]uint16 // microseconds
}
