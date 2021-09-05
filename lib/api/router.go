package api

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (env *Datastore) NewRouter() *mux.Router {
	r := mux.NewRouter()
	r.Handle("/", http.FileServer(http.Dir("./static")))
	r.HandleFunc("/create", CreateQR(env)).Methods("POST")
	r.HandleFunc("/health", Health).Methods("GET")
	r.HandleFunc("/qr/{id}", GetQR).Methods("GET")
	r.HandleFunc("/content/{filename}", GetContent).Methods("GET")
	return r
}
