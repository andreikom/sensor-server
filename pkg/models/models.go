package models

type Sensor struct {
	Id    string            `json:"id"`
	Dates map[string][]Hour `json:"dates"`
}

type Hour struct {
	Value int   `json:"hour"`
	Temp  []int `json:"temp"`
}
