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

	"github.com/eventials/go-tus"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

type uploader struct {
	client *tus.Client
}
type Uploader interface {
	UploadFile(f interface{}) error
	GogoUpload(b []byte) error
	UploadFormFile(f interface{}) error
	UploadFormFiles(f interface{}) error
	UploadFormByte(f interface{}) error
	CreateFileDB(id string, size int64) error
}

func NewUploader() Uploader {
	client, err := tus.NewClient("http://localhost:1080/files", nil)
	if err != nil {
		fmt.Println(err.Error())
	}

	return uploader{client}
}

func (c uploader) UploadFile(f interface{}) error {

	switch f.(type) {
	case []*multipart.FileHeader:
		c.UploadFormFiles(f)

	case *multipart.FileHeader:
		c.UploadFormFile(f)

	case string:
		c.UploadFormByte(f)

	default:
		return errors.New("file type not supported")
	}
	return nil
}

func (c uploader) GogoUpload(b []byte) error {

	upload := tus.NewUploadFromBytes(b)
	uploader, err := c.client.CreateUpload(upload)
	if err != nil {
		return err
	}

	fmt.Println(uploader)

	err = uploader.Upload()
	if err != nil {
		return err
	}
	c.CreateFileDB("909290390203", 320)
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

	c.GogoUpload(buf.Bytes())
	fmt.Printf("%s : upload filesuccess 123 !!\n", fileHeader.Filename)
	return nil
}
func (c uploader) UploadFormFiles(f interface{}) error {
	files, ok := f.([]*multipart.FileHeader)
	if !ok {
		return errors.New("invalid file format")
	}
	for _, fileHeader := range files {
		c.UploadFormFile(fileHeader)
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

	c.GogoUpload(imageDataBase64)
	return nil
}
func (c uploader) CreateFileDB(id string, size int64) error {

	type File struct {
		ID        bson.ObjectId `json:"id" bson:"_id,omitempty"`
		Key       string        `json:"key" bson:"key"`
		Name      string        `json:"name" bson:"name"`
		Size      int64         `json:"size" bson:"size"`
		CreatedAt time.Time     `json:"created_at" bson:"created_at"`
		UpdatedAt time.Time     `json:"updated_at" bson:"updated_at"`
	}
	fmt.Println(">>>>>>>>>>>>>>>>>", id)
	fmt.Println(">>>>>>>>>>>>>>>>>", size)

	file := File{
		Key:       id,
		Name:      "",
		Size:      size,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
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
