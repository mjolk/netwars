package router

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/mux"
	"mj0lk.be/netwars/utils"
	"net/http"
	"strings"
)

const (
	DevDomain = "localhost:8080"
	Domain    = "n3twars.appspot.com"
	KeyStr    = "agtkZXZ-bjN0d..."
)

type Router interface {
	Urls() []string
	ResponseJSON() []byte
	RequestJSON() []byte
}

type Route struct {
	Description string
	Name        []string
	Path        []string
	Method      string
	Headers     []string
	Handler     http.HandlerFunc
	Request     interface{}
	Response    interface{}
	Auth        bool
}

func routeVars(name string) []string {
	varNames := strings.Split(name, ".")
	varNames = varNames[1:]
	getVars := make([]string, len(varNames)*2, len(varNames)*2)
	cntr := 0
	for _, varName := range varNames {
		getVars[cntr] = varName
		cntr++
		if strings.Contains(varName, "id") {
			getVars[cntr] = "2343"
		} else if strings.Contains(varName, "bool") {
			getVars[cntr] = "true"
		} else {
			getVars[cntr] = KeyStr
		}
		cntr++
	}
	return getVars
}

func (r Route) Urls(router *mux.Router) (urls []string) {
	for _, name := range r.Name {
		var rVars []string
		if strings.Contains(name, ".") {
			rVars = routeVars(name)
		}
		url, err := router.Get(name).URL(rVars...)
		if err != nil {
			panic(err)
		}
		fullUrl := "https://n3twars.appspot.com" + url.String()
		urls = append(urls, fullUrl)
	}
	return
}

func prettyJson(v interface{}) []byte {
	var bf bytes.Buffer
	res, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	json.HTMLEscape(&bf, res)
	return bf.Bytes()
}

func (r Route) RequestJSON() []byte {
	if r.Request != nil {
		return prettyJson(r.Request)
	}
	return []byte{}
}

func (r Route) ResponseJSON() []byte {
	if r.Response != nil {
		return prettyJson(r.Response)
	}
	return []byte("{http 200 ok}")
}

type Routes []Route

var jsonheader = []string{"Accept", "application/json; charset=UTF-8"}

func NewRouter() *mux.Router {
	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/discover", Discover).Methods("GET")
	for prefix, routes := range API {
		//check multiple paths (optional variables require multiple paths (routes) to be
		//registered because REST url vars are not optional(in gorillatoolkit))
		subRouter := r.PathPrefix(prefix).Subrouter()
		for _, route := range routes {
			for k, path := range route.Path {
				handler := route.Handler
				if route.Auth {
					handler = utils.Validator(route.Handler)
				}
				subRouter.HandleFunc(path, handler).
					Methods(route.Method).
					Name(route.Name[k])
				if len(route.Headers) > 0 {
					subRouter.Headers(route.Headers...)
				}
			}
		}
	}
	return r
}
