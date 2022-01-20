package temperature

import (
	"context"
	"strconv"
)

type tempServiceGrpc struct {
}

var serviceGrpc *tempServiceGrpc

func (t tempServiceGrpc) GetDailyMaxTempByDateAndByID(ctx context.Context, date *SensorIdDate) (*Message, error) {
	msg := strconv.FormatInt(int64(GetTempService().GetDailyMaxTempByDateAndByID(date.SensorId, date.Date)), 10)
	return &Message{Value: msg}, nil
}

func (t tempServiceGrpc) mustEmbedUnimplementedTempServiceServer() {
}

func GetTempServiceGrpc() *tempServiceGrpc {
	once.Do(func() {
		serviceGrpc = &tempServiceGrpc{}
	})
	return serviceGrpc
}
