package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ingmarstein/miele-go/miele"
	"golang.org/x/oauth2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
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

func main() {
	flag.Parse()

	if *clientID == "" || *clientSecret == "" || *username == "" || *password == "" {
		flag.Usage()
		os.Exit(1)
	}

	var devices []device
	if *autoPower == 0 {
		configData, err := ioutil.ReadFile(*configFile)
		if err != nil {
			log.Fatalf("error reading %s: %v", *configFile, err)
		}
		if err := json.Unmarshal(configData, &devices); err != nil {
			log.Fatalf("error parsing device config: %v", err)
		}
	}

	conf := &oauth2.Config{
		ClientID:     *clientID,
		ClientSecret: *clientSecret,
		Endpoint:     miele.Endpoint,
	}

	hc := &http.Client{Transport: &miele.AuthTransport{VG: *vg}}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, hc)

	token, err := conf.PasswordCredentialsToken(ctx, *username, *password)
	if err != nil {
		log.Fatalf("error retrieving Miele token: %v", err)
	}

	oauthClient := conf.Client(context.Background(), token)

	srv := newServer(fmt.Sprintf("%s:%d", *inverterAddress, *inverterPort), *autoPower, devices, oauthClient)
	srv.printSolarEdgeInfo()

	defer srv.close()
	srv.serve()
}
