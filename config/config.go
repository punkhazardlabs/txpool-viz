package config

import (
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	_ "github.com/joho/godotenv/autoload"
	"gopkg.in/yaml.v3"
)

type Endpoint struct {
	Name        string
	RPCUrl      string						`yaml:"rpc_url"`
	Websocket   string						`yaml:"socket"`
	AuthHeaders map[string]string `yaml:"auth_headers"`
	Client			*ethclient.Client
}

type Config struct {
	Endpoints []Endpoint               `yaml:"endpoints"`
	Polling   map[string]time.Duration `yaml:"polling"`
	Filters   map[string]string        `yaml:"filters"`
}

func Load() (*Config, error) {
	userConfig := &Config{}
	cfgData, err := os.ReadFile("config.yaml")

	if err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	err = yaml.Unmarshal(cfgData, &userConfig)

	if err != nil {
		throwErr := fmt.Errorf("Error parsing config file: %v", err)
		panic(throwErr)
	}

	// Create clients for the Endpoints
	for i := range userConfig.Endpoints {
		client, err := ethclient.Dial(userConfig.Endpoints[i].RPCUrl)

		if err != nil {
			return nil, fmt.Errorf("Error connecting to client %s", userConfig.Endpoints[i].Name)
		}

		userConfig.Endpoints[i].Client = client
	}	

	return userConfig, nil
}
