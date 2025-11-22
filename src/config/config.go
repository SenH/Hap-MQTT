package config

type Device struct {
	Name         string   `yaml:"name"`
	FriendlyName string   `yaml:"friendly_name"`
	Options      []string `yaml:"options"`
}

type Config struct {
	Hap struct {
		Dbdir  string   `yaml:"db_dir"`
		Ifaces []string `yaml:"ifaces"`
		Addr   string   `yaml:"address"`
		Pin    string   `yaml:"pin"`
	} `yaml:"hap"`

	Mqtt struct {
		Broker   string `yaml:"broker"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		ClientID string `yaml:"client_id"`
	} `yaml:"mqtt"`

	Devices struct {
		ContactSensors        []Device `yaml:"contact_sensors"`
		EnOceanDimmers        []Device `yaml:"enocean_dimmers"`
		EnOceanLightbulbs     []Device `yaml:"enocean_lightbulbs"`
		ShellyDimmers         []Device `yaml:"shelly_dimmers"`
		TasmotaClimateSensors []Device `yaml:"tasmota_climate_sensors"`
		TasmotaPlugs          []Device `yaml:"tasmota_plugs"`
	} `yaml:"devices"`
}
