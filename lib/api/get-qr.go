package api

import (
	"net/http"

	"github.com/gorilla/mux"
)

func GetQR(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	http.ServeFile(w, r, "./tmp/"+id+"_qr.png")
}
