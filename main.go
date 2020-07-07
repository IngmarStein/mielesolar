package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"mielesolar/modbus"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/goburrow/modbus"
	"github.com/ingmarstein/miele-go/miele"
	"golang.org/x/oauth2"
)

var inverterAddress = flag.String("inverter", defaultString("INVERTER_ADDRESS", "192.168.188.167"), "Inverter address or IP")
var inverterPort = flag.Int("port", defaultInt("INVERTER_PORT", 502), "MODBUS over TCP port")
var pollInterval = flag.Int("interval", 5, "Polling interval in seconds")
var configFile = flag.String("config", "devices.json", "Device config file")
var clientID = flag.String("client-id", os.Getenv("MIELE_CLIENT_ID"), "Miele 3rd Party API client ID")
var clientSecret = flag.String("client-secret", os.Getenv("MIELE_CLIENT_SECRET"), "Miele 3rd Party API client secret")
var username = flag.String("user", os.Getenv("MIELE_USERNAME"), "Miele@Home user name")
var password = flag.String("password", os.Getenv("MIELE_PASSWORD"), "Miele@Home password")

func defaultString(key, value string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return value
}

func defaultInt(key string, value int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}

	return value
}

type device struct {
	ID      string  `json:"id"`
	Power   float64 `json:"power"`
	waiting bool
}

type server struct {
	mc      *miele.Client
	mb      modbus.Client
	handler *modbus.TCPClientHandler
	devices []device
}

func main() {
	flag.Parse()

	if *clientID == "" || *clientSecret == "" || *username == "" || *password == "" {
		flag.Usage()
		os.Exit(1)
	}

	configData, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("error reading %s: %v", *configFile, err)
	}
	var devices []device
	if err := json.Unmarshal(configData, &devices); err != nil {
		log.Fatalf("error parsing device config: %v", err)
	}

	conf := &oauth2.Config{
		ClientID:     *clientID,
		ClientSecret: *clientSecret,
		Endpoint:     miele.Endpoint,
	}

	ctx := context.Background()
	token, err := conf.PasswordCredentialsToken(ctx, *username, *password)
	if err != nil {
		log.Fatalf("error retrieving Miele token: %v", err)
	}

	oauthClient := conf.Client(ctx, token)

	srv := newServer(fmt.Sprintf("%s:%d", *inverterAddress, *inverterPort), devices, oauthClient)
	srv.printSolarEdgeInfo()

	defer srv.close()
	srv.serve()
}

func newServer(modbusAddress string, devices []device, httpClient *http.Client) *server {

	srv := server{
		mc:      miele.NewClient(httpClient),
		handler: modbus.NewTCPClientHandler(modbusAddress),
		devices: devices,
	}

	srv.handler.Timeout = 10 * time.Second
	srv.handler.SlaveId = 0x01
	srv.mb = modbus.NewClient(srv.handler)

	if err := srv.handler.Connect(); err != nil {
		log.Fatalf("error connecting to inverter: %s", err.Error())
	}

	return &srv
}

func (s *server) close() {
	s.handler.Close()
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
	log.Println("starting refresh")

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

// updateDevices updates all Miele appliances and returns whether
// one is waiting for SmartStart.
func (s *server) updateDevices() bool {
	var deviceWaiting bool
	for _, device := range s.devices {
		device.waiting = false
		state, err := s.mc.GetDeviceState(device.ID, miele.GetDeviceStateRequest{})
		if err != nil {
			log.Printf("error getting device state for %s: %v", device.ID, err)
			continue
		}
		if state.Status.ValueRaw == miele.DEVICE_STATUS_PROGRAMMED_WAITING_TO_START && state.RemoteEnable.SmartGrid {
			deviceWaiting = true
			device.waiting = true
		}
	}

	return deviceWaiting
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
		err := s.mc.DeviceAction(device.ID, miele.DeviceActionRequest{
			ProcessAction: miele.ACTION_START,
		})
		if err != nil {
			log.Printf("error starting device %s: %v", device.ID, err)
			continue
		}
		available -= device.Power
		log.Printf("starting device %s, remaining power: %f", device.ID, available)
		device.waiting = false
	}
}
