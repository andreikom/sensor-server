package api

import (
	"fmt"
	"github.com/andreikom/sensor-server/pkg/storage"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	readTimeout     = 5 * time.Second
	writeWorkersNum = 10
	readWorkersNum  = 10
)

var writeWorkers chan struct{}
var readWorkers chan struct{}

func init() {
	storage.InitStorage(storage.Filesystem)
	writeWorkers = make(chan struct{}, writeWorkersNum) // TODO [andreik]: consider to replace with WeightedSemaphore
	readWorkers = make(chan struct{}, readWorkersNum)   // TODO [andreik]: https://medium.com/@deckarep/gos-extended-concurrency-semaphores-part-1-5eeabfa351ce --> option 2
}

type Config struct {
	Port int
}

func Start(cfg *Config) {
	http.HandleFunc("/temp/", receiveData)
	http.ListenAndServe(":"+strconv.Itoa(cfg.Port), nil)
}

func receiveData(w http.ResponseWriter, req *http.Request) {
	endpoint := "POST /temp"
	ctx := req.Context() // TODO [andreik]: see how to use context to improve this
	if req.Method == "POST" {
		select {
		case writeWorkers <- struct{}{}:
			{
				defer func() { <-writeWorkers }()
				//if body, err := ioutil.ReadAll(req.Body); err != nil {
				//	fmt.Printf("Error while reading body on %s: %s\n", endpoint, err)
				//	http.Error(w, "Error while reading body", http.StatusInternalServerError)
				data := strings.Split(strings.TrimPrefix(req.URL.Path, "/temp/"), "/")
				if err := storage.Driver.SaveTemperature(data[0], data[1]); err != nil {
					combinedErr := fmt.Sprintf("Could not have saved to storage: %d", err)
					fmt.Println(combinedErr)
					http.Error(w, combinedErr, http.StatusInternalServerError)
				} else {
					w.WriteHeader(http.StatusOK)
				}
			}

		case <-time.After(readTimeout):
			{
				fmt.Println("Request timeout: "+endpoint, http.StatusRequestTimeout)
				http.Error(w, "Request timeout", http.StatusRequestTimeout)
			}

		case <-ctx.Done():
			{
				err := ctx.Err()
				fmt.Println("server:", err)
				http.Error(w, "Context cancelled request", http.StatusInternalServerError)
			}
			return
		}
	} else {
		errMethodNotImplemented(w, endpoint)
	}
}

func errMethodNotImplemented(w http.ResponseWriter, endpoint string) {
	w.WriteHeader(http.StatusInternalServerError)
	if _, err := w.Write([]byte(http.StatusText(http.StatusNotImplemented))); err != nil {
		fmt.Printf("Error while writing response to %s API\n", endpoint)
	}
}
