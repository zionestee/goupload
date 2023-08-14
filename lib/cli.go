package lib

import "net/http"

type Client2 struct {
	Url     string
	Version string
	Header  http.Header

	client *http.Client
}
