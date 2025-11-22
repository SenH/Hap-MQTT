package devices

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"senhaerens.be/hap-mqtt/config"
	"senhaerens.be/hap-mqtt/service"

	"github.com/brutella/hap/accessory"
	"github.com/charmbracelet/log"
	"github.com/eclipse/paho.mqtt.golang"
)

// Use pointer values so we can check for 'nil'
type SdStatus struct {
	Output     *bool `json:"output"`
	Brightness *int  `json:"brightness"`
}

type ShellyDimmer struct {
	*accessory.A
	*service.DimmableLightbulb
	config config.Device
}

func NewShellyDimmer(id int, config config.Device) *ShellyDimmer {
	name := config.Name
	model := "Dimmer"
	if config.FriendlyName != "" {
		name = config.FriendlyName
		model = fmt.Sprintf("%s (%s)", model, config.Name)
	}

	a := ShellyDimmer{}
	a.A = accessory.New(accessory.Info{
		Name:         name,
		Model:        model,
		Manufacturer: "Shelly",
	}, accessory.TypeLightbulb)
	a.Id = uint64(id)
	log.Infof("HAP Create Accessory %4d - %s", a.Id, config.Name)

	a.DimmableLightbulb = service.NewDimmableLightbulb()
	a.AddS(a.DimmableLightbulb.S)

	a.config = config

	return &a
}

func (a *ShellyDimmer) Accessory() *accessory.A {
	return a.A
}

func (a *ShellyDimmer) Listen(client mqtt.Client) {
	// MQTT -> HAP
	subLwt := fmt.Sprintf("shellies/%s/online", a.config.Name)
	client.Subscribe(subLwt, 1, func(_ mqtt.Client, msg mqtt.Message) {
		msg.Ack()
		payload := string(msg.Payload())
		log.Debugf("MQTT received %s from %s", payload, msg.Topic())

		if strings.ToLower(payload) == "false" {
			log.Infof("MQTT %s is offline", a.config.Name)
		}
	})

	subStatus := fmt.Sprintf("shellies/%s/status/light:0", a.config.Name)
	client.Subscribe(subStatus, 1, func(_ mqtt.Client, msg mqtt.Message) {
		msg.Ack()
		log.Debugf("MQTT received %s from %s", msg.Payload(), msg.Topic())
		var status SdStatus
		err := json.Unmarshal(msg.Payload(), &status)
		if err != nil {
			log.Error("Failed to decode JSON payload", "err", err)
			return
		}

		if status.Output == nil || status.Brightness == nil {
			log.Error("Dimmer status data is missing")
			return
		}

		a.On.SetValue(*status.Output)
		a.Brightness.SetValue(*status.Brightness)
	})

	// HAP -> MQTT
	pubStatus := fmt.Sprintf("shellies/%s/command/light:0", a.config.Name)
	a.Brightness.OnValueRemoteUpdate(func(brightness int) {
		payload := fmt.Sprintf("set,false,0")
		if brightness > 0 {
			payload = fmt.Sprintf("set,true,%d", brightness)
		}
		token := client.Publish(pubStatus, 1, false, payload)
		token.Wait()
		log.Debugf("MQTT published %d to %s", payload, pubStatus)
	})

	a.On.OnValueRemoteUpdate(func(on bool) {
		payload := fmt.Sprintf("set,%s", strconv.FormatBool(on))
		token := client.Publish(pubStatus, 1, false, payload)
		token.Wait()
		log.Debugf("MQTT published %s to %s", payload, pubStatus)
	})
}
