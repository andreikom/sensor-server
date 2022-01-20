package temperature

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func ReceiveData(w http.ResponseWriter, req *http.Request) {
	endpoint := "POST /temp"
	if req.Method == "POST" {
		data := strings.Split(strings.TrimPrefix(req.URL.Path, "/temp/"), "/")
		convertedTemp, err := strconv.Atoi(data[1])
		if err != nil {
			combinedErr := fmt.Sprintf("Could not have converted temp to int: %d\n", err)
			fmt.Println(combinedErr)
		}
		if err := GetTempService().SaveTemperature(data[0], convertedTemp); err != nil {
			combinedErr := fmt.Sprintf("Could not have saved to storage: %d\n", err)
			fmt.Println(combinedErr)
			http.Error(w, combinedErr, http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
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
