package api

import (
	"net/http"

	"github.com/gorilla/mux"
)

func GetContent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	filename := vars["filename"]

	http.ServeFile(w, r, "./tmp/"+filename)
}
