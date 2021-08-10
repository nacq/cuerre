package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/nicolasacquaviva/cuerre/lib"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type HttpResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

const (
	user = "djmibvor"
	password = "wXXFD7t6lb986c3t1JTkm1X1m4uMgs3x"
	dbName = "cuerre"
)

func main() {
	dbConnString := fmt.Sprintf(
		"mongodb+srv://%s:%s@cluster0.bqqes.mongodb.net/%s?retryWrites=true&w=majority",
		user,
		password,
		dbName,
	)
	db := lib.NewMongoClient(dbConnString)
	gridfs := lib.NewGridFsBucket(db)

	log.Println("Successfully connected to the database")

	fs := http.FileServer(http.Dir("./static"))

	r := mux.NewRouter()

	r.Handle("/", fs)

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

	r.HandleFunc("/create", func(w http.ResponseWriter, r *http.Request) {
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

		uploadOpts := options.GridFSUpload()
			// TODO: what metadata is needed?
			// .SetMetadata(bson.D{{ "filename", handler.Filename }})
		fileId, err := gridfs.UploadFromStream(
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
			response := HttpResponse{
				Success: true,
				Message: "File uploaded successfully",
				Data: fileId,
			}
			data, _ := json.Marshal(response)

			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(data)
		}
	}).Methods("POST")

	r.HandleFunc("/assets/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		response := HttpResponse{}

		filesCol := db.Database("cuerre").Collection("fs.chunks")
		var results bson.M
		objectId, err := primitive.ObjectIDFromHex(id)

		if err != nil {
			response.Success = false
			response.Message = err.Error()
			w.WriteHeader(http.StatusNotFound)
			data, _ := json.Marshal(response)

			w.Header().Set("content-type", "application/json")
			w.Write(data)
		}

		err = filesCol.FindOne(
			context.Background(),
			bson.M{ "files_id": objectId },
		).Decode(&results)

		if err != nil {
			response.Success = false
			response.Message = err.Error()
			w.WriteHeader(http.StatusNotFound)
		} else {
			response.Success = true
			response.Data = results

			log.Println(results)
		}

		data, _ := json.Marshal(response)

		w.Header().Set("content-type", "application/json")
		w.Write(data)

	}).Methods("GET")

	log.Fatal(http.ListenAndServe(":3000", r))
}
