package solaredge

import (
	"bytes"
	"errors"
	"github.com/u-root/u-root/pkg/uio"
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
)

// CommonModel holds the SolarEdge SunSpec Implementation for Common parameters
// from the implementation technical note:
// https://www.solaredge.com/sites/default/files/sunspec-implementation-technical-note.pdf
type CommonModel struct {
	C_SunSpec_ID     uint32
	C_SunSpec_DID    uint16
	C_SunSpec_Length uint16
	C_Manufacturer   []byte
	C_Model          []byte
	C_Version        []byte // Version defined in SunSpec implementation note as String(16) however is incorrect
	C_SerialNumber   []byte
	C_DeviceAddress  uint16
}

type CommonMeter struct {
	C_SunSpec_DID    uint16
	C_SunSpec_Length uint16
	C_Manufacturer   []byte
	C_Model          []byte
	C_Option         []byte
	C_Version        []byte // Version defined in SunSpec implementation note as String(16) however is incorrect
	C_SerialNumber   []byte
	C_DeviceAddress  uint16
}

// NewCommonModel takes a block of data read from the Modbus TCP connection and returns a new
// populated struct
func NewCommonModel(data []byte) (CommonModel, error) {
	buf := uio.NewBigEndianBuffer(data)
	if len(data) != 140 {
		return CommonModel{}, errors.New("improper data size")
	}

	var cm CommonModel
	cm.C_Manufacturer = make([]byte, 32)
	cm.C_Model = make([]byte, 32)
	cm.C_Version = make([]byte, 32)
	cm.C_SerialNumber = make([]byte, 32)

	cm.C_SunSpec_ID = buf.Read32()
	cm.C_SunSpec_DID = buf.Read16()
	cm.C_SunSpec_Length = buf.Read16()
	buf.ReadBytes(cm.C_Manufacturer[:])
	buf.ReadBytes(cm.C_Model[:])
	buf.ReadBytes(cm.C_Version[:])
	buf.ReadBytes(cm.C_SerialNumber[:])

	cm.C_Manufacturer = bytes.Trim(cm.C_Manufacturer, "\x00")
	cm.C_Model = bytes.Trim(cm.C_Model, "\x00")
	cm.C_Version = bytes.Trim(cm.C_Version, "\x00")
	cm.C_SerialNumber = bytes.Trim(cm.C_SerialNumber, "\x00")

	return cm, nil
}

