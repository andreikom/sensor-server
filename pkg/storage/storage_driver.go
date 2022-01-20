package storage

import (
	"fmt"
	"sync"
)

var once sync.Once

const (
	Filesystem = "filesystem"
)

type Driver struct{}

var driverInstance *Driver

type DriverStorage interface {
	Init()
	SaveSensorData(sensorId string, data []byte) error
	GetAvailableSensors() ([]string, error)
	GetSensorData(sensorId string) ([]byte, error)
}

func InitStorage(requestedDriver string) *Driver {
	if driverInstance != nil {
		fmt.Println("Note: a storage driver has been already initialized")
	}
	once.Do(func() {
		switch {
		case requestedDriver == Filesystem:
			{
				driverInstance = &Driver{}
				break
			}
		}
		driverInstance.Init()
		if driverInstance == nil {
			panic("No storage initialized")
		}
	})
	return driverInstance
}
