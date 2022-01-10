package api

import (
	"encoding/json"
	"fmt"
	"github.com/andreikom/sensor-server/pkg/models"
	"github.com/andreikom/sensor-server/pkg/storage"
	"sync"
	"time"
)

var once sync.Once

type tempService struct{}

var service *tempService

type TempService interface {
	Init(driver *storage.Driver)
	SaveTemperature(sensorId string, data int) error
}

var storageDriver storage.Driver

// sensorId -> date -> hour -> temp
var weeklySensorCache = make(map[string]models.Sensor, 0)

func (t tempService) Init(driver *storage.Driver) {
	storageDriver = *driver
	t.initSensorCache()
}

func (t tempService) initSensorCache() {
	sensors, err := storageDriver.GetAvailableSensors()
	if err != nil {
		panic("Could not have initialized sensor temperature cache")
	}
	for _, sensor := range sensors {
		sensorEntry, err := storageDriver.GetSensorData(sensor)
		if err != nil {
			fmt.Printf("Could not load temp for sensor %s: %s\n", sensor, err)
			continue
		}
		weeklySensorCache[sensor] = *sensorEntry
	}
	t.scheduleCacheSave()
	t.scheduleOldEntriesCleanUp()
}

func (t tempService) addDateEntryToCache(sensorEntry models.Sensor, dateToday string, now time.Time, data int) {
	newData := make([]int, 0)
	newData = append(newData, data)
	sensorEntry.Dates[dateToday] = append(sensorEntry.Dates[dateToday], models.Hour{
		Value: now.Hour(),
		Temp:  newData,
	})
}

func (t tempService) addSensorEntryToCache(sensorId string, data int, now time.Time) {
	newSensorEntry := &models.Sensor{
		Id:    sensorId,
		Dates: make(map[string][]models.Hour),
	}
	hour := &models.Hour{
		Value: now.Hour(),
		Temp:  make([]int, 0),
	}
	hour.Temp = append(hour.Temp, data)
	newSensorEntry.Dates[now.Format("01-02-2006")] = append(newSensorEntry.Dates[now.Format("01-02-2006")], *hour)
	weeklySensorCache[sensorId] = *newSensorEntry
}

func (t tempService) addHourEntryToCache(hoursInDate []models.Hour, now time.Time, data int) {
	currentHour := now.Hour()
	for _, hour := range hoursInDate {
		if hour.Value == currentHour {
			hour.Temp = append(hour.Temp, data)
			return
		}
	}
	// match was not found, adding new hour
	newHour := make([]models.Hour, 0)
	tempData := make([]int, 0)
	tempData = append(tempData, data)
	newHour = append(newHour, models.Hour{
		Value: currentHour,
		Temp:  tempData,
	})
	hoursInDate = append(hoursInDate, newHour...)
}

func (t tempService) scheduleCacheSave() {
	ticker := time.NewTicker(10 * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				for key, entry := range weeklySensorCache {
					t.saveEntryToStore(key, entry)
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (t tempService) saveNewEntryToStore(sensorId string, data int) error {
	now := time.Now()
	sensorData := &models.Sensor{
		Id:    sensorId,
		Dates: make(map[string][]models.Hour),
	}
	hour := &models.Hour{
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

func (t tempService) saveEntryToStore(sensorId string, sensorEntry models.Sensor) {
	serializedData, err := json.Marshal(sensorEntry)
	if err != nil {
		fmt.Printf("Could not have serialized sensor %s data: %s\n", sensorId, err)
	}
	err = storageDriver.SaveSensorData(sensorId, serializedData)
	if err != nil {
		fmt.Printf("Could not save data for sensor Id: %s data: %s\n", sensorId, err)
	}
}

func (t tempService) scheduleOldEntriesCleanUp() {
	ticker := time.NewTicker(12 * time.Hour)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				today := time.Now()
				lastDateToKeep := today.AddDate(0, 0, -7)
				for key, entry := range weeklySensorCache {
					t.cleanOldDailyEntry(key, entry, lastDateToKeep)
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (t tempService) cleanOldDailyEntry(sensorId string, sensorEntry models.Sensor, lastDateToKeep time.Time) {
	for date := range sensorEntry.Dates {
		parsedDate, err := time.Parse("01-02-2006", date)
		if err != nil {
			fmt.Printf("Could not have parsed previous date: %s\n", date)
			continue
		}
		if parsedDate.Before(lastDateToKeep) {
			delete(sensorEntry.Dates, date)
			fmt.Printf("Cleaned old record for date: %s, sensor Id: %s\n", date, sensorId)
		}
	}
	fmt.Printf("Old records cleaned for sensor Id: %s\n", sensorId)
}

func (t tempService) SaveTemperature(sensorId string, data int) error {
	now := time.Now()
	sensorEntry, ok := weeklySensorCache[sensorId]
	if ok {
		dateToday := now.Format("01-02-2006")
		hoursInDate, ok := sensorEntry.Dates[dateToday]
		if ok {
			t.addHourEntryToCache(hoursInDate, now, data)
		} else {
			t.addDateEntryToCache(sensorEntry, dateToday, now, data)
		}
	} else {
		t.addSensorEntryToCache(sensorId, data, now)
	}
	return nil
}

func GetTempService() *tempService {
	once.Do(func() {
		service = &tempService{}
	})
	return service
}
