package storage

import (
	"fmt"
	"github.com/andreikom/sensor-server/pkg"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type FileSystemDriver struct{}

var storePath, temperatureStorePath string

func initUserHome() string {
	store, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error occured while initiliazing storage: %s\n", err)
	}
	return store
}

func (d FileSystemDriver) Init() {
	storePath = initUserHome()
	temperatureStorePath = filepath.Join(storePath, "temperatures")
}

// TODO [andreik]: protect writing temperatures and reading to in-memory cache
func (d FileSystemDriver) SaveTemperature(sensorId string, data string) error {
	now := time.Now()
	folderPath := filepath.Join(temperatureStorePath, sensorId, now.Format("01-02-2006"),
		strconv.FormatInt(int64(now.Hour()), 10), strconv.FormatInt(int64(now.Minute()), 10))
	if err := os.MkdirAll(folderPath, os.ModePerm); err != nil {
		fmt.Println("Could not have created a folder at: " + folderPath)
	}
	filePath := filepath.Join(folderPath, data)
	file, err := os.Create(filePath)
	defer file.Close()
	if err != nil {
		fmt.Println("Could not have created a file at: " + filePath)
		return err
	}
	return nil
}

// TODO [andreik]: implement all below
func (d FileSystemDriver) GetHourlyTempsBySensorIDAndDate() (map[int]pkg.HourlyTempModel, error) {
	return make(map[int]pkg.HourlyTempModel), nil // TODO [andreik]: add initial capacity of weekly days
}

func (d FileSystemDriver) GetAllSensorsDailyTemperatures() ([]int, error) {
	return make([]int, 0), nil
}

func (d FileSystemDriver) CleanOldDailyEntry() error {
	return nil
}
