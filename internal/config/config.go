package config

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/ethclient"
	_ "github.com/joho/godotenv/autoload"
	"gopkg.in/yaml.v3"
)

type Endpoint struct {
	Name        string            `yaml:"name" json:"name"`
	RPCUrl      string            `yaml:"rpc_url" json:"rpc_url"`
	Websocket   string            `yaml:"socket" json:"socket"`
	AuthHeaders map[string]string `yaml:"auth_headers" json:"auth_headers"`
	Client      *ethclient.Client `yaml:"-" json:"-"`
}

type BeaconEndpoint struct {
	Name      string `yaml:"name" json:"name"`
	BeaconUrl string `yaml:"beacon_url" json:"beacon_url"`
}

type Config struct {
	Endpoints    []Endpoint       `yaml:"endpoints" json:"endpoints"`
	BeaconUrls   []BeaconEndpoint `yaml:"beacon_urls" json:"beacon_urls"`
	Polling      Polling          `yaml:"polling" json:"polling"`
	Filters      Filters          `yaml:"filters" json:"filters"`
	LogLevel     string           `yaml:"log_level" json:"log_level"`
	FocilEnabled string           `yaml:"focil_enabled" json:"focil_enabled"`
}

type Polling struct {
	Interval string `yaml:"interval" json:"interval"`
	Timeout  string `yaml:"timeout" json:"timeout"`
}

type Filters struct {
	MinGasPrice string `yaml:"min_gas_price" json:"min_gas_price"`
}

func Load() (*Config, error) {
	userConfig := &Config{}
	// Attempt to read config.yaml first
	cfgData, err := os.ReadFile("cfg/config.yaml")
	if err == nil {
		err = yaml.Unmarshal(cfgData, userConfig)
		if err != nil {
			return nil, fmt.Errorf("Error parsing config.yaml: %v", err.Error())
		}
	}

	// Create clients for the Endpoints
	for i := range userConfig.Endpoints {
		client, err := ethclient.Dial(userConfig.Endpoints[i].RPCUrl)

		if err != nil {
			return nil, fmt.Errorf("Error connecting to client %s. rpc url: %s", userConfig.Endpoints[i].Name, userConfig.Endpoints[i].RPCUrl)
		}

		userConfig.Endpoints[i].Client = client
	}

	return userConfig, nil
}
