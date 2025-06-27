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
		TasmotaPlugs          []Device `yaml:"tasmota_plugs"`
		EnOceanDimmers        []Device `yaml:"enocean_dimmers"`
		TasmotaClimateSensors []Device `yaml:"tasmota_climate_sensors"`
		ContactSensors        []Device `yaml:"contact_sensors"`
	} `yaml:"devices"`
}
