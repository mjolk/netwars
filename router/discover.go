package router

import (
	"html/template"
	"mj0lk.be/netwars/app"
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

func Discover(w http.ResponseWriter, r *http.Request, c app.Context) {
	discoveries := make([]Discovery, 0)
	for _, route := range API {
		disco := Discovery{}
		disco.Description = route.Description
		disco.Urls = route.Urls()
		disco.Method = route.Method
		disco.Request = template.JS(route.RequestJSON())
		disco.Response = template.JS(route.ResponseJSON())
		disco.Auth = route.Auth
		discoveries = append(discoveries, disco)
	}
	t := template.Must(template.ParseFiles("html_templates/index.tmpl"))
	err := t.Execute(w, discoveries)
	if err != nil {
		c.Errorf("template execution: %s", err)
	}
}
