package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ingmarstein/miele-go/miele"
	"log"
	"os"
	"strconv"
	"time"
	_ "time/tzdata"
)

var inverterAddress = flag.String("inverter", defaultString("INVERTER_ADDRESS", "192.168.188.167"), "Inverter address or IP")
var inverterPort = flag.Int("port", defaultInt("INVERTER_PORT", 502), "MODBUS over TCP port")
var pollInterval = flag.Int("interval", 5, "Polling interval in seconds")
var configFile = flag.String("config", "devices.json", "Device config file")
var clientID = flag.String("client-id", os.Getenv("MIELE_CLIENT_ID"), "Miele 3rd Party API client ID")
var clientSecret = flag.String("client-secret", os.Getenv("MIELE_CLIENT_SECRET"), "Miele 3rd Party API client secret")
var username = flag.String("user", os.Getenv("MIELE_USERNAME"), "Miele@Home user name")
var password = flag.String("password", os.Getenv("MIELE_PASSWORD"), "Miele@Home password")
var vg = flag.String("vg", "de-CH", "country selector")
var autoPower = flag.Int("auto", 0, "automatically start waiting devices if a minimum amount of power is available")
var verbose = flag.Bool("verbose", false, "verbose mode")

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

// https://medium.com/@mhcbinder/using-local-time-in-a-golang-docker-container-built-from-scratch-2900af02fbaf
func updateTimezone() {
	if tz := os.Getenv("TZ"); tz != "" {
		var err error
		time.Local, err = time.LoadLocation(tz)
		if err != nil {
			log.Printf("error loading location '%s': %v\n", tz, err)
		}
	}
}

func main() {
	updateTimezone()

	flag.Parse()

	if *clientID == "" || *clientSecret == "" || *username == "" || *password == "" {
		flag.Usage()
		os.Exit(1)
	}

	var devices []device
	if *autoPower == 0 {
		configData, err := os.ReadFile(*configFile)
		if err != nil {
			log.Fatalf("error reading %s: %v", *configFile, err)
		}
		if err := json.Unmarshal(configData, &devices); err != nil {
			log.Fatalf("error parsing device config: %v", err)
		}
	}

	client := miele.NewClientWithAuth(*clientID, *clientSecret, *vg, *username, *password)

	srv := newServer(fmt.Sprintf("%s:%d", *inverterAddress, *inverterPort), *autoPower, devices, *verbose, client)
	srv.printSolarEdgeInfo()

	defer srv.close()
	srv.serve()
}
