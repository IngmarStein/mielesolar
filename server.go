package main

import (
	"log"
	"math"
	"time"

	"github.com/goburrow/modbus"
	"github.com/ingmarstein/miele-go/miele"
	solaredge "github.com/ingmarstein/mielesolar/modbus"
)

type modeEnum int

const (
	ManualMode modeEnum = iota
	AutoSingleMode
	AutoAllMode
)

type server struct {
	mc         *miele.Client
	mb         modbus.Client
	handler    *modbus.TCPClientHandler
	devices    []device
	mode       modeEnum
	autoPower  int
	verbose    bool
	hasBattery bool
}

func newServer(modbusAddress string, mode modeEnum, autoPower int, devices []device, verbose bool, mieleClient *miele.Client) *server {
	srv := server{
		mc:        mieleClient,
		handler:   modbus.NewTCPClientHandler(modbusAddress),
		devices:   devices,
		mode:      mode,
		autoPower: autoPower,
		verbose:   verbose,
	}

	srv.mc.Verbose = verbose
	srv.handler.Timeout = 10 * time.Second
	srv.handler.SlaveId = 0x01
	srv.mb = modbus.NewClient(srv.handler)

	if err := srv.handler.Connect(); err != nil {
		log.Fatalf("error connecting to inverter: %s", err.Error())
	}

	return &srv
}

func (s *server) close() {
	err := s.handler.Close()
	if err != nil {
		log.Printf("error closing handler: %v\n", err)
	}
}

func (s *server) serve() {
	ticker := time.NewTicker(time.Duration(*pollInterval) * time.Second)

	for {
		<-ticker.C
		if err := s.refresh(); err != nil {
			log.Printf("attempting to reconnect")
			_ = s.handler.Close()
			time.Sleep(2 * time.Second)
			err = s.handler.Connect()
			if err != nil {
				log.Printf("error reconnecting: %v\n", err)
			}
		}
	}
}

const inverterInfoBaseAddress = 40000 // 0x9C40
const inverterDataBaseAddress = 40069 // 0x9C85
const meterInfoBaseAddress = 40121    // 0x9CB9
const meterDataBaseAddress = 40188    // 0x9CFC
const batteryInfoBaseAddress = 57600  // 0xE100
const batteryDataBaseAddress = 57664  // 0xE140

func (s *server) printSolarEdgeInfo() {
	// Collect and log common inverter data
	inverterData, err := s.mb.ReadHoldingRegisters(inverterInfoBaseAddress, 70)
	if err != nil {
		log.Fatalf("error reading inverter registers: %s", err.Error())
	}

	cm, err := solaredge.NewCommonInverter(inverterData)
	if err != nil {
		log.Fatalf("error parsing inverter data: %s", err.Error())
	}

	log.Printf("Inverter Model: %s", cm.C_Model)
	log.Printf("Inverter Serial: %s", cm.C_SerialNumber)
	log.Printf("Inverter Version: %s", cm.C_Version)

	meterData, err := s.mb.ReadHoldingRegisters(meterInfoBaseAddress, 65)
	if err != nil {
		log.Fatalf("error reading meter registers: %s", err.Error())
	}

	mm, err := solaredge.NewCommonMeter(meterData)
	if err != nil {
		log.Fatalf("error parsing meter registers: %s", err.Error())
	}
	log.Printf("Meter Manufacturer: %s", mm.C_Manufacturer)
	log.Printf("Meter Model: %s", mm.C_Model)
	log.Printf("Meter Option: %s", mm.C_Option)
	log.Printf("Meter Version: %s", mm.C_Version)
	log.Printf("Meter Serial: %s", mm.C_SerialNumber)

	batteryData, err := s.mb.ReadHoldingRegisters(batteryInfoBaseAddress, 64)
	if err != nil {
		log.Fatalf("error reading battery registers: %s", err.Error())
	}

	bm, err := solaredge.NewCommonBattery(batteryData)
	if err != nil {
		log.Fatalf("error parsing battery registers: %s", err.Error())
	}
	log.Printf("Battery Manufacturer: %s", bm.C_Manufacturer)
	log.Printf("Battery Model: %s", bm.C_Model)
	log.Printf("Battery Version: %s", bm.C_Version)
	log.Printf("Battery Serial: %s", bm.C_SerialNumber)

	s.hasBattery = bm.C_Manufacturer[0] != 0
}

