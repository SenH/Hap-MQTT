# Hap-MQTT

## Description
A bridge between MQTT devices & Apple HomeKit accessories.
This project is published as an example and a source of inspiration because it is mostly adapted to my own home automation.

## Installation
* Build binary with `go build -o ../bin/`
* Copy `data/config.example.yml` to `data/config.yml` and configure settings.
* Run with `./bin/hap-mqtt -config data/config.yml`

### Required libraries
* [Brutella/HAP](https://github.com/brutella/hap)
* [Charm_/Log](https://github.com/charmbracelet/log)
* [Paho/MQTT](https://github.com/eclipse/paho.mqtt.golang)

## Usage
```
Usage of ./hap-mqtt:
  -config string
    	Configuration filepath (default "data/config.yml")
  -debug
    	Enable debug log
  -debughap
    	Enable HAP debug log
  -printcfg
    	Print configuration
```

# Configuration
See `data/config.example.yml`.

## Contact Sensors
* MQTT subscription topic must be provided by first option in `config.yml`.

## EnOcean Dimmers
* `$DEVICE` is the device name set in `config.yml`.

#### MQTT subscription topics
* Dim value (0-100): `fhem/stat/$DEVICE/dim`
* State value (on-off): `fhem/stat/$DEVICE/state`
#### MQTT publishing topics
* Dim value (0-100): `fhem/cmnd/$DEVICE/dim`
* State value (on-off): `fhem/cmnd/$DEVICE/state`

## EnOcean Lightbulb
* `$DEVICE` is the device name set in `config.yml`.

#### MQTT subscription topic
* State value (on-off): `fhem/stat/$DEVICE/state`
#### MQTT publishing topic
* State value (on-off): `fhem/cmnd/$DEVICE/state`

## Shelly Dimmer Gen3
* `$DEVICE` is the device name set in `config.yml`.

#### MQTT subscription topic
JSON data (output, brightness): `shellies/$DEVICE/status/light:0`
#### MQTT publishing topic
string (set,$OUTPUT,$BRIGHTNESS) : `shellies/$DEVICE/command/light:0`

## Tasmota Climate Sensors
* Tasmota device with a BME280 (temperature, humidity) sensor and optional MHZ19B (CO2) sensor.
* MQTT subscription topic with JSON payload: `tele/$DEVICE/SENSOR`

## Tasmota Plugs
* `$OUTPUT` defaults to `POWER` but can be optionally set with first option in `config.yml`.

#### MQTT subscription topic
* Power value (ON-OFF): `stat/$DEVICE/$OUTPUT`
#### MQTT publishing topic
* Power value (ON-OFF): `cmnd/$DEVICE/$OUTPUT`
