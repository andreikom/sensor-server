package main

import (
	"fmt"
	"github.com/andreikom/sensor-server/pkg/api"
	"github.com/andreikom/sensor-server/pkg/utils"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
	"strconv"
)

const (
	ConfigFile = "config.yml"
)

func main() {
	cfg := resolveConfig()
	api.Start(cfg)
}

func resolveConfig() *api.Config {
	cfg := &api.Config{}
	err := resolveFromYaml(cfg)
	if err != nil {
		err := resolveEnvVars(cfg)
		if err != nil {
			return cfg
		}
		return cfg
	}
	return cfg
}

func resolveFromYaml(cfg *api.Config) error {
	userHome := utils.GetUserHome()
	configFile, err := os.Open(filepath.Join(userHome, ConfigFile))
	defer configFile.Close()
	if err != nil {
		fmt.Printf("Could not get '" + ConfigFile + "' file. Attempting to resolve from env vars\n")
		return err
	}
	decoder := yaml.NewDecoder(configFile)
	err = decoder.Decode(cfg)
	if err != nil {
		fmt.Printf("Could not get decode'" + ConfigFile + "' file. Attempting to resolve from env vars\n")
		return err
	}
	fmt.Printf("Decoded yaml config file succcesfully\n")
	return nil
}

func resolveEnvVars(cfg *api.Config) error {
	envPort := os.Getenv("sensor_server_port")
	if envPort != "" {
		if envPortInt, err := strconv.Atoi(envPort); err != nil {
			fmt.Printf("Error reading port from env: %d\n", err)
			return err
		} else {
			cfg.Server.Port = envPortInt
		}
	}
	fmt.Printf("sensor_server_port is not defined\n")
	return nil
}
