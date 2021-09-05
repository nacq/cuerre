package api

import (
	"encoding/json"
	"net/http"
)

func Health(w http.ResponseWriter, r *http.Request) {
	response := HttpResponse{
		Success: true,
		Message: "Api app and running",
	}
	data, _ := json.Marshal(response)

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
