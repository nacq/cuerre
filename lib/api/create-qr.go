package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/nicolasacquaviva/cuerre/lib"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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

func CreateQR(ds *Datastore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		config := lib.GetConfig()
		// Parse our multipart form, 10 << 20 specifies a maximum
		// upload of 10 MB files.
		r.ParseMultipartForm(10 << 20)
		// FormFile returns the first file for the given key `myFile`
		// it also returns the FileHeader so we can get the Filename,
		// the Header and the size of the file
		file, handler, err := r.FormFile("file")
		if err != nil {
			log.Panic("Error Retrieving the File", err.Error())
		}
		defer file.Close()

		fileBytes, err := ioutil.ReadAll(file)
		if err != nil {
			log.Println(err.Error())
		}

		filenameParts := strings.Split(handler.Filename, ".")
		ext := filenameParts[len(filenameParts)-1]

		// store file in grid fs
		uploadOpts := options.GridFSUpload().SetMetadata(bson.D{{
			Key:   "extension",
			Value: ext,
		}})
		fileId, err := ds.FileStore.UploadFromStream(
			handler.Filename,
			bytes.NewBuffer(fileBytes),
			uploadOpts,
		)

		if err != nil {
			http.Error(
				w,
				"Error uploading to GridFS",
				http.StatusInternalServerError,
			)
		} else {
			var response HttpResponse
			id := fileId.Hex()
			_id, err := primitive.ObjectIDFromHex(id)
			database := ds.DB.Database("cuerre")
			files := database.Collection("fs.files")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			var file File
			err = files.FindOne(ctx, bson.D{{Key: "_id", Value: _id}}).Decode(&file)

			if err != nil {
				log.Println("error finding file", err.Error())

				response.Success = false
				response.Message = err.Error()
				data, _ := json.Marshal(response)

				if err == mongo.ErrNoDocuments {
					w.WriteHeader(http.StatusNotFound)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
				}

				w.Write(data)
				return
			}

			var buf bytes.Buffer
			dsStream, err := ds.FileStore.DownloadToStreamByName(file.Filename, &buf)

			if err != nil {
				log.Println("error downloading file", err.Error())

				response.Success = false
				response.Message = err.Error()
				data, _ := json.Marshal(response)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write(data)
			}

			log.Printf("File size to download: %v\n", dsStream)

			dest := "tmp/" + id + "." + file.Metadata.Extension
			ioutil.WriteFile(dest, buf.Bytes(), 0600)

			_, err = lib.GenerateQR(id, id+"."+file.Metadata.Extension)

			if err != nil {
				log.Println("error generating QR", err.Error())

				response.Success = false
				response.Message = err.Error()
				data, _ := json.Marshal(response)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write(data)
			}
			response.Success = true
			response.Message = "File uploaded successfully"
			response.Data = config.APP_URL + "/qr/" + fileId.Hex()

			data, _ := json.Marshal(response)

			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(data)
		}
	}
}
