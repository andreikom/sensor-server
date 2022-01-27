package api

import (
	"encoding/json"
	"fmt"
	"github.com/andreikom/sensor-server/pkg/api/temperature"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

// TODO [andreik]: later we will change this to grpc client +/or rabbitmq client
type tempController struct {
	tempService *temperature.TempService
}

func (c *tempController) SaveTemp(w http.ResponseWriter, req *http.Request) {
	sensorIdTemp := &temperature.SensorIdTempJson{}
	err := json.NewDecoder(req.Body).Decode(&sensorIdTemp)
	if err != nil {
		combinedErr := fmt.Sprintf("Could not have parse the payload: %d\n", err)
		fmt.Println(combinedErr)
		http.Error(w, combinedErr, http.StatusBadRequest)
		return
	}
	if err := c.tempService.SaveTemperature(sensorIdTemp.SensorId, sensorIdTemp.Temp); err != nil {
		combinedErr := fmt.Sprintf("Could not have saved to storage: %d\n", err)
		fmt.Println(combinedErr)
		http.Error(w, combinedErr, http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func (c *tempController) GetDailyMaxTemp(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sensorId := vars["sensorId"]
	date := vars["date"]
	maxTemp, err := c.tempService.GetDailyMaxTempByDateAndById(sensorId, date)
	if err == nil {
		if _, err := w.Write([]byte(strconv.Itoa(maxTemp))); err != nil {
			fmt.Printf("Error while writing response to daily_max endpoint: %s", err)
		}
	} else {
		http.Error(w, err.Error(), http.StatusNotImplemented)
	}
}

func (c *tempController) GetWeeklyMaxTemp(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sensorId := vars["sensorId"]
	maxTemp, err := c.tempService.GetDailyWeeklyMaxTempBySensorId(sensorId)
	if err == nil {
		if _, err := w.Write([]byte(strconv.Itoa(maxTemp))); err != nil {
			fmt.Printf("Error while writing response to daily_max endpoint: %s", err)
		}
	} else {
		http.Error(w, err.Error(), http.StatusNotImplemented)
	}
}

func (c *tempController) GetDailyMinTemp(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sensorId := vars["sensorId"]
	date := vars["date"]
	minTemp, err := c.tempService.GetDailyMinTempByDateAndById(sensorId, date)
	if err == nil {
		if _, err := w.Write([]byte(strconv.Itoa(minTemp))); err != nil {
			fmt.Printf("Error while writing response to daily_min endpoint: %s", err)
		}
	} else {
		http.Error(w, err.Error(), http.StatusNotImplemented)
	}
}

func (c *tempController) GetWeeklyMinTemp(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sensorId := vars["sensorId"]
	minTemp, err := c.tempService.GetDailyWeeklyMaxTempBySensorId(sensorId)
	if err == nil {
		if _, err := w.Write([]byte(strconv.Itoa(minTemp))); err != nil {
			fmt.Printf("Error while writing response to daily_max endpoint: %s", err)
		}
	} else {
		http.Error(w, err.Error(), http.StatusNotImplemented)
	}
}

func (c *tempController) GetDailyAvgTemp(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sensorId := vars["sensorId"]
	date := vars["date"]
	maxTemp, err := c.tempService.GetDailyAvgTempByDateAndById(sensorId, date)
	if err == nil {
		if _, err := w.Write([]byte(fmt.Sprintf("%f", maxTemp))); err != nil {
			fmt.Printf("Error while writing response to daily_min endpoint: %s", err)
		}
	} else {
		http.Error(w, err.Error(), http.StatusNotImplemented)
	}
}

func (c *tempController) GetWeeklySensorAvgTemp(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sensorId := vars["sensorId"]
	maxTemp, err := c.tempService.GetDailyWeeklyAvgTempBySensorId(sensorId)
	if err == nil {
		if _, err := w.Write([]byte(fmt.Sprintf("%f", maxTemp))); err != nil {
			fmt.Printf("Error while writing response to daily_min endpoint: %s", err)
		}
	} else {
		http.Error(w, err.Error(), http.StatusNotImplemented)
	}
}

func errMethodNotImplemented(w http.ResponseWriter, endpoint string) {
	w.WriteHeader(http.StatusInternalServerError)
	if _, err := w.Write([]byte(http.StatusText(http.StatusNotImplemented))); err != nil {
		fmt.Printf("Error while writing response to %s API\n", endpoint)
	}
}
