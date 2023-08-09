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
	uploadFile(f interface{}) error
	goUpload(b []byte) error
	uploadFormFile(f interface{}) error
	uploadFormFiles(f interface{}) error
	uploadFormByte(f interface{}) error
}

func NewUploader() Uploader {
	client, err := tus.NewClient("http://localhost:1080/files", nil)
	if err != nil {
		fmt.Println(err.Error())
	}
	return uploader{client}
}

func (c uploader) uploadFile(f interface{}) error {

	switch f.(type) {
	case []*multipart.FileHeader:
		c.uploadFile(f)

	case *multipart.FileHeader:
		c.uploadFormFile(f)

	case string:
		c.uploadFormByte(f)

	default:
		return errors.New("file type not supported")
	}
	return nil
}
func (c uploader) goUpload(b []byte) error {

	upload := tus.NewUploadFromBytes(b)
	uploader, err := c.client.CreateUpload(upload)
	if err != nil {
		return err
	}

	uploader.Upload()
	return nil
}
func (c uploader) uploadFormFile(f interface{}) error {

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

	c.goUpload(buf.Bytes())
	fmt.Printf("%s : upload filesuccess !!\n", fileHeader.Filename)
	return nil
}
func (c uploader) uploadFormFiles(f interface{}) error {
	files, ok := f.([]*multipart.FileHeader)
	if !ok {
		return errors.New("invalid file format")
	}
	for _, fileHeader := range files {
		c.uploadFormFile(fileHeader)
	}
	return nil
}
func (c uploader) uploadFormByte(f interface{}) error {
	file, ok := f.(string)
	if !ok {
		return errors.New("invalid file format")
	}

	splitNameBase64 := strings.Split(file, "base64,")
	imageDataBase64, _ := base64.StdEncoding.DecodeString(splitNameBase64[1])

	c.goUpload(imageDataBase64)
	return nil
}
