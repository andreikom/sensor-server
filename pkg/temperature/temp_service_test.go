package temperature

import (
	"encoding/json"
	"fmt"
	"time"
)

// Utility method for saving an entry directly to disk
func (t tempService) saveNewEntryToStore(sensorId string, data int) error {
	now := time.Now()
	sensorData := &Sensor{
		Id:    sensorId,
		Dates: make(map[string][]Hour),
	}
	hour := &Hour{
		Value: now.Hour(),
		Temp:  make([]int, 0),
	}
	hour.Temp = append(hour.Temp, data)
	sensorData.Dates[now.Format("01-02-2006")] = append(sensorData.Dates[now.Format("01-02-2006")], *hour)
	serializedData, err := json.Marshal(sensorData)
	if err != nil {
		fmt.Printf("Could not have serialized sensor %s data: %s\n", sensorId, err)
		return err
	}
	err = storageDriver.SaveSensorData(sensorId, serializedData)
	if err != nil {
		fmt.Printf("Could not save data for sensor Id: %s data: %s\n", sensorId, err)
		return err
	}
	return nil
}
