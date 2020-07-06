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
API credentials to do so. This usually means writing an email to developer@miele.com.

### 4. Run `mielesolar`

There are several options to run the tool on a variety of machines, including Raspberry Pi or
common NAS hardware:

#### Native
```
go get github.com/IngmarStein/mielesolar
mielesolar -config devices.json -inverter $IP -port 502
```

#### In a container
```
docker run --mount type=bind,source="$(pwd)"/devices.json,target=/devices.json ingmarstein/mielesolar -inverter $IP -port
```
Alternatively, use the included `docker-compose.yml` file as a template if you prefer to use Docker Compose.
