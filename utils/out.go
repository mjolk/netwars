package utils

import (
	"appengine/blobstore"
	"encoding/json"
	"net/http"
)

var (
	ImageTypes = []string{
		"image/bmp",
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/tiff",
		"image/x-icon",
	}
)

type JSONResult struct {
	Success bool        `json:"-"`
	Error   string      `json:"error"`
	Result  interface{} `json:"result, omitempty"`
}

func (r *JSONResult) JSONf(w http.ResponseWriter) {
	json, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-type", "application/json")
	if !r.Success {
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.Write(json)
}

func IsNotImage(data *blobstore.BlobInfo) bool {
	for _, tpe := range ImageTypes {
		if data.ContentType == tpe {
			return false
		}
	}
	return true
}
