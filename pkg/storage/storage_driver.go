package storage

type Driver interface {
	SaveSensorData(sensorId string, data []byte) error
	GetAvailableSensors() ([]string, error)
	GetSensorData(sensorId string) ([]byte, error)
}
