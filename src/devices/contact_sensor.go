package devices

import (
	"fmt"

	"senhaerens.be/hap-mqtt/config"

	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/service"
	"github.com/charmbracelet/log"
	"github.com/eclipse/paho.mqtt.golang"
)

type ContactSensor struct {
	*accessory.A
	*service.ContactSensor
	config config.Device
}

func NewContactSensor(id int, config config.Device) *ContactSensor {
	name := config.Name
	model := "Contact Sensor"
	if config.FriendlyName != "" {
		name = config.FriendlyName
		model = fmt.Sprintf("%s (%s)", model, config.Name)
	}

	a := ContactSensor{}
	a.A = accessory.New(accessory.Info{
		Name:  name,
		Model: model,
	}, accessory.TypeSensor)
	a.Id = uint64(id)
	log.Infof("HAP Create Accessory %4d - %s", a.Id, config.Name)

	a.ContactSensor = service.NewContactSensor()
	a.ContactSensorState.SetValue(characteristic.ContactSensorStateContactNotDetected)
	a.AddS(a.ContactSensor.S)

	a.config = config

	return &a
}

func (a *ContactSensor) Accessory() *accessory.A {
	return a.A
}

func (a *ContactSensor) Listen(client mqtt.Client) {
	if len(a.config.Options) == 0 || a.config.Options[0] == "" {
		log.Error("Field \"Output\" in device config is missing")
		return
	}

	// MQTT -> HAP
	client.Subscribe(a.config.Options[0], 1, func(_ mqtt.Client, msg mqtt.Message) {
		msg.Ack()
		payload := string(msg.Payload())
		log.Debugf("MQTT received %s from %s", payload, msg.Topic())

		switch payload {
		case "ON":
			a.ContactSensorState.SetValue(characteristic.ContactSensorStateContactDetected)
		case "OFF":
			a.ContactSensorState.SetValue(characteristic.ContactSensorStateContactNotDetected)
		}
	})
}
