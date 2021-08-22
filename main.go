package main

import (
	"log"
	"net/http"

	"github.com/nicolasacquaviva/cuerre/lib"
	"github.com/nicolasacquaviva/cuerre/lib/db"
	"github.com/nicolasacquaviva/cuerre/lib/api"
)

func main() {
	config := lib.GetConfig()
	mongo := db.NewMongoClient(config.DB_URL)

	env := &api.Env{
		DB: mongo,
		FileStore: db.NewGridFsBucket(mongo),
	}

	router := env.NewRouter()
	log.Println("Up and listening on port", config.PORT)
	log.Fatal(http.ListenAndServe(":" + config.PORT, router))
}
