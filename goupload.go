package goupload

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/zionestee/goupload/tus"
)

type uploader struct {
	client     *tus.Client
	folder     string
	storageUrl string
}
type Cfg struct {
	StorageHost string
	StorageUrl  string
}
type FileGogo struct {
	FileName    string
	Size        int64
	ContentType string
	Path        string
}
type UploadParams struct {
	Folder string
	Body   interface{}
}
type Uploader interface {
	UploadFile(UploadParams) ([]FileGogo, error)
	UploadFormFile(interface{}) ([]FileGogo, error)
	UploadFormFiles(interface{}) ([]FileGogo, error)
	UploadFormByte(interface{}) ([]FileGogo, error)
	GogoUpload([]byte, *FileGogo) error
}

func NewUploader(cfg Cfg) Uploader {
	client, err := tus.NewClient(cfg.StorageHost, nil)

	if err != nil {
		fmt.Println(err.Error())
	}
	return uploader{client: client, folder: "", storageUrl: cfg.StorageUrl}
}

func (c uploader) UploadFile(params UploadParams) ([]FileGogo, error) {

	f := params.Body
	c.folder = params.Folder
	switch f.(type) {
	case []*multipart.FileHeader:
		return c.UploadFormFiles(f)

	case *multipart.FileHeader:
		return c.UploadFormFile(f)

	case string:
		return c.UploadFormByte(f)

	default:
		fmt.Printf(":%t", f)
		return nil, errors.New("file type not supported")
	}
}
func (c uploader) UploadFiles(params UploadParams) ([]FileGogo, error) {

	f := params.Body
	c.folder = params.Folder
	switch f.(type) {
	case []*multipart.FileHeader:
		return c.UploadFormFiles(f)

	case *multipart.FileHeader:
		return c.UploadFormFile(f)

	case string:
		return c.UploadFormByte(f)

	default:
		fmt.Printf(":%t", f)
		return nil, errors.New("file type not supported")
	}
}

func (c uploader) UploadFormFile(f interface{}) ([]FileGogo, error) {

	fileHeader, ok := f.(*multipart.FileHeader)
	if !ok {
		return nil, errors.New("invalid file format")
	}
	f2, _ := fileHeader.Open()
	buf := bytes.NewBuffer(nil)

	_, err := io.Copy(buf, f2)
	if err != nil {
		return nil, err
	}

	ContentType := fileHeader.Header.Values("Content-Type")
	fileGoHeader := FileGogo{
		FileName:    fileHeader.Filename,
		Size:        fileHeader.Size,
		ContentType: ContentType[0],
	}

	err = c.GogoUpload(buf.Bytes(), &fileGoHeader)
	if err != nil {
		return nil, err
	}
	meta := []FileGogo{}
	meta = append(meta, fileGoHeader)
	fmt.Printf("%s : upload filesuccess !!\n", fileHeader.Filename)
	return meta, nil
}
func (c uploader) UploadFormFiles(f interface{}) ([]FileGogo, error) {
	files, ok := f.([]*multipart.FileHeader)
	if !ok {
		return nil, errors.New("invalid file format")
	}
	meta := []FileGogo{}
	for _, fileHeader := range files {
		metaFile, err := c.UploadFormFile(fileHeader)
		if err != nil {
			return nil, err
		}
		meta = append(meta, metaFile[0])
	}
	return meta, nil
}
func (c uploader) UploadFormByte(f interface{}) ([]FileGogo, error) {
	fileString, ok := f.(string)
	if !ok {
		return nil, errors.New("invalid file format")
	}

	var imageDataBase64 = []byte{}
	var fileName = ""
	u, err := url.ParseRequestURI(fileString)
	if err != nil || u.Scheme == "" || u.Host == "" {
		/* เป็น base 64 */
		splitNameBase64 := strings.Split(fileString, "base64,")
		imageDataBase64, _ = base64.StdEncoding.DecodeString(splitNameBase64[1])
		if err != nil {
			fmt.Println("ไม่สามารถถอดรหัส Base64 ได้:", err)
			return nil, err
		}
	} else {
		/* url */
		response, err := http.Get(fileString)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			fmt.Printf("ไม่สามารถดาวน์โหลดรูปภาพ สถานะ: %s\n", response.Status)
			return nil, err
		}

		fileName = getFileNameFromURL(fileString)
		imageDataBase64, err = io.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}
	}

	contentType := http.DetectContentType(imageDataBase64)
	fileGoHeader := FileGogo{
		FileName:    fileName,
		Size:        int64(len(imageDataBase64)),
		ContentType: contentType,
	}
	c.GogoUpload(imageDataBase64, &fileGoHeader)
	meta := []FileGogo{}
	meta = append(meta, fileGoHeader)
	fmt.Println("base64 : upload filesuccess !!")
	return meta, nil
}

func (c uploader) GogoUpload(b []byte, fileHeader *FileGogo) error {

	metadata := map[string]string{
		"folder":       c.folder,
		"name":         fileHeader.FileName,
		"content-type": fileHeader.ContentType,
	}
	upload := tus.NewUploadFromBytes(b, metadata)
	uploader, err := c.client.CreateUpload(upload)

	key := strings.Split(uploader.Url(), "/files/")[1]
	fileHeader.Path = c.storageUrl + key

	if err != nil {
		return err
	}
	err = uploader.Upload()
	if err != nil {
		return err
	}
	return nil
}
func getFileNameFromURL(url string) string {
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}
