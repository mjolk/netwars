package router

import (
	"appengine"
	"bytes"
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"strings"
)

const (
	DevDomain = "localhost:8080"
	Domain    = "n3twars.appspot.com"
	KeyStr    = "agtkZXZ-bjN0d..."
)

var jsonheader = []string{"Accept", "application/json; charset=UTF-8"}

type Routes []Route

type AppengineHandler func(http.ResponseWriter, *http.Request, Context)

type Context struct {
	appengine.Context
	User   string
	Params httprouter.Params
}

func NewContext(r *http.Request) Context {
	return Context{appengine.NewContext(r)}
}

type Router struct {
	*httprouter.Router
}

func (r *Router) AppengineHandle(method, path string, handler AppengineHandler) {
	r.Handle(method, path,
		func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
			ctx := NewContext(req)
			ctx.Params = p
			handler(w, req, ctx)
		},
	)
}

func New() *Router {
	hr := httprouter.New()
	hr.RedirectTrailingSlash = false
	hr.RedirectFixedPath = false
	r := &Router{hr}
	r.AppengineHandle("GET", "/discover/", Discover)
	for prefix, routes := range API {
		for _, route := range routes {
			for k, path := range route.Path {
				handler := route.Handler
				if route.Auth {
					handler = Validator(route.Handler)
				}
				r.AppengineHandle(route.Method, prefix+path, route.Handler)
			}
		}
	}
	return r
}

type Discover interface {
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
			getVars[cntr] = "0"
		} else {
			getVars[cntr] = KeyStr
		}
		cntr++
	}
	return getVars
}

func (r Route) Urls() (urls []string) {
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
