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

type TasmotaPlug struct {
	*accessory.A
	*service.Lightbulb
	config config.Device
}

func NewTasmotaPlug(id int, config config.Device) *TasmotaPlug {
	name := config.Name
	model := "Plug"
	if config.FriendlyName != "" {
		name = config.FriendlyName
		model = fmt.Sprintf("%s (%s)", model, config.Name)
	}

	a := TasmotaPlug{}
	a.A = accessory.New(accessory.Info{
		Name:         name,
		Model:        model,
		Manufacturer: "Tasmota",
	}, accessory.TypeLightbulb)
	a.Id = uint64(id)
	log.Infof("HAP Create Accessory %4d - %s", a.Id, config.Name)

	a.Lightbulb = service.NewLightbulb()
	a.AddS(a.Lightbulb.S)

	a.config = config

	return &a
}

func (a *TasmotaPlug) Listen(client mqtt.Client) {
	// for Tasmota devices which have multiple outputs
	output := "POWER"
	if len(a.config.Options) > 0 && a.config.Options[0] != "" {
		output = a.config.Options[0]
	}

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

	subPower := fmt.Sprintf("stat/%s/%s", a.config.Name, output)
	client.Subscribe(subPower, 1, func(_ mqtt.Client, msg mqtt.Message) {
		msg.Ack()
		payload := string(msg.Payload())
		log.Debugf("MQTT received %s from %s", payload, msg.Topic())

		switch payload {
		case "ON":
			a.On.SetValue(true)
		case "OFF":
			a.On.SetValue(false)
		}
	})

	// HAP -> MQTT
	pubPower := fmt.Sprintf("cmnd/%s/%s", a.config.Name, output)
	a.On.OnValueRemoteUpdate(func(on bool) {
		payload := "OFF"
		if on == true {
			payload = "ON"
		}
		token := client.Publish(pubPower, 1, false, payload)
		token.Wait()
		log.Debugf("MQTT published %s to %s", payload, pubPower)
	})
}
