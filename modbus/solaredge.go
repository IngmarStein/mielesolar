package solaredge

import (
	"bytes"
	"encoding/binary"
	"errors"
)

const (
	I_STATUS_OFF           = 1 // Off
	I_STATUS_SLEEPING      = 2 // Sleeping (auto-shutdown) – Night mode
	I_STATUS_STARTING      = 3 // Grid Monitoring/wake-up
	I_STATUS_MPPT          = 4 // Inverter is ON and producing power
	I_STATUS_THROTTLED     = 5 // Production (curtailed)
	I_STATUS_SHUTTING_DOWN = 6 //  Shutting down
	I_STATUS_FAULT         = 7 // Fault
	I_STATUS_STANDBY       = 8 // Maintenance/setup

	B_STATUS_OFF         = 1
	B_STATUS_EMPTY       = 2
	B_STATUS_DISCHARGING = 3
	B_STATUS_CHARGING    = 4
	B_STATUS_FULL        = 5
	B_STATUS_HOLDING     = 6
	B_STATUS_TESTING     = 7
)

// CommonModel holds the SolarEdge SunSpec Implementation for Common parameters
// from the implementation technical note:
// https://www.solaredge.com/sites/default/files/sunspec-implementation-technical-note.pdf
type CommonModel struct {
	C_SunSpec_ID     uint32
	C_SunSpec_DID    uint16
	C_SunSpec_Length uint16
	C_Manufacturer   [32]byte
	C_Model          [32]byte
	C_Version        [32]byte // Version defined in SunSpec implementation note as String(16) however is incorrect
	C_SerialNumber   [32]byte
	C_DeviceAddress  uint16
}

type CommonMeter struct {
	C_SunSpec_DID    uint16
	C_SunSpec_Length uint16
	C_Manufacturer   [32]byte
	C_Model          [32]byte
	C_Option         [16]byte
	C_Version        [16]byte
	C_SerialNumber   [16]byte
	C_DeviceAddress  uint16
}

type CommonBattery struct {
	C_Manufacturer [16]byte
	C_Model        [16]byte
	C_Version      [16]byte
	C_SerialNumber [16]byte
}

func parseModel[M any](data []byte, order binary.ByteOrder, expectedSize int) (M, error) {
	if len(data) != expectedSize {
		return *new(M), errors.New("improper data size")
	}

	buf := bytes.NewReader(data)

	var m M
	if err := binary.Read(buf, order, &m); err != nil {
		return *new(M), err
	}

	return m, nil
}

// NewCommonModel takes a block of data read from the Modbus TCP connection and returns a new
// populated struct
func NewCommonModel(data []byte) (CommonModel, error) {
	return parseModel[CommonModel](data, binary.BigEndian, 140)
}

func NewCommonMeter(data []byte) (CommonMeter, error) {
	return parseModel[CommonMeter](data, binary.BigEndian, 130)
}

func NewCommonBattery(data []byte) (CommonBattery, error) {
	return parseModel[CommonBattery](data, binary.LittleEndian, 64)
}

// InverterModel holds the SolarEdge SunSpec Implementation for Inverter parameters
// from the implementation technical note:
// https://www.solaredge.com/sites/default/files/sunspec-implementation-technical-note.pdf
type InverterModel struct {
	SunSpec_DID      uint16
	SunSpec_Length   uint16
	AC_Current       uint16
	AC_CurrentA      uint16
	AC_CurrentB      uint16
	AC_CurrentC      uint16
	AC_Current_SF    int16
	AC_VoltageAB     uint16
	AC_VoltageBC     uint16
	AC_VoltageCA     uint16
	AC_VoltageAN     uint16
	AC_VoltageBN     uint16
	AC_VoltageCN     uint16
	AC_Voltage_SF    int16
	AC_Power         int16
	AC_Power_SF      int16
	AC_Frequency     uint16
	AC_Frequency_SF  int16
	AC_VA            int16
	AC_VA_SF         int16
	AC_VAR           int16
	AC_VAR_SF        int16
	AC_PF            int16
	AC_PF_SF         int16
	AC_Energy_WH     int32
	AC_Energy_WH_SF  uint16
	DC_Current       uint16
	DC_Current_SF    int16
	DC_Voltage       uint16
	DC_Voltage_SF    int16
	DC_Power         int16
	DC_Power_SF      int16
	Temp_Cabinet     int16
	Temp_Sink        int16
	Temp_Transformer int16
	Temp_Other       int16
	Temp_SF          int16
	Status           uint16
	Status_Vendor    uint16
}

