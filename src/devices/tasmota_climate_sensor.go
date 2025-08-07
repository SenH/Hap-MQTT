package devices

import (
	"encoding/json"
	"fmt"
	"strings"

	"senhaerens.be/hap-mqtt/config"

	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/service"
	"github.com/charmbracelet/log"
	"github.com/eclipse/paho.mqtt.golang"
)

const (
	CO2LevelsAbnormalThreshold = 1600
)

// Use pointer values so we can check for 'nil'
type TcsSensor struct {
	*TcsBme280 `json:"BME280"`
	*TcsMhz19b `json:"MHZ19B"`
}

type TcsBme280 struct {
	Temperature *float64 `json:"Temperature"`
	Humidity    *float64 `json:"Humidity"`
}

type TcsMhz19b struct {
	CarbonDioxide *float64 `json:"CarbonDioxide"`
}

type TasmotaClimateSensor struct {
	*accessory.A
	*service.TemperatureSensor
	*service.HumiditySensor
	*service.CarbonDioxideSensor
	*characteristic.CarbonDioxideLevel
	*characteristic.CarbonDioxidePeakLevel
	config config.Device
}

func NewTasmotaClimateSensor(id int, config config.Device) *TasmotaClimateSensor {
	name := config.Name
	model := "Climate Sensor"
	if config.FriendlyName != "" {
		name = config.FriendlyName
		model = fmt.Sprintf("%s (%s)", model, config.Name)
	}

	a := TasmotaClimateSensor{}
	a.A = accessory.New(accessory.Info{
		Name:         name,
		Model:        model,
		Manufacturer: "Tasmota",
	}, accessory.TypeSensor)
	a.Id = uint64(id)
	log.Infof("HAP Create Accessory %4d - %s", a.Id, config.Name)

	a.TemperatureSensor = service.NewTemperatureSensor()
	a.AddS(a.TemperatureSensor.S)

	a.HumiditySensor = service.NewHumiditySensor()
	a.AddS(a.HumiditySensor.S)

	if len(config.Options) == 0 || config.Options[0] != "noco2" {
		a.CarbonDioxideSensor = service.NewCarbonDioxideSensor()

		a.CarbonDioxideLevel = characteristic.NewCarbonDioxideLevel()
		a.CarbonDioxideSensor.AddC(a.CarbonDioxideLevel.C)

		a.CarbonDioxidePeakLevel = characteristic.NewCarbonDioxidePeakLevel()
		a.CarbonDioxideSensor.AddC(a.CarbonDioxidePeakLevel.C)

		a.AddS(a.CarbonDioxideSensor.S)
	}

	a.config = config

	return &a
}

func (a *TasmotaClimateSensor) Listen(client mqtt.Client) {
	// MQTT -> HAP
	subLwt := fmt.Sprintf("tele/%s/LWT", a.config.Name)
	client.Subscribe(subLwt, 1, func(_ mqtt.Client, msg mqtt.Message) {
		msg.Ack()
		payload := string(msg.Payload())
		log.Debugf("MQTT received %s from %s", payload, msg.Topic())

		if strings.ToLower(payload) == "offline" {
			log.Infof("MQTT %s is offline", a.config.Name)
		}
	})

	subSensor := fmt.Sprintf("tele/%s/SENSOR", a.config.Name)
	client.Subscribe(subSensor, 1, func(_ mqtt.Client, msg mqtt.Message) {
		msg.Ack()
		log.Debugf("MQTT received %s from %s", msg.Payload(), msg.Topic())

		var sensor TcsSensor
		err := json.Unmarshal(msg.Payload(), &sensor)
		if err != nil {
			log.Error("Failed to decode JSON payload", "err", err)
			return
		}
		// log.Debugf("%+v", sensor)

		// Temperature & Humidity sensor
		if sensor.Temperature == nil || sensor.Humidity == nil {
			log.Error("BME280 sensor data is missing")
			return
		}
		a.CurrentTemperature.SetValue(*sensor.Temperature)
		a.CurrentRelativeHumidity.SetValue(*sensor.Humidity)

		// CarbonDioxide sensor
		if (len(a.config.Options) > 0 && a.config.Options[0] == "noco2") || sensor.CarbonDioxide == nil {
			return
		}

		if *sensor.CarbonDioxide > CO2LevelsAbnormalThreshold {
			a.CarbonDioxideDetected.SetValue(characteristic.CarbonDioxideDetectedCO2LevelsAbnormal)
		} else {
			a.CarbonDioxideDetected.SetValue(characteristic.CarbonDioxideDetectedCO2LevelsNormal)
		}

		if *sensor.CarbonDioxide > a.CarbonDioxidePeakLevel.Value() {
			a.CarbonDioxidePeakLevel.SetValue(*sensor.CarbonDioxide)
		}

		a.CarbonDioxideLevel.SetValue(*sensor.CarbonDioxide)
	})
}
