package config

import (
	"encoding/json"
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

type Config struct {
	Endpoints    []Endpoint `yaml:"endpoints" json:"endpoints"`
	BeaconSSEUrl string     `yaml:"beacon_sse_url" json:"beacon_sse_url"`
	Polling      Polling    `yaml:"polling" json:"polling"`
	Filters      Filters    `yaml:"filters" json:"filters"`
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
	cfgData, err := os.ReadFile("config.yaml")
	if err == nil {
		err = yaml.Unmarshal(cfgData, userConfig)
		if err != nil {
			return nil, fmt.Errorf("Error parsing config.yaml: %v", err)
		}
	} else {
		// If config.yaml is not provided the tool will check CONFIG_JSON for cluster settings
		configJson := os.Getenv("CONFIG_JSON")

		if configJson == "" {
			return nil, fmt.Errorf("No config.yaml found and CONFIG_JSON not set — please provide a config.yaml")
		}

		err = json.Unmarshal([]byte(configJson), userConfig)
		if err != nil {
			return nil, fmt.Errorf("Error parsing CONFIG_JSON: %v", err)
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
