package temperature

import (
	"context"
	"strconv"
)

type TempServiceGrpc struct {
	TempService *TempService
}

func (t *TempServiceGrpc) SaveTemp(ctx context.Context, sensorIdTemp *SensorIdTemp) (*Empty, error) {
	err := t.TempService.SaveTemperature(sensorIdTemp.SensorId, int(sensorIdTemp.Temp))
	if err != nil {
		return nil, nil // TODO [andreik]: remove error?
	}
	return &Empty{}, nil
}

func (t *TempServiceGrpc) GetDailyMaxTempByDateAndById(ctx context.Context, sensorIdDate *SensorIdDate) (*Result, error) {
	res, err := t.TempService.GetDailyMaxTempByDateAndById(sensorIdDate.SensorId, sensorIdDate.Date)
	if err != nil {
		return nil, nil // TODO [andreik]: remove error?
	}
	msg := strconv.FormatInt(int64(res), 10)
	return &Result{Value: msg}, nil
}

func (t *TempServiceGrpc) GetWeeklyMaxTempById(ctx context.Context, sensorIdDate *SensorIdDate) (*Result, error) {
	res, err := t.TempService.GetDailyWeeklyMaxTempBySensorId(sensorIdDate.SensorId)
	if err != nil {
		return nil, nil // TODO [andreik]: remove error?
	}
	msg := strconv.FormatInt(int64(res), 10)
	return &Result{Value: msg}, nil
}

func (t *TempServiceGrpc) GetDailyMinTempByDateAndById(ctx context.Context, sensorIdDate *SensorIdDate) (*Result, error) {
	res, err := t.TempService.GetDailyMinTempByDateAndById(sensorIdDate.SensorId, sensorIdDate.Date)
	if err != nil {
		return nil, nil // TODO [andreik]: remove error?
	}
	msg := strconv.FormatInt(int64(res), 10)
	return &Result{Value: msg}, nil
}

func (t *TempServiceGrpc) GetWeeklyMinTempById(ctx context.Context, sensorIdDate *SensorIdDate) (*Result, error) {
	res, err := t.TempService.GetDailyWeeklyMinTempBySensorId(sensorIdDate.SensorId)
	if err != nil {
		return nil, nil // TODO [andreik]: remove error?
	}
	msg := strconv.FormatInt(int64(res), 10)
	return &Result{Value: msg}, nil
}

func (t *TempServiceGrpc) GetDailyAvgTempByDateAndById(ctx context.Context, sensorIdDate *SensorIdDate) (*Result, error) {
	res, err := t.TempService.GetDailyAvgTempByDateAndById(sensorIdDate.SensorId, sensorIdDate.Date)
	if err != nil {
		return nil, nil // TODO [andreik]: remove error?
	}
	msg := strconv.FormatInt(int64(res), 10)
	return &Result{Value: msg}, nil
}

func (t *TempServiceGrpc) GetWeeklyAvgTempById(ctx context.Context, sensorIdDate *SensorIdDate) (*Result, error) {
	res, err := t.TempService.GetDailyWeeklyAvgTempBySensorId(sensorIdDate.SensorId)
	if err != nil {
		return nil, nil // TODO [andreik]: remove error?
	}
	msg := strconv.FormatInt(int64(res), 10)
	return &Result{Value: msg}, nil
}

func (t *TempServiceGrpc) mustEmbedUnimplementedTempServiceServer() {
}
