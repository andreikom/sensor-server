package api

import (
	"fmt"
	"github.com/andreikom/sensor-server/pkg/storage"
	"github.com/andreikom/sensor-server/pkg/temperature"
	"github.com/streadway/amqp"
	"google.golang.org/grpc"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
)

// TODO [andreik]: consider to replace with WeightedSemaphore https://medium.com/@deckarep/gos-extended-concurrency-semaphores-part-1-5eeabfa351ce --> option 2
var connProcessing chan struct{}

func Start(cfg *Config) {
	err := validate(cfg)
	if err != nil {
		fmt.Printf("Falling back to default value, error: %s\n", err)
	}
	storageDriver := storage.InitStorage(storage.Filesystem) // TODO [andreik]: rethink it so that it matches the SOLID principles
	//newCoolStorage := NewCoolStorage() // TODO [andreik]: change the storage init here so that it'll seamless
	mqConn, mqChan := connectToRabbitMq()
	defer mqConn.Close()
	defer mqChan.Close()
	temperature.GetTempService().Init(storageDriver, mqChan)
	connProcessing = make(chan struct{}, maxConnections)
	wg := new(sync.WaitGroup)
	wg.Add(2)
	go startHttpServer(cfg, wg)
	go startGrpcServer(err, wg)
	wg.Wait()
	// TODO [andreik]: graceful shutdown (need to wrap the api server)
}

func connectToRabbitMq() (*amqp.Connection, *amqp.Channel) {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	rmqError(err, "Failed to connect to RabbitMQ")
	ch, err := conn.Channel()
	rmqError(err, "Failed to open a channel")
	return conn, ch
}

func rmqError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func startHttpServer(cfg *Config, wg *sync.WaitGroup) {
	http.Handle("/temp/", throttleIfNeeded(temperature.ReceiveData))
	fmt.Printf("Starting Sensor Server, port: %d\n", cfg.Server.Port)
	http.ListenAndServe(":"+strconv.Itoa(cfg.Server.Port), nil)
	wg.Done()
}

func startGrpcServer(err error, wg *sync.WaitGroup) {
	grpcListener, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("failed to listen for grpc server: %v", err)
	}
	grpcServer := grpc.NewServer()
	tempServiceGrpc := temperature.GetTempServiceGrpc()
	temperature.RegisterTempServiceServer(grpcServer, tempServiceGrpc)
	if err := grpcServer.Serve(grpcListener); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
	wg.Done()
}

func throttleIfNeeded(h http.HandlerFunc) http.Handler {
	{
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			connProcessing <- struct{}{}
			defer func() { <-connProcessing }()
			h.ServeHTTP(w, r)
		})
	}
}
