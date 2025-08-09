package devices

import (
	"fmt"
	"strconv"
	"strings"

	"senhaerens.be/hap-mqtt/config"
	"senhaerens.be/hap-mqtt/service"

	"github.com/brutella/hap/accessory"
	"github.com/charmbracelet/log"
	"github.com/eclipse/paho.mqtt.golang"
)

type EnOceanDimmer struct {
	*accessory.A
	*service.DimmableLightbulb
	config config.Device
}

func NewEnOceanDimmer(id int, config config.Device) *EnOceanDimmer {
	name := config.Name
	model := "Dimmer"
	if config.FriendlyName != "" {
		name = config.FriendlyName
		model = fmt.Sprintf("%s (%s)", model, config.Name)
	}

	a := EnOceanDimmer{}
	a.A = accessory.New(accessory.Info{
		Name:         name,
		Model:        model,
		Manufacturer: "Eltako",
	}, accessory.TypeLightbulb)
	a.Id = uint64(id)
	log.Infof("HAP Create Accessory %4d - %s", a.Id, config.Name)

	a.DimmableLightbulb = service.NewDimmableLightbulb()
	a.AddS(a.DimmableLightbulb.S)

	a.config = config

	return &a
}

func (a *EnOceanDimmer) Accessory() *accessory.A {
	return a.A
}

func (a *EnOceanDimmer) Listen(client mqtt.Client) {
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

	subDim := fmt.Sprintf("fhem/stat/%s/dim", a.config.Name)
	client.Subscribe(subDim, 1, func(_ mqtt.Client, msg mqtt.Message) {
		msg.Ack()
		payload := string(msg.Payload())
		log.Debugf("MQTT received %s from %s", payload, msg.Topic())
		brightness, _ := strconv.Atoi(payload)
		a.Brightness.SetValue(brightness)
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
	pubDim := fmt.Sprintf("fhem/cmnd/%s/dim", a.config.Name)
	a.Brightness.OnValueRemoteUpdate(func(brightness int) {
		token := client.Publish(pubDim, 1, false, fmt.Sprintf("%d", brightness))
		token.Wait()
		log.Debugf("MQTT published %d to %s", brightness, pubDim)
	})

	pubState := fmt.Sprintf("fhem/cmnd/%s/state", a.config.Name)
	a.On.OnValueRemoteUpdate(func(on bool) {
		// Only publish "off" state. "On" state is implied by dim value.
		// Otherwise the light briefly goes to 100% before going to the dim value.
		if on == false {
			payload := "off"
			token := client.Publish(pubState, 1, false, payload)
			token.Wait()
			log.Debugf("MQTT published %s to %s", payload, pubState)
		}
	})
}
