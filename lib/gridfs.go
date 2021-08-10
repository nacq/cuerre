package lib

import (
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
)

func NewGridFsBucket(client *mongo.Client) *gridfs.Bucket {
	bucket, err := gridfs.NewBucket(
		client.Database("cuerre"),
	)

	if err != nil {
		log.Panic("Error initializing GridFS", err.Error())
	}

	return bucket
}
