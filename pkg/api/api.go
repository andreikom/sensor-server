package api

import (
	"fmt"
	"github.com/andreikom/sensor-server/pkg/storage"
	"net/http"
	"strconv"
	"time"
)

const (
	defaultPort    = 8080
	readTimeout    = 5 * time.Second
	maxConnections = 10
	daysToKeep     = 7
)

var connProcessing chan struct{} // TODO [andreik]: consider to replace with WeightedSemaphore https://medium.com/@deckarep/gos-extended-concurrency-semaphores-part-1-5eeabfa351ce --> option 2

// TODO [andreik]: config.go -> add the validations here (go validator)
type Config struct {
	Server struct {
		Port int `yaml:"port"`
	}
}

func Start(cfg *Config) {
	validateConfig(cfg)
	storageDriver := storage.InitStorage(storage.Filesystem)
	//newCoolStorage := NewCoolStorage() // TODO [andreik]: change the storage init here so that it'll seamless
	GetTempService().Init(storageDriver) // TODO [andreik]: rethink it so that it matches the SOLID principles
	connProcessing = make(chan struct{}, maxConnections)
	http.Handle("/temp/", throttleIfNeeded(receiveData))
	fmt.Printf("Starting Sensor Server, port: %d\n", cfg.Server.Port)
	http.ListenAndServe(":"+strconv.Itoa(cfg.Server.Port), nil)
	// TODO [andreik]: graceful shutdown (need to wrap the api server)
}

func validateConfig(cfg *Config) {
	if cfg.Server.Port <= 1024 {
		cfg.Server.Port = defaultPort
	}
}

func throttleIfNeeded(h http.HandlerFunc) http.Handler {
	//ctx := req.Context() // TODO [andreik]: see how to use context to improve this
	//select {
	//case connProcessing <- struct{}{}:
	{
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			connProcessing <- struct{}{}
			defer func() { <-connProcessing }()
			h.ServeHTTP(w, r)
		})
	}
	//
	//case <-time.After(readTimeout):
	//	{
	//		fmt.Println("Request timeout: "+endpoint, http.StatusRequestTimeout)
	//		http.Error(w, "Request timeout", http.StatusRequestTimeout)
	//	}
	//
	//case <-ctx.Done():
	//	{
	//		err := ctx.Err()
	//		fmt.Println("server:", err)
	//		http.Error(w, "Context cancelled request", http.StatusInternalServerError)
	//	}
	//}
}
