package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/grandcat/zeroconf"
	"github.com/ingmarstein/miele-go/miele"
)

var (
	inverterAddress      = flag.String("inverter", defaultString("INVERTER_ADDRESS", ""), "Inverter address or IP")
	inverterPort         = flag.Int("port", defaultInt("INVERTER_PORT", 502), "MODBUS over TCP port")
	pollInterval         = flag.Int("interval", 5, "Polling interval in seconds")
	configFile           = flag.String("config", "devices.json", "Device config file")
	clientID             = flag.String("client-id", os.Getenv("MIELE_CLIENT_ID"), "Miele 3rd Party API client ID")
	clientSecret         = flag.String("client-secret", os.Getenv("MIELE_CLIENT_SECRET"), "Miele 3rd Party API client secret")
	username             = flag.String("user", os.Getenv("MIELE_USERNAME"), "Miele@Home user name")
	password             = flag.String("password", os.Getenv("MIELE_PASSWORD"), "Miele@Home password")
	vg                   = flag.String("vg", "de-CH", "Country selector")
	autoPower            = flag.Int("auto", 0, "Automatically start waiting devices if a minimum amount of power is available")
	autoMode             = flag.String("auto-mode", "single", "How many devices to start when the amount of power specified by -auto is available. Valid values: \"single\" or \"all\"")
	verbose              = flag.Bool("verbose", false, "Verbose mode")
	startDelay           = flag.Int("delay", defaultInt("DELAY", 300), "Delay in seconds between the start of devices")
	solarManagerUsername = flag.String("solarmanager-username", os.Getenv("SOLARMANAGER_USERNAME"), "SolarManager username")
	solarManagerPassword = flag.String("solarmanager-password", os.Getenv("SOLARMANAGER_PASSWORD"), "SolarManager password")
	solarManagerID       = flag.String("solarmanager-id", os.Getenv("SOLARMANAGER_ID"), "SolarManager ID")
)

const (
	SCAN_TIMEOUT = 60 * time.Second
)

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

	if *autoPower != 0 && *configFile != "" {
		log.Println("WARNING: configuration file is ignored in automatic mode")
	} else if *autoPower == 0 && *configFile == "" {
		log.Println("Either -auto or -config must be specified")
		flag.Usage()
		os.Exit(1)
	}

	var inverterIndex = 0
	if len(*inverterAddress) == 0 && len(*solarManagerUsername) == 0 {
		entries := make(chan *zeroconf.ServiceEntry)
		log.Println("Searching for inverter on the local network")
		resolver, err := zeroconf.NewResolver(nil)
		if err != nil {
			log.Fatalln("Failed to initialize resolver:", err.Error())
		}
		ctx, cancel := context.WithTimeout(context.Background(), SCAN_TIMEOUT)
		defer cancel()

		err = resolver.Browse(ctx, "_solaredge-modbus._tcp", "local.", entries)
		if err != nil {
			log.Fatalln("Failed to browse:", err.Error())
		}
		select {
		case entry := <-entries:
			*inverterAddress = entry.AddrIPv4[0].String()
			log.Printf("Found inverter: %s\n", *inverterAddress)
			for _, txt := range entry.Text {
				if strings.HasPrefix(txt, "MODBUS_ID=") {
					inverterIndex, err = strconv.Atoi(strings.TrimPrefix(txt, "MODBUS_ID="))
					if err != nil {
						log.Fatalf("Invalid MODBUS_ID from mDNS: %s\n", txt)
					}
					inverterIndex--
				}
			}

		case <-ctx.Done():
			log.Println("No inverter found on the local network")
			os.Exit(1)
		}
	}
	if len(*inverterAddress) > 0 && len(*solarManagerUsername) > 0 {
		log.Println("-inverter and -solarmanager-username are mutually exclusive")
		flag.Usage()
		os.Exit(1)
	}

	var mode modeEnum
	switch *autoMode {
	case "single":
		mode = AutoSingleMode
	case "all":
		mode = AutoAllMode
	default:
		flag.Usage()
		os.Exit(1)
	}

	var devices []device
	if *autoPower == 0 {
		mode = ManualMode
		configData, err := os.ReadFile(*configFile)
		if err != nil {
			log.Fatalf("error reading %s: %v", *configFile, err)
		}
		if err := json.Unmarshal(configData, &devices); err != nil {
			log.Fatalf("error parsing device config: %v", err)
		}
	}

	mieleClient := miele.NewClientWithAuth(*clientID, *clientSecret, *vg, *username, *password)

	var pp PvProvider
	if len(*inverterAddress) > 0 {
		var err error
		pp, err = newModbusProvider(fmt.Sprintf("%s:%d", *inverterAddress, *inverterPort), inverterIndex)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		pp = newSolarManagerProvider(*solarManagerUsername, *solarManagerPassword, *solarManagerID)
	}

	srv := newServer(
		mode,
		*autoPower,
		devices,
		*verbose,
		mieleClient,
		pp,
		time.Duration(*startDelay)*time.Second)
	srv.init()

	defer srv.close()
	srv.serve()
}
