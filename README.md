# Mielesolar

Start Miele@Home appliances when your SolarEdge inverter produces surplus power.

## Purpose

Why should you want to use this point-to-point integration between your inverter and household devices when more
flexible solutions like openHAB.org, FHEM, ioBroker.net, home-assistant.io, etc. exist?

- Minimal resource footprint: the Docker image has a compressed size of ~4 MB, the memory usage is ~10 MB.
- Easy to get started: it's not required to learn abstract concepts like things, connectors, bindings, channels, links,
  etc. Just provide your inverter's IP address and Miele credentials and that's it.
- No gateway (XGW3000) required to connect to your Miele devices: mielesolar uses the Miele API to talk to your
  WiFi-enabled devices.
- Realtime data: mielesolar does not use the restricted
  [SolarEdge Monitoring API](https://www.solaredge.com/sites/default/files/se_monitoring_api.pdf) (only 300 requests per
  day, data resolution of 15 minutes) or unsupported workarounds like scraping the monitoring platform, but talks to
  your inverter directly over the local network. This is fast and provides realtime data.

## Usage

### 1. Enable MODBUS over TCP on your inverter

This can be done without opening the inverter.
See [documentation](https://www.solaredge.com/sites/default/files/sunspec-implementation-technical-note.pdf).

### 2. Get Miele API credentials

The tool uses the [Miele 3rd Party API](https://developer.miele.com/) to communicate with your appliances and requires
API credentials to do so. You can request access at https://www.miele.com/f/com/en/register_api.aspx.
Pass the obtained credentials and your Miele@Home account through `MIELE_USERNAME`, `MIELE_PASSWORD`, `MIELE_CLIENT_ID`,
and `MIELE_CLIENT_SECRET` environment variables to `mielesolar`.

### 3. Run `mielesolar`

There are several options to run the tool on a variety of machines, including Raspberry Pi or common NAS hardware. Set
the `vg` parameter to the locale where you registered your Miele@Home account, e.g. `de-DE` or `de-CH`.

#### Native
```
go get github.com/IngmarStein/mielesolar
MIELE_USERNAME=xxx MIELE_PASSWORD=xxx MIELE_CLIENT_ID=xxx MIELE_CLIENT_SECRET=xxx mielesolar -inverter $IP -port 502 -vg de-DE -auto 500
```

#### In a container
```
docker run --env MIELE_USERNAME=xxx --env MIELE_PASSWORD=xxx --env MIELE_CLIENT_ID=xxx --env MIELE_CLIENT_SECRET=xxx ingmarstein/mielesolar -inverter $IP -port 502 -vg de-DE -auto 500
```
Alternatively, use the included `docker-compose.yml` file as a template if you prefer to use Docker Compose.

### 4. Start your Miele appliance

When starting your dishwasher, washing machine, tumbler, etc. use the "SmartStart" option which delays the start of a
program until enough solar power is generated or until a specified time, whichever comes first. If this is not
available, enable the SmartGrid option. `mielesolar` will find the devices in the `PROGRAMMED_WAITING_TO_START` state
and start them whenever your SolarEdge inverter signals sufficient power generation.

## Advanced configuration

A configuration file can be used to define a priority order in which to launch the appliances and to customize their
power consumption.

In order to use the configuration file, don't use the `-auto` parameter which otherwise defines a common power
consumption value for all appliances and replace it with `-config $file` using the following format:

```json
[
  {
    "id": "000xxxxxxxxx",
    "power": 200
  },
  {
    "id": "000yyyyyyyyy",
    "power": 500
  }
]
```

The order of the devices defines the priority, i.e. the order in which the devices are started if the inverter
produces surplus power.
In the example above, if 600 W power is available, `mielesolar` would start the first device consuming 200 W, but the
remaining 400 W are not enough to also start the second.

The power values don't need to be exact and should be chosen large enough to not start the devices too early.

A device's identifier is also called "serial number" or "fabnumber" and can be found in the Miele@Home app (include the
leading zeros).
