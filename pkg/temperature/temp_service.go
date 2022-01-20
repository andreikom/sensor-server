package temperature

import (
	"encoding/json"
	"fmt"
	"github.com/andreikom/sensor-server/pkg/storage"
	"github.com/streadway/amqp"
	"sync"
	"time"
)

const RcvTempQueue = "RcvTempQueue"

type tempService struct{}

var service *tempService

var once sync.Once
var mqChan *amqp.Channel
var writeLock sync.Mutex // TODO [andreik]: we can improve to lock per sensor probably inside a map/struct

type TempService interface {
	Init(driver *storage.Driver, mqChannel *amqp.Channel)
	SaveTemperature(sensorId string, data int) error
	GetDailyMaxTempByDateAndByID(sensorId string, date string) int
}

var storageDriver storage.Driver

// sensorId -> date -> hour -> temp
var weeklySensorCache = make(map[string]Sensor, 0)

func (t tempService) Init(driver *storage.Driver, mqChannel *amqp.Channel) {
	storageDriver = *driver
	mqChan = mqChannel
	t.setupQueues()
	t.initSensorCache()
	t.scheduleOldEntriesCleanUp()
	go t.consumeTempFromQueue()
}

func (t tempService) setupQueues() {
	_, err := mqChan.QueueDeclare(
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

func (t tempService) initSensorCache() {
	sensors, err := storageDriver.GetAvailableSensors()
	if err != nil {
		panic("Could not have initialized sensor temperature cache")
	}
	for _, sensor := range sensors {
		sensorEntry, err := storageDriver.GetSensorData(sensor)
		res := &Sensor{}
		err = json.Unmarshal(sensorEntry, &res)
		if err != nil {
			fmt.Printf("Error while unmarshalling sensor %s record , Error: %s\n", sensor, err)
			continue
		}
		weeklySensorCache[sensor] = *res
	}
}

func (t tempService) addDateEntryToCache(sensorEntry Sensor, dateToday string, currentHour int, data int) {
	newData := make([]int, 0)
	newData = append(newData, data)
	sensorEntry.Dates[dateToday] = append(sensorEntry.Dates[dateToday], Hour{
		Value: currentHour,
		Temp:  newData,
	})
}

func (t tempService) addSensorEntryToCache(sensorId string, data int, dateToday string, currentHour int) Sensor {
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
	weeklySensorCache[sensorId] = *newSensorEntry
	return *newSensorEntry
}

func (t tempService) addHourEntryToCache(hoursInDate []Hour, currentHour int, data int) {
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

func (t tempService) consumeTempFromQueue() {
	messages, err := mqChan.Consume(
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
			newMsg := &TempQueueMsg{}
			err = json.Unmarshal(msg.Body, &newMsg)
			if err != nil {
				errorMsg := fmt.Sprintf("Could not unmarshall a message from queue, Error: %s\n", err)
				fmt.Println(errorMsg)
				continue
			}
			writeLock.Lock()
			fmt.Println("Saving new msg to cache and disk")
			t.saveEntryToCacheAndStore(newMsg)
			writeLock.Unlock()
		}
	}()
	fmt.Println(" [*] - Waiting for messages")
	<-forever
}

func (t tempService) saveEntryToCacheAndStore(msg *TempQueueMsg) {
	sensorEntry, ok := weeklySensorCache[msg.SensorId]
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

func (t tempService) saveToDisk(sensorId string, sensorEntry Sensor) {
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
					t.cleanOldEntries(key, entry, lastDateToKeep)
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (t tempService) cleanOldEntries(sensorId string, sensorEntry Sensor, lastDateToKeep time.Time) {
	for date := range sensorEntry.Dates {
		parsedDate, err := time.Parse("01-02-2006", date)
		if err != nil {
			fmt.Printf("Could not have parsed previous date: %s\n", date)
			continue
		}
		if parsedDate.Before(lastDateToKeep) {
			writeLock.Lock()
			delete(sensorEntry.Dates, date)
			// TODO [andreik]: improve to batch date delete (instead of each date one by one)
			t.saveToDisk(sensorId, sensorEntry)
			writeLock.Unlock()
			fmt.Printf("Cleaned old record for date: %s, sensor Id: %s\n", date, sensorId)
		}
	}
	fmt.Printf("Old records cleaned for sensor Id: %s\n", sensorId)
}

func (t tempService) SaveTemperature(sensorId string, data int) error {
	now := time.Now()
	msg := &TempQueueMsg{sensorId, now.Format("01-02-2006"), now.Hour(), data}
	serializedMsg, err := json.Marshal(msg)
	if err != nil {
		fmt.Printf("Could not have serialized sensor %s data: %s\n", sensorId, err)
	}
	err = publishToQueue(serializedMsg)
	if err != nil {
		return err
	}
	return nil
}

func publishToQueue(serializedMsg []byte) error {
	// RMQClient protects publish with an inner lock
	err := mqChan.Publish(
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

func (t tempService) GetDailyMaxTempByDateAndByID(sensorId string, date string) int {
	return 20
}

func GetTempService() *tempService {
	once.Do(func() {
		service = &tempService{}
	})
	return service
}
