package utils

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/yeqown/go-qrcode"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Configuration struct {
	APP_URL string
	DB_URL  string
	MODE    string
	PORT    string
	TMP_EXP int
}

type FileMetadata struct {
	Extension string             `json:"extension"`
	LastRead  primitive.DateTime `json:"lastRead"`
	Type      string             `json:"type"`
}

type File struct {
	Id         primitive.ObjectID `json:"_id"`
	ChunkSize  int                `json:"chunkSize"`
	Filename   string             `json:"filename"`
	Length     int                `json:"length"`
	Metadata   FileMetadata       `json:"metadata"`
	UploadDate primitive.DateTime `json:"uploadDate"`
}

type Datastore struct {
	DB        *mongo.Client
	FileStore *gridfs.Bucket
}

type HttpResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// expressed in hours
const TTL = 2

var prodConfig = &Configuration{
	APP_URL: os.Getenv("CUERRE_APP_URL"),
	DB_URL:  os.Getenv("CUERRE_DB_URL"),
	MODE:    os.Getenv("CUERRE_MODE"),
	PORT:    os.Getenv("CUERRE_PORT"),
}

var defaultConfig = &Configuration{
	APP_URL: "http://localhost:3030",
	DB_URL:  "mongodb://localhost:27017/cuerre",
	MODE:    "development",
	PORT:    "3030",
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

func NewDatastore() *Datastore {
	config := GetConfig()
	mongo := NewMongoClient(config.DB_URL)

	return &Datastore{
		DB:        mongo,
		FileStore: NewGridFsBucket(mongo),
	}
}

func RemoveFile(filePath string) {
	err := os.Remove(filePath)

	if err != nil {
		log.Printf("Error removing tmp file %s %s", filePath, err.Error())
	}
}


// cleanup files from the tmp directory after a given time
func CleanupTmpFiles() {
	ds := NewDatastore()
	var filenames []string
	files, _ := ioutil.ReadDir("/tmp")

	for _, file := range files {
		filenames = append(filenames, file.Name())
	}

	database := ds.DB.Database("cuerre")
	filesColl := database.Collection("fs.files")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	cursor, err := filesColl.Find(
		ctx,
		bson.D{
			bson.E{
				Key: "filename",
				Value: bson.D{{
					Key:   "$in",
					Value: filenames,
				}},
			},
		},
	)

	if err != nil {
		log.Printf("Error finding temp files %s\n", err.Error())
		return
	}

	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var file File

		err = cursor.Decode(&file)

		if err != nil {
			log.Printf("Error decoding temp file document %s\n", err.Error())

			return
		}

		loc, _ := time.LoadLocation("UTC")
		now := time.Now().In(loc)

		if now.Sub(file.Metadata.LastRead.Time()).Hours() > TTL {
			log.Println("Removing /tmp/%s", file.Filename)
			RemoveFile("/tmp/" + file.Filename)
		}
	}
}

func GetConfig() *Configuration {
	if os.Getenv("CUERRE_MODE") == "production" {
		return prodConfig
	}

	return defaultConfig
}

func GenerateQR(id string) (string, error) {
	appConfig := GetConfig()
	config := qrcode.Config{
		EncMode: qrcode.EncModeByte,
		EcLevel: qrcode.ErrorCorrectionQuart,
	}

	qr, err := qrcode.NewWithConfig(
		appConfig.APP_URL+"/file/"+id,
		&config,
		qrcode.WithQRWidth(10),
	)

	if err != nil {
		return "", err
	}

	dest := "/tmp/" + id + "_qr.png"
	err = qr.Save(dest)

	if err != nil {
		return "", err
	}

	return dest, nil
}
