package temperature

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/andreikom/sensor-server/pkg/storage"
	"github.com/streadway/amqp"
	"sync"
	"time"
)

const RcvTempQueue = "RcvTempQueue"

type TempService struct {
	storageDriver storage.Driver
	// sensorId -> date -> hour -> temp
	weeklySensorCache map[string]Sensor
	mqChan            *amqp.Channel
	rwMutex           sync.Mutex // TODO [andreik]: we can improve to lock per sensor probably inside a map/struct
}

func NewTempService(driver storage.Driver, mqChannel *amqp.Channel) *TempService {
	cache := make(map[string]Sensor, 0)
	service := &TempService{storageDriver: driver, weeklySensorCache: cache, mqChan: mqChannel}
	service.setupQueues()
	service.initSensorCache()
	service.scheduleOldEntriesCleanUp()
	go service.consumeTempFromQueue()
	return service
}

func (t *TempService) setupQueues() {
	_, err := t.mqChan.QueueDeclare(
		RcvTempQueue, // name
		false,        // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		errorMsg := fmt.Sprintf("Could not have created the '%s' queue: %s\n", RcvTempQueue, err)
		panic(errorMsg)
	}
}

func (t *TempService) initSensorCache() {
	sensors, err := t.storageDriver.GetAvailableSensors()
	if err != nil {
		panic("Could not have initialized sensor temperature cache")
	}
	for _, sensor := range sensors {
		sensorEntry, err := t.storageDriver.GetSensorData(sensor)
		res := &Sensor{}
		err = json.Unmarshal(sensorEntry, &res)
		if err != nil {
			fmt.Printf("Error while unmarshalling sensor %s record , Error: %s\n", sensor, err)
			continue
		}
		t.weeklySensorCache[sensor] = *res
	}
}

func (t *TempService) addDateEntryToCache(sensorEntry Sensor, dateToday string, currentHour int, data int) {
	newData := make([]int, 0)
	newData = append(newData, data)
	sensorEntry.Dates[dateToday] = append(sensorEntry.Dates[dateToday], Hour{
		Value: currentHour,
		Temp:  newData,
	})
}

func (t *TempService) addSensorEntryToCache(sensorId string, data int, dateToday string, currentHour int) Sensor {
	newSensorEntry := &Sensor{
		Id:    sensorId,
		Dates: make(map[string][]Hour),
	}
	hour := &Hour{
		Value: currentHour,
		Temp:  make([]int, 0),
	}
	hour.Temp = append(hour.Temp, data)
	newSensorEntry.Dates[dateToday] = append(newSensorEntry.Dates[dateToday], *hour)
	t.weeklySensorCache[sensorId] = *newSensorEntry
	return *newSensorEntry
}

func (t *TempService) addHourEntryToCache(hoursInDate []Hour, currentHour int, data int) {
	for _, hour := range hoursInDate {
		if hour.Value == currentHour {
			hour.Temp = append(hour.Temp, data)
			return
		}
	}
	// match was not found, adding new hour
	newHour := make([]Hour, 0)
	tempData := make([]int, 0)
	tempData = append(tempData, data)
	newHour = append(newHour, Hour{
		Value: currentHour,
		Temp:  tempData,
	})
	hoursInDate = append(hoursInDate, newHour...)
}

