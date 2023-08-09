package goupload

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"strings"

	"github.com/eventials/go-tus"
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

	uploader.Upload()
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
	fmt.Printf("%s : upload filesuccess !!\n", fileHeader.Filename)
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
