package db

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func NewMongoClient(uri string) *mongo.Client {
	clientOptions := options.Client().ApplyURI(uri).SetMaxPoolSize(5)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)

	if err != nil {
		log.Panic("Error connecting to mongo", err.Error())
	}

	return client
}

func NewGridFsBucket(client *mongo.Client) *gridfs.Bucket {
	bucket, err := gridfs.NewBucket(
		client.Database("cuerre"),
	)

	if err != nil {
		log.Panic("Error initializing GridFS", err.Error())
	}

	return bucket
}
