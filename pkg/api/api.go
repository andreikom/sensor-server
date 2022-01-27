package api

import (
	"fmt"
	"github.com/andreikom/sensor-server/pkg/api/temperature"
	"github.com/andreikom/sensor-server/pkg/storage"
	"github.com/andreikom/sensor-server/pkg/utils"
	"github.com/gorilla/mux"
	"github.com/streadway/amqp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	_ "net/http/pprof"
	"strconv"
	"sync"
)

var connProcessing chan struct{}

func Start(cfg *Config) {
	err := validate(cfg)
	if err != nil {
		fmt.Printf("Falling back to default value, error: %s\n", err)
	}
	storageDriver := storage.NewFSDriver(utils.GetUserHome())
	mqConn, mqChan := connectToRabbitMq()
	defer mqConn.Close()
	defer mqChan.Close()
	tempService := temperature.NewTempService(storageDriver, mqChan)
	connProcessing = make(chan struct{}, maxConnections)
	wg := new(sync.WaitGroup)
	wg.Add(3)
	tempController := &tempController{tempService: tempService}
	go startHttpServer(tempController, cfg, wg)
	go startGrpcServer(tempService, err, wg)
	go startPprofDebugServer(wg)
	wg.Wait()
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

func startHttpServer(tempController *tempController, cfg *Config, wg *sync.WaitGroup) {
	router := mux.NewRouter()
	router.Handle("/temp/", throttleIfNeeded(tempController.SaveTemp)).Methods("POST")
	router.Handle("/temp/daily_max/{sensorId}/{date}", throttleIfNeeded(tempController.GetDailyMaxTemp)).Methods("GET")
	router.Handle("/temp/weekly_max/{sensorId}/{date}", throttleIfNeeded(tempController.GetWeeklyMaxTemp)).Methods("GET")
	router.Handle("/temp/daily_min/{sensorId}/{date}", throttleIfNeeded(tempController.GetDailyMinTemp)).Methods("GET")
	router.Handle("/temp/weekly_min/{sensorId}/{date}", throttleIfNeeded(tempController.GetWeeklyMinTemp)).Methods("GET")
	router.Handle("/temp/daily_avg/{sensorId}/{date}", throttleIfNeeded(tempController.GetDailyAvgTemp)).Methods("GET")
	router.Handle("/temp/weekly_avg/{sensorId}", throttleIfNeeded(tempController.GetWeeklySensorAvgTemp)).Methods("GET")
	fmt.Printf("Starting Sensor Server, port: %d\n", cfg.Server.Port)
	http.ListenAndServe("localhost:"+strconv.Itoa(cfg.Server.Port), router)
	wg.Done()
}

func startPprofDebugServer(wg *sync.WaitGroup) {
	debugRouter := mux.NewRouter()
	AttachProfiler(debugRouter)
	http.ListenAndServe("localhost:6060", debugRouter)
	wg.Done()
}

func AttachProfiler(router *mux.Router) {
	// PathPrefix used since Index func contains additional profile method not specified explicitly
	router.PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index)
	router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
}

func startGrpcServer(tempService *temperature.TempService, err error, wg *sync.WaitGroup) {
	grpcListener, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("failed to listen for grpc server: %v", err)
	}
	grpcServer := grpc.NewServer()
	tempServiceGrpc := &temperature.TempServiceGrpc{TempService: tempService}
	temperature.RegisterTempServiceServer(grpcServer, tempServiceGrpc)
	reflection.Register(grpcServer) // only for "dump" clients (grpcurl)
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
