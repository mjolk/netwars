package app

import (
	"appengine"
	"appengine/blobstore"
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"io"
	"io/ioutil"
	"net/http"
)

const READLIMIT = 1048576

type JSONResult struct {
	Success    bool        `json:"-"`
	StatusCode int         `json:"-"`
	Error      string      `json:"error"`
	Result     interface{} `json:"result, omitempty"`
}

func (r *JSONResult) JSONf(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if !r.Success {
		w.WriteHeader(r.StatusCode)
	}
	if err := json.NewEncoder(w).Encode(r); err != nil {
		panic(err)
	}
}

type EngineHandler func(http.ResponseWriter, *http.Request, Context)

type Router struct {
	*httprouter.Router
}

func (r *Router) AppengineHandle(method, path string, handler EngineHandler) {
	r.Handle(method, path,
		func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
			ctx := Context{appengine.NewContext(req), "", p}
			handler(w, req, ctx)
		},
	)
}

func NewRouter() *Router {
	hr := httprouter.New()
	hr.RedirectTrailingSlash = false
	hr.RedirectFixedPath = false
	return &Router{hr}
}

type Context struct {
	appengine.Context
	User   string
	Params httprouter.Params
}

func (c Context) Param(name string) string {
	return c.Params.ByName(name)
}

func DecodeJsonBody(r *http.Request, v interface{}) error {
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, READLIMIT))
	if err != nil {
		return err
	}
	if err := r.Body.Close(); err != nil {
		return err
	}
	if err := json.Unmarshal(body, v); err != nil {
		return err
	}
	return nil
}

func IsNotImage(data *blobstore.BlobInfo) bool {
	for _, tpe := range ImageTypes {
		if data.ContentType == tpe {
			return false
		}
	}
	return true
}

func NoAccess(w http.ResponseWriter) {
	res := JSONResult{Success: false, StatusCode: http.StatusUnauthorized, Error: "No Access"}
	res.JSONf(w)
}

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
