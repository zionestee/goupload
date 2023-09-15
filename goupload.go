package goupload

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
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
	Filename    string
	Size        int64
	ContentType string
	Path        string
}
type UploadParams struct {
	Folder string
	Body   interface{}
}
type MetaFile struct {
	Path      string
	Extension string
}
type Uploader interface {
	UploadFile(UploadParams) ([]MetaFile, error)
	GogoUpload([]byte, *FileGogo) error
	UploadFormFile(interface{}) ([]MetaFile, error)
	UploadFormFiles(interface{}) ([]MetaFile, error)
	UploadFormByte(interface{}) ([]MetaFile, error)
}

func NewUploader(cfg Cfg) Uploader {
	client, err := tus.NewClient(cfg.StorageHost, nil)

	if err != nil {
		fmt.Println(err.Error())
	}
	return uploader{client: client, folder: "", storageUrl: cfg.StorageUrl}
}

func (c uploader) UploadFile(params UploadParams) ([]MetaFile, error) {

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
func (c uploader) UploadFiles(params UploadParams) ([]MetaFile, error) {

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

func (c uploader) UploadFormFile(f interface{}) ([]MetaFile, error) {

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
		Filename:    fileHeader.Filename,
		Size:        fileHeader.Size,
		ContentType: ContentType[0],
	}

	err = c.GogoUpload(buf.Bytes(), &fileGoHeader)
	if err != nil {
		return nil, err
	}
	meta := []MetaFile{}
	m := MetaFile{
		Path:      fileGoHeader.Path,
		Extension: fileGoHeader.ContentType,
	}
	meta = append(meta, m)
	fmt.Printf("%s : upload filesuccess !!\n", fileHeader.Filename)
	return meta, nil
}
func (c uploader) UploadFormFiles(f interface{}) ([]MetaFile, error) {
	files, ok := f.([]*multipart.FileHeader)
	if !ok {
		return nil, errors.New("invalid file format")
	}
	meta := []MetaFile{}
	for _, fileHeader := range files {
		metaFile, err := c.UploadFormFile(fileHeader)
		if err != nil {
			return nil, err
		}
		m := MetaFile{
			Path:      metaFile[0].Path,
			Extension: metaFile[0].Extension,
		}
		meta = append(meta, m)
	}
	return meta, nil
}
func (c uploader) UploadFormByte(f interface{}) ([]MetaFile, error) {
	fileString, ok := f.(string)
	if !ok {
		return nil, errors.New("invalid file format")
	}

	imageDataBase64 := []byte{}

	// urlString := "https://www.example.com/path/to/page"

	// ใช้ ParseRequestURI เพื่อตรวจสอบว่า string เป็น URL หรือไม่
	u, err := url.ParseRequestURI(fileString)
	if err != nil || u.Scheme == "" || u.Host == "" {
		/* เป็น base 64 */
		fmt.Printf("%s ไม่เป็น URL ที่ถูกต้อง\n", fileString)
	} else {
		/* url */
		fmt.Printf("%s เป็น URL ที่ถูกต้อง\n", fileString)
		response, err := http.Get(fileString)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()

		// imageDataBase64 = response.Body

		imageDataBase64, err = io.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}
	}
	marshaled, _ := json.MarshalIndent(imageDataBase64, "", "   ")
	fmt.Println(string(marshaled))

	splitNameBase64 := strings.Split(fileString, "base64,")
	imageDataBase64, _ = base64.StdEncoding.DecodeString(splitNameBase64[1])

	fileGoHeader := FileGogo{}

	c.GogoUpload(imageDataBase64, &fileGoHeader)
	meta := []MetaFile{}
	m := MetaFile{
		Path:      fileGoHeader.Path,
		Extension: fileGoHeader.ContentType,
	}
	meta = append(meta, m)
	fmt.Println("base64 : upload filesuccess !!")
	return meta, nil
	// return nil, nil
}

func (c uploader) GogoUpload(b []byte, fileHeader *FileGogo) error {

	metadata := map[string]string{
		"folder":       c.folder,
		"name":         fileHeader.Filename,
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