func (s *server) currentPowerExport() (float64, error) {
	inverterData, err := s.mb.ReadHoldingRegisters(inverterDataBaseAddress, 40)
	if err != nil {
		log.Printf("error reading inverter registers: %s", err.Error())
		return 0, err
	}

	inverter, err := solaredge.NewInverterModel(inverterData)
	if err != nil {
		log.Printf("error parsing data: %s", err.Error())
		return 0, err
	}

	if inverter.Status != solaredge.I_STATUS_MPPT && inverter.Status != solaredge.I_STATUS_THROTTLED {
		log.Printf("current inverter status: %d\n", inverter.Status)
		//return 0, nil
	}

	inverterACPower := float64(inverter.AC_Power) * math.Pow(10.0, float64(inverter.AC_Power_SF))
	log.Printf("Inverter AC Power: %f", inverterACPower)
	inverterDCPower := float64(inverter.DC_Power) * math.Pow(10.0, float64(inverter.DC_Power_SF))
	log.Printf("Inverter DC Power: %f", inverterDCPower)

	meterData, err := s.mb.ReadHoldingRegisters(meterDataBaseAddress, 105)
	if err != nil {
		log.Printf("error reading meter data: %s", err.Error())
		return 0, err
	}

	mt, err := solaredge.NewMeterModel(meterData)
	if err != nil {
		log.Printf("error parsing meter data: %s", err.Error())
		return 0, err
	}
	meterACPower := float64(mt.M_AC_Power) * math.Pow(10.0, float64(mt.M_AC_Power_SF))
	log.Printf("Meter AC Power: %f", meterACPower)

	if s.hasBattery {
		batteryData, err := s.mb.ReadHoldingRegisters(batteryDataBaseAddress, 43)
		if err != nil {
			log.Printf("error reading battery data: %s", err.Error())
			return 0, err
		}

		b, err := solaredge.NewBatteryModel(batteryData)
		if err != nil {
			log.Printf("error parsing battery data: %s", err.Error())
			return 0, err
		}
		log.Printf("Battery Power: %f", b.InstantaneousPower)
	}

	return meterACPower, nil
}

func (s *server) refresh() error {
	if s.verbose {
		log.Println("starting refresh")
	}

	waiting := s.updateDevices()
	if !waiting {
		return nil
	}

	available, err := s.currentPowerExport()
	if err != nil {
		return err
	}

	s.consumePower(available)

	return nil
}

func (s *server) updateConfiguredDevices() bool {
	var deviceWaiting bool
	for i := 0; i < len(s.devices); i++ {
		device := &s.devices[i]
		device.waiting = false
		state, err := s.mc.GetDeviceState(device.ID, miele.GetDeviceStateRequest{})
		if err != nil {
			log.Printf("error getting device state for %s (%s): %v", device.Name, device.ID, err)
			continue
		}
		if state.Status.ValueRaw == miele.DEVICE_STATUS_PROGRAMMED_WAITING_TO_START && state.RemoteEnable.FullRemoteControl {
			deviceWaiting = true
			device.waiting = true
		}
	}

	return deviceWaiting
}

func (s *server) updateAutoDevices() bool {
	resp, err := s.mc.ListDevices(miele.ListDevicesRequest{})
	if err != nil {
		log.Printf("error listing devices: %v", err)
		return false
	}

	s.devices = []device{}
	var deviceWaiting bool
	for _, r := range resp {
		// https://www.miele.com/developer/swagger-ui/put_additional_info.html
		if r.Ident.Typ.ValueRaw != miele.DEVICE_TYPE_WASHING_MACHINE &&
			r.Ident.Typ.ValueRaw != miele.DEVICE_TYPE_TUMBLE_DRYER &&
			r.Ident.Typ.ValueRaw != miele.DEVICE_TYPE_DISHWASHER &&
			r.Ident.Typ.ValueRaw != miele.DEVICE_TYPE_WASHER_DRYER {
			continue
		}

		if r.State.Status.ValueRaw != miele.DEVICE_STATUS_PROGRAMMED_WAITING_TO_START || !r.State.RemoteEnable.FullRemoteControl {
			continue
		}

		s.devices = append(s.devices, device{
			ID:      r.Ident.DeviceIdentLabel.FabNumber,
			Name:    r.Ident.DeviceName,
			Power:   float64(*autoPower),
			waiting: true,
		})
		deviceWaiting = true
	}

	return deviceWaiting
}

// updateDevices updates all Miele appliances and returns whether one is waiting for SmartStart.
func (s *server) updateDevices() bool {
	if s.mode == ManualMode {
		return s.updateConfiguredDevices()
	}

	return s.updateAutoDevices()
}

// consumePower starts appliances in the given priority order to
// consume the surplus power.
//
// See also:
// https://github.com/demel42/IPSymconMieleAtHome
// https://www.symcon.de/forum/threads/34249-Miele-Home-XKM-3100W-Protokollanalyse
func (s *server) consumePower(available float64) {
	for i := 0; i < len(s.devices); i++ {
		device := &s.devices[i]
		if !device.waiting || device.Power > available {
			continue
		}
		log.Printf("starting device %s (%s)", device.Name, device.ID)
		err := s.mc.DeviceAction(device.ID, miele.DeviceActionRequest{
			ProcessAction: miele.ACTION_START,
		})
		if err != nil {
			log.Printf("error starting device %s (%s): %v", device.Name, device.ID, err)
			continue
		}
		if s.mode != AutoAllMode {
			available -= device.Power
		}
		log.Printf("started device %s (%s), remaining power: %f", device.Name, device.ID, available)
		device.waiting = false
	}
}
