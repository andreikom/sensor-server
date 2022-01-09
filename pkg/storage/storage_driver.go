package storage

import (
	"github.com/andreikom/sensor-server/pkg"
	"sync"
)

var once sync.Once

const (
	Filesystem = "filesystem"
)

var Driver storageDriver

type storageDriver interface {
	Init()
	SaveTemperature(sensorId string, data string) error
	GetHourlyTempsBySensorIDAndDate() (map[int]pkg.HourlyTempModel, error)
	GetAllSensorsDailyTemperatures() ([]int, error)
	CleanOldDailyEntry() error
}

func InitStorage(storageDriver string) {
	once.Do(func() {
		switch {
		case storageDriver == Filesystem:
			{
				Driver = &FileSystemDriver{}
				break
			}
		}
		Driver.Init()
		if Driver == nil {
			panic("No storage initialized")
		}
	})
}
