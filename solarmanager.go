package main

import (
	"github.com/ingmarstein/solarmanager-go/solarmanager"
	"log"
)

type solarManagerProvider struct {
	c  *solarmanager.Client
	id string
}

func newSolarManagerProvider(username string, password string, id string) *solarManagerProvider {
	return &solarManagerProvider{
		c:  solarmanager.NewClient(nil, nil, username, password),
		id: id,
	}
}

func (smp *solarManagerProvider) Open() error {
	return nil
}

func (smp *solarManagerProvider) Close() error {
	return nil
}

func (smp *solarManagerProvider) CurrentPowerExport() (float64, error) {
	gd, err := smp.c.GetGatewayData(smp.id)
	if err != nil {
		return 0, err
	}

	export := float64(gd.CurrentPvGeneration - gd.CurrentPowerConsumption + gd.CurrentBatteryChargeDischarge)

	return export, nil
}

func (smp *solarManagerProvider) Init() {
	info, err := smp.c.GetGatewayInfo(smp.id)
	if err != nil {
		log.Printf("failed to get SolarManager gateway info: %v", err)
		return
	}

	log.Printf("Connected to SolarManager gateway %s (%s)", info.Name, info.SmId)
	log.Printf("SolarManager gateway version: %s", info.Firmware)
	log.Printf("SolarManager gateway IP: %s", info.Ip)
}
