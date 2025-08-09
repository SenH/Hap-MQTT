package devices

import (
	"fmt"
	"strings"

	"senhaerens.be/hap-mqtt/config"

	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/service"
	"github.com/charmbracelet/log"
	"github.com/eclipse/paho.mqtt.golang"
)

type EnOceanLightbulb struct {
	*accessory.A
	*service.Lightbulb
	config config.Device
}

func NewEnOceanLightbulb(id int, config config.Device) *EnOceanLightbulb {
	name := config.Name
	model := "Lightbulb"
	if config.FriendlyName != "" {
		name = config.FriendlyName
		model = fmt.Sprintf("%s (%s)", model, config.Name)
	}

	a := EnOceanLightbulb{}
	a.A = accessory.New(accessory.Info{
		Name:         name,
		Model:        model,
		Manufacturer: "Eltako",
	}, accessory.TypeLightbulb)
	a.Id = uint64(id)
	log.Infof("HAP Create Accessory %4d - %s", a.Id, config.Name)

	a.Lightbulb = service.NewLightbulb()
	a.AddS(a.Lightbulb.S)

	a.config = config

	return &a
}

func (a *EnOceanLightbulb) Accessory() *accessory.A {
	return a.A
}

func (a *EnOceanLightbulb) Listen(client mqtt.Client) {
	// MQTT -> HAP
	subLwt := "fhem"
	client.Subscribe(subLwt, 1, func(_ mqtt.Client, msg mqtt.Message) {
		msg.Ack()
		payload := string(msg.Payload())
		log.Debugf("MQTT received %s from %s", payload, msg.Topic())

		if strings.ToLower(payload) == "offline" {
			log.Infof("MQTT %s is offline", a.config.Name)
		}
	})

	subState := fmt.Sprintf("fhem/stat/%s/state", a.config.Name)
	client.Subscribe(subState, 1, func(_ mqtt.Client, msg mqtt.Message) {
		msg.Ack()
		payload := string(msg.Payload())
		log.Debugf("MQTT received %s from %s", payload, msg.Topic())

		switch strings.ToLower(payload) {
		case "on":
			a.On.SetValue(true)
		case "off":
			a.On.SetValue(false)
		}
	})

	// HAP -> MQTT
	pubState := fmt.Sprintf("fhem/cmnd/%s/state", a.config.Name)
	a.On.OnValueRemoteUpdate(func(on bool) {
		payload := "off"
		if on == true {
			payload = "on"
		}
		token := client.Publish(pubState, 1, false, payload)
		token.Wait()
		log.Debugf("MQTT published %s to %s", payload, pubState)
	})
}
