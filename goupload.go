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
	EndPoint        string
	StorageUrl      string
	SecretAccessKey string
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
type DeleteParams struct {
	Key []string
}
type jsonResponse struct {
	Error string `json:"error,omitempty"`
	Data  any    `json:"data,omitempty"`
}

type Uploader interface {
	Upload(UploadParams) ([]FileGogo, error)
	DeleteObjects(DeleteParams) (interface{}, error)
	UploadFormFile(interface{}) ([]FileGogo, error)
	UploadFormFiles(interface{}) ([]FileGogo, error)
	UploadFormByte(interface{}) ([]FileGogo, error)
	GogoUpload([]byte, *FileGogo) error
}

func NewUploader(cfg Cfg) Uploader {
	client, err := tus.NewClient(cfg.EndPoint, nil)

	if err != nil {
		fmt.Println(err.Error())
	}
	return uploader{client: client, folder: "", storageUrl: cfg.StorageUrl}
}

func (c uploader) Upload(params UploadParams) ([]FileGogo, error) {

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
		base64String := ""
		if len(splitNameBase64) > 1 {
			base64String = splitNameBase64[1]
		} else {
			base64String = splitNameBase64[0]
		}
		imageDataBase64, err = base64.StdEncoding.DecodeString(base64String)
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
func (c uploader) DeleteObjects(params DeleteParams) (interface{}, error) {

	jsonBody, _ := json.Marshal(params)
	request, err := http.NewRequest("DELETE", c.storageUrl, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	b_byte, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	responseBody := jsonResponse{}
	err = json.Unmarshal(b_byte, &responseBody)

	marshaled2, _ := json.MarshalIndent(responseBody, "", "   ")
	fmt.Println(string(marshaled2))

	if err != nil {
		return nil, err
	}

	return responseBody.Data, nil
}
