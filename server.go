package main

import (
	"log"
	"math"
	solaredge "mielesolar/modbus"
	"net/http"
	"time"

	"github.com/goburrow/modbus"
	"github.com/ingmarstein/miele-go/miele"
)

type server struct {
	mc        *miele.Client
	mb        modbus.Client
	handler   *modbus.TCPClientHandler
	devices   []device
	autoPower int
	verbose   bool
}

func newServer(modbusAddress string, autoPower int, devices []device, verbose bool, httpClient *http.Client) *server {
	srv := server{
		mc:        miele.NewClient(httpClient),
		handler:   modbus.NewTCPClientHandler(modbusAddress),
		devices:   devices,
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
		select {
		case <-ticker.C:
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
}

func (s *server) printSolarEdgeInfo() {
	// Collect and log common inverter data
	inverterData, err := s.mb.ReadHoldingRegisters(40000, 70)
	if err != nil {
		log.Fatalf("error reading inverter registers: %s", err.Error())
	}

	cm, err := solaredge.NewCommonModel(inverterData)
	if err != nil {
		log.Fatalf("error parsing inverter data: %s", err.Error())
	}

	log.Printf("Inverter Model: %s", cm.C_Model)
	log.Printf("Inverter Serial: %s", cm.C_SerialNumber)
	log.Printf("Inverter Version: %s", cm.C_Version)

	meterData, err := s.mb.ReadHoldingRegisters(40121, 65)
	if err != nil {
		log.Fatalf("error reading meter registers: %s", err.Error())
	}

	cm2, err := solaredge.NewCommonMeter(meterData)
	if err != nil {
		log.Fatalf("error parsing meter registers: %s", err.Error())
	}
	log.Printf("Meter Manufacturer: %s", cm2.C_Manufacturer)
	log.Printf("Meter Model: %s", cm2.C_Model)
	log.Printf("Meter Serial: %s", cm2.C_SerialNumber)
	log.Printf("Meter Version: %s", cm2.C_Version)
	log.Printf("Meter Option: %s", cm2.C_Option)
}

func (s *server) currentPowerExport() (float64, error) {
	inverterData, err := s.mb.ReadHoldingRegisters(40069, 40)
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

	meterData, err := s.mb.ReadHoldingRegisters(40188, 105)
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
	for _, device := range s.devices {
		device.waiting = false
		state, err := s.mc.GetDeviceState(device.ID, miele.GetDeviceStateRequest{})
		if err != nil {
			log.Printf("error getting device state for %s: %v", device.ID, err)
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
			Power:   float64(*autoPower),
			waiting: true,
		})
		deviceWaiting = true
	}

	return deviceWaiting
}

// updateDevices updates all Miele appliances and returns whether
// one is waiting for SmartStart.
func (s *server) updateDevices() bool {
	if *autoPower == 0 {
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
	for _, device := range s.devices {
		if !device.waiting || device.Power > available {
			continue
		}
		log.Printf("starting device %s", device.ID)
		err := s.mc.DeviceAction(device.ID, miele.DeviceActionRequest{
			ProcessAction: miele.ACTION_START,
		})
		if err != nil {
			log.Printf("error starting device %s: %v", device.ID, err)
			continue
		}
		available -= device.Power
		log.Printf("started device %s, remaining power: %f", device.ID, available)
		device.waiting = false
	}
}
