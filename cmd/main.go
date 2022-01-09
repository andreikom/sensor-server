package main

import (
	"fmt"
	"github.com/andreikom/sensor-server/pkg/api"
	"os"
	"strconv"
)

var port = 8080

func init() {
	envPort := os.Getenv("sensor_server_port")
	if envPort != "" {
		if envPortInt, err := strconv.Atoi(envPort); err != nil {
			fmt.Printf("Error reading port from env: %d\n", err)
			return
		} else {
			port = envPortInt
		}
	}
}

func main() {
	api.Start(
		&api.Config{Port: port},
	)
	fmt.Printf("Starting Sensor Server, port: %d", port)
}
