package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
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
}

type Datastore struct {
	DB        *mongo.Client
	FileStore *gridfs.Bucket
}

type FileMetadata struct {
	Extension string `json:"extension"`
}

type File struct {
	Id         primitive.ObjectID `json:"_id"`
	ChunkSize  int                `json:"chunkSize"`
	Filename   string             `json:"filename"`
	Length     int                `json:"length"`
	Metadata   FileMetadata       `json:"metadata"`
	UploadDate primitive.DateTime `json:"uploadDate"`
}

type HttpResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

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

	dest := "tmp/" + id + "_qr.png"
	err = qr.Save(dest)

	if err != nil {
		return "", err
	}

	return dest, nil
}

func GetConfig() *Configuration {
	if os.Getenv("CUERRE_MODE") == "production" {
		return prodConfig
	}

	return defaultConfig
}

func NewDatastore() *Datastore {
	config := GetConfig()
	mongo := NewMongoClient(config.DB_URL)

	return &Datastore{
		DB:        mongo,
		FileStore: NewGridFsBucket(mongo),
	}
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

func NewGridFsBucket(client *mongo.Client) *gridfs.Bucket {
	bucket, err := gridfs.NewBucket(
		client.Database("cuerre"),
	)

	if err != nil {
		log.Panic("Error initializing GridFS", err.Error())
	}

	return bucket
}

func NewRouter() *mux.Router {
	r := mux.NewRouter()
	r.Handle("/", http.FileServer(http.Dir("./static")))

	r.HandleFunc("/create", func(w http.ResponseWriter, r *http.Request) {
		ds := NewDatastore()
		config := GetConfig()
		// Parse our multipart form, 10 << 20 specifies a maximum
		// upload of 10 MB files.
		r.ParseMultipartForm(10 << 20)
		// FormFile returns the first file for the given key `myFile`
		// it also returns the FileHeader so we can get the Filename,
		// the Header and the size of the file
		file, header, err := r.FormFile("file")

		if err != nil {
			log.Panic("Error Retrieving the File", err.Error())
		}

		defer file.Close()

		// read file content
		fileBytes, err := ioutil.ReadAll(file)

		if err != nil {
			log.Println(err.Error())
		}

		// get file name and extension to persist in db
		filenameParts := strings.Split(header.Filename, ".")
		ext := filenameParts[len(filenameParts)-1]

		// store file in grid fs with some metadata
		uploadOpts := options.GridFSUpload().SetMetadata(
			bson.D{{
				Key:   "extension",
				Value: ext,
			}, {
				Key:   "originalName",
				Value: header.Filename,
			}, {
				Key:   "type",
				Value: "file",
			}},
		)

		// store file in gridfs
		fileId, err := ds.FileStore.UploadFromStream(
			// generate random name for the new file
			primitive.NewObjectID().Hex()+"."+ext,
			bytes.NewBuffer(fileBytes),
			uploadOpts,
		)

		if err != nil {
			http.Error(
				w,
				"Error uploading to GridFS",
				http.StatusInternalServerError,
			)

			return
		}

		log.Printf("File stored in grid fs id: %s, extension %s\n", fileId, ext)

		// generate file qr file
		tmpQRFile, err := GenerateQR(fileId.Hex())

		log.Printf("QR temp file generated %s\n", tmpQRFile)

		// set qr gridfs document metadata
		uploadOpts = options.GridFSUpload().SetMetadata(bson.D{{
			Key:   "extension",
			Value: "png",
		}, {
			Key:   "type",
			Value: "qr",
		}})

		qrFile, err := os.ReadFile(tmpQRFile)

		if err != nil {
			log.Println("Error reading temp qr file", err.Error())
		}

		// store qr file in gridfs
		fileId, err = ds.FileStore.UploadFromStream(
			// NOTE: extension is hardcoded because the generated qrs are
			// png files.
			primitive.NewObjectID().Hex()+".png",
			bytes.NewBuffer(qrFile),
			uploadOpts,
		)

		if err != nil {
			log.Println("Error uploading QR", err.Error())

			response := HttpResponse{
				Success: false,
				Message: "Error uploadig file",
			}
			data, _ := json.Marshal(response)

			w.WriteHeader(http.StatusInternalServerError)
			w.Write(data)

			return
		}

		log.Printf("QR file stored in gridfs id %s\n", fileId)

		// remove temp qr file
		err = os.Remove(tmpQRFile)

		if err != nil {
			log.Printf("Error removing qr tmp file %s %s", tmpQRFile, err.Error())
		}

		response := HttpResponse{
			Success: true,
			Message: "File uploaded successfully",
			Data:    config.APP_URL + "/qr/" + fileId.Hex(),
		}
		data, _ := json.Marshal(response)

		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(data)

		return
	}).Methods("POST")

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		response := HttpResponse{
			Success: true,
			Message: "Api app and running",
		}
		data, _ := json.Marshal(response)

		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}).Methods("GET")

	r.HandleFunc("/{fileType}/{id}", func(w http.ResponseWriter, r *http.Request) {
		ds := NewDatastore()
		vars := mux.Vars(r)
		id := vars["id"]
		fileType := vars["fileType"]

		_id, err := primitive.ObjectIDFromHex(id)
		database := ds.DB.Database("cuerre")
		files := database.Collection("fs.files")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		defer cancel()

		var file File

		// get gridfs document for the requested file id
		err = files.FindOne(
			ctx,
			bson.D{{
				Key:   "_id",
				Value: _id,
			}, {
				Key:   "metadata.type",
				Value: fileType,
			}},
		).Decode(&file)

		if err != nil {
			log.Println("Error finding file", err.Error())

			response := HttpResponse{
				Success: false,
				Message: "Error uploading file",
			}
			data, _ := json.Marshal(response)

			if err == mongo.ErrNoDocuments {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}

			w.Write(data)

			return
		}

		// read the actual file content
		var buf bytes.Buffer
		dsStream, err := ds.FileStore.DownloadToStreamByName(file.Filename, &buf)

		if err != nil {
			log.Println("Error downloading file", err.Error())

			response := HttpResponse{
				Success: false,
				Message: "Error uploading file",
			}
			data, _ := json.Marshal(response)

			w.WriteHeader(http.StatusInternalServerError)
			w.Write(data)

			return
		}

		log.Printf("File size to download: %v\n", dsStream)

		// TODO: check first if the file still exist in the temporal directory before downloading
		// a new one (this will still work but it will be overriding the existing file everytime)
		// write the content of the requested file to a temp directory
		dest := "tmp/" + file.Filename
		ioutil.WriteFile(dest, buf.Bytes(), 0600)

		// set last read
		_, err = files.UpdateByID(
			ctx,
			_id,
			bson.D{
				bson.E{
					Key: "$set",
					Value: bson.D{{
						Key:   "metadata.lastRead",
						Value: time.Now(),
					}},
				},
			},
		)

		if err != nil {
			log.Println("Error updating qr with last read", err.Error())

			response := HttpResponse{
				Success: false,
				Message: "Error uploading file",
			}
			data, _ := json.Marshal(response)

			w.WriteHeader(http.StatusInternalServerError)
			w.Write(data)

			return
		}

		// serve the temporal file
		http.ServeFile(w, r, "./"+dest)
	}).Methods("GET")

	return r
}

func main() {
	config := GetConfig()
	router := NewRouter()

	log.Println("Up and listening on port", config.PORT)
	log.Fatal(http.ListenAndServe(":"+config.PORT, router))
}
