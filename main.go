package main

import (
	"log"
	"net/http"

	"github.com/nicolasacquaviva/cuerre/lib"
	"github.com/nicolasacquaviva/cuerre/lib/api"
)

func main() {
	config := lib.GetConfig()
	datastore := api.NewDatastore()

	router := datastore.NewRouter()
	log.Println("Up and listening on port", config.PORT)
	log.Fatal(http.ListenAndServe(":"+config.PORT, router))
}
