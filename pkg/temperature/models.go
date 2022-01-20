package temperature

type Sensor struct {
	Id    string            `json:"id"`
	Dates map[string][]Hour `json:"dates"`
}

type Hour struct {
	Value int   `json:"hour"`
	Temp  []int `json:"temp"`
}

type TempQueueMsg struct {
	SensorId string `json:"sensorId"`
	Date     string `json:"date"`
	Hour     int    `json:"hour"`
	Temp     int    `json:"temp"`
}
