# Mielesolar

Start Miele@Home appliances when your SolarEdge inverter produces surplus power.

## Usage

### 1. Enable MODBUS over TCP on your inverter

This can be done without opening the inverter.
See [documentation](https://www.solaredge.com/sites/default/files/sunspec-implementation-technical-note.pdf).

### 2. Create a configuration file

The configuration file (usually `devices.json`) lists your Miele@Home appliances and their power consumption.
Example:

```json
[
  {
    "id": "dishwasher",
    "power": 200
  },
  {
    "id": "washing machine",
    "power": 500
  }
]
```

The order of the devices defines the priority, i.e. the order in which the devices are started if the inverter
produces surplus power.
In the example above, if 600 W power is available, `mielesolar` would start the first device consuming 200 W, but the
remaining 400 W are not enough to also start the second.

The power values don't need to be exact and should be chosen large enough to not start the devices too early.

A device's identifier is also called "serial number" or "fabnumber" and can be found in the Miele@Home app.

### 3. Get Miele API credentials

The tool uses the [Miele 3rd Party API](https://developer.miele.com/) to communicate with your appliances and requires
API credentials to do so. This usually means writing an email to developer@miele.com. Pass the obtained credentials and
your Miele@Home account through `MIELE_USERNAME`, `MIELE_PASSWORD`, `MIELE_CLIENT_ID`, and `MIELE_CLIENT_SECRET`
environment variables to `mielesolar`.

### 4. Run `mielesolar`

There are several options to run the tool on a variety of machines, including Raspberry Pi or common NAS hardware. Set
the `vg` parameter to the locale where you registered your Miele@Home account, e.g. `de-DE` or `de-CH`.

#### Native
```
go get github.com/IngmarStein/mielesolar
MIELE_USERNAME=xxx MIELE_PASSWORD=xxx MIELE_CLIENT_ID=xxx MIELE_CLIENT_SECRET=xxx mielesolar -config devices.json -inverter $IP -port 502 -vg de-DE
```

#### In a container
```
docker run --mount type=bind,source="$(pwd)"/devices.json,target=/devices.json --env MIELE_USERNAME=xxx --env MIELE_PASSWORD=xxx --env MIELE_CLIENT_ID=xxx --env MIELE_CLIENT_SECRET=xxx ingmarstein/mielesolar -inverter $IP -port 502 -vg de-DE
```
Alternatively, use the included `docker-compose.yml` file as a template if you prefer to use Docker Compose.

### 5. Start your Miele appliance

When starting your dishwasher, washing machine, tumbler, etc. use the "SmartStart" option which delays the start of a
program until enough solar power is generated or until a specified time, whichever comes first. `mielesolar` will find
the devices in the `PROGRAMMED_WAITING_TO_START` state and start them in the configured order whenever your SolarEdge
inverter signals sufficient power generation.
