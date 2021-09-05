package api

import (
	"github.com/nicolasacquaviva/cuerre/lib"
	"github.com/nicolasacquaviva/cuerre/lib/db"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
)

type Datastore struct {
	DB        *mongo.Client
	FileStore *gridfs.Bucket
}

func NewDatastore() *Datastore {
	config := lib.GetConfig()
	mongo := db.NewMongoClient(config.DB_URL)

	return &Datastore{
		DB:        mongo,
		FileStore: db.NewGridFsBucket(mongo),
	}
}
