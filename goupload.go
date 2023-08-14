package goupload

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"strings"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/zionestee/goupload/lib"
	"github.com/zionestee/goupload/tus"
)

type uploader struct {
	client  *tus.Client
	client2 lib.Client2
}
type FileGogo struct {
	Filename string
	Size     int64
}
type Uploader interface {
	UploadFile(interface{}) error
	GogoUpload([]byte, *FileGogo) error
	UploadFormFile(interface{}) error
	UploadFormFiles(interface{}) error
	UploadFormByte(interface{}) error
	CreateFileDB(id string, size int64, name string) error
}

func NewUploader() Uploader {
	// client, err := tus.NewClient("http://localhost:8080/files", nil)
	client, err := tus.NewClient("http://localhost:1080/files", nil)
	// client, err := tus.NewClient("http://13.250.149.140:1080/files", nil)
	if err != nil {
		fmt.Println(err.Error())
	}
	return uploader{client}
}

func (c uploader) UploadFile(f interface{}) error {

	switch f.(type) {
	case []*multipart.FileHeader:
		return c.UploadFormFiles(f)

	case *multipart.FileHeader:
		return c.UploadFormFile(f)

	case string:
		return c.UploadFormByte(f)

	default:
		return errors.New("file type not supported")
	}
}

func (c uploader) GogoUpload(b []byte, fileHeader *FileGogo) error {

	metadata := map[string]string{
		"key": "/slip",
	}
	upload := tus.NewUploadFromBytes(b, metadata)
	uploader, err := c.client.CreateUpload(upload)

	if err != nil {
		return err
	}
	err = uploader.Upload()
	if err != nil {
		return err
	}

	// content_type := fileHeader.Header.Values("Content-Type")
	// fmt.Println(fileHeader.Header)
	// fmt.Println(content_type)

	// fmt.Println("3")
	// url := uploader.Url()
	// urlSplit := strings.Split(url, "files/")
	// err = c.CreateFileDB(urlSplit[1], fileHeader.Size, fileHeader.Filename)
	if err != nil {
		return err
	}
	return nil
}
func (c uploader) UploadFormFile(f interface{}) error {

	fileHeader, ok := f.(*multipart.FileHeader)
	if !ok {
		return errors.New("invalid file format")
	}
	f2, _ := fileHeader.Open()
	buf := bytes.NewBuffer(nil)

	_, err := io.Copy(buf, f2)
	if err != nil {
		return err
	}

	fileGoHeader := FileGogo{
		Filename: fileHeader.Filename,
		Size:     fileHeader.Size,
	}

	err = c.GogoUpload(buf.Bytes(), &fileGoHeader)
	if err != nil {
		return err
	}
	fmt.Printf("%s : upload filesuccess !!\n", fileHeader.Filename)
	return nil
}
func (c uploader) UploadFormFiles(f interface{}) error {
	files, ok := f.([]*multipart.FileHeader)
	if !ok {
		return errors.New("invalid file format")
	}
	for _, fileHeader := range files {
		return c.UploadFormFile(fileHeader)
	}
	return nil
}
func (c uploader) UploadFormByte(f interface{}) error {
	file, ok := f.(string)
	if !ok {
		return errors.New("invalid file format")
	}

	splitNameBase64 := strings.Split(file, "base64,")
	imageDataBase64, _ := base64.StdEncoding.DecodeString(splitNameBase64[1])

	c.GogoUpload(imageDataBase64, nil)
	return nil
}
func (c uploader) CreateFileDB(id string, size int64, name string) error {

	type File struct {
		ID          bson.ObjectId `json:"id" bson:"_id,omitempty"`
		Key         string        `json:"key" bson:"key"`
		Name        string        `json:"name" bson:"name"`
		Size        int64         `json:"size" bson:"size"`
		ContentType string        `json:"content_type" bson:"content_type"`
		CreatedAt   time.Time     `json:"created_at" bson:"created_at"`
		UpdatedAt   time.Time     `json:"updated_at" bson:"updated_at"`
	}

	file := File{
		Key:         id,
		Name:        name,
		Size:        size,
		ContentType: name,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	const (
		mongodb    = "mongodb://localhost:27017"
		DBName     = "luzio-upload"
		collection = "files"
	)
	ConnectionDB, err := mgo.Dial(mongodb)
	if err != nil {
		return err
	}
	defer ConnectionDB.Close()

	return ConnectionDB.DB(DBName).C(collection).Insert(file)
}
