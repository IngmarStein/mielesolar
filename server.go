package main

import (
	"log"
	"time"

	"github.com/ingmarstein/miele-go/miele"
)

type modeEnum int

const (
	ManualMode modeEnum = iota
	AutoSingleMode
	AutoAllMode
)

type PvProvider interface {
	Init()
	CurrentPowerExport() (float64, error)
	Open() error
	Close() error
}

type server struct {
	mc         *miele.Client
	pp         PvProvider
	devices    []device
	mode       modeEnum
	autoPower  int
	verbose    bool
	startDelay time.Duration
	nextStart  time.Time
}

func newServer(mode modeEnum, autoPower int, devices []device, verbose bool, mieleClient *miele.Client, pvProvider PvProvider, startDelay time.Duration) *server {
	srv := server{
		mc:         mieleClient,
		pp:         pvProvider,
		devices:    devices,
		mode:       mode,
		autoPower:  autoPower,
		verbose:    verbose,
		startDelay: startDelay,
		nextStart:  time.Now(),
	}

	srv.mc.Verbose = verbose

	if err := srv.pp.Open(); err != nil {
		log.Fatalf("error connecting to inverter: %v", err)
	}

	return &srv
}

func (s *server) close() {
	if err := s.pp.Close(); err != nil {
		log.Print(err)
	}
}

func (s *server) serve() {
	ticker := time.NewTicker(time.Duration(*pollInterval) * time.Second)

	for {
		<-ticker.C
		if err := s.refresh(); err != nil {
			log.Printf("attempting to reconnect")
			_ = s.pp.Close()
			time.Sleep(2 * time.Second)
			err = s.pp.Open()
			if err != nil {
				log.Printf("error reconnecting: %v\n", err)
			}
		}
	}
}

func (s *server) init() {
	s.pp.Init()
}

func (s *server) refresh() error {
	if s.verbose {
		log.Println("starting refresh")
	}

	waiting := s.updateDevices()
	if !waiting {
		return nil
	}

	available, err := s.pp.CurrentPowerExport()
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
		if time.Now().Before(s.nextStart) {
			log.Printf("delaying start of device %s (%s). Next start after %v", device.Name, device.ID, s.nextStart.Format(time.RFC1123))
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
		s.nextStart = time.Now().Add(s.startDelay)
		log.Printf("started device %s (%s), remaining power: %f", device.Name, device.ID, available)
		device.waiting = false
	}
}
