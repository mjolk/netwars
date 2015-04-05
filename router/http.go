package router

import (
	"appengine"
	"html/template"
	"net/http"
)

type Discovery struct {
	Description string
	Prefix      string
	Urls        []string
	Method      string
	Request     template.JS
	Response    template.JS
	Auth        bool
}

func Discover(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	rt := NewRouter()
	discoveries := make([]Discovery, 0)
	for key, routes := range API {
		for _, route := range routes {
			disco := Discovery{}
			disco.Description = route.Description
			disco.Prefix = key
			disco.Urls = route.Urls(rt)
			disco.Method = route.Method
			disco.Request = template.JS(route.RequestJSON())
			disco.Response = template.JS(route.ResponseJSON())
			disco.Auth = route.Auth
			//c.Debugf("discovery: %v", disco)
			discoveries = append(discoveries, disco)
		}
	}
	t := template.Must(template.ParseFiles("html_templates/index.tmpl"))
	err := t.Execute(w, discoveries)
	if err != nil {
		c.Errorf("template execution: %s", err)
	}
}
