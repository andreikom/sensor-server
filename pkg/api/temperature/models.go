package temperature

type Sensor struct {
	Id    string            `json:"id"`
	Dates map[string][]Hour `json:"dates"`
}

type Hour struct {
	Value int   `json:"hour"`
	Temp  []int `json:"temp"`
}

// TempQueueMsg - Represents the RabbitMq model
type TempQueueMsg struct {
	SensorId string `json:"sensorId"`
	Date     string `json:"date"`
	Hour     int    `json:"hour"`
	Temp     int    `json:"temp"`
}

type SensorIdTempJson struct {
	SensorId string `json:"sensorId"`
	Temp     int    `json:"temp"`
}

type NotFoundError struct {
	Name string
}

func (e *NotFoundError) Error() string {
	return e.Name + ": not found"
}
