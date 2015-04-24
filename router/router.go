package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mj0lk.be/netwars/app"
	"mj0lk.be/netwars/secure"
	"strings"
)

const (
	DevDomain = "localhost:8080"
	Domain    = "n3twars.appspot.com"
	KeyStr    = "agtkZXZ-bjN0d..."
)

type Routes []Route

func New() *app.Router {
	r := app.NewRouter()
	r.AppengineHandle("GET", "/discover/", Discover)
	for _, route := range API {
		for _, path := range route.Path {
			handler := route.Handler
			if route.Auth {
				handler = secure.Validator(route.Handler)
			}
			r.AppengineHandle(route.Method, path, handler)
		}
	}
	return r
}

type Route struct {
	Description string
	Path        []string
	Method      string
	Handler     app.EngineHandler
	Request     interface{}
	Response    interface{}
	Auth        bool
}

func routeVars(path string) string {
	pces := strings.Split(path, "/")
	for k, pce := range pces {
		if strings.Contains(pce, "_") {
			prts := strings.Split(pce, "_")
			varType := prts[1]
			switch varType {
			case "id":
				pces[k] = "6"
			case "bool":
				pces[k] = "0"
			case "key":
				pces[k] = KeyStr
			}
		}
	}
	return strings.Join(pces, "/")
}

func (r Route) Urls() (urls []string) {
	for _, path := range r.Path {
		fullUrl := fmt.Sprintf("https://%s%s", Domain, routeVars(path))
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