func (t *TempService) consumeTempFromQueue() {
	messages, err := t.mqChan.Consume(
		RcvTempQueue,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	forever := make(chan bool)
	go func() {
		for msg := range messages {
			// TODO [andreik]: consider ack again
			newMsg := &TempQueueMsg{}
			err = json.Unmarshal(msg.Body, &newMsg)
			if err != nil {
				errorMsg := fmt.Sprintf("Could not unmarshall a message from queue, Error: %s\n", err)
				fmt.Println(errorMsg)
				continue
			}
			t.rwMutex.Lock()
			fmt.Println("Saving new msg to cache and disk")
			t.saveEntryToCacheAndStore(newMsg)
			t.rwMutex.Unlock()
		}
	}()
	fmt.Println(" [*] - Waiting for messages")
	<-forever
}

func (t *TempService) saveEntryToCacheAndStore(msg *TempQueueMsg) {
	sensorEntry, ok := t.weeklySensorCache[msg.SensorId]
	if ok {
		dateToday := msg.Date
		hoursInDate, ok := sensorEntry.Dates[dateToday]
		if ok {
			t.addHourEntryToCache(hoursInDate, msg.Hour, msg.Temp)
		} else {
			t.addDateEntryToCache(sensorEntry, dateToday, msg.Hour, msg.Temp)
		}
	} else {
		sensorEntry = t.addSensorEntryToCache(msg.SensorId, msg.Temp, msg.Date, msg.Hour)
	}
	t.saveToDisk(msg.SensorId, sensorEntry)
}

func (t *TempService) saveToDisk(sensorId string, sensorEntry Sensor) {
	serializedData, err := json.Marshal(sensorEntry)
	if err != nil {
		fmt.Printf("Could not have serialized sensor %s data: %s\n", sensorId, err)
	}
	err = t.storageDriver.SaveSensorData(sensorId, serializedData)
	if err != nil {
		fmt.Printf("Could not save data for sensor Id: %s data: %s\n", sensorId, err)
	}
}

func (t *TempService) scheduleOldEntriesCleanUp() {
	ticker := time.NewTicker(12 * time.Hour)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				today := time.Now()
				lastDateToKeep := today.AddDate(0, 0, -7)
				for key, entry := range t.weeklySensorCache {
					t.cleanOldEntries(key, entry, lastDateToKeep)
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (t *TempService) cleanOldEntries(sensorId string, sensorEntry Sensor, lastDateToKeep time.Time) {
	for date := range sensorEntry.Dates {
		parsedDate, err := time.Parse("01-02-2006", date)
		if err != nil {
			fmt.Printf("Could not have parsed previous date: %s\n", date)
			continue
		}
		if parsedDate.Before(lastDateToKeep) {
			t.rwMutex.Lock()
			delete(sensorEntry.Dates, date)
			// TODO [andreik]: improve to batch date delete (instead of each date one by one)
			t.saveToDisk(sensorId, sensorEntry)
			t.rwMutex.Unlock()
			fmt.Printf("Cleaned old record for date: %s, sensor Id: %s\n", date, sensorId)
		}
	}
	fmt.Printf("Old records cleaned for sensor Id: %s\n", sensorId)
}

func (t *TempService) publishToQueue(serializedMsg []byte) error {
	// RMQClient protects publish with an inner lock
	err := t.mqChan.Publish(
		"",
		RcvTempQueue,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        serializedMsg,
		},
	)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (t *TempService) SaveTemperature(sensorId string, data int) error {
	now := time.Now()
	msg := &TempQueueMsg{sensorId, now.Format("01-02-2006"), now.Hour(), data}
	serializedMsg, err := json.Marshal(msg)
	if err != nil {
		fmt.Printf("Could not have serialized sensor %s data: %s\n", sensorId, err)
	}
	err = t.publishToQueue(serializedMsg)
	if err != nil {
		return err
	}
	return nil
}

func (t *TempService) GetDailyMaxTempByDateAndById(sensorId string, date string) (int, error) {
	dateEntry, ok := t.weeklySensorCache[sensorId].Dates[date]
	if ok {
		max := dateEntry[0].Temp[0]
		return getDailyMaxTemp(dateEntry, max), nil
	}
	return 0, errors.New("Could not have find an entry for sensorId: '" + sensorId + "' and date: '" + date + "'")
}

func (t *TempService) GetDailyWeeklyMaxTempBySensorId(sensorId string) (int, error) {
	sensorEntry, ok := t.weeklySensorCache[sensorId]
	if ok {
		weeklyMax := 1000
		for _, dateEntry := range sensorEntry.Dates {
			dailyMax := getDailyMaxTemp(dateEntry, weeklyMax)
			if dailyMax < 1000 {
				weeklyMax = dailyMax
			}
		}
		return weeklyMax, nil
	}
	return 0, errors.New("Could not have find an entry for sensorId: '" + sensorId)
}

func getDailyMaxTemp(dateEntry []Hour, max int) int {
	for _, hour := range dateEntry {
		for _, temp := range hour.Temp {
			if temp > max {
				max = temp
			}
		}
	}
	return max
}

func (t *TempService) GetDailyMinTempByDateAndById(sensorId string, date string) (int, error) {
	dateEntry, ok := t.weeklySensorCache[sensorId].Dates[date]
	if ok {
		min := dateEntry[0].Temp[0]
		return getDailyMinTemp(dateEntry, min), nil
	}
	return 0, errors.New("Could not have find an entry for sensorId: '" + sensorId + "' and date: '" + date + "'")
}

func (t *TempService) GetDailyWeeklyMinTempBySensorId(sensorId string) (int, error) {
	sensorEntry, ok := t.weeklySensorCache[sensorId]
	if ok {
		weeklyMin := 1000
		for _, dateEntry := range sensorEntry.Dates {
			dailyMin := getDailyMinTemp(dateEntry, weeklyMin)
			if dailyMin < 1000 {
				weeklyMin = dailyMin
			}
		}
		return weeklyMin, nil
	}
	return 0, errors.New("Could not have find an entry for sensorId: '" + sensorId)
}

func getDailyMinTemp(dateEntry []Hour, min int) int {
	for _, hour := range dateEntry {
		for _, temp := range hour.Temp {
			if temp < min {
				min = temp
			}
		}
	}
	return min
}

func (t *TempService) GetDailyAvgTempByDateAndById(sensorId string, date string) (float64, error) {
	dateEntry, ok := t.weeklySensorCache[sensorId].Dates[date]
	if ok {
		return calculateDailyAvg(dateEntry), nil
	}
	return 0, errors.New("Could not have find an entry for sensorId: '" + sensorId + "' and date: '" + date + "'")
}

func (t *TempService) GetDailyWeeklyAvgTempBySensorId(sensorId string) (float64, error) {
	sensorEntry, ok := t.weeklySensorCache[sensorId]
	if ok {
		weeklySum := 0.0
		for _, dateEntry := range sensorEntry.Dates {
			weeklySum = weeklySum + calculateDailyAvg(dateEntry)
		}
		return weeklySum / float64(len(sensorEntry.Dates)), nil
	}
	return 0, errors.New("Could not have find an entry for sensorId: '" + sensorId)
}

func calculateDailyAvg(dateEntry []Hour) float64 {
	entriesNum := 0.0
	tempSum := 0.0
	for _, hour := range dateEntry {
		for _, temp := range hour.Temp {
			entriesNum++
			tempSum += float64(temp)
		}
	}
	return tempSum / entriesNum
}
