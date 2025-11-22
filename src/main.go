package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"senhaerens.be/hap-mqtt/config"
	"senhaerens.be/hap-mqtt/devices"

	"github.com/brutella/hap"
	"github.com/brutella/hap/accessory"
	haplog "github.com/brutella/hap/log"
	"github.com/charmbracelet/log"
	"github.com/eclipse/paho.mqtt.golang"
	"gopkg.in/yaml.v2"
)

const (
	programName string = "hap-mqtt"
)

var (
	configPath  = flag.String("config", "data/config.yml", "Configuration filepath")
	printConfig = flag.Bool("printcfg", false, "Print configuration")
	debugLog    = flag.Bool("debug", false, "Enable debug log")
	debugHapLog = flag.Bool("debughap", false, "Enable HAP debug log")
)

func setupConfig(fpath string, print bool) config.Config {
	f, err := os.Open(fpath)
	if err != nil {
		log.Fatal("Config filepath not found", "error", err)
	}
	defer f.Close()

	var cfg config.Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		log.Fatal("Failed decoding configuration", "error", err)
	}

	if print {
		d, err := yaml.Marshal(&cfg)
		if err != nil {
			log.Fatal("Failed printing configuration", "error", err)
		}
		fmt.Printf("# %s\n%s\n", fpath, string(d))
		os.Exit(0)
	}

	return cfg
}

func setupMqtt(cfg config.Config) *mqtt.ClientOptions {
	opts := mqtt.NewClientOptions()
	if cfg.Mqtt.Broker == "" {
		log.Fatal("MQTT broker is not specified in configuration")
	}
	opts.AddBroker(cfg.Mqtt.Broker)

	if cfg.Mqtt.Username != "" {
		opts.SetUsername(cfg.Mqtt.Username)
	}
	if cfg.Mqtt.Password != "" {
		opts.SetPassword(cfg.Mqtt.Password)
	}

	if cfg.Mqtt.ClientID == "" {
		cfg.Mqtt.ClientID = programName
	}
	log.Debug("MQTT Set", "Clientid", cfg.Mqtt.ClientID)
	opts.SetClientID(cfg.Mqtt.ClientID)

	opts.OnConnect = func(_ mqtt.Client) {
		log.Info("MQTT connected", "broker", opts.Servers)
	}

	opts.OnConnectionLost = func(_ mqtt.Client, err error) {
		log.Error("MQTT connection lost", "error", err)
	}

	return opts
}

func setupSignals() context.Context {
	chanSigs := make(chan os.Signal, 1)
	signal.Notify(chanSigs, os.Interrupt, os.Kill, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sig := <-chanSigs
		log.Debug("Received", "signal", sig)
		log.Info("Stopping " + programName)
		signal.Stop(chanSigs)
		cancel()
	}()

	return ctx
}

type devicer interface {
	Listen(mqtt.Client)
	Accessory() *accessory.A
}

type deviceOptions struct {
	configs     []config.Device
	offset      int
	mqttClient  mqtt.Client
	accessories *[]*accessory.A
}

func makeDevices[T devicer](newDevice func(int, config.Device) T, opts deviceOptions) []T {
	devices := make([]T, len(opts.configs))

	for i, config := range opts.configs {
		device := newDevice(i+opts.offset, config)
		device.Listen(opts.mqttClient)
		devices[i] = device
		*opts.accessories = append(*opts.accessories, device.Accessory())
	}

	return devices
}

func main() {
	flag.Parse()

	// Do not output timestamp when running under systemd
	if a, b := os.Getenv("INVOCATION_ID"), os.Getenv("JOURNAL_STREAM"); a != "" && b != "" {
		log.SetReportTimestamp(false)
	}

	if *debugHapLog {
		haplog.Debug.Enable()
	}

	// Setup config
	cfg := setupConfig(*configPath, *printConfig)
	if *debugLog {
		log.SetLevel(log.DebugLevel)
	}

	// Setup MQTT client
	mqttOpts := setupMqtt(cfg)
	mqttClient := mqtt.NewClient(mqttOpts)
	log.Debug("Starting MQTT client")
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal("MQTT could not connect", "token", token.Error())
	}

	// Setup HAP Bridge
	hapBridge := accessory.NewBridge(accessory.Info{
		Name: programName,
	})
	hapBridge.Id = 1
	log.Infof("HAP Create Accessory %4d - %s (Bridge)", hapBridge.Id, hapBridge.A.Name())

	// Setup HAP Accessories
	var accessories []*accessory.A

	makeDevices[*devices.TasmotaPlug](devices.NewTasmotaPlug, deviceOptions{
		configs:     cfg.Devices.TasmotaPlugs,
		offset:      2,
		mqttClient:  mqttClient,
		accessories: &accessories,
	})

	makeDevices[*devices.EnOceanDimmer](devices.NewEnOceanDimmer, deviceOptions{
		configs:     cfg.Devices.EnOceanDimmers,
		offset:      100,
		mqttClient:  mqttClient,
		accessories: &accessories,
	})

	makeDevices[*devices.TasmotaClimateSensor](devices.NewTasmotaClimateSensor, deviceOptions{
		configs:     cfg.Devices.TasmotaClimateSensors,
		offset:      200,
		mqttClient:  mqttClient,
		accessories: &accessories,
	})

	makeDevices[*devices.ContactSensor](devices.NewContactSensor, deviceOptions{
		configs:     cfg.Devices.ContactSensors,
		offset:      300,
		mqttClient:  mqttClient,
		accessories: &accessories,
	})

	makeDevices[*devices.EnOceanLightbulb](devices.NewEnOceanLightbulb, deviceOptions{
		configs:     cfg.Devices.EnOceanLightbulbs,
		offset:      400,
		mqttClient:  mqttClient,
		accessories: &accessories,
	})

	makeDevices[*devices.ShellyDimmer](devices.NewShellyDimmer, deviceOptions{
		configs:     cfg.Devices.ShellyDimmers,
		offset:      500,
		mqttClient:  mqttClient,
		accessories: &accessories,
	})

	log.Debugf("%d HAP Accessories", len(accessories))

	// Setup HAP filestore
	err := os.MkdirAll(cfg.Hap.Dbdir, 0750)
	if err != nil {
		log.Fatal("Failed creating HAP dbdir", "error", err)
	}
	hapFs := hap.NewFsStore(cfg.Hap.Dbdir)

	// Setup HAP server
	hapServer, err := hap.NewServer(hapFs, hapBridge.A, accessories...)
	if err != nil {
		log.Fatal("Failed to create HAP server", "error", err)
	}

	hapServer.Ifaces = cfg.Hap.Ifaces
	hapServer.Addr = cfg.Hap.Addr
	hapServer.Pin = cfg.Hap.Pin

	ctx := setupSignals()
	log.Debug("Starting HAP server")
	log.Debugf("%d Goroutines exist", runtime.NumGoroutine())
	if err := hapServer.ListenAndServe(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("Failed to start HAP server", "error", err)
	}

	log.Debugf("%d Goroutines exist", runtime.NumGoroutine())
}
