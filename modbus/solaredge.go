package solaredge

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/simonvetter/modbus"
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

func bytesToString(b []byte) string {
	n := bytes.IndexByte(b, 0)
	if n == -1 {
		return string(b)
	}
	return string(b[:n])
}

type Model interface {
	ModelName() string
	NumRegisters() int
	BaseAddress() int
	Stride() int
}

func readModel[M Model](mb *modbus.ModbusClient, index int) (M, error) {
	var m M

	address := m.BaseAddress() + index*m.Stride()
	data, err := mb.ReadBytes(uint16(address), uint16(m.NumRegisters()*2), modbus.HOLDING_REGISTER)
	if err != nil {
		return *new(M), fmt.Errorf("error reading %s registers: %v", m.ModelName(), err)
	}

	if len(data) != m.NumRegisters()*2 {
		return m, fmt.Errorf("improper data size: expected %d but got %d", m.NumRegisters()*2, len(data))
	}

	buf := bytes.NewReader(data)
	if err := binary.Read(buf, LittleBigEndian, &m); err != nil {
		return m, fmt.Errorf("error parsing %s data: %v", m.ModelName(), err)
	}

	return m, nil
}

// InverterModel holds the SolarEdge SunSpec Implementation for Inverter parameters
// from the implementation technical note:
// https://www.solaredge.com/sites/default/files/sunspec-implementation-technical-note.pdf
type InverterModel struct {
	C_SunSpec_ID     uint32
	C_SunSpec_DID    uint16
	C_SunSpec_Length uint16
	C_Manufacturer   [32]byte
	C_Model          [32]byte
	C_Version        [32]byte // Version defined in SunSpec implementation note as String(16) however is incorrect
	C_SerialNumber   [32]byte
	C_DeviceAddress  uint16

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

func (InverterModel) ModelName() string {
	return "inverter"
}

func (InverterModel) NumRegisters() int {
	return 110
}

func (InverterModel) Stride() int {
	return 0
}

func (InverterModel) BaseAddress() int {
	return 40000 // 0x9C40
}

func (im InverterModel) Manufacturer() string {
	return bytesToString(im.C_Manufacturer[:])
}

func (im InverterModel) Model() string {
	return bytesToString(im.C_Model[:])
}

func (im InverterModel) Version() string {
	return bytesToString(im.C_Version[:])
}

func (im InverterModel) SerialNumber() string {
	return bytesToString(im.C_SerialNumber[:])
}

type MeterModel struct {
	// Common Block
	C_SunSpec_DID    uint16
	C_SunSpec_Length uint16
	C_Manufacturer   [32]byte
	C_Model          [32]byte
	C_Option         [16]byte
	C_Version        [16]byte
	C_SerialNumber   [32]byte
	C_DeviceAddress  uint16

	// Identification
	SunSpec_DID    uint16
	SunSpec_Length uint16

	// Current
	M_AC_Current    uint16
	M_AC_CurrentA   uint16
	M_AC_CurrentB   uint16
	M_AC_CurrentC   uint16
	M_AC_Current_SF int16

	// Voltage
	M_AC_VoltageLN  uint16
	M_AC_VoltageAN  uint16
	M_AC_VoltageBN  uint16
	M_AC_VoltageCN  uint16
	M_AC_VoltageLL  uint16
	M_AC_VoltageAB  uint16
	M_AC_VoltageBC  uint16
	M_AC_VoltageCA  uint16
	M_AC_Voltage_SF int16

	// Frequency
	M_AC_Frequency    uint16
	M_AC_Frequency_SF int16

	// Power
	M_AC_Power    int16
	M_AC_Power_A  int16
	M_AC_Power_B  int16
	M_AC_Power_C  int16
	M_AC_Power_SF int16
	M_AC_VA       uint16
	M_AC_VA_A     uint16
	M_AC_VA_B     uint16
	M_AC_VA_C     uint16
	M_AC_VA_SF    int16
	M_AC_VAR      uint16
	M_AC_VAR_A    uint16
	M_AC_VAR_B    uint16
	M_AC_VAR_C    uint16
	M_AC_VAR_SF   int16
	M_AC_PF       uint16
	M_AC_PF_A     uint16
	M_AC_PF_B     uint16
	M_AC_PF_C     uint16
	M_AC_PF_SF    int16

	// Accumulated Energy
	M_Exported    uint32
	M_Exported_A  uint32
	M_Exported_B  uint32
	M_Exported_C  uint32
	M_Imported    uint32
	M_Imported_A  uint32
	M_Imported_B  uint32
	M_Imported_C  uint32
	M_Energy_W_SF int16
}

func (MeterModel) ModelName() string {
	return "meter"
}

func (MeterModel) NumRegisters() int {
	return 122
}

func (MeterModel) BaseAddress() int {
	return 40121 // 0x9CB9
}

func (MeterModel) Stride() int {
	return 174 // 0xae
}

func (mm MeterModel) Manufacturer() string {
	return bytesToString(mm.C_Manufacturer[:])
}

func (mm MeterModel) Model() string {
	return bytesToString(mm.C_Model[:])
}

func (mm MeterModel) Option() string {
	return bytesToString(mm.C_Option[:])
}

func (mm MeterModel) Version() string {
	return bytesToString(mm.C_Version[:])
}

func (mm MeterModel) SerialNumber() string {
	return bytesToString(mm.C_SerialNumber[:])
}

type BatteryInfoModel struct {
	C_Manufacturer  [32]byte
	C_Model         [32]byte
	C_Version       [32]byte
	C_SerialNumber  [32]byte
	C_DeviceAddress uint16
	C_SunSpec_DID   uint16

	RatedEnergy                     float32 // Rated Energy [Wh]
	MaximumChargeContinuousPower    float32 // Maximum Charge Continuous Power [W]
	MaximumDischargeContinuousPower float32 // Maximum Discharge Continuous Power [W]
	MaximumChargePeakPower          float32 // Maximum Charge Peak Power [W]
	MaximumDischargePeakPower       float32 // Maximum Discharge Peak Power [W]
}

// byte order = big endian, word order = little endian
type littleBigEndian struct{}

func (littleBigEndian) Uint16(b []byte) uint16 {
	return binary.BigEndian.Uint16(b)
}

func (littleBigEndian) PutUint16(b []byte, v uint16) {
	binary.BigEndian.PutUint16(b, v)
}

func (littleBigEndian) Uint32(b []byte) uint32 {
	_ = b[3] // bounds check hint to compiler; see golang.org/issue/14808
	return binary.BigEndian.Uint32([]byte{b[2], b[3], b[0], b[1]})
}

func (littleBigEndian) PutUint32(b []byte, v uint32) {
	_ = b[3] // early bounds check to guarantee safety of writes below

	binary.BigEndian.PutUint32(b, v)
	// swap words
	b[0], b[1], b[2], b[3] = b[2], b[3], b[0], b[1]
}

func (littleBigEndian) Uint64(b []byte) uint64 {
	_ = b[7] // bounds check hint to compiler; see golang.org/issue/14808
	return binary.BigEndian.Uint64([]byte{b[6], b[7], b[4], b[5], b[2], b[3], b[0], b[1]})
}

func (littleBigEndian) PutUint64(b []byte, v uint64) {
	_ = b[7] // early bounds check to guarantee safety of writes below
	binary.BigEndian.PutUint64(b, v)
	// swap words
	b[0], b[1], b[2], b[3], b[4], b[5], b[6], b[7] = b[6], b[7], b[4], b[5], b[2], b[3], b[0], b[1]
}

func (littleBigEndian) String() string { return "LittleBigEndian" }

var LittleBigEndian littleBigEndian

func (BatteryInfoModel) ModelName() string {
	return "battery info"
}

func (bim BatteryInfoModel) Manufacturer() string {
	return bytesToString(bim.C_Manufacturer[:])
}

func (bim BatteryInfoModel) Model() string {
	return bytesToString(bim.C_Model[:])
}

func (bim BatteryInfoModel) Version() string {
	return bytesToString(bim.C_Version[:])
}

func (bim BatteryInfoModel) SerialNumber() string {
	return bytesToString(bim.C_SerialNumber[:])
}

func (BatteryInfoModel) NumRegisters() int {
	return 76
}

func (BatteryInfoModel) BaseAddress() int {
	return 57600 // 0xE100
}

func (BatteryInfoModel) Stride() int {
	return 256 // 0x100
}

type BatteryModel struct {
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

	//EventLog         uint16 // 0xe18a - Event Log
	//_                [12]byte
	//EventLogInternal uint16 // 0xe192 - Internal Event Log
	//_                [12]byte
}

func (BatteryModel) ModelName() string {
	return "battery"
}

func (BatteryModel) NumRegisters() int {
	return 30
}

func (BatteryModel) BaseAddress() int {
	return 57708 // 0xE16C
}

func (BatteryModel) Stride() int {
	return 256 // 0x100
}

func ReadInverter(mb *modbus.ModbusClient, index int) (InverterModel, error) {
	return readModel[InverterModel](mb, index)
}

func ReadMeter(mb *modbus.ModbusClient, index int) (MeterModel, error) {
	return readModel[MeterModel](mb, index)
}

func ReadBatteryInfo(mb *modbus.ModbusClient, index int) (BatteryInfoModel, error) {
	return readModel[BatteryInfoModel](mb, index)
}

func ReadBattery(mb *modbus.ModbusClient, index int) (BatteryModel, error) {
	return readModel[BatteryModel](mb, index)
}
