package storage

import (
	"errors"
	"fmt"
	"github.com/andreikom/sensor-server/pkg/utils"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var temperatureStorePath string

func (d Driver) Init() {
	storePath := utils.GetUserHome()
	temperatureStorePath = filepath.Join(storePath, "temperatures")
	if _, err := os.Stat(temperatureStorePath); err == nil {
		fmt.Printf("Temperatures store folder already exists: %s\n", temperatureStorePath)
	} else if errors.Is(err, os.ErrNotExist) {
		fmt.Printf("Creating temperatures folder at: %s\n", temperatureStorePath)
		if err := os.Mkdir(temperatureStorePath, 0755); err != nil {
			errorMsg := fmt.Sprintf("Could not have created a folder at: %s\n", err)
			panic(errorMsg)
		}
		fmt.Printf("Created temperatures folder\n")
	}
}

func (d Driver) SaveSensorData(sensorId string, data []byte) error {
	sensorJsonFilePath := filepath.Join(temperatureStorePath, sensorId+".json")
	err := ioutil.WriteFile(sensorJsonFilePath, data, 0644)
	if err != nil {
		fmt.Println("Could not have created a file at: " + sensorJsonFilePath)
		return err
	}
	return nil
}

func (d Driver) GetAvailableSensors() ([]string, error) {
	sensors := make([]string, 0)
	err := filepath.WalkDir(temperatureStorePath+"/", visitBySensorId(&sensors))
	if err != nil {
		fmt.Printf("Error while attempting to list sensors: %s", err)
		return nil, err
	}
	return sensors, nil
}

func visitBySensorId(sensors *[]string) fs.WalkDirFunc {
	return func(path string, entry fs.DirEntry, err error) error {
		if !entry.IsDir() {
			splitPath := strings.Split(path, ".")
			sensorId := strings.Split(splitPath[0], "/")
			*sensors = append(*sensors, sensorId[len(sensorId)-1])
		}
		if err != nil {
			return err
		}
		return nil
	}
}

func (d Driver) GetSensorData(sensorId string) ([]byte, error) {
	sensorFolderPath := filepath.Join(temperatureStorePath)
	folder, err := os.Open(sensorFolderPath) // TODO [andreik]: add existence check first
	if err != nil {
		fmt.Printf("Error while attempting to open temp folder at: %s. Error: %s\n", sensorFolderPath, err)
		return nil, err
	} else if folder != nil { // TODO [andreik]: probably dont need this
		sensorPath := filepath.Join(temperatureStorePath, sensorId)
		sensorPath = sensorPath + ".json"
		sensorFile, err := ioutil.ReadFile(sensorPath)
		if err != nil {
			fmt.Printf("Error while attempting to read a sensor record file at: %s. Error: %s\n", sensorPath, err)
			return nil, err
		}
		return sensorFile, nil
	}
	return nil, nil // the case that the file does not exist and neither an error happened
}
