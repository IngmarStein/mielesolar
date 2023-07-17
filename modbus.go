package main

import (
	"fmt"
	solaredge "github.com/ingmarstein/mielesolar/modbus"
	"github.com/simonvetter/modbus"
	"log"
	"math"
	"time"
)

type modbusProvider struct {
	c          *modbus.ModbusClient
	hasBattery bool
}

func newModbusProvider(address string) (*modbusProvider, error) {
	var p modbusProvider

	var err error
	p.c, err = modbus.NewClient(&modbus.ClientConfiguration{
		URL:     "tcp://" + address,
		Timeout: 10 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}

	return &p, err
}

func (mp *modbusProvider) Open() error {
	if err := mp.c.Open(); err != nil {
		return err
	}

	if err := mp.c.SetUnitId(0x01); err != nil {
		return fmt.Errorf("error setting unit ID: %v", err)
	}

	return nil
}

func (mp *modbusProvider) Close() error {
	if err := mp.c.Close(); err != nil {
		return fmt.Errorf("error closing modbus client: %v", err)
	}

	return nil
}

func (mp *modbusProvider) Init() {
	// Collect and log common inverter data
	inverter, err := solaredge.ReadInverter(mp.c)
	if err != nil {
		log.Fatalf("%v", err)
	}

	log.Printf("Inverter Model: %s", inverter.C_Model)
	log.Printf("Inverter Serial: %s", inverter.C_SerialNumber)
	log.Printf("Inverter Version: %s", inverter.C_Version)

	meter, err := solaredge.ReadMeter(mp.c, 0)
	if err != nil {
		log.Fatalf("error reading meter registers: %s", err.Error())
	}

	log.Printf("Meter Manufacturer: %s", meter.C_Manufacturer)
	log.Printf("Meter Model: %s", meter.C_Model)
	log.Printf("Meter Option: %s", meter.C_Option)
	log.Printf("Meter Version: %s", meter.C_Version)
	log.Printf("Meter Serial: %s", meter.C_SerialNumber)

	battery, err := solaredge.ReadBatteryInfo(mp.c, 0)
	if err != nil {
		log.Fatalf("error reading battery registers: %s", err.Error())
	}

	log.Printf("Battery Manufacturer: %s", battery.C_Manufacturer)
	log.Printf("Battery Model: %s", battery.C_Model)
	log.Printf("Battery Version: %s", battery.C_Version)
	log.Printf("Battery Serial: %s", battery.C_SerialNumber)

	mp.hasBattery = battery.C_Manufacturer[0] != 0

	if mp.hasBattery {
		log.Printf("Battery rated energy: %.0f W", battery.RatedEnergy)
		log.Printf("Battery maximum charge continuous power: %.0f W", battery.MaximumChargeContinuousPower)
		log.Printf("Battery maximum discharge continuous power: %.0f W", battery.MaximumDischargeContinuousPower)
		log.Printf("Battery maximum charge peak power: %.0f W", battery.MaximumChargePeakPower)
		log.Printf("Battery maximum discharge peak power: %.0f W", battery.MaximumDischargePeakPower)
	}
}

func (mp *modbusProvider) CurrentPowerExport() (float64, error) {
	inverter, err := solaredge.ReadInverter(mp.c)
	if err != nil {
		log.Printf("error reading inverter registers: %s", err.Error())
		return 0, err
	}

	if inverter.Status != solaredge.I_STATUS_MPPT && inverter.Status != solaredge.I_STATUS_THROTTLED {
		log.Printf("current inverter status: %d\n", inverter.Status)
		//return 0, nil
	}

	// inverter DC power = solar production
	inverterDCPower := float64(inverter.DC_Power) * math.Pow(10.0, float64(inverter.DC_Power_SF))
	log.Printf("Inverter DC Power: %f", inverterDCPower)

	// inverter AC power = production after conversion to AC
	inverterACPower := float64(inverter.AC_Power) * math.Pow(10.0, float64(inverter.AC_Power_SF))
	log.Printf("Inverter AC Power: %f", inverterACPower)

	meter, err := solaredge.ReadMeter(mp.c, 0)
	if err != nil {
		log.Printf("error reading meter data: %s", err.Error())
		return 0, err
	}

	// meter AC power = balance of production and consumption
	// positive values indicate a surplus -> export to grid
	// negative values indicate a deficit -> import from grid
	meterACPower := float64(meter.M_AC_Power) * math.Pow(10.0, float64(meter.M_AC_Power_SF))
	log.Printf("Meter AC Power: %f", meterACPower)

	powerExport := meterACPower

	// If the system has a battery installed, consider the amount of energy flowing into it
	// as surplus. That is, prioritize Miele appliances higher than the battery.
	if mp.hasBattery {
		battery, err := solaredge.ReadBattery(mp.c, 0)
		if err != nil {
			log.Printf("error reading battery data: %v", err)
			return 0, err
		}

		log.Printf("Battery Power: %f", battery.InstantaneousPower)

		powerExport += float64(battery.InstantaneousPower)
	}

	return powerExport, nil
}