func NewCommonMeter(data []byte) (CommonMeter, error) {
	buf := uio.NewBigEndianBuffer(data)
	if len(data) < 100 {
		return CommonMeter{}, errors.New("improper data size")
	}

	var cm CommonMeter
	cm.C_Manufacturer = make([]byte, 32)
	cm.C_Model = make([]byte, 32)
	cm.C_Version = make([]byte, 16)
	cm.C_Option = make([]byte, 16)
	cm.C_SerialNumber = make([]byte, 16)

	cm.C_SunSpec_DID = buf.Read16()
	cm.C_SunSpec_Length = buf.Read16()
	buf.ReadBytes(cm.C_Manufacturer[:])
	buf.ReadBytes(cm.C_Model[:])
	buf.ReadBytes(cm.C_Option[:])
	buf.ReadBytes(cm.C_Version[:])
	buf.ReadBytes(cm.C_SerialNumber[:])

	cm.C_Manufacturer = bytes.Trim(cm.C_Manufacturer, "\x00")
	cm.C_Model = bytes.Trim(cm.C_Model, "\x00")
	cm.C_Option = bytes.Trim(cm.C_Option, "\x00")
	cm.C_Version = bytes.Trim(cm.C_Version, "\x00")
	cm.C_SerialNumber = bytes.Trim(cm.C_SerialNumber, "\x00")

	return cm, nil
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

// NewInverterModel takes a block of data read from the Modbus TCP connection and returns
// a new populated struct.
func NewInverterModel(data []byte) (InverterModel, error) {
	buf := uio.NewBigEndianBuffer(data)
	if len(data) != 80 {
		return InverterModel{}, errors.New("improper data size")
	}

	im := InverterModel{
		SunSpec_DID:      buf.Read16(),
		SunSpec_Length:   buf.Read16(),
		AC_Current:       buf.Read16(),
		AC_CurrentA:      buf.Read16(),
		AC_CurrentB:      buf.Read16(),
		AC_CurrentC:      buf.Read16(),
		AC_Current_SF:    int16(buf.Read16()),
		AC_VoltageAB:     buf.Read16(),
		AC_VoltageBC:     buf.Read16(),
		AC_VoltageCA:     buf.Read16(),
		AC_VoltageAN:     buf.Read16(),
		AC_VoltageBN:     buf.Read16(),
		AC_VoltageCN:     buf.Read16(),
		AC_Voltage_SF:    int16(buf.Read16()),
		AC_Power:         int16(buf.Read16()),
		AC_Power_SF:      int16(buf.Read16()),
		AC_Frequency:     buf.Read16(),
		AC_Frequency_SF:  int16(buf.Read16()),
		AC_VA:            int16(buf.Read16()),
		AC_VA_SF:         int16(buf.Read16()),
		AC_VAR:           int16(buf.Read16()),
		AC_VAR_SF:        int16(buf.Read16()),
		AC_PF:            int16(buf.Read16()),
		AC_PF_SF:         int16(buf.Read16()),
		AC_Energy_WH:     int32(buf.Read32()),
		AC_Energy_WH_SF:  buf.Read16(),
		DC_Current:       buf.Read16(),
		DC_Current_SF:    int16(buf.Read16()),
		DC_Voltage:       buf.Read16(),
		DC_Voltage_SF:    int16(buf.Read16()),
		DC_Power:         int16(buf.Read16()),
		DC_Power_SF:      int16(buf.Read16()),
		Temp_Cabinet:     int16(buf.Read16()),
		Temp_Sink:        int16(buf.Read16()),
		Temp_Transformer: int16(buf.Read16()),
		Temp_Other:       int16(buf.Read16()),
		Temp_SF:          int16(buf.Read16()),
		Status:           buf.Read16(),
		Status_Vendor:    buf.Read16(),
	}

	return im, nil
}

func NewMeterModel(data []byte) (MeterModel, error) {
	buf := uio.NewBigEndianBuffer(data)
	if len(data) <= 10 {
		return MeterModel{}, errors.New("improper data size")
	}

	im := MeterModel{
		SunSpec_DID:       buf.Read16(),
		SunSpec_Length:    buf.Read16(),
		M_AC_Current:      buf.Read16(),
		M_AC_CurrentA:     buf.Read16(),
		M_AC_CurrentB:     buf.Read16(),
		M_AC_CurrentC:     buf.Read16(),
		M_AC_Current_SF:   int16(buf.Read16()),
		M_AC_VoltageLN:    buf.Read16(),
		M_AC_VoltageAN:    buf.Read16(),
		M_AC_VoltageBN:    buf.Read16(),
		M_AC_VoltageCN:    buf.Read16(),
		M_AC_VoltageLL:    buf.Read16(),
		M_AC_VoltageAB:    buf.Read16(),
		M_AC_VoltageBC:    buf.Read16(),
		M_AC_VoltageCA:    buf.Read16(),
		M_AC_Voltage_SF:   int16(buf.Read16()),
		M_AC_Frequency:    buf.Read16(),
		M_AC_Frequency_SF: int16(buf.Read16()),
		M_AC_Power:        int16(buf.Read16()),
		M_AC_Power_A:      int16(buf.Read16()),
		M_AC_Power_B:      int16(buf.Read16()),
		M_AC_Power_C:      int16(buf.Read16()),
		M_AC_Power_SF:     int16(buf.Read16()),
		M_AC_VA:           buf.Read16(),
		M_AC_VA_A:         buf.Read16(),
		M_AC_VA_B:         buf.Read16(),
		M_AC_VA_C:         buf.Read16(),
		M_AC_VA_SF:        int16(buf.Read16()),
		M_AC_VAR:          buf.Read16(),
		M_AC_VAR_A:        buf.Read16(),
		M_AC_VAR_B:        buf.Read16(),
		M_AC_VAR_C:        buf.Read16(),
		M_AC_VAR_SF:       int16(buf.Read16()),
		M_AC_PF:           buf.Read16(),
		M_AC_PF_A:         buf.Read16(),
		M_AC_PF_B:         buf.Read16(),
		M_AC_PF_C:         buf.Read16(),
		M_AC_PF_SF:        int16(buf.Read16()),
		M_Exported:        buf.Read32(),
		M_Exported_A:      buf.Read32(),
		M_Exported_B:      buf.Read32(),
		M_Exported_C:      buf.Read32(),
		M_Imported:        buf.Read32(),
		M_Imported_A:      buf.Read32(),
		M_Imported_B:      buf.Read32(),
		M_Imported_C:      buf.Read32(),
		M_Energy_W_SF:     int16(buf.Read16()),
	}

	return im, nil
}
