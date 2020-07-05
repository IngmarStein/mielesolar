package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"mielesolar/modbus"
	"os"
	"strconv"
	"time"

	"github.com/goburrow/modbus"
)

var inverterAddress = flag.String("inverter", defaultString("INVERTER_ADDRESS", "192.168.188.167"), "Inverter address or IP")
var inverterPort = flag.Int("port", defaultInt("INVERTER_PORT", 502), "MODBUS over TCP port")
var pollInterval = flag.Int("interval", 5, "Polling interval in seconds")
var configFile = flag.String("config", "devices.json", "Device config file")

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
	Name    string  `json:"name"`
	Power   float64 `json:"power"`
	waiting bool
}

func main() {
	configData, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("error reading %s: %v", *configFile, err)
	}
	var devices []device
	if err := json.Unmarshal(configData, &devices); err != nil {
		log.Fatalf("error parsing device config: %v", err)
	}

	handler := modbus.NewTCPClientHandler(fmt.Sprintf("%s:%d", *inverterAddress, *inverterPort))
	handler.Timeout = 10 * time.Second
	handler.SlaveId = 0x01
	if err := handler.Connect(); err != nil {
		log.Fatalf("error connecting to inverter: %s", err.Error())
	}
	defer handler.Close()
	client := modbus.NewClient(handler)

	printSolarEdgeInfo(client)

	ticker := time.NewTicker(time.Duration(*pollInterval) * time.Second)
	done := make(chan bool)

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if err := refresh(client, devices); err != nil {
				log.Printf("attempting to reconnect")
				_ = handler.Close()
				time.Sleep(2 * time.Second)
				err = handler.Connect()
				if err != nil {
					log.Printf("error reconnecting: %v\n", err)
				}
			}
		}
	}
}

func printSolarEdgeInfo(client modbus.Client) error {
	// Collect and log common inverter data
	inverterData, err := client.ReadHoldingRegisters(40000, 70)
	if err != nil {
		log.Printf("error reading inverter registers: %s", err.Error())
		return err
	}

	cm, err := solaredge.NewCommonModel(inverterData)
	if err != nil {
		log.Printf("error parsing inverter data: %s", err.Error())
		return err
	}

	log.Printf("Inverter Model: %s", cm.C_Model)
	log.Printf("Inverter Serial: %s", cm.C_SerialNumber)
	log.Printf("Inverter Version: %s", cm.C_Version)

	meterData, err := client.ReadHoldingRegisters(40121, 65)
	if err != nil {
		log.Printf("error reading meter registers: %s", err.Error())
		return err
	}

	cm2, err := solaredge.NewCommonMeter(meterData)
	if err != nil {
		log.Printf("error parsing meter registers: %s", err.Error())
		return err
	}
	log.Printf("Meter Manufacturer: %s", cm2.C_Manufacturer)
	log.Printf("Meter Model: %s", cm2.C_Model)
	log.Printf("Meter Serial: %s", cm2.C_SerialNumber)
	log.Printf("Meter Version: %s", cm2.C_Version)
	log.Printf("Meter Option: %s", cm2.C_Option)

	return nil
}

func currentPowerExport(client modbus.Client) (float64, error) {
	inverterData, err := client.ReadHoldingRegisters(40069, 40)
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

	meterData, err := client.ReadHoldingRegisters(40188, 105)
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

func refresh(client modbus.Client, devices []device) error {
	log.Println("starting refresh")

	waiting := updateDevices(devices)
	if !waiting {
		return nil
	}

	available, err := currentPowerExport(client)
	if err != nil {
		return err
	}

	consumePower(available, devices)

	return nil
}

// updateDevices updates all Miele appliances and returns whether
// one is waiting for SmartStart.
func updateDevices(devices []device) bool {
	//TODO
	for _, device := range devices {
		device.waiting = true
	}

	return true
}

// consumePower starts appliances in the given priority order to
// consume the surplus power.
func consumePower(available float64, devices []device) {
	// https://github.com/demel42/IPSymconMieleAtHome
	// https://www.symcon.de/forum/threads/34249-Miele-Home-XKM-3100W-Protokollanalyse/page22

	for _, device := range devices {
		if device.waiting && device.Power <= available {
			// TODO: start device
			available -= device.Power
			log.Printf("starting %s, remaining power: %f", device.Name, available)
			device.waiting = false
		}
	}
}