type MeterModel struct {
	SunSpec_DID       uint16
	SunSpec_Length    uint16
	M_AC_Current      uint16
	M_AC_CurrentA     uint16
	M_AC_CurrentB     uint16
	M_AC_CurrentC     uint16
	M_AC_Current_SF   int16
	M_AC_VoltageLN    uint16
	M_AC_VoltageAN    uint16
	M_AC_VoltageBN    uint16
	M_AC_VoltageCN    uint16
	M_AC_VoltageLL    uint16
	M_AC_VoltageAB    uint16
	M_AC_VoltageBC    uint16
	M_AC_VoltageCA    uint16
	M_AC_Voltage_SF   int16
	M_AC_Frequency    uint16
	M_AC_Frequency_SF int16
	M_AC_Power        int16
	M_AC_Power_A      int16
	M_AC_Power_B      int16
	M_AC_Power_C      int16
	M_AC_Power_SF     int16
	M_AC_VA           uint16
	M_AC_VA_A         uint16
	M_AC_VA_B         uint16
	M_AC_VA_C         uint16
	M_AC_VA_SF        int16
	M_AC_VAR          uint16
	M_AC_VAR_A        uint16
	M_AC_VAR_B        uint16
	M_AC_VAR_C        uint16
	M_AC_VAR_SF       int16
	M_AC_PF           uint16
	M_AC_PF_A         uint16
	M_AC_PF_B         uint16
	M_AC_PF_C         uint16
	M_AC_PF_SF        int16
	M_Exported        uint32
	M_Exported_A      uint32
	M_Exported_B      uint32
	M_Exported_C      uint32
	M_Imported        uint32
	M_Imported_A      uint32
	M_Imported_B      uint32
	M_Imported_C      uint32
	M_Energy_W_SF     int16
}

type BatteryModel struct {
	DeviceAddress uint16
	SunSpec_DID   uint16

	RatedEnergy                     float32 // Rated Energy [Wh]
	MaximumChargeContinuousPower    float32 // Maximum Charge Continuous Power [W]
	MaximumDischargeContinuousPower float32 // Maximum Discharge Continuous Power [W]
	MaximumChargePeakPower          float32 // Maximum Charge Peak Power [W]
	MaximumDischargePeakPower       float32 // Maximum Discharge Peak Power [W]

	AverageTemperature float32 // Average Temperature [°C]
	MaximumTemperature float32 // Maximum Temperature [°C]

	InstantaneousVoltage float32 // Instantaneous Voltage [V]
	InstantaneousCurrent float32 // Instantaneous Current [A]
	InstantaneousPower   float32 // Instantaneous Power [W]

	LifetimeExportEnergyCounter uint64 // Total Exported Energy [Wh]
	LifetimeImportEnergyCounter uint64 // Total Imported Energy [Wh]

	MaximumEnergy   float32 // Maximum Energy [Wh]
	AvailableEnergy float32 // Available Energy [Wh]

	SoH float32 // State of Health (SOH) [%]
	SoE float32 // State of Energy (SOE) [%]

	Status         uint32 // Status
	StatusInternal uint32 // Internal Status

	EventLog         uint16 // Event Log
	EventLogInternal uint16 // Internal Event Log
}

// NewInverterModel takes a block of data read from the Modbus TCP connection and returns
// a new populated struct.
func NewInverterModel(data []byte) (InverterModel, error) {
	return parseModel[InverterModel](data, binary.BigEndian, 80)
}

func NewMeterModel(data []byte) (MeterModel, error) {
	return parseModel[MeterModel](data, binary.BigEndian, 210)
}

func NewBatteryModel(data []byte) (BatteryModel, error) {
	return parseModel[BatteryModel](data, binary.LittleEndian, 86)
}
