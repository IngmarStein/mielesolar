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

func (s *server) printSolarEdgeInfo() {
	// Collect and log common inverter data
	inverter, err := solaredge.ReadInverter(s.mb)
	if err != nil {
		log.Fatalf("%v", err)
	}

	log.Printf("Inverter Model: %s", inverter.C_Model)
	log.Printf("Inverter Serial: %s", inverter.C_SerialNumber)
	log.Printf("Inverter Version: %s", inverter.C_Version)

	meter, err := solaredge.ReadMeter(s.mb, 0)
	if err != nil {
		log.Fatalf("error reading meter registers: %s", err.Error())
	}

	log.Printf("Meter Manufacturer: %s", meter.C_Manufacturer)
	log.Printf("Meter Model: %s", meter.C_Model)
	log.Printf("Meter Option: %s", meter.C_Option)
	log.Printf("Meter Version: %s", meter.C_Version)
	log.Printf("Meter Serial: %s", meter.C_SerialNumber)

	battery, err := solaredge.ReadBatteryInfo(s.mb, 0)
	if err != nil {
		log.Fatalf("error reading battery registers: %s", err.Error())
	}

	log.Printf("Battery Manufacturer: %s", battery.C_Manufacturer)
	log.Printf("Battery Model: %s", battery.C_Model)
	log.Printf("Battery Version: %s", battery.C_Version)
	log.Printf("Battery Serial: %s", battery.C_SerialNumber)

	s.hasBattery = battery.C_Manufacturer[0] != 0
}

func (s *server) currentPowerExport() (float64, error) {
	inverter, err := solaredge.ReadInverter(s.mb)
	if err != nil {
		log.Printf("error reading inverter registers: %s", err.Error())
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

	meter, err := solaredge.ReadMeter(s.mb, 0)
	if err != nil {
		log.Printf("error reading meter data: %s", err.Error())
		return 0, err
	}

	meterACPower := float64(meter.M_AC_Power) * math.Pow(10.0, float64(meter.M_AC_Power_SF))
	log.Printf("Meter AC Power: %f", meterACPower)

	if s.hasBattery {
		battery, err := solaredge.ReadBattery(s.mb, 0)
		if err != nil {
			log.Printf("error reading battery data: %v", err)
			return 0, err
		}

		log.Printf("Battery Power: %f", battery.InstantaneousPower)
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
