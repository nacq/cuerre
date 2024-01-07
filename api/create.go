package handler

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/nacq/cuerre/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Create(w http.ResponseWriter, r *http.Request) {
	utils.CleanupTmpFiles()

	ds := utils.NewDatastore()
	config := utils.GetConfig()
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
	tmpQRFile, err := utils.GenerateQR(fileId.Hex())

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

		response := utils.HttpResponse{
			Success: false,
			Message: "Error uploadig file",
		}
		data, _ := json.Marshal(response)

		w.WriteHeader(http.StatusInternalServerError)
		w.Write(data)

		return
	}

	log.Printf("QR file stored in gridfs id %s\n", fileId)

	utils.RemoveFile(tmpQRFile)

	response := utils.HttpResponse{
		Success: true,
		Message: "File uploaded successfully",
		Data:    config.APP_URL + "/qr/" + fileId.Hex(),
	}
	data, _ := json.Marshal(response)

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)

	return
}
