package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/nacq/cuerre/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func Get(w http.ResponseWriter, r *http.Request) {
	utils.CleanupTmpFiles()

	ds := utils.NewDatastore()
	path := strings.Split(r.URL.Path, "/")

	if len(path) < 3 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	fileType := path[1]
	id := path[2]

	log.Printf("Getting file of type %s and id %s\n", fileType, id)

	_id, err := primitive.ObjectIDFromHex(id)
	database := ds.DB.Database("cuerre")
	files := database.Collection("fs.files")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	var file utils.File

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

		response := utils.HttpResponse{
			Success: false,
			Message: err.Error(),
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

		response := utils.HttpResponse{
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
	dest := "/tmp/" + file.Filename
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

		response := utils.HttpResponse{
			Success: false,
			Message: err.Error(),
		}
		data, _ := json.Marshal(response)

		w.WriteHeader(http.StatusInternalServerError)
		w.Write(data)

		return
	}

	// serve the temporal file
	http.ServeFile(w, r, dest)
}
